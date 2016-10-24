package main

import (
	"github.com/palette-software/insight-server/lib"

	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var authHeaderRegExp = regexp.MustCompile("Token (.*)")

// Auth middleware
func AuthMiddleware(licenseKey string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := authHeaderRegExp.FindStringSubmatch(string(authHeader))
		if len(token) < 2 || strings.ToLower(token[1]) != licenseKey {
			insight_server.WriteResponse(w, http.StatusUnauthorized, "Not authorized")
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Middleware to log all incoming requests in a common format
func RequestLogMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logrus.WithFields(logrus.Fields{
			"component":     "http",
			"method":        r.Method,
			"remoteAddress": r.RemoteAddr,
			"url":           r.URL.RequestURI(),
		}).Info("==> Request")
		// also write all header we care about for proxies here
		w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Add("Pragma", "no-cache")
		w.Header().Add("Expires", "0")
		h.ServeHTTP(w, r)
	})
}

// Middleware to maintain agent list
func HeartbeatMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hostname := r.FormValue("hostname"); hostname != "" {
			insight_server.AgentHeartbeat(hostname)
		}
		h.ServeHTTP(w, r)
	})
}

// Returns the current working directory
func getCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logrus.Fatal(err)
		panic(err)
	}
	return dir

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
	//
	license := insight_server.UpdateLicense(config.LicenseKey)
	_, err := insight_server.CheckLicense(license)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"version": insight_server.GetVersion(),
			"license": config.LicenseKey,
		}).Error("Invalid or expired license, exiting.")
		os.Exit(1)
	}

	insight_server.InitCommandEndpoints()

	//
	// BACKENDS
	// --------
	// the temporary files are stored here so moving them wont result in errors
	tempDir := filepath.Join(config.UploadBasePath, "_temp")

	// make sure the temporary directory exists
	insight_server.CreateDirectoryIfNotExists(tempDir)

	// create the maxid backend
	maxIdBackend := insight_server.MakeFileMaxIdBackend(config.MaxIdDirectory)

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

	// HANDLERS
	// ========

	// CSV upload
	// declare both endpoints for now. /upload-with-meta is deprecated
	mainRouter := mux.NewRouter()
	mainRouter.Handle("/upload", AuthMiddleware(config.LicenseKey, uploadHandler))
	mainRouter.Handle("/maxid", AuthMiddleware(config.LicenseKey, insight_server.MakeMaxIdHandler(maxIdBackend)))

	// Commands
	mainRouter.HandleFunc("/commands", insight_server.AssetPageHandler("assets/agent-commands.html"))

	// v1
	apiRouter := mainRouter.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/ping", insight_server.PingHandler).Methods("GET")
	apiRouter.Handle("/license", AuthMiddleware(config.LicenseKey, insight_server.LicenseHandler(config.LicenseKey)))
	apiRouter.Handle("/agent/version", insight_server.GetAutoupdateLatestVersionHandler(config.UpdatesDirectory)).Methods("GET")
	apiRouter.Handle("/agent", http.StripPrefix("/api/v1/", http.FileServer(http.Dir(config.UpdatesDirectory)))).Methods("GET")
	apiRouter.Handle("/api/v1/agent", http.StripPrefix("/api/v1/api/v1/", http.FileServer(http.Dir(config.UpdatesDirectory)))).Methods("GET")
	apiRouter.HandleFunc("/config", insight_server.ServeConfig).Methods("GET")
	apiRouter.HandleFunc("/config", insight_server.UploadConfig).Methods("PUT")
	apiRouter.HandleFunc("/command", insight_server.AddCommandHandler).Methods("PUT")
	apiRouter.Handle("/command", insight_server.NewGetCommandHandler()).Methods("GET")
	apiRouter.HandleFunc("/agents", insight_server.AgentListHandler).Methods("GET")

	// DEPRECATING
	mainRouter.Handle("/updates/products/agent/{version}/{rest}", http.StripPrefix("/updates/products/agent/", http.FileServer(http.Dir(config.UpdatesDirectory)))).Methods("GET")

	// DEPRECATING IN v2
	// License check
	mainRouter.HandleFunc("/license-check", func(w http.ResponseWriter, req *http.Request) {
		owner, _ := insight_server.GetLicenseOwner()
		response := fmt.Sprintf("{\"owner\": \"%s\", \"valid\": true}", owner)
		insight_server.WriteResponse(w, http.StatusOK, response)
	})
	mainRouter.Handle("/updates/latest-version", insight_server.GetAutoupdateLatestVersionHandler(config.UpdatesDirectory))

	mainRouter.HandleFunc("/commands/new", insight_server.AddCommandHandler)
	mainRouter.HandleFunc("/commands/recent", insight_server.NewGetCommandHandler())

	// STARTING THE SERVER
	// ===================
	// http.Handle("/", AuthMiddleware(config.LicenseKey, mainRouter))
	handlerWithHeartbeat := HeartbeatMiddleware(mainRouter)
	handlerWithLogging := RequestLogMiddleware(handlerWithHeartbeat)
	// http.Handle("/", handlerWithLogging)

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

		err := http.ListenAndServeTLS(bindAddressWithPort, config.TlsCert, config.TlsKey, handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "PUT"}),
		)(handlerWithLogging))
		logrus.Fatal(err)
	} else {
		err := http.ListenAndServe(bindAddressWithPort, handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "PUT"}),
		)(handlerWithLogging))
		logrus.Fatal(err)
	}

}
