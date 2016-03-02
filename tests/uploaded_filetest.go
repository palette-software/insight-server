package tests

import (
	"github.com/revel/revel"
	"github.com/revel/revel/testing"

	"github.com/palette-software/insight-server/app/models"

	"bytes"
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
//"github.com/palette-software/insight-server/app/routes"
//	"mime/multipart"
)

type UploadedFileTest struct {
	testing.TestSuite
}

func (t *UploadedFileTest) TestSingleUploadedFile() {
	testContent := "Hello world"
	testReader := strings.NewReader(testContent)
	filename := "test.txt"

	// we need to do some stupid conversion from [16]byte to []byte here or
	// bytes.Equal will fail with argumenterror (fixed array vs slice)
	// reflect.DeepEqual will fail (different types dont match)
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

	revel.INFO.Printf("Removing temporary file '%v'", uploadedFile.UploadedPath)
	t.Assert(os.Remove(uploadedFile.UploadedPath) == nil)
}

//
//func (t* UploadedFileTest) TestMultifileUpload() {
//	// create a buffer for the upload multipart writer
//	reqBuffer := &bytes.Buffer{}
//	// create the writer for the multipart data
//	writer := multipart.NewWriter(reqBuffer)
//	// close the request writer
//	defer writer.Close()
//
//	contentType := writer.FormDataContentType()
//
//	// pack a file in
//	fileData := "HELLO WORLD"
//	metaData := "hello world"
//
//
//
//
//	writer.WriteField("_file", "test.txt")
//	writer.WriteField("_file.md5", fmt.Sprintf("%x", md5.Sum([]byte(fileData))))
//	writer.WriteField("_file.data", fileData)
//	writer.WriteField("_meta", "test.txt.meta")
//	writer.WriteField("_meta.md5", fmt.Sprintf("%x", md5.Sum([]byte(metaData))))
//	writer.WriteField("_meta.data", metaData)
//
//
//
//	// send the request with http auth
//	postUri := routes.CsvUpload.Upload(testPkg, testFileName)
//	postRequest := t.PostCustom(t.BaseUrl()+postUri, contentType, postReader)
//	// supplly an invalid password
//	postRequest.SetBasicAuth(testTenantUsername, testTenantPassword+"----")
//	postRequest.Send()
//}
