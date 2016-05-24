package insight_server

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"

	"bufio"
	"bytes"
	"io"
	"mime/multipart"
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

	// By putting this flag here, we can decouple the configuration of
	// this flag from its implementation
	UseOldFormatFilename bool

	// The orignal Md5 the agent sent us
	OriginalMd5 []byte
}

// Returns the file name for an upload request
func (u *UploadMeta) GetOutputFilename(baseDir string) string {
	dateUtc := u.Date.UTC()

	// The name of the output file
	var outFilePattern string

	if u.UseOldFormatFilename {
		// example:
		// countersamples-2016-04-18--14-10-08--seq0000--part0000-csv-08-00--14-00-95755b03f960d2994dbad08067504e02.csv.gz
		outFilePattern = fmt.Sprintf("%s-%s--seq%04d--part%04d-csv-%s-{{md5}}.csv",
			SanitizeName(u.TableName),
			// the current time as part of the 2nd timestamp
			dateUtc.Format("2006-01-02--15-04-05"),
			u.PartIdx,
			u.SeqIdx,
			// copy the timestamp to the latter place
			dateUtc.Format("01-02--15-04"),
		)

	} else {
		// example:
		// threadinfo-2016-04-19--12-36-58--seq000--part0000-bc2ce0e4421cd7dea704eff080bb6f43.csv.gz
		outFilePattern = fmt.Sprintf("%s-%s--seq%03d--part%04d-{{md5}}.csv",
			SanitizeName(u.TableName),
			// the current time as part of the 2nd timestamp
			u.Date.UTC().Format("2006-01-02--15-04-05"),
			u.PartIdx,
			u.SeqIdx,
		)
	}
	//
	return filepath.ToSlash(path.Join(
		baseDir,
		SanitizeName(u.Tenant),
		"uploads",
		SanitizeName(u.Pkg),
		SanitizeName(u.Host),
		outFilePattern,
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

	// The directory permissions to use when creating a new directory
	OUTPUT_DEFAULT_DIRMODE = 0755
)

// Creates an http endpoint handler where
func MakeUploadHandler(maxidBackend MaxIdBackend, tmpDir, baseDir, archivesDir string, useOldFormatFilename bool) HandlerFuncWithTenant {
	// the fallback handler to move files
	fallbackHandler := &FallbackUploadHandler{tmpDir: tmpDir, baseDir: baseDir}

	// processing handlers
	handlers := []UploadHandler{
		NewJsonServerlogsUploadHandler(tmpDir, baseDir, archivesDir),
		NewMetadataUploadHandler(tmpDir, baseDir, archivesDir),
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

		// update the filename flag from the config
		meta.UseOldFormatFilename = useOldFormatFilename

		// find the handler for this table
		if err := findUploadHandler(meta, handlers, fallbackHandler).HandleUpload(meta, mainFile); err != nil {
			writeResponse(w, http.StatusInternalServerError, fmt.Sprint(err))
			return
		}

		// get the maxid and save it if needed
		maxid, err := getUrlParam(r.URL, "maxid")
		if err == nil {
			if err := maxidBackend.SaveMaxId(meta.Tenant, meta.TableName, maxid); err != nil {
				logrus.WithFields(logrus.Fields{
					"component": "maxid",
					"tenant":    meta.Tenant,
					"table":     meta.TableName,
					"maxid":     maxid,
				}).WithError(err).Error("Failed to save maxid")
			}
		}

		writeResponse(w, http.StatusOK, "OK")
	}
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
		OriginalMd5:      fileMd5,

		Tenant: tenantName,

		Pkg:       pkg,
		Host:      sourceHost,
		TableName: tableName,

		Date:     requestTime,
		Timezone: sourceTimezone,
		SeqIdx:   seqIdx,
		PartIdx:  partIdx,

		// default to not using the old format
		UseOldFormatFilename: false,
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

// Copies a file line-by-line, changes the line endings, prefixes each line with prefix
// (if its not empty) and postfixes each line with postfix
func extendAndCopyByLines(from io.Reader, to io.Writer, prefix, prefixHeader, postfix, postfixHeader []byte) (err error) {

	// create a buffered reader on top for line-reading
	bufferedReader := bufio.NewReader(from)

	// Our new line endings
	unixEol := []byte("\n")

	hasPrefix := len(prefix) > 0
	hasPostfix := len(postfix) > 0

	// Shared code between mid-line exit & post-line exit
	writePostfix := func() error {
		if hasPostfix {
			// append the postfix column
			if _, err := to.Write([]byte(postfix)); err != nil {
				return fmt.Errorf("Error writing postfix: %v", err)
			}
		}
		return nil
	}

	// Flag to mark the first line (where we dont need to write
	// an EOL)
	isFirstLine := true
	originalPrefix := prefix
	originalPostfix := postfix

	prefix = prefixHeader
	postfix = postfixHeader

	// read the input line-by line.
	// Since Readline() does not include the EOL chars, we can
	// use this to convert the line endings
	for {
		// ============= BEGINING OF THE LINE ===================

		// try the read
		line, isPrefix, err := bufferedReader.ReadLine()

		// if we are EOF, we are done (as we are at the beginning of
		// a new line, we dont have to write a postfix either)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return fmt.Errorf("Error reading CSV: %v", err)
		}

		if hasPrefix {
			// only write the filename if there is an actual line
			if _, err := to.Write([]byte(prefix)); err != nil {
				return fmt.Errorf("Error writing CSV: %v", err)
			}
		}

		// copy
		if _, err := to.Write(line); err != nil {
			return fmt.Errorf("Error writing CSV: %v", err)
		}

		// ============= MIDDLE OF THE LINE ===================

		// if the line is not yet complete (because the buffer of
		// the reader is too small), copy the rest of it
		for isPrefix {
			line, isPrefix, err = bufferedReader.ReadLine()
			if err == io.EOF {
				// if we get an EOF in the middle of the line
				// we still have to write the postfix
				return writePostfix()
			}
			// propagate errors
			if err != nil {
				return fmt.Errorf("Error reading CSV content: %v", err)
			}
			// write out the next bit
			if _, err := to.Write(line); err != nil {
				return fmt.Errorf("Error writing CSV: %v", err)
			}
		}

		// ============= END OF THE LINE ===================

		if err := writePostfix(); err != nil {
			return err
		}

		to.Write(unixEol)

		// if we have finished the first line, use the actual
		// pre- & postfixes
		if isFirstLine {
			isFirstLine = false
			prefix = originalPrefix
			postfix = originalPostfix
		}
	}

	return fmt.Errorf("Unreadhable code reached")
}

// Helper for converting args of extendAndCopyByLines() from string to []byte
func extendAndCopyByLinesString(from io.Reader, to io.Writer, prefix, prefixHeader, postfix, postfixHeader string) (err error) {
	return extendAndCopyByLines(from, to, []byte(prefix), []byte(prefixHeader), []byte(postfix), []byte(postfixHeader))
}

// Shared handler to copy an uploaded file to a location
func copyUploadedFileTo(meta *UploadMeta, reader multipart.File, baseDir, tmpDir string, hasLoaderColumns bool) (outFileName string, md5 []byte, err error) {

	// create the output writer
	outputWriter, err := meta.GetOutputGzippedWriter(baseDir, tmpDir)
	if err != nil {
		return "", nil, fmt.Errorf("Error opening gzipped output: %v", err)
	}

	// Get the filename we'll use for the output
	outFileName = outputWriter.GetRandomFileName()

	// safety defered close to always close the file with our new filename
	defer outputWriter.CloseWithFileName(outFileName)

	// create the md5 hasher that hashes input data
	md5HashedReader := makeMd5Hasher(reader)

	// Create the pre & postfixes
	prefixColumn := fmt.Sprintf("%s\v", outFileName)
	postfixColumn := fmt.Sprintf("\v%s", time.Now().Format(GpfdistPostfixTsFormat))

	// if its a metadata file, we dont want to write pre & postfixes
	if !hasLoaderColumns {
		prefixColumn = ""
		postfixColumn = ""
	}

	if err := extendAndCopyByLinesString(md5HashedReader, outputWriter, prefixColumn, "p_filepath\v", postfixColumn, "\vp_cre_date"); err != nil {
		return "", nil, fmt.Errorf("Error copying CSV content: %v", err)
	}

	// pick up any errors during close
	if err := outputWriter.CloseWithFileName(outFileName); err != nil {
		return "", nil, fmt.Errorf("Error writing uploaded bytes to '%s': %v", outFileName, err)
	}

	logrus.WithFields(logrus.Fields{
		"component":        "copy",
		"sourceHost":       meta.Host,
		"tenant":           meta.Tenant,
		"bytesWritten":     outputWriter.BytesWritten,
		"originalFileName": meta.OriginalFilename,
		"tableName":        meta.TableName,
		"outputFile":       outFileName,
	}).Info("Copied uploaded file")

	return outFileName, md5HashedReader.GetHash(), nil
}

// Shared handler to copy an uploaded file to a location
func copyUploadedFileAndCheckMd5(meta *UploadMeta, reader multipart.File, baseDir, tmpDir string, hasLoaderColumns bool) (outFileName string, err error) {
	outFileName, fileMd5, err := copyUploadedFileTo(meta, reader, baseDir, tmpDir, hasLoaderColumns)

	// Check for errors
	if err != nil {
		return outFileName, err
	}

	// Check if the md5 isnt a match
	if bytes.Compare(meta.OriginalMd5, fileMd5) != 0 {
		return outFileName, fmt.Errorf("Invalid md5: agent sent '%032x' copy got '%032x'", meta.OriginalMd5, fileMd5)
	}

	return outFileName, nil
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

	// skip adding filenames and datetimes for metadata files
	_, err := copyUploadedFileAndCheckMd5(meta, reader, f.baseDir, f.tmpDir, true)
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
var isPlainServerlogRegexp = regexp.MustCompile("^plainlogs")

// Is this a JSON formatted log
func isJsonLog(fn string) bool {
	return isJsonServerlogRegexp.MatchString(fn)
}

// Is this a plain text log?
func isPlainLog(fn string) bool {
	return isPlainServerlogRegexp.MatchString(fn)
}

func (j *JsonServerlogsUploadHandler) CanHandle(meta *UploadMeta) bool {
	return isJsonLog(meta.TableName) || isPlainLog(meta.TableName)
}

func (j *JsonServerlogsUploadHandler) HandleUpload(meta *UploadMeta, reader multipart.File) error {
	// copy the serverlog to the archives, dont add filenames and datetimes for
	// the loader since we will be adding them later during serverlog parsing
	archivedFile, err := copyUploadedFileAndCheckMd5(meta, reader, j.archivesDir, j.tmpDir, false)
	if err != nil {
		return err
	}

	logFormat := LogFormatJson
	if isPlainLog(meta.TableName) {
		logFormat = LogFormatPlain
	}

	j.parserChan <- ServerlogInput{
		Meta:         meta,
		ArchivedFile: archivedFile,
		Format:       logFormat,
	}
	return nil
}

// Metadata revriting
// ------------------

type metadataUploadHandler struct {
	tmpDir, baseDir, archivesDir string
}

var isMetadataRegexp = regexp.MustCompile("^metadata")

func NewMetadataUploadHandler(tmpDir, baseDir, archivesDir string) UploadHandler {
	return &metadataUploadHandler{
		tmpDir:      tmpDir,
		baseDir:     baseDir,
		archivesDir: archivesDir,
	}
}

func (m *metadataUploadHandler) CanHandle(meta *UploadMeta) bool {
	return isMetadataRegexp.MatchString(meta.TableName)
}
func (m *metadataUploadHandler) HandleUpload(meta *UploadMeta, reader multipart.File) error {
	// copy the serverlog to the archives
	archivedFile, err := copyUploadedFileAndCheckMd5(meta, reader, m.archivesDir, m.tmpDir, false)
	if err != nil {
		return err
	}

	return MetadataUploadHandler(meta, m.tmpDir, m.baseDir, archivedFile)
}
