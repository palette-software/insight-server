package insight_server

//go:generate go-bindata -pkg $GOPACKAGE -o assets.go assets/

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

// GENERIC HELPERS
// ===============

// simple helper that logs an error then panics
func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

// The regexp we use for sanitizing any strings to a file name that is valid on all systems
var sanitizeRegexp = regexp.MustCompile("[^_A-Za-z0-9]")

// Returns a sanitized filename with all non-alphanumeric characters replaced by dashes
func SanitizeName(name string) string {
	return sanitizeRegexp.ReplaceAllString(name, "-")
}

// Writes the error message to the log then responds with an error message
func writeResponse(w http.ResponseWriter, status int, err string) {
	log.Printf("[http] {%v}: %s", status, err)
	http.Error(w, err, status)
	return
}

// FS HELPERS
// ==========

// Returns whether the given file or directory exists or not
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Returns true if path is a directory. If it does not exist err is returned
func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// Returns true if path is a directory. Otherwise (even if there was an error) returns false.
func isDirectoryNoFail(path string) bool {
	isDir, err := isDirectory(path)
	return (err == nil && isDir)
}

// / Helper that creates a directory if it does not exist
func createDirectoryIfNotExists(path string) error {
	exists, err := fileExists(path)
	// forward errors
	if err != nil {
		return err
	}
	// if it already exists, dont create it
	if exists {
		return nil
	}

	// create the directory
	log.Printf("[storage] Creating directory: '%s'", path)
	if err := os.MkdirAll(path, OUTPUT_DEFAULT_DIRMODE); err != nil {
		return err
	}

	return nil
}

// HTTP PACKAGE HELPERS
// ====================

// Helper to get a part from a multipart message
func getMultipartFile(form *multipart.Form, fieldName string) (file multipart.File, fileName string, err error) {

	// get the file from the form
	fn := form.File[fieldName]
	if len(fn) != 1 {
		err = fmt.Errorf("The request must have exactly 1 '%v' field (has %v).", fieldName, len(fn))
		return
	}

	// take the first one
	uploadedFile := fn[0]

	// set the filename
	fileName = uploadedFile.Filename

	// get the file reader
	file, err = uploadedFile.Open()
	if err != nil {
		return
	}

	return
}

// Helper to get a part from a multipart message
func getMultipartParam(form *multipart.Form, fieldName string) (value string, err error) {

	// get the file from the form
	fn := form.Value[fieldName]
	if len(fn) != 1 {
		err = fmt.Errorf("The request must have exactly 1 '%v' field (has %v).", fieldName, len(fn))
		return "", err
	}

	return fn[0], nil
}

// Returns an url param, or an error if no such param is available
func getUrlParam(reqUrl *url.URL, paramName string) (string, error) {

	// parse the url params
	urlParams, err := url.ParseQuery(reqUrl.RawQuery)
	if err != nil {
		return "", err
	}

	// get the package
	paramVals := urlParams[paramName]
	if len(paramVals) != 1 {
		return "", fmt.Errorf("BAD REQUEST: No '%v' parameter provided", paramName)
	}

	return paramVals[0], nil
}

// Returns a new handler that simply responds with an asset from the precompiled assets
func AssetPageHandler(assetName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := Asset(assetName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write(page)
	}
}

// Gets the version string of the server from the VERSION file in the assets directory
// (this should be filled by travis)
func GetVersion() string {
	version, err := Asset("assets/VERSION")
	if err != nil {
		return "v1.0.0"
	}
	return string(version)
}

// MD5 hashing TeeReader helper
// ----------------------------

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

// Returns the hash of the tree
func (m *Md5Hasher) GetHash() []byte {
	return m.Md5.Sum(nil)
}

// Returns the (lowercased) hex string of the Md5
func (m *Md5Hasher) GetHashString() string {
	return fmt.Sprintf("%32x", m.GetHash())
}
