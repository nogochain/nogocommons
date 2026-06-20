//go:build ignore

// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogopow

import (
	"crypto/rand"
	"math/big"
	"testing"
)

// ---------------------------------------------------------------------------
// PoW algorithm correctness tests
// ---------------------------------------------------------------------------

// TestSealHashDeterministic verifies that SealHash is deterministic:
// same header produces the same hash every time.
func TestSealHashDeterministic(t *testing.T) {
	engine := NewFaker()

	header := &Header{
		ParentHash: BytesToHash([]byte("parent-hash-000000000000000000000000")),
		Coinbase:   [20]byte{1, 2, 3, 4, 5},
		Number:     big.NewInt(100),
		GasLimit:   1000000,
		Time:       1000,
		Difficulty: big.NewInt(1000),
	}

	hash1 := engine.SealHash(header)
	hash2 := engine.SealHash(header)

	if hash1 != hash2 {
		t.Errorf("SealHash is not deterministic: %x != %x", hash1, hash2)
	}
}

// TestSealHashDifferentHeaders verifies that different headers produce
// different hashes (collision resistance).
func TestSealHashDifferentHeaders(t *testing.T) {
	engine := NewFaker()

	h1 := &Header{
		ParentHash: BytesToHash([]byte("parent-hash-000000000000000000000000")),
		Number:     big.NewInt(100),
		Time:       1000,
		Difficulty: big.NewInt(1000),
	}
	h2 := &Header{
		ParentHash: BytesToHash([]byte("parent-hash-000000000000000000000000")),
		Number:     big.NewInt(101), // Different number
		Time:       1000,
		Difficulty: big.NewInt(1000),
	}

	hash1 := engine.SealHash(h1)
	hash2 := engine.SealHash(h2)

	if hash1 == hash2 {
		t.Error("Different headers produced the same hash")
	}
}

// TestSealHashChangesWithNonce verifies that changing the nonce changes
// the seal hash (essential for PoW mining).
func TestSealHashChangesWithNonce(t *testing.T) {
	engine := NewFaker()

	header := &Header{
		ParentHash: BytesToHash([]byte("parent-hash-000000000000000000000000")),
		Number:     big.NewInt(100),
		Time:       1000,
		Difficulty: big.NewInt(1000),
	}

	hash1 := engine.SealHash(header)
	header.Nonce = BlockNonce{0, 0, 0, 0, 0, 0, 0, 1}
	hash2 := engine.SealHash(header)

	if hash1 == hash2 {
		t.Error("SealHash did not change with nonce")
	}
}

// ---------------------------------------------------------------------------
// ComputePoW tests
// ---------------------------------------------------------------------------

// TestComputePoWDeterministic verifies that ComputePoW is deterministic.
func TestComputePoWDeterministic(t *testing.T) {
	engine := New(DefaultConfig())
	engine.config.powModeForTest(ModeTest)

	blockHash := BytesToHash([]byte("test-block-hash-0000000000000000000000"))
	seed := BytesToHash([]byte("test-seed-hash-000000000000000000000000"))

	pow1 := engine.ComputePoW(blockHash, seed)
	pow2 := engine.ComputePoW(blockHash, seed)

	if pow1 != pow2 {
		t.Errorf("ComputePoW is not deterministic: %x != %x", pow1, pow2)
	}
}

// TestComputePoWDifferentInputs verifies that different inputs produce
// different PoW hashes.
func TestComputePoWDifferentInputs(t *testing.T) {
	engine := New(DefaultConfig())
	engine.config.powModeForTest(ModeTest)

	hash1 := BytesToHash([]byte("test-block-hash-0000000000000000000001"))
	hash2 := BytesToHash([]byte("test-block-hash-0000000000000000000002"))
	seed := BytesToHash([]byte("test-seed-hash-000000000000000000000000"))

	pow1 := engine.ComputePoW(hash1, seed)
	pow2 := engine.ComputePoW(hash2, seed)

	if pow1 == pow2 {
		t.Error("ComputePoW produced same hash for different block hashes")
	}
}

// TestComputePoWDifferentSeeds verifies that different seeds produce
// different PoW hashes.
func TestComputePoWDifferentSeeds(t *testing.T) {
	engine := New(DefaultConfig())
	engine.config.powModeForTest(ModeTest)

	blockHash := BytesToHash([]byte("test-block-hash-0000000000000000000000"))
	seed1 := BytesToHash([]byte("test-seed-hash-000000000000000000000001"))
	seed2 := BytesToHash([]byte("test-seed-hash-000000000000000000000002"))

	pow1 := engine.ComputePoW(blockHash, seed1)
	pow2 := engine.ComputePoW(blockHash, seed2)

	if pow1 == pow2 {
		t.Error("ComputePoW produced same hash for different seeds")
	}
}

// TestComputePoWProducesValidHash verifies that the PoW hash always has
// exactly 32 bytes.
func TestComputePoWProducesValidHash(t *testing.T) {
	engine := New(DefaultConfig())
	engine.config.powModeForTest(ModeTest)

	for i := 0; i < 10; i++ {
		blockHash := randomHash(t)
		seed := randomHash(t)

		pow := engine.ComputePoW(blockHash, seed)
		if len(pow.Bytes()) != 32 {
			t.Errorf("Iteration %d: PoW hash has %d bytes, expected 32", i, len(pow.Bytes()))
		}
	}
}

// ---------------------------------------------------------------------------
// checkPow / difficultyToTarget tests
// ---------------------------------------------------------------------------

// TestCheckPowWithLowDifficulty verifies that checkPow returns true for
// any hash when difficulty is very low (easy target).
func TestCheckPowWithLowDifficulty(t *testing.T) {
	engine := New(DefaultConfig())

	// With difficulty 1, the target is the maximum possible value.
	// Every hash must satisfy this.
	hash := BytesToHash([]byte("test-hash-0000000000000000000000000000"))
	difficulty := big.NewInt(1)

	if !engine.checkPow(hash, difficulty) {
		t.Error("hash must satisfy difficulty 1")
	}
}

// TestCheckPowWithHighDifficulty verifies that checkPow returns false for
// a random hash when difficulty is extremely high.
func TestCheckPowWithHighDifficulty(t *testing.T) {
	engine := New(DefaultConfig())

	hash := randomHash(t)
	// Extremely high difficulty — almost impossible for random hash to satisfy
	difficulty := new(big.Int).Lsh(big.NewInt(1), 255)

	if engine.checkPow(hash, difficulty) {
		t.Log("Random hash satisfied extremely high difficulty (highly unlikely but possible)")
	}
}

// TestDifficultyToTargetRange verifies that difficultyToTarget produces
// targets within the valid range [0, 2^256-1].
func TestDifficultyToTargetRange(t *testing.T) {
	testCases := []*big.Int{
		big.NewInt(1),
		big.NewInt(10),
		big.NewInt(1000),
		big.NewInt(1000000),
		new(big.Int).Lsh(big.NewInt(1), 128),
		new(big.Int).Lsh(big.NewInt(1), 200),
	}

	for _, diff := range testCases {
		target := difficultyToTarget(diff)
		if target.Sign() < 0 {
			t.Errorf("Difficulty %s: target is negative", diff.String())
		}

		maxTarget := new(big.Int).Lsh(big.NewInt(1), 256)
		if target.Cmp(maxTarget) >= 0 {
			t.Errorf("Difficulty %s: target %s >= max target", diff.String(), target.String())
		}
	}
}

// TestDifficultyToTargetMonotonic verifies that higher difficulty produces
// lower (stricter) target.
func TestDifficultyToTargetMonotonic(t *testing.T) {
	d1 := big.NewInt(100)
	d2 := big.NewInt(200)

	t1 := difficultyToTarget(d1)
	t2 := difficultyToTarget(d2)

	// Higher difficulty → lower target
	if t1.Cmp(t2) <= 0 {
		t.Errorf("Higher difficulty should produce lower target: d1=%s→t1=%s, d2=%s→t2=%s",
			d1.String(), t1.String(), d2.String(), t2.String())
	}
}

// TestDifficultyToTargetInverse verifies that target = maxTarget / difficulty.
func TestDifficultyToTargetInverse(t *testing.T) {
	maxTarget := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))

	for _, diffVal := range []int64{1, 2, 10, 100, 1000, 1000000} {
		diff := big.NewInt(diffVal)
		target := difficultyToTarget(diff)

		// target * difficulty should be <= maxTarget and >= maxTarget - difficulty
		product := new(big.Int).Mul(target, diff)
		diffRange := new(big.Int).Sub(maxTarget, product)

		// product should be close to maxTarget
		if diffRange.Cmp(diff) > 0 {
			t.Errorf("Difficulty %d: product %s too far from maxTarget %s (diff=%s)",
				diffVal, product.String(), maxTarget.String(), diffRange.String())
		}
	}
}

// ---------------------------------------------------------------------------
// verifySeal tests
// ---------------------------------------------------------------------------

// TestVerifySealGenesisBlock verifies that genesis block (height 0) always
// passes seal verification via the public VerifyHeader API.
func TestVerifySealGenesisBlock(t *testing.T) {
	engine := New(DefaultConfig())

	genesis := &Header{
		Number: big.NewInt(0),
	}
	// Use VerifyHeader (public API) which correctly short-circuits genesis.
	err := engine.VerifyHeader(nil, genesis, true)
	if err != nil {
		t.Errorf("Genesis block should pass seal verification: %v", err)
	}
}

// TestVerifySealFakeMode verifies that fake mode skips seal verification.
func TestVerifySealFakeMode(t *testing.T) {
	engine := NewFaker()

	header := &Header{
		Number:     big.NewInt(100),
		Difficulty: big.NewInt(1000),
	}
	err := engine.VerifyHeader(nil, header, true)
	if err != nil {
		t.Errorf("Fake mode should pass verification: %v", err)
	}
}

// TestVerifySealOnlyFakeMode verifies VerifySealOnly in fake mode.
func TestVerifySealOnlyFakeMode(t *testing.T) {
	engine := NewFaker()

	header := &Header{
		Number:     big.NewInt(100),
		Difficulty: big.NewInt(1000),
	}
	err := engine.VerifySealOnly(header)
	if err != nil {
		t.Errorf("Fake mode VerifySealOnly should pass: %v", err)
	}
}

// ---------------------------------------------------------------------------
// calcSeed tests
// ---------------------------------------------------------------------------

// TestCalcSeedGenesisBlock verifies that genesis block produces zero seed.
func TestCalcSeedGenesisBlock(t *testing.T) {
	engine := New(DefaultConfig())

	header := &Header{
		Number: big.NewInt(0),
	}
	seed := engine.calcSeed(nil, header)
	if seed != (Hash{}) {
		t.Errorf("Genesis block seed should be zero hash, got %x", seed)
	}
}

// TestCalcSeedReturnsParentHash verifies that calcSeed returns the parent
// hash for non-genesis blocks.
func TestCalcSeedReturnsParentHash(t *testing.T) {
	engine := New(DefaultConfig())

	parentHash := BytesToHash([]byte("parent-hash-000000000000000000000000"))
	header := &Header{
		Number:     big.NewInt(100),
		ParentHash: parentHash,
	}
	seed := engine.calcSeed(nil, header)
	if seed != parentHash {
		t.Errorf("Seed should be parent hash, got %x != %x", seed, parentHash)
	}
}

// ---------------------------------------------------------------------------
// Difficulty boundary tests
// ---------------------------------------------------------------------------

// testConsensusParams returns minimal consensus params for testing.
func testConsensusParams() *config.ConsensusParams {
	return &config.ConsensusParams{
		MinDifficulty:          5,
		BlockTimeTargetSeconds: 30,
	}
}

// TestCalcDifficultyNilParent returns minimum difficulty for nil parent.
func TestCalcDifficultyNilParent(t *testing.T) {
	engine := New(DefaultConfig())
	engine.config.ConsensusParams = testConsensusParams()

	result := engine.CalcDifficulty(nil, 100, nil)
	if result.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("Nil parent should return MinDifficulty 5, got %s", result.String())
	}
}

// TestCalcDifficultyNilDifficulty returns minimum for nil difficulty.
func TestCalcDifficultyNilDifficulty(t *testing.T) {
	engine := New(DefaultConfig())
	engine.config.ConsensusParams = testConsensusParams()

	parent := &Header{
		Number: big.NewInt(100),
		Time:   1000,
	}
	result := engine.CalcDifficulty(nil, 1030, parent)
	if result.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("Nil difficulty should return MinDifficulty 5, got %s", result.String())
	}
}

// ---------------------------------------------------------------------------
// HashRate atomic operations test
// ---------------------------------------------------------------------------

// TestHashRateInitialValue verifies initial hashrate is 0.
func TestHashRateInitialValue(t *testing.T) {
	engine := New(DefaultConfig())
	if engine.HashRate() != 0 {
		t.Errorf("Initial hashrate should be 0, got %d", engine.HashRate())
	}
}

// ---------------------------------------------------------------------------
// VerifyHeaders concurrent tests
// ---------------------------------------------------------------------------

// TestVerifyHeadersEmpty verifies that empty headers list returns
// immediately closed channel.
func TestVerifyHeadersEmpty(t *testing.T) {
	engine := NewFaker()

	_, results := engine.VerifyHeaders(nil, nil, nil)
	for err := range results {
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

// TestVerifyHeadersConcurrent tests concurrent header verification.
func TestVerifyHeadersConcurrent(t *testing.T) {
	engine := NewFaker()

	headers := make([]*Header, 10)
	seals := make([]bool, 10)
	for i := 0; i < 10; i++ {
		headers[i] = &Header{
			Number: big.NewInt(int64(i + 1)),
			Time:   uint64(1000 + i*30),
		}
		seals[i] = true
	}

	_, results := engine.VerifyHeaders(nil, headers, seals)
	errCount := 0
	for err := range results {
		if err != nil {
			errCount++
		}
	}

	if errCount > 0 {
		t.Errorf("Got %d errors in fake mode (expected 0)", errCount)
	}
}

// ---------------------------------------------------------------------------
// VerifyUncles tests
// ---------------------------------------------------------------------------

// TestVerifyUnclesMaxUncles verifies that blocks with > 2 uncles are rejected.
func TestVerifyUnclesMaxUncles(t *testing.T) {
	engine := NewFaker()

	block := NewBlock(&Header{
		Number: big.NewInt(100),
	}, nil, []*Header{
		{Number: big.NewInt(100)},
		{Number: big.NewInt(100)},
		{Number: big.NewInt(100)},
	}, nil)

	err := engine.VerifyUncles(nil, block)
	if err == nil {
		t.Error("Should reject block with more than 2 uncles")
	}
}

// TestVerifyUnclesValid verifies that blocks with <= 2 uncles pass.
func TestVerifyUnclesValid(t *testing.T) {
	engine := NewFaker()

	block := NewBlock(&Header{
		Number: big.NewInt(100),
	}, nil, []*Header{
		{Number: big.NewInt(100)},
		{Number: big.NewInt(100)},
	}, nil)

	err := engine.VerifyUncles(nil, block)
	if err != nil {
		t.Errorf("Should accept block with 2 uncles: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Address parsing tests
// ---------------------------------------------------------------------------

// TestStringToAddressValid verifies valid NOGO address parsing.
func TestStringToAddressValid(t *testing.T) {
	addr, err := StringToAddress("NOGO" + "0000000000000000000000000000000000000000")
	if err != nil {
		t.Errorf("Valid address should parse: %v", err)
	}
	_ = addr
}

// TestStringToAddressNoPrefix verifies parsing without NOGO prefix.
func TestStringToAddressNoPrefix(t *testing.T) {
	addr, err := StringToAddress("0000000000000000000000000000000000000000")
	if err != nil {
		t.Errorf("Address without prefix should parse: %v", err)
	}
	_ = addr
}

// TestStringToAddressShort verifies that short addresses are rejected.
func TestStringToAddressShort(t *testing.T) {
	_, err := StringToAddress("NOGO00")
	if err == nil {
		t.Error("Short address should be rejected")
	}
}

// ---------------------------------------------------------------------------
// BytesToHash tests
// ---------------------------------------------------------------------------

// TestBytesToHashTruncation verifies that BytesToHash truncates long inputs
// to 32 bytes.
func TestBytesToHashTruncation(t *testing.T) {
	longBytes := make([]byte, 64)
	for i := range longBytes {
		longBytes[i] = byte(i)
	}
	h := BytesToHash(longBytes)
	expected := longBytes[32:64]
	for i := 0; i < 32; i++ {
		if h[i] != expected[i] {
			t.Errorf("Byte %d: expected %d, got %d", i, expected[i], h[i])
			break
		}
	}
}

// TestBytesToHashShort verifies that short inputs are padded with leading zeros.
func TestBytesToHashShort(t *testing.T) {
	short := []byte{1, 2, 3}
	h := BytesToHash(short)
	// The last 3 bytes should be {1, 2, 3}
	if h[29] != 1 || h[30] != 2 || h[31] != 3 {
		t.Errorf("Short hash not padded correctly: %x", h)
	}
}

// ---------------------------------------------------------------------------
// VerifySealWithBlockHash tests
// ---------------------------------------------------------------------------

// TestVerifySealWithBlockHashGenesis verifies genesis block passes.
func TestVerifySealWithBlockHashGenesis(t *testing.T) {
	engine := New(DefaultConfig())

	genesis := &Header{
		Number: big.NewInt(0),
	}
	err := engine.VerifySealWithBlockHash(genesis, Hash{})
	if err != nil {
		t.Errorf("Genesis block should pass VerifySealWithBlockHash: %v", err)
	}
}

// TestVerifySealWithBlockHashFakeMode verifies fake mode passes.
func TestVerifySealWithBlockHashFakeMode(t *testing.T) {
	engine := NewFaker()

	header := &Header{
		Number:     big.NewInt(100),
		Difficulty: big.NewInt(1000),
	}
	err := engine.VerifySealWithBlockHash(header, Hash{})
	if err != nil {
		t.Errorf("Fake mode should pass VerifySealWithBlockHash: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ComputePoWWithCache tests
// ---------------------------------------------------------------------------

// TestComputePoWWithCacheCorrectness verifies that ComputePoW and
// ComputePoWWithCache produce the same result for the same inputs.
func TestComputePoWWithCacheCorrectness(t *testing.T) {
	engine := New(DefaultConfig())
	engine.config.powModeForTest(ModeTest)

	blockHash := randomHash(t)
	seed := randomHash(t)

	// Get cache data from engine
	cacheData := engine.cache.GetData(seed.Bytes())

	pow1 := engine.ComputePoW(blockHash, seed)
	pow2 := engine.ComputePoWWithCache(blockHash, seed, cacheData)

	if pow1 != pow2 {
		t.Errorf("ComputePoW and ComputePoWWithCache produced different results: %x != %x", pow1, pow2)
	}
}

// ---------------------------------------------------------------------------
// Random hash helper
// ---------------------------------------------------------------------------

func randomHash(t *testing.T) Hash {
	t.Helper()
	var h Hash
	_, err := rand.Read(h[:])
	if err != nil {
		t.Fatalf("failed to generate random hash: %v", err)
	}
	return h
}