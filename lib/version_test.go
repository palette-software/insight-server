package insight_server

import "testing"

func TestVersionComparison(t *testing.T) {
	v1 := Version{Major: 1, Minor: 3, Patch: 2}
	v2 := Version{Major: 1, Minor: 3, Patch: 3}
	v3 := Version{Major: 1, Minor: 4, Patch: 3}
	v4 := Version{Major: 2, Minor: 0, Patch: 0}

	assert(t, IsNewerVersion(v2, v1), "v2 > v1")
	assert(t, !IsNewerVersion(v1, v2), "! v2 < v1")
	assert(t, IsNewerVersion(v3, v1), "v3 > v1")
	assert(t, IsNewerVersion(v4, v3), "v4 > v3")
}
