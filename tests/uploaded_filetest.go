package tests

import (
	"github.com/revel/revel/testing"

	"github.com/palette-software/insight-server/app/models"

	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type UploadedFileTest struct {
	testing.TestSuite
}

func (t *UploadedFileTest) TestSingleUploadedFile() {
	testReader := strings.NewReader("Hello world")
	filename := "test.txt"
	md5 := "ABC"
	basePath := models.GetOutputDirectory()
	reqTime := time.Date(2016, 03, 15, 12, 00, 00, 00, time.UTC)

	uploadedFile, err := models.NewUploadedFile(basePath, filename, md5, reqTime, testReader)
	t.Assert(err == nil)

	expectedFileName := fmt.Sprintf("test-txt-12-00--00-00-%v.txt", md5)
	expectedUploadPath := filepath.ToSlash(path.Join(basePath, "2016-03-15", expectedFileName))

	t.Assert(uploadedFile.Filename == filename)
	t.Assert(uploadedFile.UploadedPath == expectedUploadPath)
	t.Assert(uploadedFile.Md5 == md5)
}
