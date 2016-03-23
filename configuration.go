package insight_server

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

// The configuration of the web service
type InsightWebServiceConfig struct {
	UploadBasePath, MaxIdDirectory, LicensesDirectory, UpdatesDirectory, BindAddress string
	BindPort                                                                         int
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

func ParseOptions() InsightWebServiceConfig {

	log.Printf("[boot] Starting palette insight-server %s", GetVersion())

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

	// after parse, return the results
	return InsightWebServiceConfig{
		UploadBasePath:    uploadBasePath,
		MaxIdDirectory:    maxIdDirectory,
		LicensesDirectory: licensesDirectory,
		UpdatesDirectory:  updatesDirectory,

		BindAddress: bindAddress,
		BindPort:    bindPort,
	}
}