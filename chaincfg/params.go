// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

// Package chaincfg defines network parameters for the NogoCore chain.
package chaincfg

import (
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/nogochain/nogocommons/address/base58"
	"github.com/nogochain/nogocommons/chainhash"
	"github.com/nogochain/nogocommons/wire"
)

// DNSSeed identifies a DNS seed node.
type DNSSeed struct {
	Host         string
	HasFiltering bool
}

// Checkpoint identifies a known good point in the block chain for fast initial sync.
type Checkpoint struct {
	Height int32
	Hash   *chainhash.Hash
}

// ConsensusDeployment is retained for BIP9-style soft fork compatibility.
type ConsensusDeployment struct {
	BitNumber                uint8
	MinActivationHeight      uint32
	CustomActivationThreshold uint32
	AlwaysActiveHeight        uint32
	DeploymentStarter         ConsensusDeploymentStarter
	DeploymentEnder           ConsensusDeploymentEnder
}

// ConsensusDeploymentStarter defines the interface for determining if a deployment has started.
type ConsensusDeploymentStarter interface {
	HasStarted(interface{}) (bool, error)
}

// ConsensusDeploymentEnder defines the interface for determining if a deployment has ended.
type ConsensusDeploymentEnder interface {
	HasEnded(interface{}) (bool, error)
}

// BlockClock is an abstraction over the past median time computation.
type BlockClock interface {
	// PastMedianTime returns the past median time from the PoV of the
	// passed block header.
	PastMedianTime(*wire.BlockHeader) (time.Time, error)
}

// ClockConsensusDeploymentStarter is a ConsensusDeploymentStarter that uses
// a BlockClock to determine if a deployment has started.
type ClockConsensusDeploymentStarter interface {
	ConsensusDeploymentStarter
	SynchronizeClock(clock BlockClock)
}

// ClockConsensusDeploymentEnder is a ConsensusDeploymentEnder that uses
// a BlockClock to determine if a deployment has ended.
type ClockConsensusDeploymentEnder interface {
	ConsensusDeploymentEnder
	SynchronizeClock(clock BlockClock)
}

// Deployment constants for BIP-style soft fork version bits.
const (
	DeploymentTestDummy = iota
	DeploymentTestDummyMinActivation
	DeploymentCSV
	DeploymentSegwit
	DeploymentTaproot
	DeploymentTestDummyAlwaysActive
	DefinedDeployments
)

// Params defines a NogoCore network configuration.
// Compatible with btcd peer/wire/btcutil modules.
type Params struct {
	Name        string
	Net         wire.BitcoinNet
	DefaultPort string
	DNSSeeds    []DNSSeed

	// Genesis
	GenesisBlock    *wire.MsgBlock
	GenesisHash  *chainhash.Hash

	// Checkpoints for fast initial sync
	Checkpoints []Checkpoint

	// Address encoding prefixes (Bitcoin-compatible for secp256k1)
	PubKeyHashAddrID        byte
	ScriptHashAddrID        byte
	PrivateKeyID            byte
	WitnessPubKeyHashAddrID byte
	WitnessScriptHashAddrID byte
	Bech32HRPSegwit         string

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte
	HDCoinType     uint32

	// Consensus
	TargetBlockTime    time.Duration
	MaxBlockSize       uint64
	MaxTxSize          uint64
	MaxTxPerBlock      int
	CoinbaseMaturity   int64

	// BIP deployments
	Deployments [DefinedDeployments]ConsensusDeployment

	// BIP activation heights (Bitcoin compatibility stubs)
	BIP0034Hash          *chainhash.Hash
	BIP0034Height        int32
	BIP0065Height        int32
	BIP0066Height        int32
	EnforceBIP94         bool
	ReduceMinDifficulty  bool

	SubsidyReductionInterval    int32
	RuleChangeActivationThreshold uint32
	MinerConfirmationWindow     int32
	TargetTimespan              time.Duration
	TargetTimePerBlock          time.Duration
	RetargetAdjustmentFactor    int64

	// NogoPow
	MinDifficulty         uint32
	MaxDifficulty         uint32
	PowLimit              *big.Int
	PowLimitBits          uint32
	GenesisDifficultyBits uint32
	GenerateSupported     bool // CPU mining support flag (false = regtest only)

	// Economic Model (NogoCore-Specific)
	PreAllocation       int64
	InitialBlockReward  int64
	AnnualReductionRate float64
	AnnualBlockCount    int64
	MinimumBlockReward  int64
	GenesisAddressShare int64
	BurnFees            bool
	GenesisAddress      string
	ShareAddress        string
}

// EffectiveAlwaysActiveHeight returns the AlwaysActiveHeight, or MaxUint32 if unset.
func (d *ConsensusDeployment) EffectiveAlwaysActiveHeight() uint32 {
	if d.AlwaysActiveHeight == 0 {
		return ^uint32(0)
	}
	return d.AlwaysActiveHeight
}

// Network magic constants.
const (
	MainNetMagic wire.BitcoinNet = 0xe3b0c442
	TestNetMagic wire.BitcoinNet = 0xf1a3b5d7
)

// NogoPow limit values.
var (
	NogoPowLimitMain = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
	NogoPowLimitTest = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
)

// HD key ID registration.
var (
	ErrUnknownHDKeyID = errors.New("unknown hd private extended key bytes")
	ErrInvalidHDKeyID = errors.New("invalid hd key id: must be exactly 4 bytes")

	hdPrivToPubKeyIDs = make(map[[4]byte][]byte)
	hdKeyIDMu         sync.Mutex
)

// RegisterHDKeyID registers public and private HD key IDs for address derivation.
func RegisterHDKeyID(hdPublicKeyID []byte, hdPrivateKeyID []byte) error {
	hdKeyIDMu.Lock()
	defer hdKeyIDMu.Unlock()

	if len(hdPublicKeyID) != 4 || len(hdPrivateKeyID) != 4 {
		return ErrInvalidHDKeyID
	}

	var keyID [4]byte
	copy(keyID[:], hdPrivateKeyID)
	hdPrivToPubKeyIDs[keyID] = hdPublicKeyID

	return nil
}

// HDPrivateKeyToPublicKeyID accepts a private hierarchical deterministic
// extended key id and returns the associated public key id.
func HDPrivateKeyToPublicKeyID(id []byte) ([]byte, error) {
	hdKeyIDMu.Lock()
	defer hdKeyIDMu.Unlock()

	if len(id) != 4 {
		return nil, ErrUnknownHDKeyID
	}

	var key [4]byte
	copy(key[:], id)

	pubBytes, ok := hdPrivToPubKeyIDs[key]
	if !ok {
		return nil, ErrUnknownHDKeyID
	}

	return pubBytes, nil
}

// genesisCoinbaseAddress is the NOGO address that receives the genesis
// block coinbase output (1M total supply) and 1% per-block share.
const genesisCoinbaseAddress = "1Dqbr8L29Yirs8EVhNmmikB4DLMwU5Akm7"

// genesisSubsidy is the total premine in satoshis (1M NOGO).
const genesisSubsidy int64 = 100000000000000 // 1,000,000 NOGO

// nocoGenesisCoinbaseScript is the P2PKH script built from genesisCoinbaseAddress.
var nocoGenesisCoinbaseScript []byte

func init() {
	// Decode genesis coinbase address and build P2PKH script.
	pubkeyHash, _, err := base58.CheckDecode(genesisCoinbaseAddress)
	if err != nil || len(pubkeyHash) != 20 {
		panic("genesisCoinbaseAddress: invalid P2PKH address: " + genesisCoinbaseAddress)
	}
	// P2PKH: OP_DUP OP_HASH160 <20 bytes> OP_EQUALVERIFY OP_CHECKSIG
	nocoGenesisCoinbaseScript = make([]byte, 25)
	nocoGenesisCoinbaseScript[0] = 0x76 // OP_DUP
	nocoGenesisCoinbaseScript[1] = 0xa9 // OP_HASH160
	nocoGenesisCoinbaseScript[2] = 0x14 // push 20 bytes
	copy(nocoGenesisCoinbaseScript[3:23], pubkeyHash)
	nocoGenesisCoinbaseScript[23] = 0x88 // OP_EQUALVERIFY
	nocoGenesisCoinbaseScript[24] = 0xac // OP_CHECKSIG

	// Patch genesis block TxOut PkScript — the var initializer ran before
	// init(), so nocoGenesisCoinbaseScript was nil at block construction time.
	nocoGenesisBlock.Transactions[0].TxOut[0].PkScript = nocoGenesisCoinbaseScript

	// Recompute Merkle root (changed because genesis TxOut was added).
	nocoGenesisBlock.Header.MerkleRoot = nocoGenesisBlock.Transactions[0].TxHash()

	genesisHash := nocoGenesisBlock.BlockHash()
	MainNetParams.GenesisHash = &genesisHash
	TestNet3Params.GenesisHash = &genesisHash
	TestNet4Params.GenesisHash = &genesisHash
	RegressionNetParams.GenesisHash = &genesisHash
	SigNetParams.GenesisHash = &genesisHash
	SimNetParams.GenesisHash = &genesisHash

	RegisterHDKeyID(MainNetParams.HDPublicKeyID[:], MainNetParams.HDPrivateKeyID[:])
	RegisterHDKeyID(TestNet3Params.HDPublicKeyID[:], TestNet3Params.HDPrivateKeyID[:])
	RegisterHDKeyID(TestNet4Params.HDPublicKeyID[:], TestNet4Params.HDPrivateKeyID[:])
	// Regtest/SimNet/SigNet share TestNet BIP32 IDs.
	RegisterHDKeyID(RegressionNetParams.HDPublicKeyID[:], RegressionNetParams.HDPrivateKeyID[:])
	RegisterHDKeyID(SigNetParams.HDPublicKeyID[:], SigNetParams.HDPrivateKeyID[:])
	RegisterHDKeyID(SimNetParams.HDPublicKeyID[:], SimNetParams.HDPrivateKeyID[:])
}

// MainNetParams defines the NogoCore main network configuration.
// nocoGenesisBlock is the hardcoded genesis block for NogoCore.
var nocoGenesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},
		MerkleRoot: chainhash.Hash{},
		Timestamp:  time.Unix(1750262400, 0), // 2026-06-19T00:00:00Z
		Bits:       0x207fffff,
		Nonce:      0,
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: wire.MaxPrevOutIndex,
				},
				SignatureScript: []byte("NogoCore Genesis Block"),
			}},
			TxOut: []*wire.TxOut{{
				Value:    genesisSubsidy,       // 1,000,000 NOGO
				PkScript: nocoGenesisCoinbaseScript,
			}},
		},
	},
}

var MainNetParams = Params{
	Name:        "mainnet",
	Net:         MainNetMagic,
	DefaultPort: "19444",
	DNSSeeds:    []DNSSeed{},
	GenesisBlock: &nocoGenesisBlock,

	// Address encoding (Bitcoin-compatible prefixes for secp256k1)
	PubKeyHashAddrID:        0x00, // starts with 1
	ScriptHashAddrID:        0x05, // starts with 3 (P2SH)
	PrivateKeyID:            0x80, // starts with 5 or K/L
	WitnessPubKeyHashAddrID: 0x06, // starts with p2
	WitnessScriptHashAddrID: 0x0A, // starts with 7Xh
	Bech32HRPSegwit:         "nc", // NogoCore mainnet Bech32 HRP (nc = NogoCash)

	// BIP32 extended key magics (compatible with Bitcoin mainnet)
	HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4}, // xprv
	HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e}, // xpub
	HDCoinType:     0,

	// Consensus
	TargetBlockTime:        60 * time.Second,
	TargetTimespan:         14 * 24 * time.Hour, // 2 weeks
	TargetTimePerBlock:     60 * time.Second,
	RetargetAdjustmentFactor: 4,
	MaxBlockSize:           8 * 1024 * 1024,  // 8MB
	MaxTxSize:        1 * 1024 * 1024,  // 1MB
	MaxTxPerBlock:    4096,
	CoinbaseMaturity: 100,

	// NogoPow — start at minimum difficulty for fast bootstrap, then
	// difficulty retarget converges to 60-second blocks automatically.
	//   GenesisDifficultyBits = 0x207fffff → target 2^255 (easiest)
	//   Each block: ratio = 60s / actualTime, clamped to [¼, 4×]
	//   Fast blocks → difficulty rises; slow blocks → difficulty falls
	MinDifficulty:         1,
	MaxDifficulty:         1 << 30,
	ReduceMinDifficulty:   false,          // use full retarget logic
	PowLimit:              NogoPowLimitMain,
	PowLimitBits:          0x207fffff,     // 2^255 (easiest possible)
	GenesisDifficultyBits: 0x207fffff,     // start at minimum difficulty

	// Economic Model
	PreAllocation:       1_000_000 * 100_000_000, // 1,000,000 NOGO
	InitialBlockReward:  8 * 100_000_000,         // 8 NOGO
	AnnualReductionRate: 0.9,                     // 10% reduction per year
	AnnualBlockCount:    525_600,                 // blocks per year (60s blocks)
	MinimumBlockReward:  20_000_000,              // 0.2 NOGO floor
	GenesisAddressShare: 1,                       // 1% genesis address share
	BurnFees:            true,                    // burn all transaction fees
	GenesisAddress:      genesisCoinbaseAddress,  // 1M NOGO premine + 1% share
	ShareAddress:        genesisCoinbaseAddress,  // 1% per-block share target
}

// TestNet3Params defines the NogoCore test network configuration.
var TestNet3Params = Params{
	Name:        "testnet",
	Net:         TestNetMagic,
	DefaultPort: "19555",
	DNSSeeds:    []DNSSeed{},
	GenesisBlock: &nocoGenesisBlock,

	// Address encoding (Bitcoin testnet prefixes)
	PubKeyHashAddrID:        0x6f, // starts with m or n
	ScriptHashAddrID:        0xc4, // starts with 2 (testnet P2SH)
	PrivateKeyID:            0xef, // starts with 9 or c
	WitnessPubKeyHashAddrID: 0x06,
	WitnessScriptHashAddrID: 0x0A,
	Bech32HRPSegwit:         "tn", // NogoCore testnet Bech32 HRP

	// BIP32 extended key magics (compatible with Bitcoin testnet)
	HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94}, // tprv
	HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf}, // tpub
	HDCoinType:     1,

	// Consensus (faster testnet)
	TargetBlockTime:        30 * time.Second,
	TargetTimespan:         7 * 24 * time.Hour, // 1 week
	TargetTimePerBlock:     30 * time.Second,
	RetargetAdjustmentFactor: 4,
	MaxBlockSize:           8 * 1024 * 1024,
	MaxTxSize:        1 * 1024 * 1024,
	MaxTxPerBlock:    4096,
	CoinbaseMaturity: 10,

	// NogoPow (testnet: minimum difficulty → auto-converge to 30s)
	MinDifficulty:          1,
	MaxDifficulty:          1 << 30,
	ReduceMinDifficulty:    false,          // use full retarget logic
	PowLimit:               NogoPowLimitTest,
	PowLimitBits:           0x207fffff,     // 2^255 (easiest possible)
	GenesisDifficultyBits:  0x207fffff,     // start at minimum difficulty

	// Economic Model (testnet: fast decay)
	PreAllocation:       10_000_000 * 100_000_000, // 10,000,000 NOGO
	InitialBlockReward:  8 * 100_000_000,
	AnnualReductionRate: 0.5,      // fast decay for testing
	AnnualBlockCount:    1_051_200, // 30s blocks
	MinimumBlockReward:  100_000_000, // 1 NOGO floor
	GenesisAddressShare: 1,
	BurnFees:            true,
	GenesisAddress:      genesisCoinbaseAddress,
	ShareAddress:        genesisCoinbaseAddress,
}

// knownBech32Prefixes contains the HRP prefixes for known network Bech32 addresses.
var knownBech32Prefixes = map[string]struct{}{
	MainNetParams.Bech32HRPSegwit:   {},
	TestNet3Params.Bech32HRPSegwit: {},
}

// IsBech32SegwitPrefix returns whether the prefix is a known prefix for segwit
// Bech32 addresses on any known network.
func IsBech32SegwitPrefix(prefix string) bool {
	prefix = prefix[:len(prefix)-1] // strip the trailing '1' separator
	_, ok := knownBech32Prefixes[prefix]
	return ok
}

// regressionNetTemplate returns a Params template for non-mainnet
// regression/testing networks based on TestNet3Params defaults.
func regressionNetTemplate(name string, net wire.BitcoinNet, port string, hrpSegwit string,
	witnessPubKeyHashAddrID, witnessScriptHashAddrID byte,
	hdPrivateKeyID, hdPublicKeyID [4]byte, hdCoinType uint32) Params {
	return Params{
		Name:                    name,
		Net:                     net,
		DefaultPort:             port,
		DNSSeeds:                []DNSSeed{},
		GenesisBlock:            &nocoGenesisBlock,
		PubKeyHashAddrID:        0x6f,
		ScriptHashAddrID:        0xc4,
		PrivateKeyID:            0xef,
		WitnessPubKeyHashAddrID: witnessPubKeyHashAddrID,
		WitnessScriptHashAddrID: witnessScriptHashAddrID,
		Bech32HRPSegwit:         hrpSegwit,
		HDPrivateKeyID:          hdPrivateKeyID,
		HDPublicKeyID:           hdPublicKeyID,
		HDCoinType:              hdCoinType,
		TargetBlockTime:         30 * time.Second,
		TargetTimespan:          7 * 24 * time.Hour,
		TargetTimePerBlock:      30 * time.Second,
		RetargetAdjustmentFactor: 4,
		MaxBlockSize:            8 * 1024 * 1024,
		MaxTxSize:               1 * 1024 * 1024,
		MaxTxPerBlock:           4096,
		CoinbaseMaturity:        100,
		MinDifficulty:           1,
		MaxDifficulty:           1 << 31,
		PowLimit:                NogoPowLimitTest,
		PowLimitBits:            0x207fffff,
		GenesisDifficultyBits:   0x207fffff,
		PreAllocation:           10_000_000 * 100_000_000,
		InitialBlockReward:      8 * 100_000_000,
		AnnualReductionRate:     0.5,
		AnnualBlockCount:        1_051_200,
		MinimumBlockReward:      100_000_000,
		GenesisAddressShare:     1,
		BurnFees:                true,
	}
}

// RegressionNetParams defines the NogoCore regression test network.
var RegressionNetParams = regressionNetTemplate(
	"regtest", wire.TestNet, "18444", "bcrt",
	0x06, 0x0A,
	[4]byte{0x04, 0x35, 0x83, 0x94},
	[4]byte{0x04, 0x35, 0x87, 0xcf},
	1,
)

// TestNet4Params defines the NogoCore test network (version 4).
var TestNet4Params = regressionNetTemplate(
	"testnet4", wire.TestNet4, "48333", "tb",
	0x03, 0x28,
	[4]byte{0x04, 0x35, 0x83, 0x94},
	[4]byte{0x04, 0x35, 0x87, 0xcf},
	1,
)

// SigNetParams defines the NogoCore default public signet network.
var SigNetParams = regressionNetTemplate(
	"signet", wire.SigNet, "38333", "tb",
	0x03, 0x28,
	[4]byte{0x04, 0x35, 0x83, 0x94},
	[4]byte{0x04, 0x35, 0x87, 0xcf},
	1,
)

// SimNetParams defines the NogoCore simulation test network.
var SimNetParams = regressionNetTemplate(
	"simnet", wire.SimNet, "18555", "sb",
	0x19, 0x28,
	[4]byte{0x04, 0x20, 0xb9, 0x00},
	[4]byte{0x04, 0x20, 0xbd, 0x3a},
	115,
)

// DefaultSignetChallenge is the default challenge for the public signet.
var DefaultSignetChallenge = []byte{}

// DefaultSignetDNSSeeds is the list of DNS seeds for the public signet.
var DefaultSignetDNSSeeds = []DNSSeed{}

// CustomSignetParams creates custom signet parameters. Compat stub.
func CustomSignetParams(challenge []byte, dnsSeeds []DNSSeed) Params {
	return SigNetParams
}
