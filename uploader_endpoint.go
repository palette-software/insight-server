package insight_server

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"time"

	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

type UploadMeta struct {
	// The filename as submitted by the uploader
	OriginalFilename string

	// The tenant this file belongs to
	Tenant string

	// The package this file was uploaded into
	Pkg string

	// The host where we got this table
	Host string

	// The original name of the uploaded table
	TableName string

	// The time the agent created this file
	Date time.Time

	// The timezone the host is from
	Timezone *time.Location

	// The part and seq indices originally sent
	SeqIdx  int
	PartIdx int

	// The orignal Md5 the agent sent us
	Md5 []byte
}

// Returns the file name for an upload request
func (u *UploadMeta) GetOutputFilename(baseDir string) string {
	return filepath.ToSlash(path.Join(
		baseDir,
		SanitizeName(u.Tenant),
		"uploads",
		SanitizeName(u.Pkg),
		SanitizeName(u.Host),
		fmt.Sprintf("%s-%s--seq%03d--part%04d-{{md5}}.csv",
			SanitizeName(u.TableName),
			// the current time as part of the 2nd timestamp
			u.Date.UTC().Format("2006-01-02--15-04-05"),
			u.PartIdx,
			u.SeqIdx,
		),
	))
}

// Gets a proper writer for this file
func (u *UploadMeta) GetOutputGzippedWriter(baseDir, tmpDir string) (*GzippedFileWriterWithTemp, error) {
	return NewGzippedFileWriterWithTemp(u.GetOutputFilename(baseDir), tmpDir)
}

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

	// The name of the file as it was uploaded
	OriginalFileName string

	// The host that uploaded this file
	Host string
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
	tmpDir := uploader.TempDirectory()
	if err := CreateDirectoryIfNotExists(tmpDir); err != nil {
		return nil, fmt.Errorf("Error creating temporary directory '%s': %v", tmpDir, err)
	}

	return uploader, nil
}

// Returns the temp directory to use for storing transient files.
// This should be on the same device as the final destination
func (u *basicUploader) TempDirectory() string {
	return path.Join(u.baseDir, "_temp")
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

	// get the table name
	tableName, seqIdx, partIdx, err := getTableInfoFromFilename(req.filename)
	if err != nil {
		return nil, err
	}

	// build the upload metadata
	meta := &UploadMeta{
		Pkg:       req.pkg,
		Host:      req.sourceHost,
		TableName: tableName,

		Date:    req.requestTime,
		SeqIdx:  seqIdx,
		PartIdx: partIdx,
	}

	// Get a gzipped writer for the file
	tmpOutputFile, err := meta.GetOutputGzippedWriter(u.baseDir, u.TempDirectory())
	if err != nil {
		return nil, fmt.Errorf("Error opening Gzipped output writer for %s: %v", req.filename, err)
	}

	// write the data to the temp file (and hash in the meantime)
	bytesWritten, err := io.Copy(tmpOutputFile, req.reader)
	if err != nil {
		return nil, err
	}
	log.Printf("[upload] written %d bytes to", bytesWritten)

	//// create the hasher that will hash the contents during the write
	//md5Hasher := makeMd5Hasher(req.reader)
	//// The prefix for the temporary file name. Useful
	//// if we want to re-processed unprocessed files that
	//// are stuck in the temporary directory because of an error.
	//tmpFilePrefix := fmt.Sprintf("uploaded---%s---", req.filename)
	//
	//// create a temp file to move the bytes to (we do not yet know the hash of the file)
	//tmpFile, err := ioutil.TempFile(u.TempDirectory(), tmpFilePrefix)
	//if err != nil {
	//	return nil, err
	//}
	//defer tmpFile.Close()
	//
	//// Create a gzip writer to write a compressed output
	//gzipWriter := gzip.NewWriter(tmpFile)
	//defer gzipWriter.Close()
	//
	//// write the data to the temp file (and hash in the meantime)
	//bytesWritten, err := io.Copy(gzipWriter, md5Hasher.Reader)
	//if err != nil {
	//	return nil, err
	//}
	//log.Printf("[upload] written %v bytes to '%v'\n", bytesWritten, tmpFile.Name())
	//
	//fileHash := md5Hasher.GetHash()
	//
	//// generate the output file name, and mark that its a gzipped one
	//outputPath := fmt.Sprintf("%s.gz", u.getUploadPathForFile(req, fileHash))
	//
	//// create the directory of the uploaded file
	//if err := CreateDirectoryIfNotExists(filepath.Dir(outputPath)); err != nil {
	//	return nil, err
	//}
	//
	//// Get the temp file name before closing it
	//tempFilePath := tmpFile.Name()
	//
	//// close the temp file, so writes get flushed
	//gzipWriter.Close()
	//tmpFile.Close()

	return nil, nil
	//return &UploadedFile{
	//	Filename:     req.filename,
	//	UploadedPath: tmpOutputFile.tempFilePath,
	//	TargetPath:   outputPath,
	//	Md5:          fileHash,
	//}, nil
}

func (u *basicUploader) AddCallback(c *UploadCallback) {
	u.callbacks = append(u.callbacks, c)
}

// Applies callbacks after a file is uploaded succesfully
func (u *basicUploader) ApplyCallbacks(pkg, filename string, ctx *UploadCallbackCtx) error {

	// function that wraps invoking the handler
	invokeHandler := func(name string, handler UploadCallbackFn) error {
		log.Printf("[uploader.callbacks] Invoking callback: %s ", name)
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
		err := invokeHandler("move", MoveHandler)
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

	// get the params
	urlParams, err := getUrlParams(req.URL, "pkg", "host", "tz")
	if err != nil {
		writeResponse(w, http.StatusBadRequest, fmt.Sprint(err))
	}

	pkg, sourceHost, timezoneName := urlParams[0], urlParams[1], urlParams[2]

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
		writeResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error while saving uploaded file: %v", err))
		return
	}

	// Compare the md5 with the sent one
	// ---------------------------------

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

	// Save the maxid if needed
	// ------------------------

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

	// Apply the callbacks

	// apply any callbacks
	err = uploader.ApplyCallbacks(pkg, fileName,
		&UploadCallbackCtx{
			SourceFile: uploadedFile.UploadedPath,
			OutputDir:  filepath.Dir(uploadedFile.TargetPath),
			OutputFile: uploadedFile.TargetPath,
			Basedir:    uploader.TempDirectory(),
			Timezone:   timezoneName,

			// add some source information
			OriginalFileName: fileName,
			Host:             sourceHost,
		})

	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "Error in upload callbacks")
		return
	}

	// signal that everything went ok
	writeResponse(w, http.StatusOK, "")

}

// Creates an http endpoint handler where
func MakeUploadHandler(uploader Uploader, maxidBackend MaxIdBackend, tmpDir, baseDir, archivesDir string) HandlerFuncWithTenant {
	// the fallback handler to move files
	fallbackHandler := &FallbackUploadHandler{tmpDir: tmpDir, baseDir: baseDir}

	// processing handlers
	handlers := []UploadHandler{
		NewJsonServerlogsUploadHandler(tmpDir, baseDir, archivesDir),
	}

	return func(w http.ResponseWriter, r *http.Request, tenant User) {
		//uploadHandlerInner(w, r, tenant, uploader, maxidBackend)

		// Convert the request to metadata for handling
		meta, mainFile, err := makeMetaFromRequest(r, tenant.GetUsername())
		if err != nil {
			writeResponse(w, http.StatusBadRequest, fmt.Sprint(err))
			return
		}
		defer mainFile.Close()

		// find the handler for this table
		if err := findUploadHandler(meta, handlers, fallbackHandler).HandleUpload(meta, mainFile); err != nil {
			writeResponse(w, http.StatusInternalServerError, fmt.Sprint(err))
			return
		}

		writeResponse(w, http.StatusOK, "OK")
	}
}

// DEFAULT UPLOAD HANDLERS
// =======================

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

// Soring callbacks
// ----------------

// Converts an upload request to its metadata equivalent
func makeMetaFromRequest(req *http.Request, tenantName string) (*UploadMeta, multipart.File, error) {
	requestTime := time.Now()

	// parse the multipart form
	err := req.ParseMultipartForm(multipartMaxSize)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot parse multipart form: %v", err)
	}

	// get the params
	urlParams, err := getUrlParams(req.URL, "pkg", "host", "tz")
	if err != nil {
		return nil, nil, err
	}

	pkg, sourceHost, timezoneName := urlParams[0], urlParams[1], urlParams[2]

	// parse the timezone
	// try to parse the timezone name
	sourceTimezone, err := time.LoadLocation(timezoneName)
	if err != nil {
		return nil, nil, fmt.Errorf("Unknown time zone for agent  '%s': %v", timezoneName, err)
	}

	// check the md5
	md5Fields := req.MultipartForm.Value["_md5"]
	if len(md5Fields) != 1 {
		return nil, nil, fmt.Errorf("Only one instance of the '_md5' field allowed in the request, got: %v", len(md5Fields))
	}

	fileMd5, err := base64.StdEncoding.DecodeString(md5Fields[0])
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot Base64 decode the submitted MD5 '%s': %v", md5Fields[0], fileMd5)
	}

	// get the actual file
	mainFile, fileName, err := getMultipartFile(req.MultipartForm, "_file")
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot find the field '_file' in the upload request")
	}

	// get the table name
	tableName, seqIdx, partIdx, err := getTableInfoFromFilename(fileName)
	if err != nil {
		// Close the file if we have errors here
		mainFile.Close()
		return nil, nil, err
	}

	// build the upload metadata
	return &UploadMeta{
		OriginalFilename: fileName,

		Tenant: tenantName,

		Pkg:       pkg,
		Host:      sourceHost,
		TableName: tableName,

		Date:     requestTime,
		Timezone: sourceTimezone,
		SeqIdx:   seqIdx,
		PartIdx:  partIdx,
	}, mainFile, nil
}

type UploadHandler interface {
	HandleUpload(meta *UploadMeta, reader multipart.File) error
	// Returns true if this handler can handle this request
	CanHandle(meta *UploadMeta) bool
}

// Finds the upload handler to be used.
// If no suitable handler is found, the fallback is returned
func findUploadHandler(meta *UploadMeta, handlers []UploadHandler, fallback UploadHandler) UploadHandler {
	for _, handler := range handlers {
		if handler.CanHandle(meta) {
			return handler
		}
	}
	return fallback
}

// Helpers
// -------

// Shared handler to copy an uploaded file to a location
func copyUploadedFileTo(meta *UploadMeta, reader multipart.File, baseDir, tmpDir string) (outFileName string, err error) {

	// create the output writer
	outputWriter, err := meta.GetOutputGzippedWriter(baseDir, tmpDir)
	if err != nil {
		return "", fmt.Errorf("Error opening gzipped output: %v", err)
	}
	// safety defered close to always close the file
	defer outputWriter.Close()

	// copy the data to the output
	bytesWritten, err := io.Copy(outputWriter, reader)
	if err != nil {
		return "", fmt.Errorf("Error copying data to output: %v", err)
	}

	// pick up any errors during close
	if err := outputWriter.Close(); err != nil {
		return "", fmt.Errorf("Error writing uploaded bytes to '%s': %v", outputWriter.GetFileName(), err)
	}

	log.Printf("[copy] Written %d bytes for '%s' to '%s'", bytesWritten, meta.OriginalFilename, outputWriter.GetFileName())
	return outputWriter.GetFileName(), nil
}

// Simple move handler
// ===================

type FallbackUploadHandler struct {
	tmpDir, baseDir string
}

func (f *FallbackUploadHandler) CanHandle(meta *UploadMeta) bool {
	// fallback can always handle an upload
	return true
}

func (f *FallbackUploadHandler) HandleUpload(meta *UploadMeta, reader multipart.File) error {
	_, err := copyUploadedFileTo(meta, reader, f.baseDir, f.tmpDir)
	return err
}

// Serverlog parsing handlers
// ==========================

type JsonServerlogsUploadHandler struct {
	tmpDir, baseDir, archivesDir string

	parserChan chan ServerlogInput
}

func NewJsonServerlogsUploadHandler(tmpDir, baseDir, archivesDir string) UploadHandler {
	serverlogParser := MakeServerlogsParser(tmpDir, baseDir, archivesDir, 256)
	return &JsonServerlogsUploadHandler{
		tmpDir:      tmpDir,
		baseDir:     baseDir,
		archivesDir: archivesDir,
		parserChan:  serverlogParser,
	}
}

var isJsonServerlogRegexp = regexp.MustCompile("^(server|json)logs")

func (j *JsonServerlogsUploadHandler) CanHandle(meta *UploadMeta) bool {
	return isJsonServerlogRegexp.MatchString(meta.TableName)
}

func (j *JsonServerlogsUploadHandler) HandleUpload(meta *UploadMeta, reader multipart.File) error {
	// copy the serverlog to the archives
	archivedFile, err := copyUploadedFileTo(meta, reader, j.archivesDir, j.tmpDir)
	if err != nil {
		return err
	}

	j.parserChan <- ServerlogInput{
		Meta:         meta,
		ArchivedFile: archivedFile,
		Format:       LogFormatJson,
	}
	return nil
}
