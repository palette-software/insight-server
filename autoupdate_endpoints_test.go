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
}

func TestVersionSorting(t *testing.T) {
	vers := VersionList{
		&Version{1, 3, 2},
		&Version{1, 3, 3},
		&Version{1, 4, 1},
		&Version{2, 0, 1},
	}

	sort.Sort(vers)

	// check the last one in the version list

	assertInt(t, vers[3].Major, 2, "Sort 1.: Bad major version")
	assertInt(t, vers[3].Minor, 0, "Sort 1.: Bad minor version")
	assertInt(t, vers[3].Patch, 1, "Sort 1.: Bad patch version")

	assertInt(t, vers[2].Major, 1, "Sort 2.: Bad major version")
	assertInt(t, vers[2].Minor, 4, "Sort 2.: Bad minor version")
	assertInt(t, vers[2].Patch, 1, "Sort 2.: Bad patch version")

	assertInt(t, vers[1].Major, 1, "Sort 3.: Bad major version")
	assertInt(t, vers[1].Minor, 3, "Sort 3.: Bad minor version")
	assertInt(t, vers[1].Patch, 3, "Sort 3.: Bad patch version")

	assertInt(t, vers[0].Major, 1, "Sort 4.: Bad major version")
	assertInt(t, vers[0].Minor, 3, "Sort 4.: Bad minor version")
	assertInt(t, vers[0].Patch, 2, "Sort 4.: Bad patch version")
}
