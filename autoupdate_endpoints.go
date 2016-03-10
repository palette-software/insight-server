package insight_server

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
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

// version list sorting
// --------------------

func (v VersionList) Len() int {
	return len([]*Version(v))
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (v VersionList) Less(i, j int) bool {
	l := []*Version(v)
	a := l[i]
	b := l[j]
	if a.Major < b.Major {
		return true
	}

	if a.Major == b.Major {
		if a.Minor == b.Minor {
			return a.Patch < b.Patch
		}
		return a.Minor < b.Minor
	}
	return a.Major < b.Major
}

// Swap swaps the elements with indexes i and j.
func (v VersionList) Swap(i, j int) {
	l := []*Version(v)
	a := l[i]
	b := l[j]

	l[j] = a
	l[i] = b
}

// Autoupdater implementation
// --------------------------

type ProductVersions map[string]map[string]*UpdateVersion

type baseAutoUpdater struct {
	// The base path where updates are stored
	basePath string

	productVersions map[string]map[string]*UpdateVersion
}

// Creates a new autoupdater implementation
func NewBaseAutoUpdater(basePath string) (AutoUpdater, error) {
	if err := createDirectoryIfNotExists(basePath); err != nil {
		return nil, err
	}

	return &baseAutoUpdater{
		basePath:        basePath,
		productVersions: map[string]map[string]*UpdateVersion{},
	}, nil
}

// The file name we use to store a verison inside its own folder
const CONTENTS_FILE_NAME = "contents.bin"

// Gets the path where an update binary is stored
func (a *baseAutoUpdater) updatePath(product, versionStr string) string {
	return path.Join(a.basePath, SanitizeName(product), versionStr, CONTENTS_FILE_NAME)
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

	return &UpdateVersion{
		Version: *version,
		Product: product,
		Md5:     fmt.Sprintf("%32x", md5Hasher.Md5.Sum(nil)),
	}, nil
}

// Returns a reader for the update file for the given version
func (a *baseAutoUpdater) FileForVersion() (io.Reader, error) {
	return nil, nil
}

// Returns the latest version of a product
func (a *baseAutoUpdater) LatestVersion(product string) (*UpdateVersion, error) {
	return nil, nil
}

// HTTP Handler
// ------------

type Md5Hasher struct {
	Md5    hash.Hash
	Reader io.Reader
}

func makeMd5Hasher(r io.Reader) *Md5Hasher {

	hash := md5.New()

	// create a TeeReader that automatically forwards bytes read from the file to
	// the md5 hasher's reader
	readerWithMd5 := io.TeeReader(r, hash)

	return &Md5Hasher{hash, readerWithMd5}
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

func VersionUploadPagetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(UPLOAD_PAGE))
}

const UPLOAD_PAGE = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- The above 3 meta tags *must* come first in the head; any other head content must come *after* these tags -->
    <title>Adding Users</title>

    <!-- Bootstrap -->

    <!-- Latest compiled and minified CSS -->
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap.min.css" integrity="sha384-1q8mTJOASx8j1Au+a5WDVnPi2lkFfwwEAa8hDDdjZlpLegxhjVME1fgjWPGmkzs7" crossorigin="anonymous">

    <!-- Optional theme -->
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.6/css/bootstrap-theme.min.css" integrity="sha384-fLW2N01lMqjakBkx3l/M9EahuwpSfeNvV63J5ezn3uZzapT0u7EYsXMjQV+0En5r" crossorigin="anonymous">

    <!-- HTML5 shim and Respond.js for IE8 support of HTML5 elements and media queries -->
    <!-- WARNING: Respond.js doesn't work if you view the page via file:// -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.2/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->
  </head>
  <body>


    <div class="container">
        <div class="row">
            <div class="col-sm-12">
                <div class="page-header">
                    <h1>Add a update version</h1>
                </div>

                <form method="POST" action="/updates/add-version" enctype="multipart/form-data">

                  <div class="form-group">
                    <label for="product">Product</label>
                    <select name="product" class="form-control">
                    	<option value="agent">Agent</option>
                    </select>
                  </div>

                  <div class="form-group">
                    <label for="product">Version</label>
                    <input type="text" class="form-control" name="version" value="v1.3.3" />
                  </div>

                  <div class="form-group">
                    <label for="file">File</label>
                    <input type="file" class="form-control" name="file" />
                  </div>


                  <button type="submit" class="btn btn-default">Submit</button>
                </form>
        </div>
        </div>
    </div>

    <!-- jQuery (necessary for Bootstrap's JavaScript plugins) -->
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/1.11.3/jquery.min.js"></script>
    <!-- Include all compiled plugins (below), or include individual files as needed -->
    <script src="js/bootstrap.min.js"></script>
  </body>
</html>
`
