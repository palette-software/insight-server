package main

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"path"
	"strings"

	"flag"

	"github.com/palette-software/insight-server"
)

// Utility that tries to re-parse the serverlogs

// A file that is gzipped
/////////////////////////

type GzippedFileReader struct {
	File   *os.File
	Reader *gzip.Reader
}

func NewGzippedFileReader(filename string) (*GzippedFileReader, error) {

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	gzr, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	return &GzippedFileReader{File: f, Reader: gzr}, nil
}

func (g *GzippedFileReader) Close() {
	// just make sure that things are truly closed
	defer g.File.Close()
	defer g.Reader.Close()
}

/////////////////////////////////

// A list of packages we are willing to re-check for serverlogs
var packagesToCheck []string = []string{"public"}

func checkTenants(config *insight_server.InsightWebServiceConfig) ([]string, error) {

	return filepath.Glob(filepath.Join(
		config.UploadBasePath,
		"*", // tenant
		"uploads",
		"*", // package
		"*", // host
		"errors_serverlogs*.csv.gz"))
}

func reloadServerlogs(outputPath, tempdir, filename string, sourceTimezoneName string) error {
	gzr, err := NewGzippedFileReader(filename)
	if err != nil {
		return fmt.Errorf("Error while opening file '%s': %v", filename, err)
	}
	defer gzr.Close()

	log.Printf("Parsing: %s", filename)

	hostName := getHostName(filename)

	rows, errorRows, err := insight_server.ParseServerlogsWithFn(gzr.Reader, sourceTimezoneName, func(records []string) (string, string, string, error) {
		// the line is the second entry
		return hostName, "SERVERLOGS-REPARSED", records[1], nil
	})

	if err != nil {
		return fmt.Errorf("Error while parsing '%s': %v", filename, err)
	}
	gzr.Close()

	insight_server.WriteServerlogsCsv(tempdir, strings.Replace(filename, "errors_serverlogs", "fixed_serverlogs", -1), rows)

	log.Printf("Parse output: %d rows, %d errorRows", len(rows), len(errorRows))

	destinationPath := filepath.Join(outputPath, hostName, filepath.Base(filename))

	insight_server.CreateDirectoryIfNotExists(filepath.Dir(destinationPath))

	if err := os.Rename(filename, destinationPath); err != nil {
		return fmt.Errorf("Error moving '%s' to '%s': %v", filename, destinationPath, err)
	}

	return nil
}

// Returns the host name for a file (aka. the basename of the directory of
// the file
func getHostName(filename string) string {
	return filepath.Base(filepath.Dir(filename))
}

func main() {
	var backupPath string
	flag.StringVar(&backupPath, "backup",
		filepath.Join(os.Getenv("TEMP"), "uploads"),
		"The root directory for the backups to go into.",
	)

	config := insight_server.ParseOptions()

	log.Printf("Config: %v", config)
	log.Printf("BackupPath: %s", backupPath)

	// create the authenticator

	// All files we care about are in the base path

	errorFiles, err := checkTenants(&config)
	if err != nil {
		panic(err)
	}

	for _, errFile := range errorFiles {
		err := reloadServerlogs(backupPath, path.Join(config.UploadBasePath, "_temp"), errFile, "UTC")
		if err != nil {
			panic(err)
		}
	}
}
