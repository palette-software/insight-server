package insight_server

import (
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"

	"github.com/namsral/flag"
	"strings"
)

// The configuration of the web service
type InsightWebServiceConfig struct {
	LicenseKey                          string
	UploadBasePath, MaxIdDirectory      string
	LicensesDirectory, UpdatesDirectory string
	AgentConfigsDirectory               string
	BindAddress                         string
	BindPort                            int
	TlsKey, TlsCert                     string
	UseTls                              bool

	// can be either 'json' or 'text'. Defaults to text
	LogFormat string

	LogLevel string

	// The archive path for the serverlogs
	ServerlogsArchivePath string

	// Should the filenames use the old format?
	// like 'countersamples-2016-04-18--14-10-08--seq0000--part0000-csv-08-00--14-00-95755b03f960d2994dbad08067504e02.csv.gz'
	// (with double timestamp)
	UseOldFormatFilename bool
}

// Returns the current working directory
func getCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logrus.WithField("component", "config").WithError(err).Fatal("Error getting current path")
		panic(err)
	}
	return dir
}

func ParseOptions() InsightWebServiceConfig {

	var licenseKey, uploadBasePath, maxIdDirectory, licensesDirectory, updatesDirectory, agentConfigsDirectory string
	var bindAddress, archivePath, logFormat, logLevel string
	var bindPort int

	// License info
	// ==========

	flag.StringVar(&licenseKey, "license_key", "", "License key for Palette Insight")

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

	flag.StringVar(&agentConfigsDirectory, "agent_configs_path",
		filepath.Join(getCurrentPath(), "agent_configs"),
		"The base directory where the agent configuration files are stored.",
	)

	flag.StringVar(&logFormat, "logformat", "text", "The log format to use ('json' or 'text' or 'color')")
	flag.StringVar(&logLevel, "loglevel", "info", "The log level to use ('info', 'warn' or 'debug')")

	flag.StringVar(&archivePath, "archive_path", "", "The directory where the uploaded serverlogs are archived.")
	flag.IntVar(&bindPort, "port", 9000, "The port the server is binding itself to")
	flag.StringVar(&bindAddress, "bind_address", "", "The address to bind to. Leave empty for default .")

	// SSL / HTTPS
	// ===========

	var useTls bool
	var tlsCert, tlsKey string

	flag.BoolVar(&useTls, "tls", false, "Use TLS for serving through HTTPS.")
	flag.StringVar(&tlsCert, "cert", "cert.pem", "The TLS certificate file to use when tls is set.")
	flag.StringVar(&tlsKey, "key", "key.pem", "The TLS certificate key file to use when tls is set.")

	// MISC
	// ====
	var useOldFormatFilename bool

	flag.BoolVar(&useOldFormatFilename, "old_filename", false, "Use the old output filename format")

	// CONFIG FILE
	// ===========

	flag.String("config", "", "Configuration file to use.")

	flag.Parse()

	// Set the archive path if its unset
	if archivePath == "" {
		archivePath = filepath.Join(uploadBasePath, "..", "serverlogs-archives")
	}

	// after parse, return the results
	return InsightWebServiceConfig{
		LicenseKey:            strings.ToLower(licenseKey),
		UploadBasePath:        uploadBasePath,
		MaxIdDirectory:        maxIdDirectory,
		LicensesDirectory:     licensesDirectory,
		UpdatesDirectory:      updatesDirectory,
		AgentConfigsDirectory: agentConfigsDirectory,

		BindAddress: bindAddress,
		BindPort:    bindPort,

		UseTls:  useTls,
		TlsCert: tlsCert,
		TlsKey:  tlsKey,

		LogFormat: logFormat,
		LogLevel:  logLevel,

		ServerlogsArchivePath: archivePath,
		UseOldFormatFilename:  useOldFormatFilename,
	}
}
