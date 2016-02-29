package tests

import (
	"github.com/palette-software/insight-server/app/models"
	"github.com/revel/revel/testing"

	"fmt"
	"reflect"
	"strings"
)

const (
	DEBUG_LICENSE = `
S5iIoWH6oXXLxbHZaN4DDJADwTsOf3veblwPU2VQjwr60cHQ2+o9lWRw5Vvo
Udj3Gfy5sDN6eT1AnJsp7HZFC+6pyZEBIlBhbGV0dGUgQWdlbnQgREVWFnBh
bGV0dGUtZGV2gAGAAslgqCGwwZzLLak5vW3DzMmTe6JDupKDAFYHKGLz3k4K
cVRSxbPx//2sZM660C7VYp3fYNJHaMY33ajKqZg6105aL0sZcQcWoXyll1bD
yC8oMcbEFH18wgmNG+gLJ1ZkcYx/X6HivPGWyL5WodBwcxR4oXfSdcPR3mxC
i1UfC3olxKeXiws=
`
)

var DEBUG_TOKEN []byte = []byte{201, 96, 168, 33, 176, 193, 156, 203, 45, 169, 57, 189, 109, 195, 204, 201, 147, 123, 162, 67, 186, 146, 131, 0, 86, 7, 40, 98, 243, 222, 78, 10, 113, 84, 82, 197, 179, 241, 255, 253, 172, 100, 206, 186, 208, 46, 213, 98, 157, 223, 96, 210, 71, 104, 198, 55, 221, 168, 202, 169, 152, 58, 215, 78, 90, 47, 75, 25, 113, 7, 22, 161, 124, 165, 151, 86, 195, 200, 47, 40, 49, 198, 196, 20, 125, 124, 194, 9, 141, 27, 232, 11, 39, 86, 100, 113, 140, 127, 95, 161, 226, 188, 241, 150, 200, 190, 86, 161, 208, 112, 115, 20, 120, 161, 119, 210, 117, 195, 209, 222, 108, 66, 139, 85, 31, 11, 122, 37}

type LicenseTest struct {
	testing.TestSuite
}

func (t *LicenseTest) TestLicenseReading() {
	r := strings.NewReader(DEBUG_LICENSE)
	license, err := models.ReadLicense(r)
	if err != nil {
		panic(err)
	}

	t.Assert(license.Seed == 152644215)
	t.Assert(license.Owner == "Palette Agent DEV")
	t.Assert(license.CoreCount == 64)
	t.Assert(license.LicenseId == "palette-dev")
	t.Assert(reflect.DeepEqual(license.Token, DEBUG_TOKEN))
}
