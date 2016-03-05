package main

import (
	"fmt"
	"net/http"

	"github.com/palette-software/insight-server"

// Command line
	"github.com/namsral/flag"
	"os"
	"path/filepath"
	"log"
)

func pingHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("[HTTP] {%v} Request arrived to: %v\n", req.Method, req.URL)
	fmt.Fprintf(w, "PONG")
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


// Adds basic request logging to the wrapped handler
func withRequestLog(name string, innerHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[http] (handler:%s) {%v} %v%v?%v", name, r.Method, r.URL.Host, r.URL.Path, r.URL.RawQuery)
		innerHandler(w, r)
	}
}

func main() {

	var uploadBasePath, maxIdDirectory, licensesDirectory, bindAddress string
	var bindPort int

	flag.StringVar(&uploadBasePath, "upload_path",
		filepath.Join(os.Getenv("TEMP"), "uploads"),
		"The root directory for the uploads to go into.",
	)
	// Since we have to provide defaults to flag before they are parsed
	// we cannot have paths dependent on one another
	flag.StringVar(&maxIdDirectory, "maxid_path",
		filepath.Join(os.Getenv("TEMP"), "uploads", "maxid"),
		"The root directory for the maxid files to go into.",
	)

	flag.StringVar(&licensesDirectory, "licenses_path",
		filepath.Join(getCurrentPath(), "licenses"),
		"The directory the licenses are loaded from on start.",
	)

	flag.IntVar(&bindPort, "port", 9000, "The port the server is binding itself to")
	flag.StringVar(&bindAddress, "bind_address", "", "The address to bind to. Leave empty for default .")

	flag.String("config", "", "Configuration file to use.")

	flag.Parse()

	http.HandleFunc("/", pingHandler)

	// create the uploader
	uploader := insight_server.MakeBasicUploader(filepath.ToSlash(uploadBasePath))

	// create the maxid backend
	maxIdBackend := insight_server.MakeFileMaxIdBackend(maxIdDirectory)

	// create the authenticator
	authenticator := insight_server.NewLicenseAuthenticator(licensesDirectory)

	// create the upload endpoint
	authenticatedUploadHandler := withRequestLog("upload",
		insight_server.MakeUserAuthHandler(
			authenticator,
			insight_server.MakeUploadHandler(uploader, maxIdBackend),
		),
	)
	// create the maxid handler
	maxIdHandler := withRequestLog("maxid",
		insight_server.MakeUserAuthHandler(
			authenticator,
			insight_server.MakeMaxIdHandler(maxIdBackend),
		),
	)

	// declare both endpoints for now. /upload-with-meta is deprecated
	http.HandleFunc("/upload-with-meta", authenticatedUploadHandler)
	http.HandleFunc("/upload", authenticatedUploadHandler)
	http.HandleFunc("/maxid", maxIdHandler)

	//bindAddress := getBindAddress()
	bindAddressWithPort := fmt.Sprintf("%s:%v", bindAddress, bindPort)
	fmt.Printf("Webservice starting on %v\n", bindAddressWithPort)
	http.ListenAndServe(bindAddressWithPort, nil)
}
