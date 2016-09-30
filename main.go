package main

import (
	"github.com/palette-software/insight-server/lib"

	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// Returns the current working directory
func getCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logrus.Fatal(err)
		panic(err)
	}
	return dir

}

// Adds basic request logging to the wrapped handler
func withRequestLog(name string, innerHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logrus.WithFields(logrus.Fields{
			"component":     "http",
			"method":        r.Method,
			"remoteAddress": r.RemoteAddr,
			"url":           r.URL.RequestURI(),
			"handler":       name,
		}).Info("==> Request")
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
	config := insight_server.ParseOptions()

	// setup the log timezone to be UTC (and keep any old flags)
	insight_server.SetupLogging(config.LogFormat, config.LogLevel)
	logrus.WithFields(logrus.Fields{
		"component": "boot",
		"version":   insight_server.GetVersion(),
		"path":      getCurrentPath(),
	}).Info("Starting palette insight-server")

	license := insight_server.UpdateLicense(config.LicenseKey)
	licenseOK, _ := insight_server.CheckLicense(config.LicenseKey, license)
	if !licenseOK {
		logrus.WithFields(logrus.Fields{
			"version": insight_server.GetVersion(),
			"license": config.LicenseKey,
		}).Error("Invalid or expired license, exiting.")

		os.Exit(1)
	}

	// BACKENDS
	// --------
	// the temporary files are stored here so moving them wont result in errors
	tempDir := filepath.Join(config.UploadBasePath, "_temp")

	// make sure the temporary directory exists
	insight_server.CreateDirectoryIfNotExists(tempDir)

	// create the maxid backend
	maxIdBackend := insight_server.MakeFileMaxIdBackend(config.MaxIdDirectory)

	// create the autoupdater backend
	autoUpdater := insight_server.NewBaseAutoUpdater()

	// for now, put the commands file in the updates directory (should be skipped by the updater)
	commandBackend := insight_server.NewFileCommandsEndpoint(config.UpdatesDirectory)

	// ENDPOINTS
	// ---------

	uploadHandler, err := insight_server.MakeUploadHandler(maxIdBackend, tempDir, config.UploadBasePath, config.ServerlogsArchivePath, config.UseOldFormatFilename)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"component": "uploads",
		}).Fatal("Error during upload handler creation")
		// Fail with an error here
		os.Exit(-1)
	}

	// create the upload endpoint
	authenticatedUploadHandler := withRequestLog("upload",
		insight_server.MakeUserAuthHandler(uploadHandler),
	)
	// create the maxid handler
	maxIdHandler := withRequestLog("maxid",
		insight_server.MakeUserAuthHandler(
			insight_server.MakeMaxIdHandler(maxIdBackend),
		),
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

	// Commands
	http.HandleFunc("/commands/new", newCommandHandler)
	http.HandleFunc("/commands/recent", getCommandHandler)
	http.HandleFunc("/commands", staticHandler("new-command", "assets/agent-commands.html"))

	// auto-update distribution: The updates should be publicly accessable
	logrus.WithFields(logrus.Fields{
		"component": "http",
		"directory": config.UpdatesDirectory,
	}).Info("Serving static content for updates")

	// v1
	mainRouter := mux.NewRouter()
	apiRouter := mainRouter.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/ping", insight_server.PingHandler).Methods("GET")
	apiRouter.HandleFunc("/license", insight_server.LicenseHandler(config.LicenseKey))
	apiRouter.HandleFunc("/agent/version", withRequestLog("update-latest-version", insight_server.AutoupdateLatestVersionHandler(autoUpdater))).Methods("GET")
	apiRouter.Handle("/agent", http.StripPrefix("/api/v1/", http.FileServer(http.Dir(config.UpdatesDirectory)))).Methods("GET")

	// DEPRECATING IN v2
	mainRouter.Handle("/updates/products/agent/{version}/{rest}", http.StripPrefix("/updates/products/agent/", http.FileServer(http.Dir(config.UpdatesDirectory)))).Methods("GET")

	http.Handle("/", mainRouter)

	// DEPRECATING IN v2
	// License check
	http.HandleFunc("/license-check", func(w http.ResponseWriter, req *http.Request) {
		hostname, _ := os.Hostname()
		response := fmt.Sprintf("{\"owner\": \"%s\", \"valid\": true}", hostname)
		insight_server.WriteResponse(w, http.StatusOK, response)
	})
	http.HandleFunc("/updates/latest-version", withRequestLog("update-latest-version", insight_server.AutoupdateLatestVersionHandler(autoUpdater)))

	// STARTING THE SERVER
	// ===================

	bindAddressWithPort := fmt.Sprintf("%s:%v", config.BindAddress, config.BindPort)
	logrus.WithFields(logrus.Fields{
		"component": "http",
		"address":   bindAddressWithPort,
	}).Info("Webservice starting")

	if config.UseTls {
		logrus.WithFields(logrus.Fields{
			"component": "http",
			"cert":      config.TlsCert,
			"key":       config.TlsKey,
		}).Info("Using TLS cert")

		err := http.ListenAndServeTLS(bindAddressWithPort, config.TlsCert, config.TlsKey, nil)
		logrus.Fatal(err)
	} else {
		err := http.ListenAndServe(bindAddressWithPort, nil)
		logrus.Fatal(err)
	}

}
