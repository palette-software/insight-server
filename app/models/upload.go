package models

import (
	"github.com/revel/revel"

	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	OUTPUT_DEFAULT_MODE    = 0644
	OUTPUT_DEFAULT_DIRMODE = 0755
)

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
	Uploader Tenant

	// The package this upload is part of
	Package string

	// Indicates if there is metadata coming in with this upload
	HasMeta bool
}

// Gets the path where a certain tenants files for the given package reside
func getUploadBasePath(tenantHomeDir, pkg string) string {
	return filepath.ToSlash(path.Join(GetOutputDirectory(), tenantHomeDir, "uploads", SanitizeName(pkg)))
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

// tries to create a new uploaded file. This method does not check the md5, just uses it as part of the filename
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
	revel.INFO.Printf("[Upload] written %v bytes to '%v'", bytesWritten, tmpFile.Name())

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

	revel.INFO.Printf("[Upload] Moved '%v' to '%v'", tempFilePath, outputPath)

	return &UploadedFile{
		Filename:     filename,
		UploadedPath: outputPath,
		Md5:          fileHash,
	}, nil
}

// Create a new UploadedCsv struct from the provided parameters.
func NewUploadedCsv(tenant *Tenant, pkg, filename, filemd5 string, requestTime time.Time, fileReader, metadataReader io.Reader) (*UploadedCsv, error) {

	// get the base path for uploads
	basePath := getUploadBasePath(tenant.HomeDirectory, pkg)

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
		Uploader: *tenant,
		Package:  pkg,
		HasMeta:  true,
	}, nil
}
