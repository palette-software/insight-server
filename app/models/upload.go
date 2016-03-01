package models

import (
	"github.com/revel/revel"

	"fmt"
	"io"
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
	Md5 string
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

///// Returns the path where a file needs to be placed
//func getUploadPath(uploadBasePath, filename string, requestTime time.Time, fileHash string) string {
//// the folder name is only the date
//folderTimestamp := requestTime.Format("2006-01-02")
//// the file name gets the timestamp appended (only time)
//fileTimestamp := requestTime.Format("15-04--05-00")

//// get the extension and basename
//fileBaseName := SanitizeName(filename)
//fileExtName := SanitizeName(path.Ext(filename))
//fullFileName := fmt.Sprintf("%v-%v-%v.%v", fileBaseName, fileTimestamp, fileHash, fileExtName[1:])

//// the file name is the sanitized file name
//return filepath.ToSlash(path.Join(uploadBasePath, folderTimestamp, fullFileName))
//}

// tries to create a new uploaded file. This method does not check the md5, just uses it as part of the filename
func NewUploadedFile(uploadBasePath, filename, md5 string, requestTime time.Time, reader io.Reader) (*UploadedFile, error) {
	//outputPath := getUploadPath(tenant.HomeDirectory, pkg, filename, requestTime, md5)
	outputPath := filepath.ToSlash(path.Join(uploadBasePath, getUploadPathForFile(filename, md5, requestTime)))

	// create the directory of the file
	if err := os.MkdirAll(filepath.Dir(outputPath), OUTPUT_DEFAULT_DIRMODE); err != nil {
		return nil, err
	}

	// Create the output file
	newFile, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	defer newFile.Close()

	// Copy the bytes to destination from source
	bytesWritten, err := io.Copy(newFile, reader)
	if err != nil {
		return nil, err
	}
	revel.INFO.Printf("Copied %d bytes to '%v'", bytesWritten, outputPath)

	return &UploadedFile{
		Filename:     filename,
		UploadedPath: outputPath,
		Md5:          md5,
	}, nil
}
