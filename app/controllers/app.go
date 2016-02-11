package controllers

import (
	"crypto/md5"
	"fmt"
	"github.com/revel/revel"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
type UploadResponse struct {
	UploadPath string
	UploadTime time.Time
	Md5        string
}

// The application controller itself
type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
	return c.Render()
}

// Returns a sanitized filename with all non-alphanumeric characters replaced by dashes
func sanitizeName(name string) string {
	// sanitize the filename
	// TODO: refactor this to a static if golang regexp is thread-safe / re-enterant
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		revel.ERROR.Printf("Error compiling regexp: %v", err)
	}

	return reg.ReplaceAllString(name, "-")
}

// get the hash of the contents of the file, so that even files
// uploaded at the same time can be differentiated (this is important for the
// tests)
func getContentHash(fileContents []byte) string {
	return fmt.Sprintf("%x", md5.Sum(fileContents))

}

/// Returns the path where a file needs to be placed
func getUploadPath(tenant string, pkg string, filename string, requestTime time.Time, fileHash string) string {
	// the folder name is only the date
	folderTimestamp := requestTime.Format("2006-01-02")
	// the file name gets the timestamp appended (only time)
	fileTimestamp := requestTime.Format("15-04--05-00")

	// get the extension and basename
	fileBaseName := sanitizeName(path.Base(filename))
	fileExtName := sanitizeName(path.Ext(filename))
	fullFileName := fmt.Sprintf("%v-%v-%v.%v", fileBaseName, fileTimestamp, fileHash, fileExtName[1:])

	// the file name is the sanitized file name
	return filepath.ToSlash(path.Join(OUTPUT_DIR, sanitizeName(tenant), "uploads", sanitizeName(pkg), folderTimestamp, fullFileName))
}

// Handle an actual upload
func (c App) Upload(tenant string, pkg string, filename string) revel.Result {

	// get the request time
	requestTime := time.Now()

	requestBody := c.Request.Body

	// read the contents of the post body
	content, err := ioutil.ReadAll(requestBody)
	if err != nil {
		c.RenderError(err)
	}

	// calculate the hash
	fileHash := getContentHash(content)

	// get the output path
	outputPath := getUploadPath(tenant, pkg, filename, requestTime, fileHash)

	revel.INFO.Printf("got %v bytes for %v / %v / %v -- hash is: %v", len(content), tenant, pkg, filename, fileHash)

	// create the directory of the file
	err = os.MkdirAll(filepath.Dir(outputPath), OUTPUT_DEFAULT_DIRMODE)
	if err != nil {
		c.RenderError(err)
	}

	// write out the contents to a new file
	err = ioutil.WriteFile(outputPath, content, OUTPUT_DEFAULT_MODE)
	if err != nil {
		return c.RenderError(err)
	}

	// render some nice output
	return c.RenderJson(UploadResponse{outputPath, requestTime, fileHash})
}
