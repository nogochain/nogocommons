// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"fmt"
	"log"
	"math"
	"math/big"
	"sort"
	"sync"
)

const (
	defaultWindowSize     = 30 // v2.0: increased from 10 for ~15 min history, smoother response
	maxReasonableTimeDiff = int64(3600)
)

// PI controller tuning constants — v2.0 optimized for stability.
const (
	defaultKp = 0.25 // increased from 0.10: faster proportional response to block time error
	defaultKi = 0.05 // increased from 0.02: faster elimination of systematic bias

	integralDecay = 0.95 // faster decay from 0.97: strong-CPU impact fades in ~20 blocks

	integralClampMin = -1.0 // tightened from -1.5: reduce overshoot during correction
	integralClampMax = 1.0  // tightened from 1.5: reduce overshoot during strong-CPU events

	scanDepth = 100

	// Double exponential smoothing coefficients.
	smoothingAlpha = 0.2 // reduced from 0.3
	smoothingBeta  = 0.05 // reduced from 0.1

	maxSmoothingWindow = 50

	// Per-block adjustment caps — asymmetric: slow up, fast down.
	normalModeCapUp   = 1.12 // max +12% per block (was ±8%, now +12% for faster catch-up)
	normalModeCapDown = 0.85 // max -15% per block (faster difficulty decrease when slow)

	// Deadband: no adjustment when error is within ±10%.
	deadbandErrorRatio = 0.05 // tightened from 0.10: ±5% deadband for quicker response

	// PI/DES blend weights.
	piWeight  = 0.4 // reduced from 0.7
	desWeight = 0.6 // increased from 0.3

	// Emergency recovery mode.
	emergencyThreshold  = int64(5) // block interval ≥ 5× target triggers emergency
	emergencyFloorRatio = 0.15     // single-block minimum difficulty ratio
)

// GetAncestorFunc retrieves a block header by its height on the canonical chain.
// Returns nil if the height is not available.
type GetAncestorFunc func(height uint64) *Header

// DifficultyAdjuster implements deterministic difficulty adjustment using a
// PI controller. All state is computed fresh from chain data, making it
// fully deterministic: given the same chain state, different nodes compute
// the exact same difficulty.
//
// Mathematical foundation: Proportional-Integral (PI) controller
//   - Proportional term (Kp): Responds to current error
//   - Integral term (Ki): Accumulates past errors with decay
//   - Formula: output = Kp * error + Ki * integral(error)
//   - Deterministic: integral is computed from chain history, not from a
//     running accumulator. This prevents validator state contamination
//     when processing fork blocks.
type DifficultyAdjuster struct {
	mu sync.Mutex // Protects CalcDifficulty state

	consensusParams  *ConsensusParams
	integralGain     float64 // Ki coefficient
	proportionalGain float64 // Kp coefficient
	windowSize       int     // Number of blocks for average time window

	// getAncestor provides access to ancestor block headers for deterministic
	// difficulty computation. Must be set before calling CalcDifficulty.
	// This is not protected by mu since it's set once during initialization.
	getAncestor GetAncestorFunc
}

// NewDifficultyAdjuster creates a new deterministic difficulty adjuster.
// Unlike the previous stateful implementation, this version computes all
// values from chain data, ensuring identical results across all nodes.
func NewDifficultyAdjuster(consensusParams *ConsensusParams) *DifficultyAdjuster {
	if consensusParams == nil {
		consensusParams = &ConsensusParams{
			BlockTimeTargetSeconds:     30,
			MaxDifficultyChangePercent: 100,
		}
	}

	windowSize := defaultWindowSize
	if consensusParams.DifficultyAdjustmentInterval > 1 {
		windowSize = int(consensusParams.DifficultyAdjustmentInterval)
	}

	return &DifficultyAdjuster{
		consensusParams:  consensusParams,
		integralGain:     defaultKi,
		proportionalGain: defaultKp,
		windowSize:       windowSize,
	}
}

// SetAncestorFunc sets the ancestor lookup function. Must be called before
// CalcDifficulty when using deterministic mode.
func (da *DifficultyAdjuster) SetAncestorFunc(fn GetAncestorFunc) {
	da.getAncestor = fn
}

// GetAncestorFunc returns the current ancestor lookup function.
// Returns nil if not set, in which case CalcDifficulty falls back to
// a simplified proportional-only calculation.
func (da *DifficultyAdjuster) GetAncestorFunc() GetAncestorFunc {
	return da.getAncestor
}

// CalcDifficulty calculates difficulty for the next block using a deterministic
// PI controller. The calculation relies solely on the parent block header and
// the ancestor lookup function (if set).
//
// When getAncestor is set (recommended), the calculation is fully deterministic:
//   - Average block time: computed from parent and its (windowSize)-th ancestor
//   - Integral: computed from the last N blocks' individual time errors
//   - No node-local state affects the result
//
// When getAncestor is nil (backward compatibility), falls back to a simplified
// proportional-only calculation using the single parent time difference.
//
// Parameters:
//   - currentTime: Unix timestamp of the new block (or current time for templates)
//   - parent: Previous block header
//
// Returns:
//   - *big.Int: New difficulty value
func (da *DifficultyAdjuster) CalcDifficulty(currentTime uint64, parent *Header) *big.Int {
	// Lock protection (concurrency safety)
	da.mu.Lock()
	defer da.mu.Unlock()

	if parent == nil || parent.Difficulty == nil {
		minDiff := big.NewInt(1)
		if da.consensusParams.MinDifficulty > 0 {
			minDiff = big.NewInt(int64(da.consensusParams.MinDifficulty))
		}
		log.Printf("[Difficulty] Genesis block: using minimum difficulty %d", minDiff)
		return minDiff
	}

	parentDiff := new(big.Int).Set(parent.Difficulty)
	targetTime := int64(da.consensusParams.BlockTimeTargetSeconds)

	// Compute average block time from chain data
	avgBlockTime := da.computeAverageBlockTime(currentTime, parent)

	// Compute deterministic difficulty
	newDifficulty := da.calculateDeterministicDifficulty(avgBlockTime, targetTime, parentDiff, parent)

	// Log with full big.Int string to avoid truncation for values > 2^64.
	log.Printf("[Difficulty] Deterministic: parentDiff=%s, avgTime=%ds, target=%ds, calculated=%s",
		parentDiff.String(), avgBlockTime, targetTime, newDifficulty.String())

	newDifficulty = da.enforceBoundaryConditionsLocked(newDifficulty, parentDiff)

	var changePct float64
	if parentDiff.Sign() > 0 {
		parentFloat, _ := new(big.Float).SetInt(parentDiff).Float64()
		newFloat, _ := new(big.Float).SetInt(newDifficulty).Float64()
		if parentFloat > 0 {
			changePct = (newFloat - parentFloat) / parentFloat * 100
		}
	}

	log.Printf("[Difficulty] Result: %s -> %s (%.1f%% change)",
		parentDiff.String(), newDifficulty.String(), changePct)

	return newDifficulty
}

// computeAverageBlockTime computes the median block time from chain data.
// Uses median instead of simple average to resist outlier blocks.
// Falls back to simple time difference if chain data is insufficient.
func (da *DifficultyAdjuster) computeAverageBlockTime(currentTime uint64, parent *Header) int64 {
	// If ancestor function is available, collect recent block times and compute median
	if da.getAncestor != nil && parent != nil && parent.Number != nil {
		height := parent.Number.Uint64()
		window := da.windowSize
		if window > 50 {
			window = 50 // Limit window size for performance
		}

		// Collect time diffs from recent blocks
		timeDiffs := make([]int64, 0, window)
		for i := uint64(1); i <= uint64(window) && height >= i; i++ {
			curr := da.getAncestor(height - i + 1)
			prev := da.getAncestor(height - i)
			if curr == nil || prev == nil || prev.Time == 0 || curr.Time <= prev.Time {
				continue
			}
			diff := int64(curr.Time - prev.Time)
			// Clamp time diff to reasonable range [1s, 300s]
			if diff < 1 {
				diff = 1
			}
			if diff > 300 {
				diff = 300
			}
			timeDiffs = append(timeDiffs, diff)
		}

		if len(timeDiffs) > 0 {
			// Compute trimmed mean (discard top/bottom 20%) for resistance to outliers.
			sort.Slice(timeDiffs, func(i, j int) bool { return timeDiffs[i] < timeDiffs[j] })
			mean := trimmedMean(timeDiffs, 0.1) // reduced from 0.2: keep more slow blocks in average

			log.Printf("[TrimmedMean] window=%d, mean=%ds, min=%ds, max=%ds",
				len(timeDiffs), mean, timeDiffs[0], timeDiffs[len(timeDiffs)-1])

			return mean
		}
	}

	// Fallback: use single block interval if parent time and current time are available
	if parent != nil && parent.Time > 0 && currentTime > parent.Time {
		timeDiff := int64(currentTime - parent.Time)
		if timeDiff > 0 && timeDiff < maxReasonableTimeDiff {
			return timeDiff
		}
	}

	// Ultimate fallback: return target time
	return int64(da.consensusParams.BlockTimeTargetSeconds)
}

// calculateDeterministicDifficulty implements the core PI controller
// combined with double exponential smoothing. v2.0 adds emergency recovery
// mode to prevent chain stall when strong miners leave.
func (da *DifficultyAdjuster) calculateDeterministicDifficulty(avgTime int64, targetTime int64, parentDiff *big.Int, parent *Header) *big.Int {
	// Emergency recovery: when blocks are extremely slow (≥5× target),
	// apply proportional difficulty reduction to prevent chain stall.
	if avgTime >= targetTime*emergencyThreshold {
		ratio := float64(targetTime) / float64(avgTime)
		if ratio < emergencyFloorRatio {
			ratio = emergencyFloorRatio
		}
		newDiffFloat := new(big.Float).Mul(new(big.Float).SetInt(parentDiff), big.NewFloat(ratio))
		result, _ := newDiffFloat.Int(nil)
		if result.Sign() <= 0 {
			result = big.NewInt(1)
		}
		log.Printf("[EmergencyRecovery] blockTime=%ds target=%ds ratio=%.3f diff %d→%d",
			avgTime, targetTime, ratio, parentDiff.Uint64(), result.Uint64())
		return enforceMinDiff(result, uint32(da.consensusParams.MinDifficulty))
	}

	// Normal mode: PI controller with deadband.
	var errVal float64
	if avgTime > 0 && targetTime > 0 {
		errVal = float64(targetTime-avgTime) / float64(targetTime)
	}
	errVal = clampFloat64(errVal, -0.75, 3.0)

	// Deadband: no adjustment for small errors.
	if math.Abs(errVal) < deadbandErrorRatio {
		return new(big.Int).Set(parentDiff)
	}

	integral := da.computeChainIntegral(parent)
	piOutput := da.proportionalGain*errVal + da.integralGain*integral
	smoothedOutput := da.calculateDoubleExponentialSmoothing(avgTime, targetTime, parent)
	multiplier := 1.0 + piWeight*piOutput + desWeight*smoothedOutput

	// Asymmetric clamping: slow up, slow down (emergency handles fast down).
	multiplier = clampFloat64(multiplier, normalModeCapDown, normalModeCapUp)

	newDiffFloat := new(big.Float).Mul(
		new(big.Float).SetInt(parentDiff),
		big.NewFloat(multiplier),
	)
	ceiled := new(big.Float).Add(newDiffFloat, big.NewFloat(0.999999))
	newDifficulty, _ := ceiled.Int(nil)

	if newDifficulty.Sign() < 0 {
		newDifficulty = big.NewInt(0)
	}

	log.Printf("[DeterministicPI] avgTime=%ds target=%ds | err=%.3f integral=%.3f | smoothed=%.3f | mult=%.3f",
		avgTime, targetTime, errVal, integral, smoothedOutput, multiplier)

	return newDifficulty
}

// computeChainIntegral computes the integral term from chain history.
// Scans back up to scanDepth blocks from the parent block's height,
// accumulating each block's time error with exponential decay.
//
// This is the key to determinism: instead of maintaining a running
// accumulator that diverges across nodes, we recompute the integral
// fresh from chain data each time. Given the same chain state, all
// nodes compute the exact same integral value.
func (da *DifficultyAdjuster) computeChainIntegral(parent *Header) float64 {
	if da.getAncestor == nil || parent == nil || parent.Number == nil {
		return 0
	}

	height := parent.Number.Uint64()
	if height == 0 {
		return 0
	}

	targetTime := int64(da.consensusParams.BlockTimeTargetSeconds)
	integral := 0.0
	count := 0

	// Scan back from parent, accumulating errors with decay
	for i := uint64(0); i < uint64(scanDepth) && height > i; i++ {
		block := da.getAncestor(height - i)
		if block == nil {
			break
		}

		var prev *Header
		if height > i+1 {
			prev = da.getAncestor(height - i - 1)
		}
		if prev == nil || prev.Time == 0 {
			continue
		}

		timeDiff := int64(block.Time - prev.Time)
		if timeDiff <= 0 {
			continue
		}

		// Clamp time diff to prevent extreme outliers
		if timeDiff > targetTime*4 {
			timeDiff = targetTime * 4
		}

		// Compute error with sign consistency: positive = too fast.
		err := float64(targetTime-timeDiff) / float64(targetTime)
		err = clampFloat64(err, -0.75, 3.0)

		// Apply decay: older blocks contribute less
		if count > 0 {
			integral = integral*integralDecay + err
		} else {
			integral = err
		}
		count++
	}

	// Apply anti-windup clamp
	integral = clampFloat64(integral, integralClampMin, integralClampMax)

	if count > 0 {
		log.Printf("[ChainIntegral] scanned %d blocks, integral=%.3f", count, integral)
	}

	return integral
}

// trimmedMean computes the mean after discarding the given fraction from both
// the top and bottom of the sorted slice. A trimFraction of 0.2 drops the
// fastest 20% and slowest 20% of block times, providing outlier resistance
// while preserving the central tendency.
func trimmedMean(times []int64, trimFraction float64) int64 {
	if len(times) == 0 {
		return 0
	}
	trim := int(float64(len(times)) * trimFraction)
	if trim*2 >= len(times) {
		trim = 0
	}
	trimmed := times[trim : len(times)-trim]
	var sum int64
	for _, t := range trimmed {
		sum += t
	}
	return sum / int64(len(trimmed))
}

// calculateDoubleExponentialSmoothing computes double exponential smoothing
// deterministically from chain history. Unlike the previous stateful version,
// this recomputes smoothing state from recent block data, ensuring identical
// results across all nodes given the same chain state.
func (da *DifficultyAdjuster) calculateDoubleExponentialSmoothing(avgTime int64, targetTime int64, parent *Header) float64 {
	if da.getAncestor == nil || parent == nil || parent.Number == nil || targetTime <= 0 {
		// Fallback: simple proportional error when chain data unavailable
		if avgTime > 0 {
			return clampFloat64(float64(targetTime-avgTime)/float64(targetTime), -0.75, 3.0)
		}
		return 0
	}

	// Collect per-block errors from chain history
	height := parent.Number.Uint64()
	window := da.windowSize
	if window < 5 {
		window = 5
	}
	if window > maxSmoothingWindow {
		window = maxSmoothingWindow
	}

	errors := make([]float64, 0, window)
	for i := uint64(0); i < uint64(window) && height > i; i++ {
		block := da.getAncestor(height - i)
		if block == nil {
			break
		}
		var prev *Header
		if height > i+1 {
			prev = da.getAncestor(height - i - 1)
		}
		if prev == nil || prev.Time == 0 || block.Time <= prev.Time {
			continue
		}
		timeDiff := int64(block.Time - prev.Time)
		if timeDiff <= 0 || timeDiff > targetTime*10 {
			timeDiff = targetTime // clamp extreme outliers to target
		}
		err := float64(targetTime-timeDiff) / float64(targetTime)
		err = clampFloat64(err, -0.75, 3.0)
		errors = append(errors, err)
	}

	if len(errors) == 0 {
		return 0
	}

	// Apply double exponential smoothing from oldest to newest.
	// errors[0] = most recent block, errors[len-1] = oldest block.
	// Process oldest first for proper DES convergence.
	level := errors[len(errors)-1]
	trend := 0.0

	for i := len(errors) - 2; i >= 0; i-- {
		oldLevel := level
		level = smoothingAlpha*errors[i] + (1-smoothingAlpha)*(level+trend)
		trend = smoothingBeta*(level-oldLevel) + (1-smoothingBeta)*trend
	}

	output := level + trend
	output = clampFloat64(output, -0.75, 3.0)

	log.Printf("[DoubleExpSmoothing_Deterministic] errors=%d, level=%.3f, trend=%.3f, output=%.3f",
		len(errors), level, trend, output)

	return output
}

// clampFloat64 clamps f to [min, max].
func clampFloat64(f, min, max float64) float64 {
	if f < min {
		return min
	}
	if f > max {
		return max
	}
	return f
}

// enforceMinDiff ensures the difficulty is not below the minimum.
func enforceMinDiff(diff *big.Int, minBits uint32) *big.Int {
	minDiff := new(big.Int).SetUint64(uint64(minBits))
	if diff.Cmp(minDiff) < 0 {
		return minDiff
	}
	return diff
}

// enforceBoundaryConditionsLocked applies safety constraints.
// Single-block adjustment capped at ±25% for stability.
// Uses ceiling for max bound to prevent deadlock at small difficulty values
// (e.g. parentDiff=3 → ceil(3*1.25)=4, not floor(3.75)=3).
func (da *DifficultyAdjuster) enforceBoundaryConditionsLocked(newDifficulty, parentDiff *big.Int) *big.Int {
	// 1. Minimum difficulty
	minDiff := big.NewInt(int64(da.consensusParams.MinDifficulty))
	if newDifficulty.Cmp(minDiff) < 0 {
		newDifficulty.Set(minDiff)
	}

	// 2. Maximum difficulty (256-bit cap)
	maxDiff := new(big.Int).Lsh(big.NewInt(1), 256)
	if newDifficulty.Cmp(maxDiff) > 0 {
		newDifficulty.Set(maxDiff)
	}

	// 3. Single-block adjustment cap: ±25%
	// CRITICAL: Use ceiling (add 0.999999 then truncate) for max bound
	// to match the ceiling strategy in calculateDeterministicDifficulty.
	// Without ceiling: parentDiff=3 → floor(3.75)=3 → deadlock (never increases).
	// With ceiling: parentDiff=3 → ceil(3.75)=4 → can increase.
	const maxSingleAdjustment = 1.25
	const minSingleAdjustment = 0.75
	const ceilOffset = 0.999999

	// Compute max allowed with ceiling: ceil(parentDiff * 1.25)
	maxAllowed := new(big.Int).Set(parentDiff)
	maxMultiplier := new(big.Float).SetFloat64(maxSingleAdjustment)
	maxAllowedFloat := new(big.Float).Mul(new(big.Float).SetInt(maxAllowed), maxMultiplier)
	maxCeiled := new(big.Float).Add(maxAllowedFloat, big.NewFloat(ceilOffset))
	maxAllowedInt, _ := maxCeiled.Int(nil)

	// Compute min allowed with ceiling: ceil(parentDiff * 0.75)
	// Use ceiling here too so that the min bound does not overly restrict decreases.
	minAllowed := new(big.Int).Set(parentDiff)
	minMultiplier := new(big.Float).SetFloat64(minSingleAdjustment)
	minAllowedFloat := new(big.Float).Mul(new(big.Float).SetInt(minAllowed), minMultiplier)
	minCeiled := new(big.Float).Add(minAllowedFloat, big.NewFloat(ceilOffset))
	minAllowedInt, _ := minCeiled.Int(nil)

	// Enforce single-block cap
	if newDifficulty.Cmp(maxAllowedInt) > 0 {
		log.Printf("[Boundary] Difficulty capped at +25%%: %v → %v", newDifficulty, maxAllowedInt)
		newDifficulty.Set(maxAllowedInt)
	}
	if newDifficulty.Cmp(minAllowedInt) < 0 {
		log.Printf("[Boundary] Difficulty capped at -25%%: %v → %v", newDifficulty, minAllowedInt)
		newDifficulty.Set(minAllowedInt)
	}

	// 4. Global min difficulty (re-check after cap)
	configMinDiff := big.NewInt(int64(da.consensusParams.MinDifficulty))
	if newDifficulty.Cmp(configMinDiff) < 0 {
		newDifficulty.Set(configMinDiff)
	}

	return newDifficulty
}

// ValidateDifficulty validates difficulty against consensus rules.
func (da *DifficultyAdjuster) ValidateDifficulty(difficulty *big.Int, parent *Header) bool {
	if difficulty == nil || difficulty.Sign() <= 0 {
		return false
	}

	minDiff := big.NewInt(int64(da.consensusParams.MinDifficulty))
	if difficulty.Cmp(minDiff) < 0 {
		return false
	}

	if parent != nil && parent.Difficulty != nil && parent.Difficulty.Sign() > 0 {
		// Loose safety net for already-sealed blocks: max difficulty 2.048x of parent.
		// This is intentionally wider than the ±25% single-step cap used in CalcDifficulty.
		// The 25% cap governs what miners produce; the 2.048x bound here is a defensive
		// check that prevents extreme historical difficulty jumps from rejecting valid
		// blocks that were sealed under previous consensus rules.
		boundDivisor := int64(2048)
		maxAllowed := new(big.Int).Mul(parent.Difficulty, big.NewInt(boundDivisor))
		maxAllowed.Div(maxAllowed, big.NewInt(1000))

		if difficulty.Cmp(maxAllowed) > 0 {
			return false
		}
	}

	return true
}

// GetParameters returns PI controller parameters.
func (da *DifficultyAdjuster) GetParameters() (kp, ki float64) {
	return da.proportionalGain, da.integralGain
}

// SetIntegralGain sets Ki parameter (thread-safe).
func (da *DifficultyAdjuster) SetIntegralGain(ki float64) {
	da.mu.Lock()
	defer da.mu.Unlock()
	da.integralGain = ki
}

// validatePIParameters validates PI controller parameters.
func validatePIParameters(kp, ki float64) error {
	if math.IsNaN(kp) || math.IsInf(kp, 0) {
		return fmt.Errorf("invalid proportional gain: %v", kp)
	}
	if math.IsNaN(ki) || math.IsInf(ki, 0) {
		return fmt.Errorf("invalid integral gain: %v", ki)
	}
	if kp < 0 || kp > 10.0 {
		return fmt.Errorf("proportional gain out of range [0, 10]: %f", kp)
	}
	if ki < 0 || ki > 1.0 {
		return fmt.Errorf("integral gain out of range [0, 1]: %f", ki)
	}
	return nil
}
