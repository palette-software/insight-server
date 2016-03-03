package insight_server

import (

	"net/http"
	"mime/multipart"
	"fmt"
	"time"
	"encoding/base64"
	"bytes"
	"net/url"
	"log"

	"path/filepath"
	"os"
	"path"
	"regexp"
	"io"
	"crypto/md5"
	"io/ioutil"
)

const (

	// The key in the Environment where the files will be uploaded
	// if no such value is set in the env, use ENV["TEMP"]
	UploadPathEnvKey = "INSIGHT_UPLOAD_HOME"

)

// HELPERS
// =======

// simple helper that logs an error then panics
func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

// The regexp we use for sanitizing any strings to a file name that is valid on all systems
var sanitizeRegexp = regexp.MustCompile("[^A-Za-z0-9]+")

// Returns a sanitized filename with all non-alphanumeric characters replaced by dashes
func SanitizeName(name string) string {
	return sanitizeRegexp.ReplaceAllString(name, "-")
}

// Writes the error message to the log then responds with an error message
func logError(w http.ResponseWriter, status int, err string) {
	log.Println(err)
	http.Error(w, err, status)
	return
}


// INITIALIZERS
// ============

// Called on start of the server
func Boot() {
	initLicenses()
}

const (
	OUTPUT_DEFAULT_DIRMODE = 0755
)


// DATA MODELS
// ===========

// A single file that was sent to us by the client
type UploadedFile struct {
	// The file name this client has sent us
	Filename string

	// The path where the server has stored this file
	UploadedPath string

	// The md5 of the file
	Md5 []byte
}

// Represents an uploaded CSV file with its metadata
type UploadedCsv struct {

	// The data file that has been uploaded
	Csv UploadedFile

	// The metadata file that was uploaded
	Metadata UploadedFile

	// The person uploading this file
	// TODO: since this contains the hashed auth token of the tenant, it should be better to exclude that somehow
	Uploader string

	// The package this upload is part of
	Package string

	// Indicates if there is metadata coming in with this upload
	HasMeta bool
}

// Gets the path where a certain tenants files for the given package reside
func getUploadBasePath(tenantHomeDir, pkg string) string {
	uploadBaseDir := os.Getenv(UploadPathEnvKey)
	if uploadBaseDir == "" {
		uploadBaseDir = path.Join(os.Getenv("TEMP"), "uploads")
	}
	return filepath.ToSlash(path.Join(uploadBaseDir, tenantHomeDir, "uploads", SanitizeName(pkg)))
}

// Gets the file path inside the upload directory
func getUploadPathForFile(filename, fileHash string, requestTime time.Time) string {
	// the folder name is only the date
	folderTimestamp := requestTime.Format("2006-01-02")
	// the file name gets the timestamp appended (only time)
	fileTimestamp := requestTime.Format("15-04--05-00")

	// get the extension and basename
	fileBaseName := SanitizeName(filename)
	fileExtName := SanitizeName(path.Ext(filename))
	fullFileName := fmt.Sprintf("%v-%v-%v.%v", fileBaseName, fileTimestamp, fileHash, fileExtName[1:])

	return filepath.ToSlash(path.Join(folderTimestamp, fullFileName))
}


// Central function tries to create a new uploaded file.
// The purpose of this method is to provide a unified upload capability.
func NewUploadedFile(uploadBasePath, filename string, requestTime time.Time, reader io.Reader) (*UploadedFile, error) {

	hash := md5.New()

	// create a TeeReader that automatically forwards bytes read from the file to
	// the md5 hasher's reader
	readerWithMd5 := io.TeeReader(reader, hash)

	// create a temp file to move the bytes to (we do not yet know the hash of the file)
	tmpFile, err := ioutil.TempFile("", "temporary-file-contents")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	// write the data to the temp file (and hash in the meantime)
	bytesWritten, err := io.Copy(tmpFile, readerWithMd5)
	if err != nil {
		return nil, err
	}
	log.Printf("[Upload] written %v bytes to '%v'\n", bytesWritten, tmpFile.Name())

	// get the hash from the teewriter
	fileHash := hash.Sum(nil)
	// make a hex string out of the md5
	md5str := fmt.Sprintf("%x", fileHash)

	// generate the output file name
	outputPath := filepath.ToSlash(path.Join(uploadBasePath, getUploadPathForFile(filename, md5str, requestTime)))

	// create the output file path
	if err := os.MkdirAll(filepath.Dir(outputPath), OUTPUT_DEFAULT_DIRMODE); err != nil {
		return nil, err
	}

	// Get the temp file name before closing it
	tempFilePath := tmpFile.Name()

	// close the temp file, so writes get flushed
	tmpFile.Close()

	// move the output file to the new path with the new name
	err = os.Rename(tempFilePath, outputPath)
	if err != nil {
		return nil, err
	}

	log.Printf("[Upload] Moved '%v' to '%v'\n", tempFilePath, outputPath)

	return &UploadedFile{
		Filename:     filename,
		UploadedPath: outputPath,
		Md5:          fileHash,
	}, nil
}

// Create a new UploadedCsv struct from the provided parameters.
func NewUploadedCsv(username, pkg, filename string, requestTime time.Time, fileReader, metadataReader io.Reader) (*UploadedCsv, error) {

	// get the base path for uploads
	basePath := getUploadBasePath(username, pkg)

	mainFile, err := NewUploadedFile(basePath, filename, requestTime, fileReader)
	if err != nil {
		return nil, err
	}

	metaFile, err := NewUploadedFile(basePath, fmt.Sprintf("%s.meta", filename), requestTime, metadataReader)
	if err != nil {
		return nil, err
	}

	return &UploadedCsv{
		Csv:      *mainFile,
		Metadata: *metaFile,
		Uploader: username,
		Package:  pkg,
		HasMeta:  true,
	}, nil
}
// AUTH
// ====

// UPLOAD HANDLING
// ===============


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

// The actual upload handler
func UploadHanlder(w http.ResponseWriter, req *http.Request, tenant *License) {

	// parse the multipart form
	err := req.ParseMultipartForm(128 * 1024 * 1024)
	if err != nil {
		panic(err)
	}

	// parse the url params
	urlParams, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		panic(err)
	}

	// get the package
	pkgs := urlParams["pkg"]
	if len(pkgs) != 1 {
		http.Error(w, "BAD REQUEST: No 'pkg' parameter provided", 400)
	}
	pkg := pkgs[0]

	// get the tenant


	// get the actual file
	mainFile, fileName, err := getMultipartFile(req.MultipartForm, "_file")
	if err != nil {
		panic(err)
	}
	defer mainFile.Close()

	// get the metadata file
	metaFile, _, err := getMultipartFile(req.MultipartForm, "_meta")
	if err != nil {
		panic(err)
	}
	defer metaFile.Close()

	requestTime := time.Now()
	newUploadedPack, err := NewUploadedCsv(tenant.LicenseId, pkg, fileName, requestTime, mainFile, metaFile)
	if err != nil {
		panic(err)
	}

	// check the md5
	md5Fields := req.MultipartForm.Value["_md5"]
	if len(md5Fields) != 1 {
		panic(fmt.Errorf("Only one instance of the '_md5' field allowed in the request, got: %v", len(md5Fields)))
	}

	fileMd5, err := base64.StdEncoding.DecodeString(md5Fields[0])
	if err != nil {
		panic(err)
	}

	// compare the md5
	if !bytes.Equal(fileMd5, newUploadedPack.Csv.Md5) {
		logError(w, http.StatusConflict, "CONFLICT: Md5 Error")
		return
	}

	return
}
