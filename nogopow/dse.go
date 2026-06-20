// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"encoding/binary"
	"fmt"
	"math/big"
)

// DSE (Deterministic Simple Encoding) provides a fixed-size, deterministic
// serialization for block headers. Unlike RLP, DSE prepends a 2-byte
// big-endian length prefix to every variable-length field, eliminating
// ambiguity in deserialization.
//
// Encoding format:
//
//	ParentHash  [32 bytes]  fixed
//	Coinbase    [32 bytes]  fixed (20-byte address padded to 32)
//	StateRoot   [32 bytes]  fixed
//	TxHash      [32 bytes]  fixed (merkle root)
//	Number      [2B len][var bytes]  big-endian
//	GasLimit    [8 bytes]   fixed
//	Time        [8 bytes]   fixed
//	Extra       [2B len][var bytes]
//	Nonce       [32 bytes]  fixed
//	Difficulty  [2B len][var bytes]  big-endian
//
// Minimum size: 184 bytes (32*4 + 2+1 + 8+8 + 2+0 + 32 + 2+1)
// Typical size: ~200 bytes

const (
	// DSEFixedFieldSizes defines the fixed-size fields in the header.
	DSEParentHashSize = 32
	DSECoinbaseSize   = 32
	DSEStateRootSize  = 32
	DSETxHashSize     = 32
	DSEGasLimitSize   = 8
	DSETimeSize       = 8
	DSENonceSize      = 32
	DSEFixedOverhead  = DSEParentHashSize + DSECoinbaseSize + DSEStateRootSize +
		DSETxHashSize + DSEGasLimitSize + DSETimeSize + DSENonceSize
	DSELengthPrefixSize = 2
	DSEMinHeaderSize    = DSEFixedOverhead + DSELengthPrefixSize + 1 + // Number min
		DSELengthPrefixSize + 0 + // Extra empty
		DSELengthPrefixSize + 1 // Difficulty min
)

// DSEEncodeHeader serializes a Header into deterministic DSE format.
// Returns the byte slice for hashing (SealHash input).
func DSEEncodeHeader(h *Header) ([]byte, error) {
	if h == nil {
		return nil, fmt.Errorf("DSEEncodeHeader: nil header")
	}

	// Estimate buffer size: fixed overhead + 3 variable fields with prefixes.
	bufSize := DSEFixedOverhead + 3*DSELengthPrefixSize + 8 + len(h.Extra) + 32
	buf := make([]byte, 0, bufSize)

	// Fixed fields.
	buf = append(buf, h.ParentHash[:]...)
	coinbase := PadAddressTo32(h.Coinbase)
	buf = append(buf, coinbase[:]...)
	buf = append(buf, h.Root[:]...)
	buf = append(buf, h.TxHash[:]...)

	// Number — variable length, big-endian with 2-byte length prefix.
	numBytes := h.Number.Bytes()
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(numBytes)))
	buf = append(buf, numBytes...)

	// GasLimit — fixed 8 bytes, big-endian.
	buf = binary.BigEndian.AppendUint64(buf, h.GasLimit)

	// Time — fixed 8 bytes, big-endian.
	buf = binary.BigEndian.AppendUint64(buf, h.Time)

	// Extra — variable length with 2-byte length prefix.
	extraLen := len(h.Extra)
	buf = binary.BigEndian.AppendUint16(buf, uint16(extraLen))
	buf = append(buf, h.Extra...)

	// Nonce — fixed 32 bytes.
	buf = append(buf, h.Nonce[:]...)

	// Difficulty — variable length, big-endian with 2-byte length prefix.
	diffBytes := h.Difficulty.Bytes()
	buf = binary.BigEndian.AppendUint16(buf, uint16(len(diffBytes)))
	buf = append(buf, diffBytes...)

	return buf, nil
}

// DSEDecodeHeader deserializes DSE-encoded bytes back into a Header.
func DSEDecodeHeader(data []byte) (*Header, error) {
	if len(data) < DSEMinHeaderSize {
		return nil, fmt.Errorf("DSEDecodeHeader: data too short (%d < %d)", len(data), DSEMinHeaderSize)
	}

	pos := 0

	h := &Header{}

	// ParentHash [32].
	copy(h.ParentHash[:], data[pos:pos+DSEParentHashSize])
	pos += DSEParentHashSize

	// Coinbase [32] (padded address).
	var coinbase32 [32]byte
	copy(coinbase32[:], data[pos:pos+DSECoinbaseSize])
	h.Coinbase = UnpadAddressFrom32(coinbase32)
	pos += DSECoinbaseSize

	// StateRoot [32].
	copy(h.Root[:], data[pos:pos+DSEStateRootSize])
	pos += DSEStateRootSize

	// TxHash [32].
	copy(h.TxHash[:], data[pos:pos+DSETxHashSize])
	pos += DSETxHashSize

	// Number [2B len][var].
	numLen := int(binary.BigEndian.Uint16(data[pos : pos+DSELengthPrefixSize]))
	pos += DSELengthPrefixSize
	if pos+numLen > len(data) {
		return nil, fmt.Errorf("DSEDecodeHeader: number field exceeds data bounds")
	}
	h.Number = new(big.Int).SetBytes(data[pos : pos+numLen])
	pos += numLen

	// GasLimit [8].
	h.GasLimit = binary.BigEndian.Uint64(data[pos : pos+DSEGasLimitSize])
	pos += DSEGasLimitSize

	// Time [8].
	h.Time = binary.BigEndian.Uint64(data[pos : pos+DSETimeSize])
	pos += DSETimeSize

	// Extra [2B len][var].
	extraLen := int(binary.BigEndian.Uint16(data[pos : pos+DSELengthPrefixSize]))
	pos += DSELengthPrefixSize
	if pos+extraLen > len(data) {
		return nil, fmt.Errorf("DSEDecodeHeader: extra field exceeds data bounds")
	}
	h.Extra = make([]byte, extraLen)
	copy(h.Extra, data[pos:pos+extraLen])
	pos += extraLen

	// Nonce [32].
	copy(h.Nonce[:], data[pos:pos+DSENonceSize])
	pos += DSENonceSize

	// Difficulty [2B len][var].
	diffLen := int(binary.BigEndian.Uint16(data[pos : pos+DSELengthPrefixSize]))
	pos += DSELengthPrefixSize
	if pos+diffLen > len(data) {
		return nil, fmt.Errorf("DSEDecodeHeader: difficulty field exceeds data bounds")
	}
	h.Difficulty = new(big.Int).SetBytes(data[pos : pos+diffLen])
	// pos += diffLen — not needed, end of header

	return h, nil
}

// PadAddressTo32 pads a 20-byte Address to 32 bytes for DSE encoding.
// The padding prepends 12 zero bytes.
func PadAddressTo32(addr Address) [32]byte {
	var padded [32]byte
	copy(padded[12:], addr[:])
	return padded
}

// UnpadAddressFrom32 extracts a 20-byte Address from a 32-byte padded field.
func UnpadAddressFrom32(padded [32]byte) Address {
	var addr Address
	copy(addr[:], padded[12:])
	return addr
}
