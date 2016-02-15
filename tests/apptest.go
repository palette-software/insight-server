package tests

import (
	"github.com/palette-software/insight-webservice-go/app/controllers"
	"github.com/palette-software/insight-webservice-go/app/models"
	"github.com/palette-software/insight-webservice-go/app/routes"
	"github.com/revel/revel/testing"

	"bytes"
	"encoding/json"
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

	testFileContents  = "HELLO WORLD"
	testFileContents2 = "hello world 2"
)

type AppTest struct {
	testing.TestSuite

	tenant *models.Tenant
}

func (t *AppTest) Before() {
	println("Set up")
	t.tenant = createTestTenant(testTenantUsername, testTenantPassword)
}

func (t *AppTest) After() {
	println("Tear down")
	deleteTestTenant(t.tenant)
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

const (
	testTenantUsername = "testTenant"
	testTenantPassword = "testTenantPw"
)

func createTestTenant(username string, password string) *models.Tenant {
	return models.CreateTenant(controllers.Dbm, username, password, "Test User")
}

func deleteTestTenant(tenant *models.Tenant) {
	models.DeleteTenant(controllers.Dbm, tenant)
}

// Tries to upload the contents of a file then returns the possible uploaded path
func sendAsUpload(t *AppTest, tenant string, password string, pkg string, filename string, contents string) string {
	postReader := strings.NewReader(contents)

	// send the request with http auth
	postUri := routes.App.Upload(tenant, pkg, filename)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
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

	return uploadPath
}

// TEST CASES
// ==========

// check for a simple upload
func (t *AppTest) TestIncorrectPasswordShouldNotWork() {

	postReader := strings.NewReader("HELLO WORLD")

	// send the request with http auth
	postUri := routes.App.Upload(testTenantUsername, testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
	// supplly an invalid password
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword+"----")
	postRequest.Send()

	t.AssertStatus(403)
}

// check for a simple upload
func (t *AppTest) TestIncorrectUserShouldNotWork() {

	postReader := strings.NewReader("HELLO WORLD")

	// send the request with http auth
	postUri := routes.App.Upload(testTenantUsername, testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
	// supplly an invalid password
	postRequest.SetBasicAuth(testTenantUsername+"----", testTenantPassword)
	postRequest.Send()

	t.AssertStatus(403)
}

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

//func (t *AppTest) TestThatUsernamePasswordIsRequired() {
//postReader := strings.NewReader("HELLO WORLD")

//t.Post(routes.App.Upload(tenant, pkg, filename), "text/plain", postReader)
//t.AssertOk()

//}
