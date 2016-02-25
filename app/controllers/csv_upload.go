package controllers

import (
	"github.com/palette-software/insight-server/app/models"
	"github.com/revel/revel"

	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	_      = iota
	KB int = 1 << (10 * iota)
	MB
	GB

	OUTPUT_DEFAULT_MODE    = 0644
	OUTPUT_DEFAULT_DIRMODE = 0755

	MAXIMUM_MULTIPART_SIZE = 32 << 20

	// The name of the multipart field containing the JSON manifest
	// for multiple uploads
	MANIFEST_FIELD_NAME = "_manifest"
)

//
// DATA STRUCTURES
// ===============
//

// A generic structure desribing an uploaded file
type UploadFile struct {
	Name string
	Md5  string
}

// The manifest structure for uploading many files
type UploadManyRequest struct {
	Files []UploadFile
}

// The result of an upload operation is of this type.
// We are returning this as JSON when replying to a valid Upload() request
type UploadResponse struct {
	// The filename of the upload
	UploadFile
	UploadPath string
	UploadTime time.Time
	Status     int
	Error      string
}

const (
	HTTP_OK = 200
)

// Returns true if the response is an ok one
func (r *UploadResponse) IsOK() bool {
	return r.Status == HTTP_OK
}

// Quick accessor to update the status and the error string
func (r UploadResponse) SetStatusAndErrorMessage(status int, err string) (UploadResponse, error) {
	r.Status = status
	r.Error = err
	return r, errors.New(err)
}

// Quick accessor to update the status and the error string
func (r UploadResponse) SetStatusAndError(status int, err error) (UploadResponse, error) {
	r.Status = status
	r.Error = fmt.Sprintf("%v", err)
	return r, err
}

// The result of an uploadMany operation.
type UploadManyResponse struct {
	Files []UploadResponse
}

// The application controller itself
type CsvUpload struct {
	*revel.Controller
	Tenant *models.Tenant
}

//
// MISC SMALL METHODS
// ==================
//

// get the hash of the contents of the file, so that even files
// uploaded at the same time can be differentiated (this is important for the
// tests)
func getContentHash(fileContents []byte) string {
	return fmt.Sprintf("%x", md5.Sum(fileContents))

}

// Creates an authentication error response
func (c *CsvUpload) respondWith(status int) revel.Result {
	c.Response.Status = status
	return c.RenderText("")
}

// Creates an authentication error response
func (c *CsvUpload) respondWithText(status int, text string) revel.Result {
	c.Response.Status = status
	return c.RenderText(text)
}

// Interceptor filter for all actions in controllers that require authentication
//
// Checks the auth information from the request, and fails if it isnt there or the auth
// info does not correspond to the
func (c *CsvUpload) CheckUserAuth() revel.Result {
	username, password, authOk := c.Request.BasicAuth()
	if !authOk {
		revel.INFO.Printf("[AUTH] No auth information provided in request")
		return c.respondWith(401)
	}

	// check password / username and get the tenant
	tenant := models.TenantFromAuthentication(Dbm, username, password)
	if tenant == nil {
		revel.INFO.Printf("[auth] not a valid user: %v", username)
		return c.respondWith(401)
	}

	revel.TRACE.Printf("[auth] User: %v", username)

	// set the controllers tenant to the freshly loaded one
	c.Tenant = tenant

	return nil
}

//
// UPLOAD HELPERS
// ==============
//

/// Returns the path where a file needs to be placed
func getUploadPath(tenantHome, pkg, filename string, requestTime time.Time, fileHash string) string {
	// the folder name is only the date
	folderTimestamp := requestTime.Format("2006-01-02")
	// the file name gets the timestamp appended (only time)
	fileTimestamp := requestTime.Format("15-04--05-00")

	// get the extension and basename
	tenantHomeDir := models.SanitizeName(tenantHome)
	pkgDir := models.SanitizeName(pkg)
	fileBaseName := models.SanitizeName(path.Base(filename))
	fileExtName := models.SanitizeName(path.Ext(filename))
	fullFileName := fmt.Sprintf("%v-%v-%v.%v", fileBaseName, fileTimestamp, fileHash, fileExtName[1:])

	// the file name is the sanitized file name
	return filepath.ToSlash(path.Join(models.GetOutputDirectory(), tenantHomeDir, "uploads", pkgDir, folderTimestamp, fullFileName))
}

// checks if the "md5" URL parameter sent matches fileHash (if there is such a parameter)
// Since url.Values.Get() returns an empty string if the given
// parameter value is not found, we check against that
func checkSentMd5(sentMd5, fileHash string) bool {

	// If the client hasnt sent an Md5, we cosider it a valid hash
	if sentMd5 == "" {
		return true
	}

	// decode the bytes
	sentMd5Bytes, sentErr := hex.DecodeString(sentMd5)
	localMd5Bytes, localErr := hex.DecodeString(fileHash)

	if sentErr != nil || localErr != nil {
		revel.INFO.Printf("Md5 decode error strings are: '%v', '%v' errors are: %v // %v", sentMd5, fileHash, sentErr, localErr)
		return false
	}

	return bytes.Compare(sentMd5Bytes, localMd5Bytes) == 0
}

// parses the manifest out of the quest body
func (c *CsvUpload) parseManifest() (*UploadManyRequest, error) {

	// try to parse the form as multipart
	// TODO: check for maxmimum size, and use c.Request.FormFile() if the size is too large
	if err := c.Request.ParseMultipartForm(MAXIMUM_MULTIPART_SIZE); err != nil {
		return nil, err
	}

	formValues := c.Request.MultipartForm.Value

	manifestJSON, hasManifest := formValues[MANIFEST_FIELD_NAME]
	// no manifest? cannot do anything :(
	if !hasManifest {
		return nil, fmt.Errorf("No _manifest field provided.")
	}

	// as a form may have more fields of the same name, check it
	if len(manifestJSON) != 1 {
		return nil, fmt.Errorf("More then one _manifest provided")
	}

	// create a reader for the json from the first field
	manifestReader := strings.NewReader(manifestJSON[0])

	// create a dummy value for parsing into
	manifest := &UploadManyRequest{[]UploadFile{UploadFile{"", ""}}}
	// Try to decode the manifest JSON
	err := json.NewDecoder(manifestReader).Decode(manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

// internal helper that handles the uploading of a file
func (c *CsvUpload) handleUpload(pkg, filename string, content []byte, dataMD5 string, requestTime time.Time) (response UploadResponse, outErr error) {

	// make a local copy the tenant
	tenant := c.Tenant
	fileHash := getContentHash(content)
	homeDir := tenant.HomeDirectory
	outputPath := getUploadPath(tenant.HomeDirectory, pkg, filename, requestTime, fileHash)
	response = UploadResponse{UploadFile{filename, fileHash}, outputPath, requestTime, 200, ""}

	// check the MD5 if the client sent it to us
	if !checkSentMd5(dataMD5, fileHash) {
		return response.SetStatusAndErrorMessage(409, fmt.Sprintf("Md5 for '%v' does not match '%s' vs '%s'", filename, dataMD5, fileHash))
	}

	// do some minimal logging
	revel.INFO.Printf("[UPLOAD] got %v bytes for home directory '%v': %v / %v -- hash is: %v", len(content), homeDir, pkg, filename, fileHash)

	// create the directory of the file
	if err := os.MkdirAll(filepath.Dir(outputPath), OUTPUT_DEFAULT_DIRMODE); err != nil {
		return response.SetStatusAndError(500, err)
	}

	// write out the contents to a new file
	if err := ioutil.WriteFile(outputPath, content, OUTPUT_DEFAULT_MODE); err != nil {
		return response.SetStatusAndError(500, err)
	}

	// log that we were successful
	revel.INFO.Printf("[UPLOAD] Saved to '%v'", outputPath)

	return response.SetStatusAndError(200, nil)
}

//
// HTTP HANDLERS
// =============
//

// Handle an actual upload
func (c *CsvUpload) Upload(pkg, filename string) revel.Result {

	// parse the full request, so we can use the Request.Form parameters that
	// are passed to us
	if err := c.Request.ParseForm(); err != nil {
		return c.RenderError(err)
	}

	// read the contents of the post body
	content, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		return c.RenderError(err)
	}

	// Do the actual uploading
	response, err := c.handleUpload(pkg, filename, content, c.Request.Form.Get("md5"), time.Now())
	if err != nil {
		c.Response.Status = response.Status
		return c.RenderError(err)
	}

	return c.RenderJson(response)
}

// Upload many files in one go.
func (c *CsvUpload) UploadMany(pkg string) revel.Result {
	// get the request time
	requestTime := time.Now()

	// parse the manifest
	manifest, err := c.parseManifest()
	if err != nil {
		return c.RenderError(err)
	}

	// cache the uploaded & parsed map
	formValues := c.Request.MultipartForm.Value

	// create the storage for the results
	results := UploadManyResponse{make([]UploadResponse, len(manifest.Files))}

	for i, uploadedFile := range manifest.Files {
		// check if we have this file in the request
		fileData, hasFile := formValues[uploadedFile.Name]
		ok := true

		response := UploadResponse{uploadedFile, "<NO PATH>", requestTime, 404, ""}

		if !hasFile {
			ok = false
			response.SetStatusAndError(404, fmt.Errorf("Missing file in manifest from upload: '%v'", uploadedFile.Name))
		}

		// check if the file count is ok
		if ok && (len(fileData) != 1) {
			ok = false
			response.SetStatusAndError(404, fmt.Errorf("File '%v' listed more then once in the request body", uploadedFile.Name))
		}

		// do the actual upload if we are fine
		if ok {
			// upload this file
			// if we have an error, we should handle that error by saving it in the response
			response, err = c.handleUpload(pkg, uploadedFile.Name, []byte(fileData[0]), uploadedFile.Md5, requestTime)
		}

		// add it to the list of files
		results.Files[i] = response

	}
	return c.RenderJson(results)
}
