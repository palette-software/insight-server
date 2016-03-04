package insight_server

import (
	"net/http"
	"fmt"
	"time"
	"encoding/base64"
	"bytes"
	"log"

	"path/filepath"
	"os"
	"path"
	"io"
	"crypto/md5"
	"io/ioutil"
)

const (
// The key in the Environment where the files will be uploaded
// if no such value is set in the env, use ENV["TEMP"]
	UploadPathEnvKey = "INSIGHT_UPLOAD_HOME"
)

const (
	OUTPUT_DEFAULT_DIRMODE = 0755
)

// A single file that was sent to us by the client
type UploadedFile struct {
	// The file name this client has sent us
	Filename     string

	// The path where the server has stored this file
	UploadedPath string

	// The md5 of the file
	Md5          []byte
}

// Parameters for an upload request
type uploadRequest struct {
	username    string
	pkg         string
	filename    string

	requestTime time.Time
	reader      io.Reader
}

// A generic interface implementing the saving of a file
type Uploader interface {
	SaveFile(req *uploadRequest) (*UploadedFile, error)
}


// IMPLEMENTATIONS
// ===============

type basicUploader struct {
	// The directory where the files are uploaded
	baseDir string
}

// Creates a basic uploader
func MakeBasicUploader(basePath string) Uploader {
	log.Printf("[uploader] Using path '%v' for upload root", basePath)
	return &basicUploader{
		baseDir: basePath,
	}
}


// Gets the file path inside the upload directory
func (u *basicUploader) getUploadPathForFile(req *uploadRequest, fileHash []byte) string {
	// the folder name is only the date
	folderTimestamp := req.requestTime.Format("2006-01-02")
	// the file name gets the timestamp appended (only time)
	fileTimestamp := req.requestTime.Format("15-04--05-00")

	filename := req.filename
	// get the extension and basename
	fullFileName := fmt.Sprintf("%v-%v-%x.%v",
		SanitizeName(filename),
		fileTimestamp,
		fileHash,
		SanitizeName(path.Ext(filename)),
	)

	return filepath.ToSlash(path.Join(u.baseDir, folderTimestamp, fullFileName))
}


// Central function tries to create a new uploaded file.
// The purpose of this method is to provide a unified upload capability.
func (u *basicUploader) SaveFile(req *uploadRequest) (*UploadedFile, error) {

	hash := md5.New()

	// create a TeeReader that automatically forwards bytes read from the file to
	// the md5 hasher's reader
	readerWithMd5 := io.TeeReader(req.reader, hash)

	// create a temp file to move the bytes to (we do not yet know the hash of the file)
	tmpFile, err := ioutil.TempFile("", "temporary-file-contents-")
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

	// generate the output file name
	outputPath := u.getUploadPathForFile(req, fileHash)

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
		Filename:     req.filename,
		UploadedPath: outputPath,
		Md5:          fileHash,
	}, nil
}


// UPLOAD HANDLING
// ===============

// provides an actual implementation of the upload functionnality
func uploadHandlerInner(w http.ResponseWriter, req *http.Request, tenant User, uploader Uploader) {
	log.Printf("[HTTP] {%v} Request arrived to: %v\n", req.Method, req.URL)

	// parse the multipart form
	err := req.ParseMultipartForm(128 * 1024 * 1024)
	if err != nil {
		logError(w, http.StatusBadRequest, "Cannot parse multipart form")
		return
	}

	pkg, err := getUrlParam(req.URL, "pkg")
	if err != nil {
		logError(w, http.StatusBadRequest, "No _pkg parameter provided")
		return
	}

	// get the actual file
	mainFile, fileName, err := getMultipartFile(req.MultipartForm, "_file")
	if err != nil {
		logError(w, http.StatusBadRequest, "Cannot find the field '_file' in the upload request")
		return
	}
	defer mainFile.Close()

	requestTime := time.Now()

	uploadedFile, err := uploader.SaveFile(&uploadRequest{
		username: tenant.GetUsername(),
		pkg: pkg,
		filename: fileName,
		requestTime: requestTime,
		reader: mainFile,

	})

	if err != nil {
		logError(w, http.StatusBadRequest, fmt.Sprintf("Error while saving uploaded file: %v", err))
		return
	}

	// check the md5
	md5Fields := req.MultipartForm.Value["_md5"]
	if len(md5Fields) != 1 {
		logError(w, http.StatusBadRequest, fmt.Sprintf("Only one instance of the '_md5' field allowed in the request, got: %v", len(md5Fields)))
		return
	}

	fileMd5, err := base64.StdEncoding.DecodeString(md5Fields[0])
	if err != nil {
		logError(w, http.StatusBadRequest, "Cannot Base64 decode the submitted MD5")
		return
	}

	// compare the md5
	if !bytes.Equal(fileMd5, uploadedFile.Md5) {
		logError(w, http.StatusConflict, "CONFLICT: Md5 Error")
		return
	}

}

// Creates an http endpoint handler where
func MakeUploadHandler(uploader Uploader) HandlerFuncWithTenant {
	return func(w http.ResponseWriter, r *http.Request, tenant User) {
		uploadHandlerInner(w, r, tenant, uploader)
	}
}
