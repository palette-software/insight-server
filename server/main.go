package main

import (
	"github.com/namsral/flag"
	"github.com/palette-software/insight-server"

	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

func staticHandler(name, assetPath string) http.HandlerFunc {
	return withRequestLog(name, insight_server.AssetPageHandler(assetPath))
}

func main() {

	// setup the log timezone to be UTC (and keep any old flags)
	log.SetFlags(log.Flags() | log.LUTC)

	log.Printf("[boot] Starting palette insight-server %s", insight_server.GetVersion())

	var uploadBasePath, maxIdDirectory, licensesDirectory, updatesDirectory, bindAddress string
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

	flag.StringVar(&updatesDirectory, "updates_path",
		filepath.Join(getCurrentPath(), "updates"),
		"The directory where the update files for the agent are stored.",
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

	// BACKENDS
	// --------

	// create the uploader
	uploader, err := insight_server.MakeBasicUploader(filepath.ToSlash(uploadBasePath))
	if err != nil {
		// log the error and exit
		log.Fatalf("Error during creating the uploader: %v", err)
	}

	// create the maxid backend
	maxIdBackend := insight_server.MakeFileMaxIdBackend(maxIdDirectory)

	// create the authenticator
	authenticator := insight_server.NewLicenseAuthenticator(licensesDirectory)

	// create the server logs parser
	serverlogsParser := insight_server.MakeServerlogParser(16)

	// create the autoupdater backend
	autoUpdater, err := insight_server.NewBaseAutoUpdater(updatesDirectory)
	if err != nil {
		log.Fatalf("Error during creation of Autoupdater: %v", err)
	}

	// for now, put the commands file in the updates directory (should be skipped by the updater)
	commandBackend := insight_server.NewFileCommandsEndpoint(updatesDirectory)

	// UPLOADER CALLBACKS
	// ------------------

	uploader.AddCallback(&insight_server.UploadCallback{
		Name:     "Serverlogs parsing",
		Pkg:      regexp.MustCompile(""),
		Filename: regexp.MustCompile("^serverlogs-"),
		Handler: func(c *insight_server.UploadCallbackCtx) error {
			serverlogsParser <- insight_server.ServerlogToParse{c.SourceFile, c.OutputFile, c.Basedir, c.Timezone}
			return nil
		},
	})

	uploader.AddCallback(&insight_server.UploadCallback{
		Name:     "Serverlogs metadata addition",
		Pkg:      regexp.MustCompile(""),
		Filename: regexp.MustCompile("^metadata-"),
		Handler:  insight_server.MetadataUploadHandler,
	})

	// ENDPOINTS
	// ---------

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

	autoUpdatesAddHandler := withRequestLog("autoupdate-add",
		insight_server.NewAutoupdateHttpHandler(autoUpdater),
	)

	newCommandHandler := withRequestLog("commands-new", insight_server.NewAddCommandHandler(commandBackend))
	getCommandHandler := withRequestLog("commands-get", insight_server.NewGetCommandHandler(commandBackend))

	// HANDLERS
	// ========

	// CSV upload
	// declare both endpoints for now. /upload-with-meta is deprecated
	http.HandleFunc("/upload-with-meta", authenticatedUploadHandler)
	http.HandleFunc("/upload", authenticatedUploadHandler)
	http.HandleFunc("/maxid", maxIdHandler)

	// auto-updates
	//http.HandleFunc("/updates/new-version", withRequestLog("new-version", insight_server.AssetPageHandler("assets/upload-new-version.html")))
	http.HandleFunc("/updates/new-version", staticHandler("new-version", "assets/upload-new-version.html"))
	http.HandleFunc("/updates/add-version", autoUpdatesAddHandler)
	http.HandleFunc("/updates/latest-version", withRequestLog("update-latest-version", insight_server.AutoupdateLatestVersionHandler(autoUpdater)))

	// Commands
	http.HandleFunc("/commands/new", newCommandHandler)
	http.HandleFunc("/commands/recent", getCommandHandler)
	http.HandleFunc("/commands", staticHandler("new-command", "assets/agent-commands.html"))

	// auto-update distribution: The updates should be publicly accessable
	log.Printf("[http] Serving static content for updates from: %s", updatesDirectory)
	http.Handle("/updates/products/", http.StripPrefix("/updates/products/", http.FileServer(http.Dir(updatesDirectory))))

	// STARTING THE SERVER
	// ===================

	bindAddressWithPort := fmt.Sprintf("%s:%v", bindAddress, bindPort)
	log.Printf("[http] Webservice starting on %v\n", bindAddressWithPort)

	if useTls {
		log.Printf("[http] Using TLS cert: '%s' and key: '%s'", tlsCert, tlsKey)
		err := http.ListenAndServeTLS(bindAddressWithPort, tlsCert, tlsKey, nil)
		log.Fatal(err)
	} else {
		err := http.ListenAndServe(bindAddressWithPort, nil)
		log.Fatal(err)
	}

}
