package insight_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// ENPOINTS
// ========
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

// The regexp we'll use for parsing version strings
var versionCompiler *regexp.Regexp = regexp.MustCompile("^v([0-9]+).([0-9]+).([0-9a-zA-Z]+)$")

// Parses a string to a Version struct or returns an error if it cannot
func StringToVersion(verStr string) (*Version, error) {
	if versionCompiler.MatchString(verStr) {
		matches := versionCompiler.FindStringSubmatch(verStr)

		versionParts, err := parseAllInts(matches[1:])
		if err != nil {
			return nil, fmt.Errorf("Error parsing version string '%s': %v", verStr, err)
		}

		return &Version{
			Major: versionParts[0],
			Minor: versionParts[1],
			Patch: versionParts[2],
		}, nil

	}
	return nil, fmt.Errorf("Cannot parse version string: %s", verStr)
}

// Define a versionlist type for sorting by version
type VersionList []*Version

type AutoUpdater interface {
	// Returns the latest version of a product
	LatestVersion(product string) (*UpdateVersion, error)

	// Adds a new version to the stored versions
	AddNewVersion(product string, version *Version, src io.Reader) (*UpdateVersion, error)

	// Returns a reader for the update file for the given version
	FileForVersion() (io.Reader, error)
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
	// The base path where updates are stored
	basePath string

	latestVersions map[string]*UpdateVersion
}

// Creates a new autoupdater implementation
func NewBaseAutoUpdater(basePath string) (AutoUpdater, error) {
	// create the versions directory
	if err := createDirectoryIfNotExists(basePath); err != nil {
		return nil, err
	}

	// update the latest version list
	latestVersions, err := loadLatestVersions(basePath)
	if err != nil {
		return nil, err
	}

	return &baseAutoUpdater{
		basePath:       basePath,
		latestVersions: latestVersions,
	}, nil
}

// The file name we use to store a verison inside its own folder
const CONTENTS_FILE_NAME = "contents.bin"

// Gets the path where an update binary is stored
func (a *baseAutoUpdater) updatePath(product, versionStr string) string {
	return path.Join(a.basePath, SanitizeName(product), versionStr, fmt.Sprintf("%s-%s", product, versionStr))
}

// Adds a new version to the list of available versions
func (a *baseAutoUpdater) AddNewVersion(product string, version *Version, src io.Reader) (*UpdateVersion, error) {
	// get the storage path
	storagePath := a.updatePath(product, version.String())

	// Check if we already have this version
	versionExists, err := fileExists(storagePath)
	if err != nil {
		return nil, err
	}
	if versionExists {
		return nil, fmt.Errorf("Version '%s' of product '%s' already exists", version, product)
	}

	// save the update binary
	// ----------------------

	// Create the directory of the update
	if err := createDirectoryIfNotExists(filepath.Dir(storagePath)); err != nil {
		return nil, fmt.Errorf("Error while creating update directory for '%s': %v", storagePath, err)
	}

	// Put the update there
	f, err := os.Create(storagePath)
	if err != nil {
		return nil, fmt.Errorf("Cannot create file '%s': %s", storagePath, err)
	}
	defer f.Close()

	// create an md5 teereader
	md5Hasher := makeMd5Hasher(src)

	// copy the contents
	if _, err := io.Copy(f, md5Hasher.Reader); err != nil {
		return nil, fmt.Errorf("Error while saving update: %v", err)
	}

	log.Printf("[autoupdate] Copied new version '%s' of product '%s' to '%s'", version, product, storagePath)

	// save the metadata
	// ------------------
	metaData := &UpdateVersion{
		Version: *version,
		Product: product,
		Md5:     fmt.Sprintf("%32x", md5Hasher.Md5.Sum(nil)),
		Url:     fmt.Sprintf("/updates/products/%s/%s/%s-%s", product, version, product, version),
	}

	metaFileName := fmt.Sprintf("%s.meta.json", storagePath)
	metaFile, err := os.Create(metaFileName)
	if err != nil {
		return nil, fmt.Errorf("Cannot create metadata file '%s': %s", metaFileName, err)
	}
	defer metaFile.Close()

	// encode the metadata as json
	if err := json.NewEncoder(metaFile).Encode(metaData); err != nil {
		return nil, fmt.Errorf("Error while saving metadata: %v", err)
	}

	// Update the products latest version with the new metadata
	if err := a.updateExistingVersions(); err != nil {
		return nil, fmt.Errorf("Error while updating version list: %v", err)
	}

	return metaData, nil
}

// Returns a reader for the update file for the given version
func (a *baseAutoUpdater) FileForVersion() (io.Reader, error) {
	return nil, nil
}

// Returns the latest version of a product
func (a *baseAutoUpdater) LatestVersion(product string) (*UpdateVersion, error) {
	latestsVersion, hasProduct := a.latestVersions[product]
	if !hasProduct {
		return nil, fmt.Errorf("Cannot find product '%s'", product)
	}
	return latestsVersion, nil
}

// updates the versions list from the file system
func (a *baseAutoUpdater) updateExistingVersions() error {
	// Update the products latest version with the new metadata
	latestVersions, err := loadLatestVersions(a.basePath)
	if err != nil {
		return fmt.Errorf("Error while updating version list: %v", err)
	}

	a.latestVersions = latestVersions
	return nil
}

func loadLatestVersions(basePath string) (map[string]*UpdateVersion, error) {
	// load all products
	products, err := ioutil.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("Error while loading product names: %v", err)
	}

	productVersions := make(map[string]*UpdateVersion)

	// go through each product
	for _, productDir := range products {
		product := productDir.Name()

		versionDirs, err := ioutil.ReadDir(path.Join(basePath, product))
		if err != nil {
			return nil, fmt.Errorf("Error while loading product versions for '%s': %v", product, err)
		}

		// find the latest version
		versionNames := make([]string, len(versionDirs))
		for i, version := range versionDirs {
			// add each version
			versionNames[i] = version.Name()
		}
		sort.StringSlice(versionNames).Sort()

		// find the latest version
		newest := versionNames[len(versionNames)-1]
		metaFilePath := path.Join(basePath, product, newest, fmt.Sprintf("%s-%s.meta.json", product, newest))
		metafile, err := os.Open(metaFilePath)
		if err != nil {
			return nil, fmt.Errorf("Error while opening metadata file '%s': %v", metaFilePath, err)
		}
		defer metafile.Close()

		// deserialize the meta and update the latest version
		u := &UpdateVersion{}
		if err := json.NewDecoder(metafile).Decode(u); err != nil {
			return nil, fmt.Errorf("Error while deserializing metadata '%s': %v", metaFilePath, err)
		}

		log.Printf("[autoupdate] Found product: '%s' with versions: %v using: %s", product, versionNames, u.String())
		productVersions[product] = u

	}

	return productVersions, nil

}

// HTTP Handler
// ------------

func AutoupdateLatestVersionHandler(a AutoUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productName, err := getUrlParam(r.URL, "product")
		if err != nil {
			writeResponse(w, http.StatusBadRequest, "Missing 'product' parameter")
			return
		}

		latestVersion, err := a.LatestVersion(productName)
		if err != nil {
			writeResponse(w, http.StatusNotFound, fmt.Sprintf("Cannot find product '%s': %v", productName, err))
			return
		}

		if err := json.NewEncoder(w).Encode(latestVersion); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	}
}

func NewAutoupdateHttpHandler(u AutoUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(multipartMaxSize)
		if err != nil {
			writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Error while parsing multipart form: %v", err))
		}

		// parse the product name and version information
		productName, err := getMultipartParam(r.MultipartForm, "product")
		if err != nil {
			writeResponse(w, http.StatusBadRequest, "Missing 'product' parameter!")
			return
		}

		versionName, err := getMultipartParam(r.MultipartForm, "version")
		if err != nil {
			writeResponse(w, http.StatusBadRequest, "Missing 'version' parameter")
			return
		}

		version, err := StringToVersion(versionName)
		if err != nil {
			writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Cannot parse version '%s': %v", versionName, err))
			return
		}

		// get the request file
		sentFile, _, err := getMultipartFile(r.MultipartForm, "file")
		if err != nil {
			writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Error while extracting file: %v", err))
		}

		// delay closing the file body
		defer sentFile.Close()

		newVersion, err := u.AddNewVersion(productName, version, sentFile)
		if err != nil {
			writeResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error while saving new version: %v", err))
			return
		}

		wb := &bytes.Buffer{}
		if err := json.NewEncoder(wb).Encode(newVersion); err != nil {

		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(wb.Bytes())
	}
}
