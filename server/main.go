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

// The key in the env where the maxid files are stored
	MaxidDirectoryKey = "INSIGHT_MAXID_PATH"
// The path to the directory where the maxid is stored
	MaxIdBasePath = "_maxid"
)

func pingHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("[HTTP] {%v} Request arrived to: %v\n", req.Method, req.URL)
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


// tries to get a string from the Environment and returns
// defaults when its not set
func getFromEnv(key, defaults string) string {
	// get the licenses root directory from the env if possible
	v := os.Getenv(key)
	if v == "" {
		v = defaults
	}
	return v
}

func main() {

	http.HandleFunc("/", pingHandler)

	// create the maxid backend
	maxIdBackendDirectory := getFromEnv(MaxidDirectoryKey, path.Join(getUploadBasePath(), MaxIdBasePath))
	maxIdBackend := insight_server.MakeFileMaxIdBackend(maxIdBackendDirectory)

	// create the authenticator
	authenticator := insight_server.NewLicenseAuthenticator(getLicensesDirectory())
	uploader := insight_server.MakeBasicUploader(getUploadBasePath())

	// create the upload endpoint
	authenticatedUploadHandler := insight_server.MakeUserAuthHandler(
		authenticator,
		insight_server.MakeUploadHandler(uploader, maxIdBackend),
	)
	// create the maxid handler
	maxIdHandler := insight_server.MakeUserAuthHandler(
		authenticator,
		insight_server.MakeMaxIdHandler(maxIdBackend),
	)

	// declare both endpoints for now. /upload-with-meta is deprecated
	http.HandleFunc("/upload-with-meta", authenticatedUploadHandler)
	http.HandleFunc("/upload", authenticatedUploadHandler)
	http.HandleFunc("/maxid", maxIdHandler)

	bindAddress := getBindAddress()
	fmt.Printf("Webservice starting on %v\n", bindAddress)
	http.ListenAndServe(bindAddress, nil)
}
