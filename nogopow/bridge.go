// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"fmt"
	"math/big"
)

// Bridge provides adaptation between btcd's blockchain types and nogopow types.
// It converts btcd wire/chainhash representations to nogopow Header/Hash
// for consensus verification and back.

// BTCDBlockHeader represents a minimal btcd-style block header for bridging.
type BTCDBlockHeader struct {
	Version    int32
	PrevBlock  [32]byte
	MerkleRoot [32]byte
	Timestamp  int64
	Bits       uint32
	Nonce      [32]byte
	Height     int64
	Coinbase   [20]byte
	StateRoot  [32]byte
	ExtraData  []byte
}

// FromBTCDHeader converts a btcd-style block header to a nogopow Header.
func FromBTCDHeader(btcd *BTCDBlockHeader, difficulty *big.Int) *Header {
	if btcd == nil {
		return nil
	}

	var parentHash Hash
	copy(parentHash[:], btcd.PrevBlock[:])

	var coinbase Address
	copy(coinbase[:], btcd.Coinbase[:])

	var stateRoot Hash
	copy(stateRoot[:], btcd.StateRoot[:])

	var txHash Hash
	copy(txHash[:], btcd.MerkleRoot[:])

	var nonce BlockNonce
	copy(nonce[:], btcd.Nonce[:])

	return &Header{
		ParentHash: parentHash,
		Coinbase:   coinbase,
		Root:       stateRoot,
		TxHash:     txHash,
		Number:     big.NewInt(btcd.Height),
		GasLimit:   uint64(btcd.Bits),
		Time:       uint64(btcd.Timestamp),
		Extra:      btcd.ExtraData,
		Nonce:      nonce,
		Difficulty: difficulty,
	}
}

// ToBTCDHeader converts a nogopow Header to a btcd-style block header.
func ToBTCDHeader(h *Header) (*BTCDBlockHeader, error) {
	if h == nil {
		return nil, fmt.Errorf("ToBTCDHeader: nil header")
	}
	if h.Number == nil {
		return nil, fmt.Errorf("ToBTCDHeader: nil block number")
	}

	var prevBlock [32]byte
	copy(prevBlock[:], h.ParentHash[:])

	var coinbase [20]byte
	copy(coinbase[:], h.Coinbase[:])

	var stateRoot [32]byte
	copy(stateRoot[:], h.Root[:])

	var merkleRoot [32]byte
	copy(merkleRoot[:], h.TxHash[:])

	var nonce [32]byte
	copy(nonce[:], h.Nonce[:])

	extra := make([]byte, len(h.Extra))
	copy(extra, h.Extra)

	return &BTCDBlockHeader{
		Version:    1,
		PrevBlock:  prevBlock,
		MerkleRoot: merkleRoot,
		Timestamp:  int64(h.Time),
		Bits:       uint32(h.GasLimit & 0xFFFFFFFF),
		Nonce:      nonce,
		Height:     h.Number.Int64(),
		Coinbase:   coinbase,
		StateRoot:  stateRoot,
		ExtraData:  extra,
	}, nil
}

// VerifyBTCDBlock checks a btcd-style block header's NogoPow proof-of-work.
// Returns nil if valid, or an error describing the failure.
func (engine *NogopowEngine) VerifyBTCDBlock(btcd *BTCDBlockHeader, difficulty *big.Int) error {
	if btcd == nil {
		return fmt.Errorf("VerifyBTCDBlock: nil header")
	}

	hdr := FromBTCDHeader(btcd, difficulty)
	if hdr == nil {
		return fmt.Errorf("VerifyBTCDBlock: failed to convert header")
	}

	// SealHash(hdr) is the blockHash (varies with Nonce), not the cache seed.
	// VerifySealWithBlockHash internally calls calcSeed() which derives the
	// correct seed from header.ParentHash.
	blockHash := engine.SealHash(hdr)
	return engine.VerifySealWithBlockHash(hdr, blockHash)
}

// ComputeBTCDSealHash computes the SealHash for a btcd-style block header.
func ComputeBTCDSealHash(btcd *BTCDBlockHeader, difficulty *big.Int) Hash {
	hdr := FromBTCDHeader(btcd, difficulty)
	if hdr == nil {
		return Hash{}
	}
	// SealHash is a method on NogopowEngine, compute via Hash().
	return hdr.Hash()
}
