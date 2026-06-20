// Copyright (c) 2026 NogoChain Contributors
// Use of this source code is governed by an ISC license.

package nogoutil

import (
	"github.com/nogochain/nogocommons/address"
)

// --- Type Aliases ---

// Address is an interface type for NogoChain addresses.
type Address = address.Address

// AddressPubKeyHash is an Address for a pay-to-pubkey-hash (P2PKH) transaction.
type AddressPubKeyHash = address.AddressPubKeyHash

// AddressScriptHash is an Address for a pay-to-script-hash (P2SH) transaction.
type AddressScriptHash = address.AddressScriptHash

// AddressWitnessPubKeyHash is an Address for a pay-to-witness-pubkey-hash
// (P2WPKH) transaction.
type AddressWitnessPubKeyHash = address.AddressWitnessPubKeyHash

// AddressWitnessScriptHash is an Address for a pay-to-witness-script-hash
// (P2WSH) transaction.
type AddressWitnessScriptHash = address.AddressWitnessScriptHash

// AddressTaproot is an Address for a taproot (P2TR) transaction.
type AddressTaproot = address.AddressTaproot

// AddressPubKey is an Address for a pay-to-pubkey (P2PK) transaction.
type AddressPubKey = address.AddressPubKey

// AddressSegWit is a generic segwit address.
type AddressSegWit = address.AddressSegWit

// --- Function Re-exports ---

// Hash160 calculates the hash ripemd160(sha256(b)).
var Hash160 = address.Hash160

// DecodeAddress decodes the string encoding of an address and returns
// the Address.
var DecodeAddress = address.DecodeAddress

// NewAddressPubKeyHash returns a new AddressPubKeyHash.
var NewAddressPubKeyHash = address.NewAddressPubKeyHash

// NewAddressScriptHash returns a new AddressScriptHash.
// pkScript is the script to be hashed.
var NewAddressScriptHash = address.NewAddressScriptHash

// NewAddressScriptHashFromHash returns a new AddressScriptHash from
// a precomputed script hash.
var NewAddressScriptHashFromHash = address.NewAddressScriptHashFromHash

// NewAddressWitnessPubKeyHash returns a new AddressWitnessPubKeyHash.
var NewAddressWitnessPubKeyHash = address.NewAddressWitnessPubKeyHash

// NewAddressWitnessScriptHash returns a new AddressWitnessScriptHash.
var NewAddressWitnessScriptHash = address.NewAddressWitnessScriptHash

// NewAddressTaproot returns a new AddressTaproot.
var NewAddressTaproot = address.NewAddressTaproot

// NewAddressPubKey returns a new AddressPubKey.
var NewAddressPubKey = address.NewAddressPubKey
