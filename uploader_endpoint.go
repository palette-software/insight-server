package insight_server

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"crypto/md5"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

const (
	// The maximum size of the message we are willing to parse
	// when dealing with multipart messages
	multipartMaxSize = 128 * 1024 * 1024

	// The directory persions to use when creating a new directory
	OUTPUT_DEFAULT_DIRMODE = 0755
)

// A single file that was sent to us by the client
type UploadedFile struct {
	// The file name this client has sent us
	Filename string

	// The path where the server has stored this file
	UploadedPath string

	// The path where the output should go
	TargetPath string

	// The md5 of the file
	Md5 []byte
}

// Parameters for an upload request
type uploadRequest struct {
	sourceHost string
	username   string
	pkg        string
	filename   string
	timezone   string

	requestTime time.Time
	reader      io.Reader
}

type UploadCallbackCtx struct {
	SourceFile, OutputDir, OutputFile, Basedir, Timezone string
}

type UploadCallbackFn func(ctx *UploadCallbackCtx) error
type UploadCallback struct {
	Name          string
	Pkg, Filename *regexp.Regexp
	Handler       UploadCallbackFn
}

// A generic interface implementing the saving of a file
type Uploader interface {
	SaveFile(req *uploadRequest) (*UploadedFile, error)

	// returns the temporary directory path to use for storing files
	TempDirectory() string

	// Registers a callback that gets called with the uploaded filename if
	// both packageRegexp matches the package and filenameRegexp matches the
	// filename
	AddCallback(callback *UploadCallback)

	ApplyCallbacks(pkg, filename string, ctx *UploadCallbackCtx) error
}

// IMPLEMENTATIONS
// ===============

type basicUploader struct {
	// The directory where the files are uploaded
	baseDir string

	callbacks []*UploadCallback
}

// Creates a basic uploader
func MakeBasicUploader(basePath string) (Uploader, error) {
	log.Printf("[uploader] Using path '%v' for upload root", basePath)

	// create the uploader
	uploader := &basicUploader{
		baseDir:   basePath,
		callbacks: []*UploadCallback{},
	}
	// create the temp directory
	if err := uploader.createTempDirectory(); err != nil {
		return nil, err
	}

	return uploader, nil
}

// Returns the temp directory to use for storing transient files.
// This should be on the same device as the final destination
func (u *basicUploader) TempDirectory() string {
	return path.Join(u.baseDir, "_temp")
}

// Creates the temporary directory (this should prevent
// errors arising from non-existing temp path)
func (u *basicUploader) createTempDirectory() error {
	tmpDir := u.TempDirectory()

	tmpDirExists, err := fileExists(tmpDir)

	// if there was an error, forward it
	if err != nil {
		return err
	}

	if !tmpDirExists {
		// create the temporary path
		log.Printf("[uploader] Creating temp directory: %s", tmpDir)
		if err := os.MkdirAll(tmpDir, OUTPUT_DEFAULT_DIRMODE); err != nil {
			return err
		}
	}
	// just signal that we are using the existing path
	log.Printf("[uploader] Using path '%s' for temporary files", tmpDir)

	return nil
}

// Gets the file path inside the upload directory
func (u *basicUploader) getUploadPathForFile(req *uploadRequest, fileHash []byte) string {
	// the file name gets the timestamp appended (only time)
	fileTimestamp := req.requestTime.Format("15-04--05-00")

	filename := req.filename
	fileExt := path.Ext(filename)
	// get the extension and basename
	fullFileName := fmt.Sprintf("%v-%v-%032x.%v",
		SanitizeName(filename),
		fileTimestamp,
		fileHash,
		// remove the '.' from the file extension
		SanitizeName(fileExt[1:]),
	)

	return filepath.ToSlash(path.Join(
		u.baseDir,
		SanitizeName(req.username),
		"uploads",
		req.pkg,
		req.sourceHost,
		// the folder name is only the date
		// TODO: this may be necessary later
		//req.requestTime.Format("2006-01-02"),
		fullFileName,
	))
}

// Central function tries to create a new uploaded file.
// The purpose of this method is to provide a unified upload capability.
func (u *basicUploader) SaveFile(req *uploadRequest) (*UploadedFile, error) {

	hash := md5.New()

	// create a TeeReader that automatically forwards bytes read from the file to
	// the md5 hasher's reader
	readerWithMd5 := io.TeeReader(req.reader, hash)

	// create a temp file to move the bytes to (we do not yet know the hash of the file)
	tmpFile, err := ioutil.TempFile(u.TempDirectory(), "temporary-file-contents-")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	// write the data to the temp file (and hash in the meantime)
	bytesWritten, err := io.Copy(tmpFile, readerWithMd5)
	if err != nil {
		return nil, err
	}
	log.Printf("[upload] written %v bytes to '%v'\n", bytesWritten, tmpFile.Name())

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

	return &UploadedFile{
		Filename:     req.filename,
		UploadedPath: tempFilePath,
		TargetPath:   outputPath,
		Md5:          fileHash,
	}, nil
}

// The default fallback handler that gets invoked if
// no handlers are found
func MoveHandler(c *UploadCallbackCtx) error {

	// move the output file to the new path with the new name
	err := os.Rename(c.SourceFile, c.OutputFile)
	if err != nil {
		return err
	}

	log.Printf("[upload] Moved '%v' to '%v'\n", c.SourceFile, c.OutputFile)
	return nil
}

func (u *basicUploader) AddCallback(c *UploadCallback) {
	u.callbacks = append(u.callbacks, c)
}

// Applies callbacks after a file is uploaded succesfully
func (u *basicUploader) ApplyCallbacks(pkg, filename string, ctx *UploadCallbackCtx) error {

	// function that wraps invoking the handler
	invokeHandler := func(name string, handler UploadCallbackFn) error {
		log.Printf("[uploader.callbacks] Invoking callback: %s with '%v'", name, ctx)
		err := handler(ctx)
		if err != nil {
			log.Printf("[uploader.callbacks] Error during running '%s' for file %s::%s -- %v", name, pkg, filename, err)
			return err
		}
		return nil
	}

	// if we have this handled
	handled := false

	// try each added handler
	for _, callback := range u.callbacks {
		if callback.Pkg.MatchString(pkg) && callback.Filename.MatchString(filename) {
			err := invokeHandler(callback.Name, callback.Handler)
			if err != nil {
				return err
			}
			handled = true
		}
	}

	// fallback handler
	if !handled {
		err := invokeHandler("fallback", MoveHandler)
		if err != nil {
			return err
		}
	}

	return nil
}

// UPLOAD HANDLING
// ===============

// provides an actual implementation of the upload functionnality
func uploadHandlerInner(w http.ResponseWriter, req *http.Request, tenant User, uploader Uploader, maxidbackend MaxIdBackend) {

	// parse the multipart form
	err := req.ParseMultipartForm(multipartMaxSize)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Cannot parse multipart form: %v", err))
		return
	}

	// get the package from the URL
	pkg, err := getUrlParam(req.URL, "pkg")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "No 'pkg' parameter provided")
		return
	}

	// get the source host from the URL
	sourceHost, err := getUrlParam(req.URL, "host")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "No 'host' parameter provided")
		return
	}

	// get the package from the URL
	timezoneName, err := getUrlParam(req.URL, "tz")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "No 'tz' parameter provided")
		return
	}

	// get the actual file
	mainFile, fileName, err := getMultipartFile(req.MultipartForm, "_file")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Cannot find the field '_file' in the upload request")
		return
	}
	defer mainFile.Close()

	requestTime := time.Now()

	uploadedFile, err := uploader.SaveFile(&uploadRequest{
		sourceHost:  sourceHost,
		username:    tenant.GetUsername(),
		pkg:         pkg,
		filename:    fileName,
		requestTime: requestTime,
		reader:      mainFile,
		timezone:    timezoneName,
	})

	if err != nil {
		writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Error while saving uploaded file: %v", err))
		return
	}

	// check the md5
	md5Fields := req.MultipartForm.Value["_md5"]
	if len(md5Fields) != 1 {
		writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Only one instance of the '_md5' field allowed in the request, got: %v", len(md5Fields)))
		return
	}

	fileMd5, err := base64.StdEncoding.DecodeString(md5Fields[0])
	if err != nil {
		writeResponse(w, http.StatusBadRequest, "Cannot Base64 decode the submitted MD5")
		return
	}

	// compare the md5
	if !bytes.Equal(fileMd5, uploadedFile.Md5) {
		writeResponse(w, http.StatusConflict, "CONFLICT: Md5 Error")
		return
	}

	// get the maxid if any
	maxid, err := getUrlParam(req.URL, "maxid")
	if err == nil {
		tableName, tnError := getTableNameFromFilename(fileName)
		if tnError != nil {
			writeResponse(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
			return
		}
		// if we have the maxid parameter, save it
		maxidbackend.SaveMaxId(tenant.GetUsername(), tableName, maxid)
	}

	// apply any callbacks
	err = uploader.ApplyCallbacks(pkg, fileName,
		&UploadCallbackCtx{
			SourceFile: uploadedFile.UploadedPath,
			OutputDir:  filepath.Dir(uploadedFile.TargetPath),
			OutputFile: uploadedFile.TargetPath,
			Basedir:    uploader.TempDirectory(),
			Timezone:   timezoneName,
		})
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Error in upload callbacks")
		return
	}

	// signal that everything went ok
	writeResponse(w, http.StatusOK, "")

}

// Creates an http endpoint handler where
func MakeUploadHandler(uploader Uploader, maxidBackend MaxIdBackend) HandlerFuncWithTenant {
	return func(w http.ResponseWriter, r *http.Request, tenant User) {
		uploadHandlerInner(w, r, tenant, uploader, maxidBackend)
	}
}
