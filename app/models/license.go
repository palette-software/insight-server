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

const (
	LICENSE_AVRO_SCHEMA = `
	{
		"type":"record",
		"name":"License",
		"fields":
				[
						{ "name":"seed", "type":"int" },
						{ "name":"owner", "type":"string" },
						{ "name":"licenseId", "type":"string" },
						{ "name":"coreCount", "type":"int" },
						{ "name":"token", "type":"bytes" },
						{ "name":"validUntilUTC", "type":"long" }
				]
	}
	`
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

//// Creates an Avro codec for desr
//func createLicenseAvroCodec() (goavro.Codec, error) {
//codec, err := goavro.NewCodec(LICENSE_AVRO_SCHEMA)
//if err != nil {
//return nil, err
//}
//return codec, nil
//}

//var licenseCodecInstance goavro.Codec
//var once sync.Once

//func getLicenseCodecInstance() goavro.Codec {
//once.Do(func() {
//// unless we declare err here, doing a := would create
//// a new variable named licenseCodecInstance here.
//var err error
//licenseCodecInstance, err = createLicenseAvroCodec()
//if err != nil {
//// if we cannot deserialize licenses, we should fail
//// immidately
//panic(err)
//}
//})
//return licenseCodecInstance
//}

// Helper to get fields from avro data without running into
// the issue of Get() returning two values
// As this method should only be used in the context of the license,
// we can safely panic here.
//func getAvroField(r *goavro.Record, field string) interface{} {
//f, err := r.Get(field)
//if err != nil {
//panic(err)
//}

//return f
//}

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

	//// create a reader for the license bytes
	//licenseReader := bytes.NewReader(licenseAvroData)

	//// read the license as avro reacord
	//licenseData, err := getLicenseCodecInstance().Decode(licenseReader)
	//if err != nil {
	//return nil, err
	//}

	//license := licenseData.(*goavro.Record)

	//// convert it to a proper struct
	//return &License{
	//Owner:     getAvroField(license, "owner").(string),
	//Seed:      getAvroField(license, "seed").(int32),
	//LicenseId: getAvroField(license, "licenseId").(string),
	//CoreCount: getAvroField(license, "coreCount").(int32),
	//Token:     getAvroField(license, "token").([]byte),
	//// convert the validity date the same way as the license generator does
	//ValidUntilUTC: time.Unix(getAvroField(license, "validUntilUTC").(int64), 0),
	//}, nil
}
