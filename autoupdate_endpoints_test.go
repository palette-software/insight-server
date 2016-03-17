package insight_server

import "testing"

func TestVersionParsing(t *testing.T) {

	ver, err := StringToVersion("v1.3.2")
	assert(t, err == nil, "Should parse the version string")

	assertInt(t, ver.Major, 1, "Parse: Bad major version")
	assertInt(t, ver.Minor, 3, "Parse: Bad minor version")
	assertInt(t, ver.Patch, 2, "Parse: Bad patch version")

	assertString(t, ver.String(), "v1.3.2", "Version stringification mismatch")

}

func TestWindowsServiceVersionParsing(t *testing.T) {

	ver, err := StringToVersion("1.8.177.0")
	assert(t, err == nil, "Should parse the version string")

	assertInt(t, ver.Major, 1, "Parse: Bad major version")
	assertInt(t, ver.Minor, 8, "Parse: Bad minor version")
	assertInt(t, ver.Patch, 177, "Parse: Bad patch version")

	assertString(t, ver.String(), "v1.8.177", "Version stringification mismatch")

}

func TestWindowsServiceVersionParsingNonNumeric(t *testing.T) {

	ver, err := StringToVersion("1.8.177-rc2")
	assert(t, err == nil, "Should parse the version string")

	assertInt(t, ver.Major, 1, "Parse: Bad major version")
	assertInt(t, ver.Minor, 8, "Parse: Bad minor version")
	assertInt(t, ver.Patch, 177, "Parse: Bad patch version")

	assertString(t, ver.String(), "v1.8.177", "Version stringification mismatch")

}
func TestVersionComparison(t *testing.T) {
	v1, _ := StringToVersion("v1.3.2")
	v2, _ := StringToVersion("v1.3.3")
	v3, _ := StringToVersion("v1.4.3")
	v4, _ := StringToVersion("v2.0.0")

	assert(t, IsNewerVersion(v2, v1), "v2 > v1")
	assert(t, !IsNewerVersion(v1, v2), "! v2 < v1")
	assert(t, IsNewerVersion(v3, v1), "v3 > v1")
	assert(t, IsNewerVersion(v4, v3), "v4 > v3")
}
