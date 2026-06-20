// Package txscript provides compat stubs for the removed Bitcoin script engine.
package txscript

import ("fmt"; "github.com/nogochain/nogocommons/address"
	"github.com/nogochain/nogocommons/nogoec"
	"github.com/nogochain/nogocommons/chaincfg"
	"github.com/nogochain/nogocommons/wire"
)

var ErrUnsupportedScriptType = fmt.Errorf("unsupported script type")

type ScriptBuilder []byte
type SigCache struct{}
type HashCache struct{}
type TxSigHashes struct{}
type MultiPrevOutFetcher struct {
	prevOuts map[wire.OutPoint]*wire.TxOut
}
func NewMultiPrevOutFetcher(tx *wire.MsgTx) *MultiPrevOutFetcher {
	return &MultiPrevOutFetcher{prevOuts: make(map[wire.OutPoint]*wire.TxOut)}
}
func (f *MultiPrevOutFetcher) AddPrevOut(op wire.OutPoint, txOut *wire.TxOut) {
	if f.prevOuts == nil {
		f.prevOuts = make(map[wire.OutPoint]*wire.TxOut)
	}
	f.prevOuts[op] = txOut
}
func (f *MultiPrevOutFetcher) FetchPrevOutput(op wire.OutPoint) *wire.TxOut {
	if f.prevOuts == nil {
		return nil
	}
	return f.prevOuts[op]
}
type KeyDB interface{}
type ScriptDB interface{}
type SecretsSource interface {
	ChainParams() *chaincfg.Params
	GetKey(addr address.Address) (*nogoec.PrivateKey, bool, error)
}
type SigHashType uint32
// GetScriptClass returns the class of the script. Compat stub.
func GetScriptClass(script []byte) (ScriptClass, error) {
	return NonStandardTy, nil
}

// PushedData extracts all data pushed in a script. Compat stub.
func PushedData(script []byte) ([][]byte, error) {
	return nil, nil
}

type ScriptClass int

// String returns the string representation of the ScriptClass.
func (s ScriptClass) String() string {
	switch s {
	case NonStandardTy:
		return "nonstandard"
	case PubKeyTy:
		return "pubkey"
	case PubKeyHashTy:
		return "pubkeyhash"
	case WitnessV0PubKeyHashTy:
		return "witness_v0_keyhash"
	case ScriptHashTy:
		return "scripthash"
	case WitnessV0ScriptHashTy:
		return "witness_v0_scripthash"
	case MultiSigTy:
		return "multisig"
	case NullDataTy:
		return "nulldata"
	case WitnessV1TaprootTy:
		return "witness_v1_taproot"
	default:
		return "unknown"
	}
}
type UtxoViewpoint struct{}
type TapLeaf struct {
	LeafVersion uint8
	Script      []byte
}
func (t *TapLeaf) TapHash() []byte { return nil }
type ControlBlock struct {
	InternalKey *nogoec.PublicKey
	RootNode    *TapLeaf
}
func (c *ControlBlock) RootHash(script []byte) []byte { return nil }
func (c *ControlBlock) ToBytes() ([]byte, error) { return nil, nil }
type TapscriptType uint8

const (
	SigHashAll    SigHashType = 1
	SigHashDefault SigHashType = 0
	StandardVerifyFlags = 0
	NonStandardTy          ScriptClass = 0
	PubKeyTy               ScriptClass = 1
	PubKeyHashTy           ScriptClass = 2
	WitnessV0PubKeyHashTy  ScriptClass = 3
	ScriptHashTy           ScriptClass = 4
	WitnessV0ScriptHashTy  ScriptClass = 5
	MultiSigTy             ScriptClass = 6
	NullDataTy             ScriptClass = 7
	WitnessV1TaprootTy     ScriptClass = 8
	TapscriptLeafVersion uint8 = 0xc0
	TapscriptTypeFullTree TapscriptType = 0
)

func PayToAddrScript(addr address.Address) ([]byte, error) { return nil, ErrUnsupportedScriptType }
func NewTxSigHashes(tx *wire.MsgTx, fetcher PrevOutputFetcher) *TxSigHashes { return &TxSigHashes{} }
func GetPreciseSigOpCount(a, b []byte, c bool) int { return 0 }
func CalcMultiSigStats(script []byte) (int, int, error) { return 0, 0, nil }
func IsPayToScriptHash(script []byte) bool { return false }
func IsPayToWitnessPubKeyHash(script []byte) bool { return false }
func IsPayToTaproot(script []byte) bool { return false }
func IsPayToAnchorScript(script []byte) bool { return false }
func IsWitnessProgram(script []byte) bool { return false }
func IsUnspendable(script []byte) bool { return false }
func IsPushOnlyScript(script []byte) bool { return false }
func IsCoinBaseTx(tx *wire.MsgTx) bool { return len(tx.TxIn) == 1 && tx.TxIn[0].PreviousOutPoint.Index == wire.MaxPrevOutIndex }
func ExtractPkScriptAddrs(script []byte, params *chaincfg.Params) (ScriptClass, []address.Address, int, error) { return NonStandardTy, nil, 0, nil }
func Hash160(data []byte) []byte { h := address.Hash160(data); return h[:] }
func SignTxOutput(chainParams *chaincfg.Params, tx *wire.MsgTx, idx int, pkScript []byte, hashType SigHashType, kdb KeyDB, sdb ScriptDB, previousScript []byte) ([]byte, error) { return nil, nil }
func GetKey(s SecretsSource, addr address.Address) (*nogoec.PrivateKey, bool, error) { return nil, false, nil }
func ValidateTransactionScripts(tx *wire.MsgTx, view *UtxoViewpoint, flags int, sigCache *SigCache, hashCache *HashCache) error { return nil }
func AssembleTaprootScriptTree(leaves ...TapLeaf) *ControlBlock { return &ControlBlock{RootNode: &TapLeaf{}} }
func ComputeTaprootKeyNoScript(internalKey *nogoec.PublicKey) *nogoec.PublicKey { return internalKey }
func ComputeTaprootOutputKey(internalKey *nogoec.PublicKey, scriptRoot []byte) *nogoec.PublicKey { return internalKey }
func ParseControlBlock(b []byte) (*ControlBlock, error) { return &ControlBlock{}, nil }

// ComputedPkScript wraps the result of ComputePkScript, providing
// an Address() method compatible with newer btcd API.
type ComputedPkScript struct {
	script  []byte
	witness wire.TxWitness
}

// Address returns the address derived from the computed pkScript.
func (c *ComputedPkScript) Address(params *chaincfg.Params) (address.Address, error) {
	return nil, ErrUnsupportedScriptType
}

// ComputePkScript reconstructs the pkScript from a signature script and
// optional witness data. Compat stub.
func ComputePkScript(sigScript []byte, witness wire.TxWitness) (*ComputedPkScript, error) {
	return &ComputedPkScript{script: sigScript, witness: witness}, nil
}

// WitnessSignature creates an input witness stack for tx to spend BTC sent
// to a P2WPKH / P2WSH output. Compat wrapper stub.
func WitnessSignature(tx *wire.MsgTx, sigHashes *TxSigHashes, idx int, amt int64,
	subscript []byte, hashType SigHashType, privKey *nogoec.PrivateKey,
	compress bool) ([][]byte, error) {
	return nil, ErrUnsupportedScriptType
}

// TaprootWitnessSignature returns a valid witness stack that can be used to
// spend the key-spend path of a taproot output. Compat wrapper stub.
func TaprootWitnessSignature(tx *wire.MsgTx, sigHashes *TxSigHashes, idx int,
	amt int64, pkScript []byte, hashType SigHashType,
	privKey *nogoec.PrivateKey) ([][]byte, error) {
	return nil, ErrUnsupportedScriptType
}

// --- OP codes ---
const (
	OP_0                   = 0x00
	OP_FALSE               = 0x00
	OP_PUSHDATA1           = 0x4c
	OP_PUSHDATA2           = 0x4d
	OP_PUSHDATA4           = 0x4e
	OP_1                   = 0x51
	OP_TRUE                = 0x51
	OP_IF                  = 0x63
	OP_ELSE                = 0x67
	OP_ENDIF               = 0x68
	OP_RETURN              = 0x6a
	OP_DUP                 = 0x76
	OP_EQUAL               = 0x87
	OP_EQUALVERIFY         = 0x88
	OP_2DUP                = 0x6e
	OP_HASH160             = 0xa9
	OP_CHECKSIG            = 0xac
	OP_CHECKSIGVERIFY      = 0xad
	OP_CHECKMULTISIG       = 0xae
	OP_CHECKMULTISIGVERIFY = 0xaf
	OP_INVALIDOPCODE       = 0xff

	// SigHash types
	SigHashSingle          SigHashType = 3
	SigHashNone            SigHashType = 2
	SigHashAnyOneCanPay    SigHashType = 0x80
)

// --- ScriptBuilder methods (pointer receiver for chainable API) ---

// NewScriptBuilder returns a new ScriptBuilder.
func NewScriptBuilder() *ScriptBuilder {
	return &ScriptBuilder{}
}

// AddOp adds the passed opcode to the script builder.
func (b *ScriptBuilder) AddOp(op byte) *ScriptBuilder { return b }

// AddData adds the passed data as a data push to the script builder.
func (b *ScriptBuilder) AddData(data []byte) *ScriptBuilder { return b }

// Script returns the built script. Compat stub.
func (b *ScriptBuilder) Script() ([]byte, error) { return nil, nil }

// GetSigOpCount returns the number of signature operations in a script. Compat stub.
func GetSigOpCount(script []byte) int { return 0 }

// RawTxInSignature generates a raw signature for the transaction input. Compat stub.
func RawTxInSignature(tx *wire.MsgTx, idx int, subScript []byte,
	hashType SigHashType, key *nogoec.PrivateKey) ([]byte, error) {
	return nil, nil
}

// --- Engine stubs ---

// ScriptFlags defines flags for script execution.
type ScriptFlags = int

// Engine represents a virtual machine that executes scripts. Compat stub.
type Engine struct{}

// NewEngine returns a new script engine. Compat stub.
func NewEngine(scriptPubKey []byte, tx *wire.MsgTx, txIdx int, flags ScriptFlags,
	sigCache *SigCache, hashCache *TxSigHashes, inputAmount int64,
	prevOutFetcher PrevOutputFetcher) (*Engine, error) {
	return &Engine{}, nil
}

// Execute executes the scripts in the engine. Compat stub.
func (e *Engine) Execute() error {
	return nil
}

// PrevOutputFetcher defines an interface for fetching previous output info.
type PrevOutputFetcher interface {
	FetchPrevOutput(wire.OutPoint) *wire.TxOut
}

// KeyClosure is a closure function type that provides keys.
type KeyClosure func(address.Address) (*nogoec.PrivateKey, bool, error)

// ScriptClosure is a closure function type that provides scripts.
type ScriptClosure func(address.Address) ([]byte, error)

// MultiSigScript creates a multisig script. Compat stub.
func MultiSigScript(pubKeys interface{}, nRequired int) ([]byte, error) {
	return nil, ErrUnsupportedScriptType
}

// DisasmString returns a human-readable disassembly of the script.
// Compat stub — script validation is removed per NogoCore design.
func DisasmString(script []byte) (string, error) {
	return "script-disabled", nil
}
