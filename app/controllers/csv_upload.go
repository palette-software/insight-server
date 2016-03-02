package controllers

import (
	"github.com/palette-software/insight-server/app/models"
	"github.com/revel/revel"

	"bytes"
	"fmt"
	"time"
	"mime/multipart"
	"encoding/base64"
)

// The application controller itself
type CsvUpload struct {
	*revel.Controller
	Tenant *models.Tenant
}

//
// MISC SMALL METHODS
// ==================
//

// Creates an authentication error response
func (c *CsvUpload) respondWithText(status int, text string) revel.Result {
	c.Response.Status = status
	// render the error as proper json
	return c.RenderJson(text)
}

// Interceptor filter for all actions in controllers that require authentication
//
// Checks the auth information from the request, and fails if it isnt there or the auth
// info does not correspond to the
func (c *CsvUpload) CheckUserAuth() revel.Result {
	username, password, authOk := c.Request.BasicAuth()
	if !authOk {
		revel.INFO.Printf("[AUTH] No auth information provided in request")
		return c.respondWithText(401, "No auth information provided")
	}

	// check password / username and get the tenant
	tenant := models.TenantFromAuthentication(Dbm, username, password)
	if tenant == nil {
		revel.INFO.Printf("[auth] not a valid user: %v", username)
		return c.respondWithText(401, "Not a valid user.")
	}

	// set the controllers tenant to the freshly loaded one
	c.Tenant = tenant

	return nil
}

//
// UPLOAD HELPERS
// ==============

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

// Hanlder for uploading a file with its metadata and validate the md5
func (c *CsvUpload) UploadWithMetadata(pkg string) revel.Result {

	req := c.Request

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
	newUploadedPack, err := models.NewUploadedCsv(c.Tenant, pkg, fileName, requestTime, mainFile, metaFile )
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
		c.Response.Status = 409
		return c.RenderJson("Md5 Error")
	}


	return c.RenderJson(newUploadedPack)
}
