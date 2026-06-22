// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"math/big"
	"testing"
)

// defaultTestConsensusParams returns default consensus params for testing
func defaultTestConsensusParams() *ConsensusParams {
	return &ConsensusParams{
		ChainID:                      1,
		DifficultyEnable:             true,
		BlockTimeTargetSeconds:       30,
		MinDifficulty:                1,
		MaxDifficultyChangePercent:   50,
		DifficultyAdjustmentInterval: 10,
	}
}

// TestDeterministicDifficulty_Basic tests basic deterministic difficulty calculation.
func TestDeterministicDifficulty_Basic(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	parent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}

	// Call CalcDifficulty twice with same inputs.
	// v4.0 Kp‑P controller is stateless — same inputs → same output.
	result1 := adjuster.CalcDifficulty(1030, parent)
	result2 := adjuster.CalcDifficulty(1030, parent)

	// Same inputs must produce same outputs (deterministic).
	if result1.Cmp(result2) != 0 {
		t.Errorf("Deterministic property violated: same inputs produced different results %d vs %d",
			result1, result2)
	}

	// Result must be within [0.5×, 2.0×] of parent (clamp bounds).
	minExpected := new(big.Int).Div(parent.Difficulty, big.NewInt(2))
	maxExpected := new(big.Int).Mul(parent.Difficulty, big.NewInt(2))
	if result1.Cmp(minExpected) < 0 {
		t.Errorf("Difficulty %d below minimum %d", result1, minExpected)
	}
	if result1.Cmp(maxExpected) > 0 {
		t.Errorf("Difficulty %d above maximum %d", result1, maxExpected)
	}
}

// TestDeterministicDifficulty_OnTarget tests difficulty stays at parent level
// when the block interval is exactly the target (deadband threshold).
func TestDeterministicDifficulty_OnTarget(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	parent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}

	// 30 s block time = target → ratio = 1.0 → deadband → no change.
	newDiff := adjuster.CalcDifficulty(1030, parent)

	if newDiff.Cmp(big.NewInt(1000)) != 0 {
		t.Errorf("Expected difficulty near 1000 when on target, got %d", newDiff)
	}
}

// TestDeterministicDifficulty_SlowBlocks tests difficulty decreases when blocks are slow.
func TestDeterministicDifficulty_SlowBlocks(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	parent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}

	// 120 s block time = 4× target → ratio = 0.25
	// adj = 1 + Kp × (0.25 − 1) = 1 + 0.5 × (−0.75) = 0.625
	// newDiff = ceil(1000 × 0.625) = 625 (< 1000, within clamp).
	newDiff := adjuster.CalcDifficulty(1120, parent)

	if newDiff.Cmp(parent.Difficulty) >= 0 {
		t.Errorf("Expected difficulty decrease for slow blocks, got increase: %d -> %d",
			parent.Difficulty, newDiff)
	}

	// Must not drop below 50% of parent (clamp floor).
	minAllowed := new(big.Int).Div(parent.Difficulty, big.NewInt(2))
	if newDiff.Cmp(minAllowed) < 0 {
		t.Errorf("Difficulty %d dropped below 50%% floor %d", newDiff, minAllowed)
	}
}

// TestDeterministicDifficulty_FastBlocks tests difficulty increases when blocks are fast.
func TestDeterministicDifficulty_FastBlocks(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	parent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}

	// 5 s block time, target = 30 s → ratio = 6.0
	// adj = 1 + Kp × (6 − 1) = 1 + 0.5 × 5 = 3.5 → clamped to 2.0
	// newDiff = ceil(1000 × 2.0) = 2000.
	newDiff := adjuster.CalcDifficulty(1005, parent)

	if newDiff.Cmp(parent.Difficulty) <= 0 {
		t.Errorf("Expected difficulty increase for fast blocks, got %d -> %d",
			parent.Difficulty, newDiff)
	}

	// Must not exceed 2× of parent (clamp ceiling).
	maxAllowed := new(big.Int).Mul(parent.Difficulty, big.NewInt(2))
	if newDiff.Cmp(maxAllowed) > 0 {
		t.Errorf("Difficulty %d exceeded 2× ceiling %d", newDiff, maxAllowed)
	}
}

// TestDeterministicDifficulty_Genesis tests genesis block returns minimum difficulty
func TestDeterministicDifficulty_Genesis(t *testing.T) {
	params := defaultTestConsensusParams()
	params.MinDifficulty = 5
	adjuster := NewDifficultyAdjuster(params)

	// Nil parent → genesis
	result := adjuster.CalcDifficulty(100, nil)
	expected := big.NewInt(int64(params.MinDifficulty))
	if result.Cmp(expected) != 0 {
		t.Errorf("Expected minimum difficulty %d for genesis, got %d", expected, result)
	}
}

// TestDeterministicDifficulty_Boundaries tests boundary conditions
func TestDeterministicDifficulty_Boundaries(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	// Test with minimum difficulty parent
	minParent := &Header{
		Difficulty: big.NewInt(1),
		Time:       1000,
	}

	result := adjuster.CalcDifficulty(1001, minParent)
	if result.Cmp(big.NewInt(1)) < 0 {
		t.Errorf("Difficulty %d below minimum", result)
	}

	// Test with extreme time difference (very slow)
	slowParent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}
	slowResult := adjuster.CalcDifficulty(1000+3600*2, slowParent) // 2 hours
	if slowResult.Cmp(big.NewInt(0)) <= 0 {
		t.Errorf("Difficulty must be positive, got %d", slowResult)
	}
	// Must not drop below 50%
	minAllowed := new(big.Int).Div(slowParent.Difficulty, big.NewInt(2))
	if slowResult.Cmp(minAllowed) < 0 {
		t.Errorf("Difficulty %d dropped below floor %d", slowResult, minAllowed)
	}
}

// TestDeterministicDifficulty_DifferentAdjusters tests that two independently
// created adjusters produce identical results for the same (parent, time) pair.
//
// v4.0 Kp‑P controller is stateless — no ancestor function is required or
// supported, so any two adjusters with the same params must agree exactly.
func TestDeterministicDifficulty_DifferentAdjusters(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster1 := NewDifficultyAdjuster(params)
	adjuster2 := NewDifficultyAdjuster(params)

	parent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}

	result1 := adjuster1.CalcDifficulty(1030, parent)
	result2 := adjuster2.CalcDifficulty(1030, parent)

	if result1.Cmp(result2) != 0 {
		t.Errorf("Two adjusters with same params produced different results: %d vs %d",
			result1, result2)
	}

	// Both must be positive
	if result1.Cmp(big.NewInt(0)) <= 0 {
		t.Errorf("Result must be positive, got %d", result1)
	}
}

// TestDeterministicDifficulty_DoubleInvocation tests that calling CalcDifficulty
// twice with the same inputs produces the same result (no state contamination)
func TestDeterministicDifficulty_DoubleInvocation(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	parent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}

	// First call
	first := adjuster.CalcDifficulty(1030, parent)

	// Second call with SAME parent (simulating fork validation scenario)
	second := adjuster.CalcDifficulty(1030, parent)

	if first.Cmp(second) != 0 {
		t.Errorf("State contamination detected: first call=%d, second call=%d (must be identical)",
			first, second)
	}
}

// TestDeterministicDifficulty_ValidateDifficulty tests the ValidateDifficulty method
func TestDeterministicDifficulty_ValidateDifficulty(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	parent := &Header{
		Difficulty: big.NewInt(1000),
		Time:       1000,
	}

	// Valid difficulty
	if !adjuster.ValidateDifficulty(big.NewInt(1000), parent) {
		t.Error("Difficulty 1000 should be valid")
	}

	// Zero difficulty should be invalid
	if adjuster.ValidateDifficulty(big.NewInt(0), parent) {
		t.Error("Zero difficulty should be invalid")
	}

	// Negative difficulty should be invalid
	if adjuster.ValidateDifficulty(big.NewInt(-1), parent) {
		t.Error("Negative difficulty should be invalid")
	}

	// Very high difficulty should be invalid (exceeds tolerance)
	if adjuster.ValidateDifficulty(big.NewInt(1000000), parent) {
		t.Error("Very high difficulty should be invalid")
	}

	// Nil parent should still validate basic checks
	if !adjuster.ValidateDifficulty(big.NewInt(50), nil) {
		t.Error("Difficulty 50 should be valid with nil parent")
	}
	if adjuster.ValidateDifficulty(big.NewInt(-5), nil) {
		t.Error("Negative difficulty should be invalid even with nil parent")
	}
}

// TestDeterministicDifficulty_Parameters tests Kp‑P controller parameters.
func TestDeterministicDifficulty_Parameters(t *testing.T) {
	params := defaultTestConsensusParams()
	adjuster := NewDifficultyAdjuster(params)

	gotKp, gotDeadband := adjuster.GetParameters()
	if gotKp != kpGain {
		t.Errorf("Expected kpGain=%f, got %f", kpGain, gotKp)
	}
	if gotDeadband != kpDeadband {
		t.Errorf("Expected deadband=%f, got %f", kpDeadband, gotDeadband)
	}
}