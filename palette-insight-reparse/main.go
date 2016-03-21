package main

import (
	"compress/gzip"
	"fmt"
	"github.com/palette-software/insight-server"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Utility that tries to re-parse the serverlogs

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

func reloadServerlogs(filename string, sourceTimezone *time.Location) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	log.Printf("Parsing: %s", f.Name())

	csvReader := insight_server.MakeCsvReader(gzr)
	csvReader.FieldsPerRecord = 0
	records, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("Readall: %v", err)
	}

	for rowIdx, row := range records {
		//errormsg := row[0]
		line := row[1]

		outerJson, innerStr, err := insight_server.ParseOuterJson(insight_server.UnescapeGreenPlumCSV(line), sourceTimezone)
		if err != nil {
			log.Printf("Error during re-parse of line %d '%s': %v", rowIdx, line, err)
			continue
		}

		log.Printf("Outer: %v, inner: %s", outerJson, innerStr)

	}

	return nil
}

func main() {
	config := insight_server.ParseOptions()

	log.Printf("Config: %v", config)

	// create the authenticator

	// All files we care about are in the base path

	errorFiles, err := checkTenants(&config)
	if err != nil {
		panic(err)
	}

	for _, errFile := range errorFiles {
		log.Printf("File: %s", errFile)
		err := reloadServerlogs(errFile, time.UTC)
		if err != nil {
			panic(err)
		}
	}
}
