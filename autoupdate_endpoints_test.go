package insight_server

import (
	"sort"
	"testing"
)

func TestVersionParsing(t *testing.T) {

	ver, err := StringToVersion("v1.3.2")
	assert(t, err == nil, "Should parse the version string")

	assertInt(t, ver.Major, 1, "Parse: Bad major version")
	assertInt(t, ver.Minor, 3, "Parse: Bad minor version")
	assertInt(t, ver.Patch, 2, "Parse: Bad patch version")

	assertString(t, ver.String(), "v1.3.2", "Version stringification mismatch")
}
