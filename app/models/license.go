package models

/*
// #cgo LDFLAGS: vendor/libsodium-win64/lib/libsodium.a
// #include "vendor/libsodium-win64/include/sodium.h"
import "C"
*/

import (
	"gopkg.in/yaml.v2"

	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"time"
)

// Faking the crypto
////////////////////

// As messages signed by sodium have a similar structure:
//
// |SIGNATURE                    |MESSAGE                  |
// |crypto_sign_BYTES bytes      |messageLen bytes         |
// +-----------------------------+-------------------------+
//
// (crypto_sign_BYTES is 64 for ED25519)
//
// So if we do not care about cryptographic integrity (which
// we do not in case of the server) we simply skip the
// signature in the message, and read the bytes as-is.

const (
	crypto_sign_ed25519_BYTES = 64
)

// Removes the signature bytes from an Ed25519 signed message
func removeEd25519Signature(signedMsg []byte) ([]byte, error) {
	if len(signedMsg) <= crypto_sign_ed25519_BYTES {
		return []byte{}, fmt.Errorf("Signed message is too short (shorter then the signature itself)")
	}

	return signedMsg[crypto_sign_ed25519_BYTES:], nil
}

// License
////////////////

// The YAML-serialized format of the license
type yamlLicense struct {
	Seed          int32     `yaml:"seed"`
	Owner         string    `yaml:"owner"`
	LicenseId     string    `yaml:"licenseId"`
	CoreCount     int32     `yaml:"coreCount"`
	Token         string    `yaml:"token"`
	ValidUntilUTC time.Time `yaml:"validUntilUTC"`
}

// The License data structure
type License struct {
	Seed          int32
	Owner         string
	LicenseId     string
	CoreCount     int32
	Token         []byte
	ValidUntilUTC time.Time
}

/// Tries to read and deserialize a license
func ReadLicense(r io.Reader) (*License, error) {

	// read all the license in
	licenseString, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// first, decode the message (we convert to string as the
	// .Decode() interace with bytes is a stateful & ugly one
	base64Decoded, err := base64.StdEncoding.DecodeString(string(licenseString))
	if err != nil {
		return nil, err
	}

	// remove the Ed25519 signature
	licenseAvroData, err := removeEd25519Signature(base64Decoded)
	if err != nil {
		return nil, err
	}

	// the yaml format license
	serializedLicense := yamlLicense{}

	// read the license as YAML
	err = yaml.Unmarshal(licenseAvroData, &serializedLicense)
	if err != nil {
		return nil, err
	}

	// decode the base64 encoded token
	token, err := base64.StdEncoding.DecodeString(serializedLicense.Token)

	return &License{
		Owner:     serializedLicense.Owner,
		Seed:      serializedLicense.Seed,
		LicenseId: serializedLicense.LicenseId,
		CoreCount: serializedLicense.CoreCount,
		Token:     token,

		ValidUntilUTC: serializedLicense.ValidUntilUTC,
	}, nil

}
