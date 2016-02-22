package tests

import (
	"github.com/palette-software/insight-server/app/controllers"
	"github.com/palette-software/insight-server/app/routes"

	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"mime/multipart"
)

type MultiFileInput struct {
	Filename string
	Contents string
}

// packages a manifest
func packageManifest(files []MultiFileInput, writer *multipart.Writer) error {
	manifest := controllers.UploadManyRequest{make([]controllers.UploadFile, len(files))}

	// add all data passed to the request
	for i, f := range files {
		// calc the file MD5
		md5 := fmt.Sprintf("%x", md5.Sum([]byte(f.Contents)))
		manifest.Files[i] = controllers.UploadFile{Name: f.Filename, Md5: md5}
	}

	// create a writer for the manifest field
	manifestWriter, err := writer.CreateFormField(controllers.MANIFEST_FIELD_NAME)
	if err != nil {
		return err
	}

	// serialize the manifest into a JSON and write it to the request
	if err := json.NewEncoder(manifestWriter).Encode(&manifest); err != nil {
		return err
	}

	return nil
}

// Builds a new multipart request body for many files
func packageMultipleUploads(files []MultiFileInput) (reqBuffer *bytes.Buffer, contentType string) {
	// create a buffer for the upload multipart writer
	reqBuffer = &bytes.Buffer{}
	// create the writer for the multipart data
	writer := multipart.NewWriter(reqBuffer)
	// close the request writer
	defer writer.Close()

	contentType = writer.FormDataContentType()

	// add all data passed to the request
	for _, f := range files {
		// add the file data to the request
		err := writer.WriteField(f.Filename, f.Contents)
		if err != nil {
			panic(err)
		}
	}

	if err := packageManifest(files, writer); err != nil {
		panic(err)
	}
	return
}

// UPLOAD MULTIPLE FILES

func sendManyFilesAsUpload(t *CsvUploadTest, tenant string, password string, pkg string, files []MultiFileInput) controllers.UploadManyResponse {
	// generate the request body
	reqBody, contentType := packageMultipleUploads(files)
	// send the request with http auth
	postUri := routes.CsvUpload.UploadMany(pkg)
	postRequest := t.PostCustom(t.BaseUrl()+postUri, contentType, reqBody)

	postRequest.SetBasicAuth(tenant, password)
	postRequest.Send()

	// check if the rquest is successful
	t.AssertOk()

	t.Assertf(len(t.ResponseBody) > 0, "The response for uploads must be larger then 0 bytes")
	// create a string reader for the response body so we can decode it
	bodyReader := bytes.NewReader(t.ResponseBody)
	// create a dummy value for parsing into
	uploadResponse := controllers.UploadManyResponse{}
	// try to deserialize the request body
	err := json.NewDecoder(bodyReader).Decode(&uploadResponse)
	if err != nil {
		panic(err)
	}

	for i, fileResponse := range uploadResponse.Files {
		// get the original one here
		input := files[i]
		// compare with the uploaded one
		checkUpload(t, &fileResponse, input.Filename, input.Contents)
	}

	return uploadResponse
}

// We should be able to upload more then one files, with all contents intact
func (t *CsvUploadTest) TestUploadMultipleFiles() {
	sendManyFilesAsUpload(t, testTenantUsername, testTenantPassword, testPkg, []MultiFileInput{
		MultiFileInput{"hello1.txt", "Hello 1 text"},
		MultiFileInput{"hello2.txt", "hello 2 text"},
	})

}
