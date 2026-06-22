// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"log"
	"math"
	"math/big"
	"sync"
)

// GetAncestorFunc retrieves a block header by its height.
// Retained for API compatibility; v4.0 Kp‑P controller does not use
// ancestor lookups — difficulty is a pure function of parent header
// and block interval.
type GetAncestorFunc func(height uint64) *Header

// Kp-P controller tuning constants — v4.0 simplified architecture.
//
// Replaces the former v3.0 EWMA+P‑only controller with a pure Kp‑P
// (proportional gain) controller.  The gain Kp < 1.0 smooths near‑target
// adjustments while the [0.5, 2.0] clamp serves as a safety net for
// extreme intervals.
//
// Core formula:  adj = 1 + Kp × (ratio − 1)
//                where ratio = targetTime / blockInterval
//
// With Kp = 0.5 and a 60 s target the behaviour is:
//
//	Interval │ Ratio  │ adj (Kp=0.5) │ Clamped? │ Effect
//	─────────┼────────┼───────────────┼──────────┼────────────────
//	  5 s    │ 12.00  │     6.50      │  2.00    │ diff × 2.00  (extreme speed clamp)
//	 40 s    │  1.50  │     1.25      │   —      │ diff × 1.25  (slightly fast)
//	57–63 s  │0.95–1.05│ 0.975–1.025 │   —      │ deadband — no change
//	 90 s    │  0.67  │     0.833     │   —      │ diff × 0.833 (slightly slow)
//	120 s    │  0.50  │     0.75      │   —      │ diff × 0.75  (slow)
//	300 s    │  0.20  │     0.60      │   —      │ diff × 0.60  (chain stalled)
//
// Zero hidden state.  Difficulty is a pure function of the parent header
// and block interval — all nodes reach identical results.
const (
	// kpGain is the proportional gain factor.
	// Kp < 1.0 damps near‑target adjustments while the clamp handles
	// extremes.  A value of 0.5 means a 2× ratio error produces only a
	// 1.5× difficulty change.
	kpGain = 0.5

	// kpClampMin / kpClampMax are the absolute safety bounds on the
	// difficulty adjustment multiplier.  Difficulty cannot change by
	// more than 2× or less than 0.5× in a single block, regardless of
	// how extreme the interval is.
	kpClampMax = 2.0
	kpClampMin = 0.5

	// kpDeadband in ratio space: when |ratio − 1.0| < kpDeadband the
	// difficulty is left unchanged to suppress micro‑oscillations.
	// ±5% corresponds to [57, 63] s for a 60 s target.
	kpDeadband = 0.05
)

// DifficultyAdjuster implements deterministic difficulty adjustment using
// a Kp‑P (proportional gain) controller.
//
// v4.0 Kp‑P architecture:
//  1. ratio = targetTime / blockInterval
//  2. adj   = 1 + Kp × (ratio − 1)
//  3. deadband: |ratio − 1.0| < 5 % → no adjustment
//  4. clamp: [0.5, 2.0] safety net
//
// The controller has zero hidden state — difficulty depends solely on the
// parent header and the block time interval, making it fully deterministic
// across all nodes.  Unlike the former EWMA architecture there are no
// ancestor lookups, eliminating the risk of state divergence between nodes.
type DifficultyAdjuster struct {
	mu sync.Mutex // Protects CalcDifficulty state

	consensusParams *ConsensusParams
}

// NewDifficultyAdjuster creates a new deterministic Kp‑P difficulty adjuster.
func NewDifficultyAdjuster(consensusParams *ConsensusParams) *DifficultyAdjuster {
	if consensusParams == nil {
		consensusParams = &ConsensusParams{
			BlockTimeTargetSeconds:   60,
			MaxDifficultyChangePercent: 100,
		}
	}

	return &DifficultyAdjuster{
		consensusParams: consensusParams,
	}
}

// CalcDifficulty calculates difficulty for the next block using the Kp‑P
// controller.  The result is a pure function of the parent header and the
// block time interval.
//
// v4.0 control chain:
//  1. ratio = targetTime / blockInterval
//  2. adj   = 1 + Kp × (ratio − 1)
//  3. deadband: |ratio − 1.0| < 5 % → return parent difficulty unchanged
//  4. clamp: adj ∈ [0.5, 2.0]
//  5. output: newDiff = parentDiff × adj  (ceiling for integer safety)
//
// Parameters:
//   - currentTime: Unix timestamp of the new block (seconds since epoch)
//   - parent:      Previous block header (nil for genesis)
//
// Returns:
//   - *big.Int: New difficulty value (arbitrary‑precision integer)
func (da *DifficultyAdjuster) CalcDifficulty(currentTime uint64, parent *Header) *big.Int {
	// Lock protection — concurrency safety.
	da.mu.Lock()
	defer da.mu.Unlock()

	// Genesis / nil parent: return minimum difficulty.
	if parent == nil || parent.Difficulty == nil {
		minDiff := big.NewInt(1)
		if da.consensusParams.MinDifficulty > 0 {
			minDiff = big.NewInt(int64(da.consensusParams.MinDifficulty))
		}
		log.Printf("[Difficulty] Genesis block: using minimum difficulty %s", minDiff.String())
		return minDiff
	}

	parentDiff := new(big.Int).Set(parent.Difficulty)
	if parentDiff.Sign() <= 0 {
		parentDiff = big.NewInt(1)
	}

	targetTime := int64(da.consensusParams.BlockTimeTargetSeconds)
	if targetTime <= 0 {
		targetTime = 60 // defensive fallback
	}

	// ── 1. Block interval ──
	interval := int64(currentTime) - int64(parent.Time)
	if interval < 1 {
		interval = 1 // defensive: minimum 1 s
	}

	// ── 2. Ratio ──
	// ratio > 1 → blocks too fast → increase difficulty
	// ratio < 1 → blocks too slow → decrease difficulty
	ratio := float64(targetTime) / float64(interval)

	// ── 3. Deadband ±5 % ──
	// Suppress micro‑oscillations when the interval is near target.
	if math.Abs(ratio-1.0) < kpDeadband {
		log.Printf("[Difficulty] deadband: interval=%ds, target=%ds, ratio=%.3f (no change)",
			interval, targetTime, ratio)
		return parentDiff
	}

	// ── 4. Kp‑P controller ──
	// adj = 1 + Kp × (ratio − 1)
	// With Kp = 0.5 a 2× ratio error only produces a 1.5× adjustment,
	// avoiding over‑correction near target while the clamp handles
	// extreme intervals.
	adj := 1.0 + kpGain*(ratio-1.0)

	// ── 5. Clamp [0.5, 2.0] safety net ──
	if adj > kpClampMax {
		adj = kpClampMax
	} else if adj < kpClampMin {
		adj = kpClampMin
	}

	// ── 6. Output: parentDiff × adj with ceiling ──
	newDiffFloat := new(big.Float).Mul(
		new(big.Float).SetInt(parentDiff),
		big.NewFloat(adj),
	)
	ceiled := new(big.Float).Add(newDiffFloat, big.NewFloat(0.999999))
	result, _ := ceiled.Int(nil)
	if result.Sign() <= 0 {
		result = big.NewInt(1)
	}

	// ── 7. Minimum difficulty enforcement ──
	if da.consensusParams.MinDifficulty > 0 {
		minDiff := big.NewInt(int64(da.consensusParams.MinDifficulty))
		if result.Cmp(minDiff) < 0 {
			result.Set(minDiff)
		}
	}

	// Percentage change for logging.
	var changePct float64
	parentFloat, _ := new(big.Float).SetInt(parentDiff).Float64()
	resultFloat, _ := new(big.Float).SetInt(result).Float64()
	if parentFloat > 0 {
		changePct = (resultFloat - parentFloat) / parentFloat * 100
	}

	log.Printf("[Difficulty] interval=%ds, target=%ds, ratio=%.3f, adj=%.3f, diff %s→%s (%.1f%%)",
		interval, targetTime, ratio, adj, parentDiff.String(), result.String(), changePct)

	return result
}

// ValidateDifficulty validates difficulty against consensus rules.
// This is a loose check for already‑sealed blocks: the 2.048× bound
// prevents rejecting valid blocks that were sealed under previous
// consensus rules.
func (da *DifficultyAdjuster) ValidateDifficulty(difficulty *big.Int, parent *Header) bool {
	if difficulty == nil || difficulty.Sign() <= 0 {
		return false
	}

	if da.consensusParams.MinDifficulty > 0 {
		minDiff := big.NewInt(int64(da.consensusParams.MinDifficulty))
		if difficulty.Cmp(minDiff) < 0 {
			return false
		}
	}

	if parent != nil && parent.Difficulty != nil && parent.Difficulty.Sign() > 0 {
		// Defensive check: max 2.048× of parent.
		// This is intentionally wider than the [0.5, 2.0] production
		// clamp used in CalcDifficulty.  It prevents extreme historical
		// difficulty jumps from rejecting valid blocks sealed under
		// previous consensus.
		boundDivisor := int64(2048)
		maxAllowed := new(big.Int).Mul(parent.Difficulty, big.NewInt(boundDivisor))
		maxAllowed.Div(maxAllowed, big.NewInt(1000))

		if difficulty.Cmp(maxAllowed) > 0 {
			return false
		}
	}

	return true
}

// GetAncestorFunc returns nil — v4.0 Kp‑P controller has no ancestor
// dependency. Retained for API backward compatibility.
func (da *DifficultyAdjuster) GetAncestorFunc() GetAncestorFunc {
	return nil
}

// SetAncestorFunc is a no-op — v4.0 Kp‑P controller does not use
// ancestor lookups. Retained for API backward compatibility.
func (da *DifficultyAdjuster) SetAncestorFunc(fn GetAncestorFunc) {
	// No-op: Kp‑P controller is stateless.
}

// GetParameters returns the Kp‑P controller parameters:
//
//	kpGain (proportional gain) and kpDeadband (ratio‑space threshold).
//
// Retained for backward compatibility with diagnostic callers.
func (da *DifficultyAdjuster) GetParameters() (gain float64, deadband float64) {
	return kpGain, kpDeadband
}
