package main

import (
	"github.com/palette-software/insight-server/lib"

	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/palette-software/go-log-targets"

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
			insight_server.WriteResponse(w, http.StatusUnauthorized, "Not authorized", r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// Middleware to log all incoming requests in a common format
func RequestLogMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Request: method=%s remoteAddress=%s url=%s", r.Method, r.RemoteAddr, r.URL.RequestURI())
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
		log.Error("Error while geting current path: ", err)
		panic(err)
	}
	return dir

}

func main() {
	config := insight_server.ParseOptions()

	log.AddTarget(os.Stdout, log.LevelDebug)

	license := insight_server.UpdateLicense(config.LicenseKey)
	_, err := insight_server.CheckLicense(license)
	if err != nil {
		log.Errorf("Invalid or expired license - exiting. license=%s version=%s err=%s", config.LicenseKey, insight_server.GetVersion(), err)
		os.Exit(1)
	}

	log.Infof("License is registered to: %s", license.Name)

	insight_server.InitCommandEndpoints()

	// setup the log timezone to be UTC (and keep any old flags)
	// insight_server.SetupLogging(config.LogFormat, config.LogLevel)
	log.Infof("Starting up. version=%s path=%s", insight_server.GetVersion(), getCurrentPath())

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
		log.Error("Error during upload handler creation", err)
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
	mainRouter.HandleFunc("/commands/new", insight_server.AddCommandHandler)
	mainRouter.HandleFunc("/commands/recent", insight_server.NewGetCommandHandler())

	// STARTING THE SERVER
	// ===================
	// http.Handle("/", AuthMiddleware(config.LicenseKey, mainRouter))
	handlerWithHeartbeat := HeartbeatMiddleware(mainRouter)
	handlerWithLogging := RequestLogMiddleware(handlerWithHeartbeat)
	// http.Handle("/", handlerWithLogging)

	bindAddressWithPort := fmt.Sprintf("%s:%v", config.BindAddress, config.BindPort)
	log.Infof("Starting webservice: address=%s port=%d", config.BindAddress, config.BindPort)

	if config.UseTls {
		log.Infof("Using TLS cert: cert=%s key=%s", config.TlsCert, config.TlsKey)

		err := http.ListenAndServeTLS(bindAddressWithPort, config.TlsCert, config.TlsKey, handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "PUT"}),
		)(handlerWithLogging))
		log.Errorf("Exiting. err=%s", err)
	} else {
		err := http.ListenAndServe(bindAddressWithPort, handlers.CORS(
			handlers.AllowedOrigins([]string{"*"}),
			handlers.AllowedMethods([]string{"GET", "PUT"}),
		)(handlerWithLogging))
		log.Errorf("Exiting. err=%s", err)
	}

}
