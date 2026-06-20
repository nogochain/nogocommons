// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"fmt"
	"math"
	"math/big"
)

// BlockchainCompatibility provides compatibility layer for blockchain package
// This allows the blockchain to use nogopow difficulty adjustment seamlessly

// BlockHeader represents a minimal block header for difficulty calculation
type BlockHeader struct {
	Height         uint64
	TimestampUnix  int64
	DifficultyBits uint32
	PrevHash       []byte
	Hash           []byte
}

// DifficultyCalculator provides difficulty calculation services
type DifficultyCalculator struct {
	adjuster        *DifficultyAdjuster
	consensusParams *ConsensusParams
}

// NewDifficultyCalculator creates a new difficulty calculator
func NewDifficultyCalculator(consensusParams *ConsensusParams) *DifficultyCalculator {
	if consensusParams == nil {
		consensusParams = &ConsensusParams{
			BlockTimeTargetSeconds:     30,
			MaxDifficultyChangePercent: 100,
		}
	}

	return &DifficultyCalculator{
		adjuster:        NewDifficultyAdjuster(consensusParams),
		consensusParams: consensusParams,
	}
}

// SetAncestorFunc sets the ancestor lookup function for deterministic
// difficulty computation. When set, the calculator computes difficulty
// from chain data rather than using a running accumulator.
func (dc *DifficultyCalculator) SetAncestorFunc(fn GetAncestorFunc) {
	dc.adjuster.SetAncestorFunc(fn)
}

// CalcNextDifficulty calculates difficulty for next block given parent block
func (dc *DifficultyCalculator) CalcNextDifficulty(parent *BlockHeader, currentTime uint64) uint32 {
	if parent == nil {
		return uint32(dc.consensusParams.MinDifficulty)
	}

	// Convert parent to nogopow.Header format
	parentHeader := &Header{
		Number:     big.NewInt(int64(parent.Height)),
		Time:       uint64(parent.TimestampUnix),
		Difficulty: big.NewInt(int64(parent.DifficultyBits)),
	}

	// Calculate new difficulty using PI controller
	newDifficulty := dc.adjuster.CalcDifficulty(currentTime, parentHeader)

	// Guard against difficulty values that overflow uint32.
	// big.Int.Uint64() returns low 64 bits; prevent silent truncation
	// by checking BitLen first for values beyond 32-bit range.
	var bits uint64
	if newDifficulty.BitLen() > 32 {
		bits = math.MaxUint32
	} else {
		bits = newDifficulty.Uint64()
		if bits > math.MaxUint32 {
			bits = math.MaxUint32
		}
	}
	if bits < 1 {
		bits = 1
	}

	return uint32(bits)
}

// ValidateDifficulty validates block difficulty against expected value
func (dc *DifficultyCalculator) ValidateDifficulty(parent, current *BlockHeader) error {
	if parent == nil || current == nil {
		return nil
	}

	// Calculate expected difficulty
	expected := dc.CalcNextDifficulty(parent, uint64(current.TimestampUnix))

	// Check if actual difficulty matches expected
	if current.DifficultyBits != expected {
		return &DifficultyMismatchError{
			Height:   current.Height,
			Expected: expected,
			Got:      current.DifficultyBits,
		}
	}

	return nil
}

// DifficultyMismatchError represents a difficulty validation error
type DifficultyMismatchError struct {
	Height   uint64
	Expected uint32
	Got      uint32
}

func (e *DifficultyMismatchError) Error() string {
	return fmt.Sprintf("bad difficulty at height %d: expected %d got %d", e.Height, e.Expected, e.Got)
}

// GetMinimumDifficulty returns the minimum difficulty value
func (dc *DifficultyCalculator) GetMinimumDifficulty() uint32 {
	return uint32(dc.consensusParams.MinDifficulty)
}

// GetMaximumDifficulty returns the maximum difficulty value (256 bits)
func (dc *DifficultyCalculator) GetMaximumDifficulty() uint32 {
	return maxDifficultyBits
}

// Constants for compatibility
const maxDifficultyBits = uint32(math.MaxUint32)
