// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync/atomic"

	"golang.org/x/crypto/sha3"
)

// nodeDebugLogging controls diagnostic log output for the node-side nogopow types.
// Default off; enabled via NOGO_NODE_DEBUG=1.
var nodeDebugLogging atomic.Bool

func init() {
	if os.Getenv("NOGO_NODE_DEBUG") == "1" {
		nodeDebugLogging.Store(true)
	}
}

// Address represents a 20-byte NogoChain address
type Address [20]byte

// Bytes returns address as byte slice
func (a Address) Bytes() []byte { return a[:] }

// Hash represents a 32-byte hash
type Hash [32]byte

// Bytes returns hash as byte slice
func (h Hash) Bytes() []byte { return h[:] }

// Hex returns hex string representation
func (h Hash) Hex() string {
	return fmt.Sprintf("%x", h[:])
}

// BlockNonce represents a 32-byte nonce for mining
type BlockNonce [32]byte

// Header represents a block header
type Header struct {
	ParentHash Hash
	Coinbase   Address
	Root       Hash
	TxHash     Hash
	Number     *big.Int
	GasLimit   uint64
	Time       uint64
	Extra      []byte
	Nonce      BlockNonce
	Difficulty *big.Int
}

// Hash computes the hash of the header
func (h *Header) Hash() Hash {
	// ── DIAGNOSTIC: Log every header field before computing hash ──
	if nodeDebugLogging.Load() {
		log.Printf("[SealHash-DIAG-NODE] === Node SealHash Input ===")
		log.Printf("[SealHash-DIAG-NODE] ParentHash:   %x", h.ParentHash)
		log.Printf("[SealHash-DIAG-NODE] Coinbase:     %x", h.Coinbase)
		log.Printf("[SealHash-DIAG-NODE] Root(StateR):  %x", h.Root)
		log.Printf("[SealHash-DIAG-NODE] TxHash(Merkle):%x", h.TxHash)
		if h.Number != nil {
			log.Printf("[SealHash-DIAG-NODE] Number(Height):%d (bytes=%x)", h.Number.Uint64(), h.Number.Bytes())
		} else {
			log.Printf("[SealHash-DIAG-NODE] Number(Height): nil")
		}
		log.Printf("[SealHash-DIAG-NODE] GasLimit:     %d (0x%x)", h.GasLimit, h.GasLimit)
		log.Printf("[SealHash-DIAG-NODE] Time:         %d (0x%x)", h.Time, h.Time)
		log.Printf("[SealHash-DIAG-NODE] Extra:        %x (len=%d)", h.Extra, len(h.Extra))
		log.Printf("[SealHash-DIAG-NODE] Nonce:        %x", h.Nonce)
		if h.Difficulty != nil {
			log.Printf("[SealHash-DIAG-NODE] Difficulty:   %s (bytes=%x)", h.Difficulty.String(), h.Difficulty.Bytes())
		} else {
			log.Printf("[SealHash-DIAG-NODE] Difficulty:   nil")
		}
	}

	hasher := sha3.NewLegacyKeccak256()
	rlpEncode(hasher, h)
	result := BytesToHash(hasher.Sum(nil))
	if nodeDebugLogging.Load() {
		log.Printf("[SealHash-DIAG-NODE] === Node SealHash OUTPUT: %x ===", result[:])
	}
	return result
}

// Block represents a complete block
type Block struct {
	header       *Header
	transactions []*Transaction
	uncles       []*Header
}

// Header returns the block header
func (b *Block) Header() *Header { return b.header }

// Transactions returns the block transactions
func (b *Block) Transactions() []*Transaction { return b.transactions }

// Uncles returns the block uncles
func (b *Block) Uncles() []*Header { return b.uncles }

// Number returns the block number
func (b *Block) Number() *big.Int { return b.header.Number }

// NewBlock creates a new block
func NewBlock(header *Header, txs []*Transaction, uncles []*Header, receipts []*Receipt) *Block {
	return &Block{
		header:       header,
		transactions: txs,
		uncles:       uncles,
	}
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Type       TransactionType
	ChainID    uint64
	FromPubKey []byte
	ToAddress  string
	Amount     uint64
	Fee        uint64
	Nonce      uint64
	Data       string
	Signature  []byte
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TxCoinbase TransactionType = "coinbase"
	TxTransfer TransactionType = "transfer"
)

// Receipt represents a transaction receipt
type Receipt struct {
	Status uint64
}

// ChainHeaderReader defines the interface for header access
type ChainHeaderReader interface {
	GetHeaderByHash(hash Hash) *Header
}

// ChainReader defines the interface for chain access
type ChainReader interface {
	ChainHeaderReader
}

// StateDB defines the interface for state access
type StateDB interface {
	AddBalance(addr Address, amount *big.Int)
	IntermediateRoot(v bool) Hash
}

// rlpEncode encodes header fields sequentially
func rlpEncode(w interface{}, v interface{}) {
	// For Header: encode all fields sequentially
	// Uses custom RLP-like encoding for block headers

	header, ok := v.(*Header)
	if !ok {
		return
	}

	// Write all header fields to the writer
	writer, ok := w.(interface{ Write([]byte) (int, error) })
	if !ok {
		return
	}

	// Encode each field
	if nodeDebugLogging.Load() {
		log.Printf("[rlpEncode-DIAG-NODE] 1.ParentHash(%d): %x", len(header.ParentHash.Bytes()), header.ParentHash.Bytes())
	}
	writer.Write(header.ParentHash.Bytes())
	if nodeDebugLogging.Load() {
		log.Printf("[rlpEncode-DIAG-NODE] 2.Coinbase(%d):   %x", len(header.Coinbase.Bytes()), header.Coinbase.Bytes())
	}
	writer.Write(header.Coinbase.Bytes())
	if nodeDebugLogging.Load() {
		log.Printf("[rlpEncode-DIAG-NODE] 3.Root(%d):       %x", len(header.Root.Bytes()), header.Root.Bytes())
	}
	writer.Write(header.Root.Bytes())
	if nodeDebugLogging.Load() {
		log.Printf("[rlpEncode-DIAG-NODE] 4.TxHash(%d):     %x", len(header.TxHash.Bytes()), header.TxHash.Bytes())
	}
	writer.Write(header.TxHash.Bytes())

	// Number as big.Int bytes
	if header.Number != nil {
		numBytes := header.Number.Bytes()
		if nodeDebugLogging.Load() {
			log.Printf("[rlpEncode-DIAG-NODE] 5.Number(%d):      %x", len(numBytes), numBytes)
		}
		writer.Write(numBytes)
	} else {
		if nodeDebugLogging.Load() {
			log.Printf("[rlpEncode-DIAG-NODE] 5.Number:          nil")
		}
	}

	// GasLimit as 8 bytes
	gasBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasBytes, header.GasLimit)
	if nodeDebugLogging.Load() {
		log.Printf("[rlpEncode-DIAG-NODE] 6.GasLimit(%d):    %x", len(gasBytes), gasBytes)
	}
	writer.Write(gasBytes)

	// Time as 8 bytes
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, header.Time)
	if nodeDebugLogging.Load() {
		log.Printf("[rlpEncode-DIAG-NODE] 7.Time(%d):        %x", len(timeBytes), timeBytes)
	}
	writer.Write(timeBytes)

	// Extra data
	if len(header.Extra) > 0 {
		if nodeDebugLogging.Load() {
			log.Printf("[rlpEncode-DIAG-NODE] 8.Extra(%d):       %x", len(header.Extra), header.Extra)
		}
		writer.Write(header.Extra)
	} else {
		if nodeDebugLogging.Load() {
			log.Printf("[rlpEncode-DIAG-NODE] 8.Extra:          (empty, skipped)")
		}
	}

	// Nonce
	if nodeDebugLogging.Load() {
		log.Printf("[rlpEncode-DIAG-NODE] 9.Nonce(%d):       %x", len(header.Nonce[:]), header.Nonce[:])
	}
	writer.Write(header.Nonce[:])

	// Difficulty as big.Int bytes
	if header.Difficulty != nil {
		diffBytes := header.Difficulty.Bytes()
		if nodeDebugLogging.Load() {
			log.Printf("[rlpEncode-DIAG-NODE] 10.Difficulty(%d): %x", len(diffBytes), diffBytes)
		}
		writer.Write(diffBytes)
	} else {
		if nodeDebugLogging.Load() {
			log.Printf("[rlpEncode-DIAG-NODE] 10.Difficulty:     nil")
		}
	}
}

// BytesToHash converts bytes to hash
func BytesToHash(b []byte) Hash {
	var h Hash
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(h[32-len(b):], b)
	return h
}

// BigToHash converts big.Int to hash
func BigToHash(b *big.Int) Hash {
	if b == nil {
		return Hash{}
	}
	return BytesToHash(b.Bytes())
}

// StringToAddress converts NOGO-prefixed or raw hex string to Address.
// Strips "NOGO" prefix, hex-decodes, and validates length.
// Returns an error when the input is malformed, preventing silent zero-address
// assignments that would cause coinbase rewards to be permanently lost.
func StringToAddress(s string) (Address, error) {
	var a Address
	encoded := s

	// Strip "NOGO" prefix if present
	if len(s) >= 4 && s[:4] == "NOGO" {
		encoded = s[4:]
	}

	// Hex-decode and copy first 20 bytes
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return a, fmt.Errorf("invalid address hex: %w", err)
	}
	if len(decoded) < 20 {
		return a, fmt.Errorf("invalid address length: got %d, need >= 20", len(decoded))
	}

	copy(a[:], decoded[:20])
	return a, nil
}

// StringToAddressChecked is an alias for StringToAddress.
// Deprecated: use StringToAddress which now returns (Address, error).
func StringToAddressChecked(s string) (Address, error) {
	return StringToAddress(s)
}

// CompactToBig converts a compact representation of a whole number N to an
// unsigned 32-bit number. The representation is similar to IEEE754 floating point.
//
// Like IEEE754 floating point, there are three basic components: the sign,
// the exponent, and the mantissa. They are broken out as follows:
//
//	* the most significant 8 bits represent the unsigned base 256 exponent
//	* bit 23 (the 24th bit) represents the sign bit
//	* the least significant 23 bits represent the mantissa
//
//	    -------------------------------------------------
//	    |   Exponent     |    Sign    |    Mantissa     |
//	    -------------------------------------------------
//	    | 8 bits [31-24] | 1 bit [23] | 23 bits [22-00] |
//	    -------------------------------------------------
//
// The formula to calculate N is:
//
//	N = (-1^sign) * mantissa * 256^(exponent-3)
func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number. So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly. This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// BigToCompact converts a whole number N to a compact representation using
// an unsigned 32-bit number. The compact representation only provides 23 bits
// of precision, so values larger than (2^23 - 1) only encode the most
// significant digits of the number. See CompactToBig for details.
func BigToCompact(n *big.Int) uint32 {
	// No need to do any work if it's zero.
	if n.Sign() == 0 {
		return 0
	}

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes. So, shift the number right or left
	// accordingly. This is equivalent to:
	// mantissa = mantissa * 256^(exponent-3)
	var mantissa uint32
	exponent := uint(len(n.Bytes()))

	// Note: the mantissa is truncated to 23 bits when it overflows.
	if exponent <= 3 {
		mantissa = uint32(n.Bits()[0])
		mantissa <<= 8 * (3 - exponent)
	} else {
		// Use a copy to avoid modifying n's underlying array.
		tn := new(big.Int).Set(n)
		mantissa = uint32(tn.Rsh(tn, 8*(exponent-3)).Bits()[0])
	}

	// Truncate to the 23-bit mantissa.
	mantissa &= 0x007fffff
	compact := uint32(exponent<<24) | mantissa
	if n.Sign() < 0 {
		compact |= 0x00800000
	}
	return compact
}
