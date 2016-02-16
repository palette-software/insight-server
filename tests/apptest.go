package tests

import (
	"github.com/palette-software/insight-webservice-go/app/controllers"
	"github.com/palette-software/insight-webservice-go/app/models"
	"github.com/palette-software/insight-webservice-go/app/routes"
	"github.com/revel/revel/testing"

	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/revel/revel"
)

const (
	testTenant   = "testTenant"
	testPkg      = "testPkg"
	testFileName = "log.txt"

	testTenantUsername = "testTenant"
	testTenantPassword = "testTenantPw"

	testFileContents  = "HELLO WORLD"
	testFileContents2 = "hello world 2"
)

// TEST GROUP SETUP
// ================

type AppTest struct {
	testing.TestSuite

	tenant *models.Tenant
}

func (t *AppTest) Before() {
	// create a tenant for the test run
	var err error = nil
	t.tenant, err = models.CreateTenant(controllers.Dbm, testTenantUsername, testTenantPassword, "Test User", testTenantUsername)

	if err != nil {
		panic(err)
	}
}

func (t *AppTest) After() {
	// delete the existing test tenant, so the DB stays relatively clean
	models.DeleteTenant(controllers.Dbm, t.tenant)
}

// SIMPLE HELPERS
// ==============

// Returns true if fileName exists or false otherwise
func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}

// Checks if the contents of fileName match contents
func fileCheckContents(fileName string, contents string) bool {

	// read the file
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return false
	}

	revel.TRACE.Printf("conetnts: %v", fileContents)

	// check the contents
	return bytes.Compare([]byte(contents), fileContents) == 0
}

// TESTING HELPERS
// ===============
// Tries to upload the contents of a file then returns the possible uploaded path
func sendAsUpload(t *AppTest, tenant string, password string, pkg string, filename string, contents string) controllers.UploadResponse {
	postReader := strings.NewReader(contents)

	// send the request with http auth
	postUri := routes.App.Upload(pkg, filename)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
	revel.INFO.Printf("----> URL: %v", t.BaseUrl()+postUri)

	postRequest.SetBasicAuth(tenant, password)
	postRequest.Send()

	// check if the rquest is successful
	t.AssertOk()

	t.Assertf(len(t.ResponseBody) > 0, "The response for uploads must be larger then 0 bytes")
	// create a string reader for the response body so we can decode it
	bodyReader := bytes.NewReader(t.ResponseBody)
	// make a nonsensical time for marking invalid responses
	nonsenseTime := time.Unix(int64(0), 0)
	// create a dummy value for parsing into
	uploadResponse := controllers.UploadResponse{"", nonsenseTime, ""}
	// try to deserialize the request body
	err := json.NewDecoder(bodyReader).Decode(&uploadResponse)
	if err != nil {
		panic(err)
	}
	//extract the upload time
	uploadPath := uploadResponse.UploadPath
	// check the returned time
	t.Assertf(uploadResponse.UploadTime != nonsenseTime, "Invalid time returned by the service: %v", string(t.ResponseBody))
	// check for the existance of the uploaded file
	t.Assertf(fileExists(uploadPath), "Output file '%v' has not been created", uploadPath)
	// check the contents of the uploaded file
	t.Assertf(fileCheckContents(uploadPath, contents), "Contents of output file '%v' does not match the test content", uploadPath)

	return uploadResponse
}

// AUTH TESTS
// ----------

// check for a simple upload
func (t *AppTest) TestIncorrectPasswordShouldNotWork() {

	postReader := strings.NewReader("HELLO WORLD")

	// send the request with http auth
	postUri := routes.App.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
	// supplly an invalid password
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword+"----")
	postRequest.Send()

	t.AssertStatus(401)
}

// check for a simple upload
func (t *AppTest) TestIncorrectUserShouldNotWork() {

	postReader := strings.NewReader("HELLO WORLD")

	// send the request with http auth
	postUri := routes.App.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
	// supplly an invalid password
	postRequest.SetBasicAuth(testTenantUsername+"----", testTenantPassword)
	postRequest.Send()

	t.AssertStatus(401)
}

//// Check if we can use a users login credentials to write to another users
//// logs
//func (t *AppTest) TestUsernameShouldMatchTenant() {

//postReader := strings.NewReader("HELLO WORLD")

//// send the request with http auth
//postUri := routes.App.Upload(testTenantUsername+"-alt", testPkg, testFileName)
//postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
//// supplly an invalid password
//postRequest.SetBasicAuth(testTenantUsername, testTenantPassword)
//postRequest.Send()

//t.AssertStatus(403)
//}

// UPLOAD TESTS
// ------------

// check for a simple upload
func (t *AppTest) TestThatFilesCanBeUploaded() {
	sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "HELLO WORLD")
}

// check for uploading the same file name multiple times, but the contents and files must be
// different
func (t *AppTest) TestMultipleFilesSameName() {
	// check if both files upload properly
	uploadPath1 := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "HELLO WORLD")
	uploadPath2 := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "Hello world 2")

	t.Assertf(uploadPath1 != uploadPath2, "Multiple uploads must result in different uploaded file names")
}

// SIGNATURE CHECKS
// ================

// check for a simple upload
func (t *AppTest) TestFileSiginitureOk() {

	data := "HELLO WORLD"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	response := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, data)
	t.Assertf(response.Md5 == hash, "Hash of sent data does not match reply")
}

// check for a simple upload
func (t *AppTest) TestFileSiginitureFail() {

	data := "HELLO WORLD 2"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	response := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "HELLO WORLD")
	t.Assertf(response.Md5 != hash, "Hash of sent data should not match reply")
}

// Tests if sending a signature with the request properly rejects wrong data
func (t *AppTest) TestSendingMd5SignatureRejection() {

	data := "HELLO WORLD"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	// modify the data, so the MD5 must also change
	postReader := strings.NewReader(data + "--")

	// send the request with http auth
	postUri := routes.App.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri+"?md5="+hash, "text/plain", postReader)
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword)
	postRequest.Send()

	// He are expecting a 409 - Conflict here
	t.AssertStatus(409)
}

// Tests if sending a signature with the request properly rejects wrong data
func (t *AppTest) TestSendingMd5SignatureAcceptance() {

	data := "HELLO WORLD"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	postReader := strings.NewReader(data)

	// send the request with http auth
	postUri := routes.App.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri+"?md5="+hash, "text/plain", postReader)
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword)
	postRequest.Send()

	// since the data and the md5 hash matches, we should be ok here
	t.AssertOk()
}

// TENANT OUTPUT CONFIGURATION
// ===========================

func (t *AppTest) TestTenantOutputDirectory() {
	const TEST_DIR = "test-home"
	// create a temporary tenant
	tenant, err := models.CreateTenant(controllers.Dbm, testTenantUsername+"-", testTenantPassword, "Test User", TEST_DIR)
	if err != nil {
		panic(err)
	}
	// delete the created tenant on exit
	defer models.DeleteTenant(controllers.Dbm, tenant)

	response := sendAsUpload(t, testTenantUsername+"-", testTenantPassword, testPkg, testFileName, "HELLO WORLD")
	t.Assertf(strings.Contains(response.UploadPath, TEST_DIR), "Upload path '%v' does not contain the users home directory '%v'", response.UploadPath, TEST_DIR)
}
