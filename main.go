package main

import (
	"github.com/palette-software/insight-server/lib"

	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
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

	// BACKENDS
	// --------
	// the temporary files are stored here so moving them wont result in errors
	tempDir := filepath.Join(config.UploadBasePath, "_temp")

	// make sure the temporary directory exists
	insight_server.CreateDirectoryIfNotExists(tempDir)

	// create the maxid backend
	maxIdBackend := insight_server.MakeFileMaxIdBackend(config.MaxIdDirectory)

	// create the authenticator
	authenticator := insight_server.NewLicenseAuthenticator(config.LicensesDirectory)

	// create the autoupdater backend
	autoUpdater, err := insight_server.NewBaseAutoUpdater(config.UpdatesDirectory)
	if err != nil {
		logrus.Fatalf("Error during creation of Autoupdater: %v", err)
	}

	// for now, put the commands file in the updates directory (should be skipped by the updater)
	commandBackend := insight_server.NewFileCommandsEndpoint(config.UpdatesDirectory)

	// ENDPOINTS
	// ---------

	// create the upload endpoint
	authenticatedUploadHandler := withRequestLog("upload",
		insight_server.MakeUserAuthHandler(
			authenticator,
			insight_server.MakeUploadHandler(maxIdBackend, tempDir, config.UploadBasePath, config.ServerlogsArchivePath, config.UseOldFormatFilename),
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

	licenseCheckHandler := withRequestLog("license-check", insight_server.LicenseCheckHandler())

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

	// License check
	http.HandleFunc("/license-check", licenseCheckHandler)

	// auto-update distribution: The updates should be publicly accessable
	logrus.WithFields(logrus.Fields{
		"component": "http",
		"directory": config.UpdatesDirectory,
	}).Info("Serving static content for updates")
	http.Handle("/updates/products/", http.StripPrefix("/updates/products/", http.FileServer(http.Dir(config.UpdatesDirectory))))

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