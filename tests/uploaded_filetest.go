package tests

import (
	"github.com/revel/revel"
	"github.com/revel/revel/testing"

	"github.com/palette-software/insight-server/app/models"
	"github.com/palette-software/insight-server/app/routes"

	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"mime/multipart"
	"github.com/palette-software/insight-server/app/controllers"
	"io/ioutil"
	"encoding/json"
	"encoding/base64"
)

type UploadedFileTest struct {
	testing.TestSuite

	tenant *models.Tenant
}

/////////////
func (t *UploadedFileTest) Before() {
	t.tenant = makeTestTenant(controllers.Dbm)
}

func (t *UploadedFileTest) After() {
	deleteTestTenant(controllers.Dbm, t.tenant)
}

// Checks if the contents of fileName match contents
func checkFileContents(fileName string, contents string) bool {
	// read the file
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return false
	}
	// check the contents
	return bytes.Equal([]byte(contents), fileContents)
}

func (t *UploadedFileTest) TestSingleUploadedFile() {
	testContent := "Hello world"
	testReader := strings.NewReader(testContent)
	filename := "test.txt"

	fileMd5a := md5.Sum([]byte(testContent))
	fileMd5 := fileMd5a[0:]

	// create a testing directory inside the uploads directory for
	basePath := path.Join(models.GetOutputDirectory(), "testing")
	reqTime := time.Date(2016, 03, 15, 12, 00, 00, 00, time.UTC)

	uploadedFile, err := models.NewUploadedFile(basePath, filename, reqTime, testReader)
	t.Assert(err == nil)

	expectedFileName := fmt.Sprintf("test-txt-12-00--00-00-%x.txt", fileMd5)
	expectedUploadPath := filepath.ToSlash(path.Join(basePath, "2016-03-15", expectedFileName))

	t.Assert(uploadedFile.Filename == filename)
	t.Assert(uploadedFile.UploadedPath == expectedUploadPath)
	t.Assert(bytes.Equal(uploadedFile.Md5, fileMd5))

	t.Assert(checkFileContents(uploadedFile.UploadedPath, testContent))

	revel.INFO.Printf("Removing temporary file '%v'", uploadedFile.UploadedPath)
	t.Assert(os.Remove(uploadedFile.UploadedPath) == nil)
}



//
func (t* UploadedFileTest) TestMultifileUpload() {
	// create a buffer for the upload multipart writer
	reqBuffer := &bytes.Buffer{}
	// create the writer for the multipart data
	writer := multipart.NewWriter(reqBuffer)

	// pack a file in
	fileData := "HELLO WORLD"
	metaData := "hello world"

	// add the file
	fw, err := writer.CreateFormFile("_file", "test.txt")
	if  err != nil {
		panic(err)
	}
	fw.Write([]byte(fileData))

	// add the metadata
	mfw, err := writer.CreateFormFile("_meta", "test.txt.meta")
	if  err != nil {
		panic(err)
	}
	mfw.Write([]byte(metaData))

	// add the md5
	md5Sum := md5.Sum([]byte(fileData))
	writer.WriteField("_md5", base64.StdEncoding.EncodeToString(md5Sum[0:]))

	// close the writer so the request buffer gets filled
	writer.Close()



	// send the request with http auth
	postUri := routes.CsvUpload.UploadWithMetadata(testPkg)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, writer.FormDataContentType(), reqBuffer)
	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword)
	// supplly an invalid password
	//postRequest.SetBasicAuth(testTenantUsername, testTenantPassword+"----")
	postRequest.Send()

	t.AssertOk()

	// check the response
	resp := models.UploadedCsv{}
	// try to deserialize the request body
	err = json.NewDecoder(bytes.NewReader(t.ResponseBody)).Decode(&resp)
	if err != nil {
		panic(err)
	}

	// check the main file
	t.Assert( resp.Csv.Filename == "test.txt")
	t.Assert( checkFileContents( resp.Csv.UploadedPath, fileData))


	// check the metadata file
	t.Assert( resp.Metadata.Filename == "test.txt.meta")
	t.Assert( checkFileContents( resp.Metadata.UploadedPath, metaData))

}
