package tests

/*
import (
	"github.com/palette-software/insight-server/app/controllers"
	"github.com/palette-software/insight-server/app/routes"
	"github.com/revel/revel"
	"github.com/revel/revel/testing"

	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"mime/multipart"
)

type CsvMultiUploadTest struct {
	testing.TestSuite
	parent CsvUploadTest
}

type MultiFileInput struct {
	Filename string
	Contents string
}

/////////////
func (t *CsvMultiUploadTest) Before() {
	revel.INFO.Printf("Creating test tenant")
	t.parent.tenant = makeTestTenant(controllers.Dbm)
}

func (t *CsvMultiUploadTest) After() {
	// delete the existing test tenant, so the DB stays relatively clean
	deleteTestTenant(controllers.Dbm, t.parent.tenant)
}

// Creates a manifest data from a list of files
func manifestFromFiles(files []MultiFileInput) controllers.UploadManyRequest {
	manifest := controllers.UploadManyRequest{make([]controllers.UploadFile, len(files))}

	// add all data passed to the request
	for i, f := range files {
		// calc the file MD5
		md5 := fmt.Sprintf("%x", md5.Sum([]byte(f.Contents)))
		manifest.Files[i] = controllers.UploadFile{Name: f.Filename, Md5: md5}
	}

	return manifest
}

// Writes the manifest to the request
func packageManifest(writer *multipart.Writer, manifest controllers.UploadManyRequest) error {

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
func packageMultipleUploads(files []MultiFileInput, manifest controllers.UploadManyRequest) (reqBuffer *bytes.Buffer, contentType string) {
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

	if err := packageManifest(writer, manifest); err != nil {
		panic(err)
	}
	return
}

// UPLOAD MULTIPLE FILES

func sendManyFilesWithManifest(t *CsvMultiUploadTest, tenant string, password string, pkg string, files []MultiFileInput, manifest controllers.UploadManyRequest) controllers.UploadManyResponse {
	// generate the request body
	reqBody, contentType := packageMultipleUploads(files, manifest)
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

	return uploadResponse
}

// Checks if all files specified by files are uploaded and have the correct md5.
// Returns two lists of files: successfully uploaded and failed ones
func checkMultiUploadResponse(t *CsvMultiUploadTest, files []MultiFileInput, uploadResponse controllers.UploadManyResponse, doAssert bool) (success, failed []string) {
	// initialize the lists
	success = make([]string, 0, len(files))
	failed = make([]string, 0, len(files))

	for i, input := range files {
		// get the original one here
		fileName := input.Filename
		// compare with the uploaded one
		checkerFn := t.parent.checkUploadNoAssert
		if doAssert {
			checkerFn = t.parent.checkUpload
		}

		if len(uploadResponse.Files) <= i {
			failed = append(failed, fileName)
			continue
		}
		if checkerFn(&uploadResponse.Files[i], fileName, input.Contents) {
			success = append(success, fileName)
		} else {
			failed = append(failed, fileName)
		}
	}

	return
}

// Uploads a list of files with auto-generating the manifest and checking the upload ok status after the uploads
func sendManyFilesAsUpload(t *CsvMultiUploadTest, tenant string, password string, pkg string, files []MultiFileInput) controllers.UploadManyResponse {
	uploadResponse := sendManyFilesWithManifest(t, tenant, password, pkg, files, manifestFromFiles(files))
	_, failed := checkMultiUploadResponse(t, files, uploadResponse, true)
	t.Assertf(len(failed) == 0, "Errors during upload")
	return uploadResponse
}

// TESTS
// =====

// We should be able to upload more then one files, with all contents intact
func (t *CsvMultiUploadTest) TestUploadMultipleFiles() {
	sendManyFilesAsUpload(t, testTenantUsername, testTenantPassword, testPkg, []MultiFileInput{
		MultiFileInput{"hello1.txt", "Hello 1 text"},
		MultiFileInput{"hello2.txt", "hello 2 text"},
	})

}

// Checks if files missing from the manifest are not uploaded and the response contains only the files listsed in
// the manifest
func (t *CsvMultiUploadTest) TestFilesMissingFromManifestShouldNotUpload() {
	origFiles := []MultiFileInput{
		MultiFileInput{"hello1.txt", "Hello 1 text"},
		MultiFileInput{"hello2.txt", "hello 2 text"},
	}

	manifestFiles := []MultiFileInput{
		origFiles[0],
	}

	uploadResponse := sendManyFilesWithManifest(t, testTenantUsername, testTenantPassword, testPkg, origFiles, manifestFromFiles(manifestFiles))

	success, failed := checkMultiUploadResponse(t, origFiles, uploadResponse, false)

	t.Assertf(len(success) == 1, "There should be a single file uploaded")
	t.Assertf(len(failed) == 1, "There should be a single file not uploaded")

	t.Assertf(success[0] == manifestFiles[0].Filename, "The successfully uploaded file has name '%v' instead of '%v'", success[0], manifestFiles[0].Filename)
	t.Assertf(failed[0] == origFiles[1].Filename, "The failed uploaded file has name '%v' instead of '%v'", failed[0], origFiles[1].Filename)
}

// Checks if files missing from the manifest are not uploaded and the response contains only the files listsed in
// the manifest
func (t *CsvMultiUploadTest) TestMissingFilesListedInManifestShouldFail() {
	origFiles := []MultiFileInput{
		MultiFileInput{"hello1.txt", "Hello 1 text"},
	}

	manifestFiles := []MultiFileInput{
		origFiles[0],
		MultiFileInput{"hello2.txt", "hello 2 text"},
	}

	uploadResponse := sendManyFilesWithManifest(t, testTenantUsername, testTenantPassword, testPkg, origFiles, manifestFromFiles(manifestFiles))

	success, failed := checkMultiUploadResponse(t, manifestFiles, uploadResponse, false)

	t.Assertf(len(success) == 1, "There should be a single file uploaded")
	t.Assertf(len(failed) == 1, "There should be a single file not uploaded")

	t.Assertf(success[0] == manifestFiles[0].Filename, "The successfully uploaded file has name '%v' instead of '%v'", success[0], manifestFiles[0].Filename)
	t.Assertf(failed[0] == manifestFiles[1].Filename, "The failed uploaded file has name '%v' instead of '%v'", failed[0], manifestFiles[1].Filename)
}
*/
