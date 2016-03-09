package insight_server

import (
	"testing"

	"bytes"
	"strings"
)

const (
	DEBUG_LICENSE = `
jh6HOVLY4E3oPbUxoyNf2T9xZ2Ljab4dRrkwuACIACE9v/Fz6Noxw0+vVSkm
Bv9pvht+vpqNjJBeo4U95EeGDHNlZWQ6IDg3OTA2Mjc3Mg0Kb3duZXI6IFBh
bGV0dGUgQWdlbnQgREVWDQpsaWNlbnNlSWQ6IHBhbGV0dGUtZGV2DQpjb3Jl
Q291bnQ6IDY0DQp0b2tlbjogZXpDRE1RczllOFlUYVRWQnk4UndKZ21VUGNU
dkZpTFpub0xSaDdYaFdNUmlKREtpa0hCT0RBYmY3ZlZPTkZEWVoybENPMmF0
QUdVTnFwS0liUFFRTWhGdzNxYVN6WmFZUlRHK0R5a3RpbW94K0I5TG5oeGV0
U1BObGViZHhaOFN6cnJrM2xXcVJwclRxZlUwNy9BODN4OG9WblBVeFpQMEVp
bUN4VGl2dTRBPQ0KdmFsaWRVbnRpbFVUQzogMjAxNi0wMy0wMVQxMTozOTo0
OS4zNDk4NTI2Wg0K
`
)

var DEBUG_TOKEN []byte = []byte{123, 48, 131, 49, 11, 61, 123, 198, 19, 105, 53, 65, 203, 196, 112, 38, 9, 148, 61, 196, 239, 22, 34, 217, 158, 130, 209, 135, 181, 225, 88, 196, 98, 36, 50, 162, 144, 112, 78, 12, 6, 223, 237, 245, 78, 52, 80, 216, 103, 105, 66, 59, 102, 173, 0, 101, 13, 170, 146, 136, 108, 244, 16, 50, 17, 112, 222, 166, 146, 205, 150, 152, 69, 49, 190, 15, 41, 45, 138, 106, 49, 248, 31, 75, 158, 28, 94, 181, 35, 205, 149, 230, 221, 197, 159, 18, 206, 186, 228, 222, 85, 170, 70, 154, 211, 169, 245, 52, 239, 240, 60, 223, 31, 40, 86, 115, 212, 197, 147, 244, 18, 41, 130, 197, 56, 175, 187, 128}

func TestLicenseReading(t *testing.T) {
	r := strings.NewReader(DEBUG_LICENSE)
	license, err := ReadLicense(r)
	if err != nil {
		panic(err)
	}

	assert(t, license.Seed == 879062772, "Bad seed")
	assert(t, license.Owner == "Palette Agent DEV", "Bad Owner")
	assert(t, license.CoreCount == 64, "Bad corecount")
	assert(t, license.LicenseId == "palette-dev", "Bad licenseId")
	assert(t, bytes.Equal(license.Token, DEBUG_TOKEN), "Bad token")
}
