package insight_server

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	log "github.com/palette-software/go-log-targets"
)

// The base structure for a SemVer like version
type Version struct {
	// The version according to SemVer
	Major, Minor, Patch int
}

// Converts a version to its string equivalent
func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
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

// The file name we use to store a verison inside its own folder
const CONTENTS_FILE_NAME = "contents.bin"

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
func LatestVersion(updateDirectory string) (*UpdateVersion, error) {
	latestVersion, err := getLatestAgentVersion(updateDirectory)
	if err != nil {
		log.Error("Error querying Agent version.", err)
		return nil, fmt.Errorf("No latest version found yet")
	}
	return latestVersion, nil
}

// Returns true if version a is newer then version b
func IsNewerVersion(a, b Version) bool {
	if a.Major == b.Major {
		if a.Minor == b.Minor {
			if a.Patch == b.Patch {
				return false
			}
			return a.Patch > b.Patch
		}
		return a.Minor > b.Minor
	}
	return a.Major > b.Major
}

var versionRegExp = regexp.MustCompile("(\\d+)\\.(\\d+)\\.(\\d+)")

func computeMd5(filePath string) ([]byte, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return result, err
	}

	return hash.Sum(result), nil
}

// Tries to load all valid versions from a product directory
func getLatestAgentVersion(updatePath string) (*UpdateVersion, error) {
	versionString, err := exec.Command("rpm", "-qa", "--queryformat", "'%{version}\n'", "palette-insight-agent").Output()
	// versionString, err := exec.Command("echo", "2.0.1\n").Output()
	if err != nil {
		return nil, err
	}
	version := versionRegExp.FindStringSubmatch(string(versionString))
	if len(version) < 4 {
		return nil, fmt.Errorf("Invalid version received from RPM: %s", versionString)
	}
	major, err := strconv.ParseInt(version[1], 10, 32)
	if err != nil {
		return nil, err
	}
	minor, err := strconv.ParseInt(version[2], 10, 32)
	if err != nil {
		return nil, err
	}
	patch, err := strconv.ParseInt(version[3], 10, 32)
	if err != nil {
		return nil, err
	}
	packageMd5, err := computeMd5(filepath.Join(updatePath, "agent"))
	if err != nil {
		return nil, err
	}

	return &UpdateVersion{
		Version: Version{
			Major: int(major),
			Minor: int(minor),
			Patch: int(patch),
		},
		Md5:     fmt.Sprintf("%32x", packageMd5),
		Product: "Agent",
		Url:     "/api/v1/agent",
	}, nil
}

func GetAutoupdateLatestVersionHandler(updateDirectory string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		latestVersion, err := LatestVersion(updateDirectory)
		if err != nil {
			WriteResponse(w, http.StatusNotFound, fmt.Sprintf("%v", err), r)
			return
		}

		if err := json.NewEncoder(w).Encode(latestVersion); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}
}
