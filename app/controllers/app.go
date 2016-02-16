package controllers

import (
	"crypto/md5"
	"fmt"
	"github.com/palette-software/insight-webservice-go/app/models"
	"github.com/revel/revel"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	_      = iota
	KB int = 1 << (10 * iota)
	MB
	GB

	OUTPUT_DIR = "c:\\tmp\\uploads"

	OUTPUT_DEFAULT_MODE    = 0644
	OUTPUT_DEFAULT_DIRMODE = 777
)

// The result of an upload operation is of this type.
// We are returning this as JSON when replying to a valid Upload() request
type UploadResponse struct {
	UploadPath string
	UploadTime time.Time
	Md5        string
}

// The application controller itself
type App struct {
	*revel.Controller
	Tenant *models.Tenant
}

func (c *App) Index() revel.Result {
	return c.Render()
}

// get the hash of the contents of the file, so that even files
// uploaded at the same time can be differentiated (this is important for the
// tests)
func getContentHash(fileContents []byte) string {
	return fmt.Sprintf("%x", md5.Sum(fileContents))

}

// Creates an authentication error response
func (c *App) respondWith(status int) revel.Result {
	c.Response.Status = status
	return c.Render()
}

/// Returns the path where a file needs to be placed
func getUploadPath(tenantHome, pkg, filename string, requestTime time.Time, fileHash string) string {
	// the folder name is only the date
	folderTimestamp := requestTime.Format("2006-01-02")
	// the file name gets the timestamp appended (only time)
	fileTimestamp := requestTime.Format("15-04--05-00")

	// get the extension and basename
	fileBaseName := models.SanitizeName(path.Base(filename))
	fileExtName := models.SanitizeName(path.Ext(filename))
	fullFileName := fmt.Sprintf("%v-%v-%v.%v", fileBaseName, fileTimestamp, fileHash, fileExtName[1:])

	// the file name is the sanitized file name
	return filepath.ToSlash(path.Join(OUTPUT_DIR, models.SanitizeName(tenantHome), "uploads", models.SanitizeName(pkg), folderTimestamp, fullFileName))
}

// Interceptor filter for all actions in controllers that require authentication
//
// Checks the auth information from the request, and fails if it isnt there or the auth
// info does not correspond to the
func (c *App) CheckUserAuth() revel.Result {
	username, password, authOk := c.Request.BasicAuth()
	if !authOk {
		revel.INFO.Printf("[AUTH] No auth information provided in request")
		return c.respondWith(401)
	}

	// check password / username
	tenant, isValidTenant := models.IsValidTenant(Dbm, username, password)
	if !isValidTenant {
		revel.INFO.Printf("[auth] not a valid user: %v", username)
		return c.respondWith(401)
	}

	revel.TRACE.Printf("[auth] User: %v", username)

	// set the controllers tenant to the freshly loaded one
	c.Tenant = tenant

	return nil
}

// checks if the "md5" URL parameter sent matches fileHash (if there is such a parameter)
// Since url.Values.Get() returns an empty string if the given
// parameter value is not found, we check against that
func checkSentMd5(sentMd5, fileHash string) bool {

	// if we are provided with an md5 parameter, check it if the hash is correct
	if sentMd5 != "" && sentMd5 != fileHash {
		return false
	}

	// otherwise we are ok
	return true
}

// Handle an actual upload
func (c *App) Upload(pkg, filename string) revel.Result {

	// parse the full request, so we can use the Request.Form parameters that
	// are passed to us
	if err := c.Request.ParseForm(); err != nil {
		return c.RenderError(err)
	}

	// make a local copy the tenant
	tenant := c.Tenant

	// get the request time
	requestTime := time.Now()

	requestBody := c.Request.Body

	// read the contents of the post body
	content, err := ioutil.ReadAll(requestBody)
	if err != nil {
		return c.RenderError(err)
	}

	// calculate the hash
	fileHash := getContentHash(content)

	// check the MD5 if the client sent it to us
	if !checkSentMd5(c.Request.Form.Get("md5"), fileHash) {
		// fail here with a 409 - Conflict for Md5 mismatches
		return c.respondWith(409)
	}

	// get the output path
	outputPath := getUploadPath(tenant.HomeDirectory, pkg, filename, requestTime, fileHash)

	// do some minimal logging
	revel.INFO.Printf("got %v bytes for tenant '%v': %v / %v -- hash is: %v", len(content), tenant.Username, pkg, filename, fileHash)

	// create the directory of the file
	err = os.MkdirAll(filepath.Dir(outputPath), OUTPUT_DEFAULT_DIRMODE)
	if err != nil {
		return c.RenderError(err)
	}

	// write out the contents to a new file
	err = ioutil.WriteFile(outputPath, content, OUTPUT_DEFAULT_MODE)
	if err != nil {
		return c.RenderError(err)
	}

	// render some nice output
	return c.RenderJson(UploadResponse{outputPath, requestTime, fileHash})
}
