package main

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"path"
	"strings"

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

/*
type GzippedFileWriter struct {
	File *os.File
	Writer *gzip.Writer

	// The temporary storage location for the temporary file
	TempDirectory string
	// Where we want to move this file when we are finished
	OutputFile string
}

func NewGzippedFileWriter(tempDirectory, namePrefix, outFilename string) (*GzippedFileWriter, error) {

	f, err := ioutil.TempFile(tempDirectory, namePrefix)
	if err != nil {
		return nil, err
	}

	gzr := gzip.NewWriter(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	return &GzippedFileWriter{File: f, Writer: gzr, TempDirectory: tempDirectory, OutputFile:outFilename}
}

// Moves the temporary file into place
func (g* GzippedFileWriter) Finalize() error {

	// make sure the file is closed at this point
	g.File.Close()

	// rename the file

	if err := os.Rename(g.File.Name(), g.OutputFile); err != nil {
		return fmt.Errorf("Error while moving '%s' to '%s': %v", g.File.Name(), g.OutputFile, err)
	}
}

// Closes the gzipped writer and file, but does not move the temporary file into place
func (g *GzippedFileWriter) Close() error {
	// make sure we close the file even if we have flushing problems
	defer g.File.Close()

	// Close & flush the gzip writer
	if err := g.Writer.Close(); err != nil {
		return fmt.Errorf("Error while flushing gzip writer for '%s': %v", g.File.Name(), err)
	}

	//// make sure the file is closed at this point
	//g.File.Close()
	//
	//// rename the file
	//
	//if err := os.Rename(g.File.Name(), g.OutputFile); err != nil {
	//	return fmt.Errorf("Error while moving '%s' to '%s': %v", g.File.Name(), g.OutputFile, err)
	//}

	return nil
}

*/
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
	defer gzr.Close()

	log.Printf("Parsing: %s", gzr.File.Name())

	hostName := getHostName(filename)

	rows, errorRows, err := insight_server.ParseServerlogsWithFn(gzr.Reader, sourceTimezoneName, func(records []string) (string, string, string, error) {
		// the line is the second entry
		return hostName, "SERVERLOGS-REPARSED", records[1], nil
	})

	if err != nil {
		return fmt.Errorf("Error while parsing '%s': %v", filename, err)
	}
	gzr.Close()

	insight_server.WriteServerlogsCsv(tempdir, strings.Replace(filename, "errors_serverlogs", "serverlogs", -1), rows)

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
	if len(os.Args) != 2 {
		log.Printf("Usage: %s <OUTPUT DIRECTORY>", os.Args[0])
		return
	}

	outputPath := os.Args[1]
	config := insight_server.ParseOptions()

	log.Printf("Config: %v", config)

	// create the authenticator

	// All files we care about are in the base path

	errorFiles, err := checkTenants(&config)
	if err != nil {
		panic(err)
	}

	for _, errFile := range errorFiles {
		err := reloadServerlogs(outputPath, path.Join(config.UploadBasePath, "_temp"), errFile, "UTC")
		if err != nil {
			panic(err)
		}
	}
}
