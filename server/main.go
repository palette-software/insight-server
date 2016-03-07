package main

import (
	"github.com/palette-software/insight-server"
	"github.com/namsral/flag"

	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"log"
	"regexp"
)


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
		// also write all header we care about for proxies here
		w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Add("Pragma", "no-cache")
		w.Header().Add("Expires", "0")
		innerHandler(w, r)
	}
}

func main() {

	var uploadBasePath, maxIdDirectory, licensesDirectory, bindAddress string
	var bindPort int

	// Path setup
	// ==========

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

	// SSL / HTTPS
	// ===========

	var useTls bool
	var tlsCert, tlsKey string

	flag.BoolVar(&useTls, "tls", false, "Use TLS for serving through HTTPS.")
	flag.StringVar(&tlsCert, "cert", "cert.pem", "The TLS certificate file to use when tls is set.")
	flag.StringVar(&tlsKey, "key", "key.pem", "The TLS certificate key file to use when tls is set.")



	// CONFIG FILE
	// ===========

	flag.String("config", "", "Configuration file to use.")

	flag.Parse()

	// create the uploader
	uploader := insight_server.MakeBasicUploader(filepath.ToSlash(uploadBasePath))

	// create the maxid backend
	maxIdBackend := insight_server.MakeFileMaxIdBackend(maxIdDirectory)

	// create the authenticator
	authenticator := insight_server.NewLicenseAuthenticator(licensesDirectory)


	// create the server logs parser
	serverlogsParser := insight_server.MakeServerlogParser(16)

	uploader.AddCallback(&insight_server.UploadCallback{
		Name: "Serverlogs parsing",
		Pkg: regexp.MustCompile(""),
		Filename: regexp.MustCompile("^serverlogs-"),
		Handler: func(c *insight_server.UploadCallbackCtx) error {
			serverlogsParser <- insight_server.ServerlogToParse{c.SourceFile, c.OutputFile}
			return nil
		},
	})

	uploader.AddCallback(&insight_server.UploadCallback{
		Name: "Serverlogs metadata addition",
		Pkg: regexp.MustCompile(""),
		Filename: regexp.MustCompile("^metadata-"),
		Handler: insight_server.MetadataUploadHandler,
	})
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

	// HANDLERS
	// ========
	http.HandleFunc("/", withRequestLog("ping", insight_server.PingHandler))
	// declare both endpoints for now. /upload-with-meta is deprecated
	http.HandleFunc("/upload-with-meta", authenticatedUploadHandler)
	http.HandleFunc("/upload", authenticatedUploadHandler)
	http.HandleFunc("/maxid", maxIdHandler)

	//bindAddress := getBindAddress()
	bindAddressWithPort := fmt.Sprintf("%s:%v", bindAddress, bindPort)
	fmt.Printf("Webservice starting on %v\n", bindAddressWithPort)

	if useTls {
		err := http.ListenAndServeTLS(bindAddressWithPort, tlsCert, tlsKey, nil)
		log.Fatal(err)
	} else {

		err := http.ListenAndServe(bindAddressWithPort, nil)
		log.Fatal(err)
	}

}
