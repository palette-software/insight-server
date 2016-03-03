package main

import (
	"fmt"
	"net/http"

	"github.com/palette-software/insight-server"
	"os"
	"path"
	"path/filepath"
	"log"
)

const (
// The key in ENV where the license files are looked up.
// If this key isnt provided, the 'licenses' subdirectory of the working directory is used
	LicenseDirectoryKey = "INSIGHT_LICENSES_PATH"
// The address where the server will bind itself
	BindAddress = ":9000"
// The key in the Environment where the files will be uploaded
// if no such value is set in the env, use ENV["TEMP"]
	UploadPathEnvKey = "INSIGHT_UPLOAD_HOME"
)

func pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "PONG")
}

func getBindAddress() string {
	port := os.Getenv("PORT")
	if port == "" {
		return BindAddress
	}
	return fmt.Sprintf(":%v", port)
}


// Gets the path where a certain tenants files for the given package reside
func getUploadBasePath() string {
	uploadBaseDir := os.Getenv(UploadPathEnvKey)
	if uploadBaseDir == "" {
		return path.Join(os.Getenv("TEMP"), "uploads")
	}
	return uploadBaseDir
}

// Returns the current working directory
func getCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	return dir

}

func getLicensesDirectory() string {

	// get the licenses root directory from the env if possible
	licensesRoot := os.Getenv(LicenseDirectoryKey)
	if licensesRoot == "" {
		licensesRoot = path.Join(getCurrentPath(), "licenses")
	}
	return licensesRoot
}

func main() {

	http.HandleFunc("/", pingHandler)

	authenticator := insight_server.NewLicenseAuthenticator(getLicensesDirectory())
	uploader := insight_server.MakeBasicUploader(getUploadBasePath())
	uploadHandler := insight_server.MakeUploadHandler(uploader)

	authenticatedUploadHandler := insight_server.MakeUserAuthHandler(authenticator, uploadHandler)

	// declare both endpoints for now. /upload-with-meta is deprecated
	http.HandleFunc("/upload-with-meta", authenticatedUploadHandler)
	http.HandleFunc("/upload", authenticatedUploadHandler)

	bindAddress := getBindAddress()
	fmt.Printf("Webservice starting on %v\n", bindAddress)
	http.ListenAndServe(bindAddress, nil)
}
