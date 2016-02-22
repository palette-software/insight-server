package tests

import (
	"github.com/palette-software/insight-server/app/controllers"
	"github.com/palette-software/insight-server/app/models"
	"github.com/palette-software/insight-server/app/routes"
	"github.com/revel/revel/testing"

	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
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

type CsvUploadTest struct {
	testing.TestSuite

	tenant *models.Tenant
}

func (t *CsvUploadTest) Before() {
	// create a tenant for the test run
	var err error = nil
	t.tenant, err = models.CreateTenant(controllers.Dbm, testTenantUsername, testTenantPassword, "Test User", testTenantUsername)

	if err != nil {
		panic(err)
	}
}

func (t *CsvUploadTest) After() {
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

	// check the contents
	return bytes.Compare([]byte(contents), fileContents) == 0
}

// checks if a file is available at
func checkUpload(t *CsvUploadTest, uploadResponse *controllers.UploadResponse, filename, contents string) bool {
	//extract the upload time
	uploadPath := uploadResponse.UploadPath
	// check if the Name field is correct in the reply
	t.Assertf(uploadResponse.Name == filename, "The output response must return the same filename '%v' as we have sent", filename)
	// check the returned time
	t.Assertf(uploadResponse.UploadTime != models.NonsenseTime, "Invalid time returned by the service: %v", string(t.ResponseBody))
	// check for the existance of the uploaded file
	t.Assertf(fileExists(uploadPath), "Output file '%v' has not been created", uploadPath)
	// check the contents of the uploaded file
	t.Assertf(fileCheckContents(uploadPath, contents), "Contents of output file '%v' does not match the test content", uploadPath)

	return true
}

// TESTING HELPERS
// ===============
// Tries to upload the contents of a file then returns the possible uploaded path
func sendAsUpload(t *CsvUploadTest, tenant string, password string, pkg string, filename string, contents string) controllers.UploadResponse {
	postReader := strings.NewReader(contents)

	// send the request with http auth
	postUri := routes.CsvUpload.Upload(pkg, filename)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)

	postRequest.SetBasicAuth(tenant, password)
	postRequest.Send()

	// check if the rquest is successful
	t.AssertOk()

	t.Assertf(len(t.ResponseBody) > 0, "The response for uploads must be larger then 0 bytes")
	// create a string reader for the response body so we can decode it
	bodyReader := bytes.NewReader(t.ResponseBody)
	// create a dummy value for parsing into
	uploadResponse := controllers.UploadResponse{controllers.UploadFile{"", ""}, "", models.NonsenseTime}
	// try to deserialize the request body
	err := json.NewDecoder(bodyReader).Decode(&uploadResponse)
	if err != nil {
		panic(err)
	}

	checkUpload(t, &uploadResponse, filename, contents)

	return uploadResponse
}

// AUTH TESTS
// ----------

// check for a simple upload
func (t *CsvUploadTest) TestIncorrectPasswordShouldNotWork() {

	postReader := strings.NewReader("HELLO WORLD")

	// send the request with http auth
	postUri := routes.CsvUpload.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
	// supplly an invalid password
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword+"----")
	postRequest.Send()

	t.AssertStatus(401)
}

// check for a simple upload
func (t *CsvUploadTest) TestIncorrectUserShouldNotWork() {

	postReader := strings.NewReader("HELLO WORLD")

	// send the request with http auth
	postUri := routes.CsvUpload.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, "text/plain", postReader)
	// supplly an invalid password
	postRequest.SetBasicAuth(testTenantUsername+"----", testTenantPassword)
	postRequest.Send()

	t.AssertStatus(401)
}

// UPLOAD TESTS
// ------------

// check for a simple upload
func (t *CsvUploadTest) TestThatFilesCanBeUploaded() {
	sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "HELLO WORLD")
}

// check for uploading the same file name multiple times, but the contents and files must be
// different
func (t *CsvUploadTest) TestMultipleFilesSameName() {
	// check if both files upload properly
	uploadPath1 := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "HELLO WORLD")
	uploadPath2 := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "Hello world 2")

	t.Assertf(uploadPath1 != uploadPath2, "Multiple uploads must result in different uploaded file names")
}

// SIGNATURE CHECKS
// ================

// check for a simple upload
func (t *CsvUploadTest) TestFileSiginitureOk() {

	data := "HELLO WORLD"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	response := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, data)
	t.Assertf(response.Md5 == hash, "Hash of sent data does not match reply")
}

// check for a simple upload
func (t *CsvUploadTest) TestFileSiginitureFail() {

	data := "HELLO WORLD 2"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	response := sendAsUpload(t, testTenantUsername, testTenantPassword, testPkg, testFileName, "HELLO WORLD")
	t.Assertf(response.Md5 != hash, "Hash of sent data should not match reply")
}

// Tests if sending a signature with the request properly rejects wrong data
func (t *CsvUploadTest) TestSendingMd5SignatureRejection() {

	data := "HELLO WORLD"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	// modify the data, so the MD5 must also change
	postReader := strings.NewReader(data + "--")

	// send the request with http auth
	postUri := routes.CsvUpload.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri+"?md5="+hash, "text/plain", postReader)
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword)
	postRequest.Send()

	// He are expecting a 409 - Conflict here
	t.AssertStatus(409)
}

// Tests if sending a signature with the request properly rejects wrong data
func (t *CsvUploadTest) TestSendingMd5SignatureAcceptance() {

	data := "HELLO WORLD"
	hash := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	postReader := strings.NewReader(data)

	// send the request with http auth
	postUri := routes.CsvUpload.Upload(testPkg, testFileName)
	postRequest := t.PostCustom(t.BaseUrl()+postUri+"?md5="+hash, "text/plain", postReader)
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword)
	postRequest.Send()

	// since the data and the md5 hash matches, we should be ok here
	t.AssertOk()
}

// TENANT OUTPUT CONFIGURATION
// ===========================

func (t *CsvUploadTest) TestTenantOutputDirectory() {
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
