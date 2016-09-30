package insight_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path"
	"regexp"
	"strconv"

	"github.com/Sirupsen/logrus"
)

// ENDPOINTS
// =========
//
// GET: /updates/agent/latest-version => 200 OK
// {version: "v1.3.2", major: 1, minor: 3, patch: 2, url: "https://.../updates/agent/versions/v1.3.2", md5: "....."}

//
// GET: /updates/agent/versions/v1.3.2 => 200 OK
// palette-insight-v1.3.2.msi
//
//=> 404 NOT FOUND

// Public API
// ==========

// The base structure for a SemVer like version
type Version struct {
	// The version according to SemVer
	Major, Minor, Patch int
}

// Converts a version to its string equivalent
func (v *Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

type AutoUpdater interface {
	// Returns the latest version of a product
	LatestVersion() (*UpdateVersion, error)
}

// Implementation
// ==============

// Tries to parse a list of string to a list of integers
func parseAllInts(strs []string) ([]int, error) {
	o := make([]int, len(strs))
	for i, s := range strs {
		// parse the version as 32 bit wide, based on the prefix of the string
		// (defaults to decimal)
		verPart, err := strconv.ParseInt(s, 0, 32)
		// make errors break the loop
		if err != nil {
			return nil, fmt.Errorf("Error parsing integer: '%s' %v", s, err)
		}
		o[i] = int(verPart)
	}
	return o, nil
}

// Autoupdater implementation
// --------------------------

type baseAutoUpdater struct {
}

// Creates a new autoupdater implementation
func NewBaseAutoUpdater() AutoUpdater {
	return &baseAutoUpdater{}
}

// The file name we use to store a verison inside its own folder
const CONTENTS_FILE_NAME = "contents.bin"

func (a *baseAutoUpdater) updatePath(product, versionStr string) string {
	return path.Join("/tmp", SanitizeName(product), versionStr, fmt.Sprintf("%s-%s", product, versionStr))
}

// Combines a version with an actual product and a file
type UpdateVersion struct {
	Version
	// The name of the product
	Product string
	// The Md5 checksum of this update
	Md5 string
	// The url where this update can be downloaded from
	Url string
}

// Returns the latest version of a product
func (a *baseAutoUpdater) LatestVersion() (*UpdateVersion, error) {
	latestVersion, err := getLatestAgentVersion()
	if err != nil {
		logrus.WithError(err).Error("Error querying Agent version")
		return nil, fmt.Errorf("No latest version found yet")
	}
	return latestVersion, nil
}

// Tries to load all valid versions from a product directory
func getLatestAgentVersion() (*UpdateVersion, error) {
	version, err := exec.Command("rpm", "-qa", "--queryformat", "'%{version}\n'", "palette-insight-agent").Output()
	// version, err := exec.Command("echo", "v1.0.96\n").Output()
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile("v(\\d+)\\.(\\d+)\\.(\\d+)")
	r2 := re.FindStringSubmatch(string(version))
	if len(r2) < 4 {
		return nil, fmt.Errorf("Invalid version received from RPM: %s", version)
	}
	major, err := strconv.ParseInt(r2[1], 10, 32)
	if err != nil {
		return nil, err
	}
	minor, err := strconv.ParseInt(r2[2], 10, 32)
	if err != nil {
		return nil, err
	}
	patch, err := strconv.ParseInt(r2[3], 10, 32)
	if err != nil {
		return nil, err
	}
	return &UpdateVersion{
		Version: Version{
			Major: int(major),
			Minor: int(minor),
			Patch: int(patch),
		},
		Product: "Agent",
		Url:     "/api/v1/agent",
	}, nil
}

// HTTP Handler
// ------------

func AutoupdateLatestVersionHandler(a AutoUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latestVersion, err := a.LatestVersion()
		if err != nil {
			WriteResponse(w, http.StatusNotFound, fmt.Sprintf("%v", err))
			return
		}

		if err := json.NewEncoder(w).Encode(latestVersion); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}
}
