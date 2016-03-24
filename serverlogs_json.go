package insight_server

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The outer Json wrapper
type ServerlogOuterJson struct {
	Ts, Sev, Req, Sess, Site, User, K string
	V                                 interface{}
	Pid                               int
	Tid                               string
}

// The inner json wrapper
type ServerlogOutputRow struct {
	Filename, Hostname string

	Outer ServerlogOuterJson
	Inner string
}

type ErrorRow struct {
	Json               string
	Filename, Hostname string
	Error              string
}

type ServerlogToParse struct {
	SourceFile, OutputFile, TmpDir, Timezone string
}

// Creates a new parser that accepts filenames on the channel returned.
// Any passed files are stored in the directory pointed to by archivePath.
func MakeServerlogParser(bufferSize int, archivePath string) chan ServerlogToParse {
	input := make(chan ServerlogToParse, bufferSize)
	go func() {
		for {
			serverlog := <-input
			err := parseServerlogFile(archivePath, serverlog)
			if err != nil {
				// log the error but keep on spinning
				log.Printf("[serverlogs] ERROR: %s", err)
			}

		}
	}()
	return input
}

//////////////////////////////////

// The date format we'll use for creating the subfolders in the archives for the serverlogs
const archiveDirectoryDateFormatString = "2006-01-02"

// Moves a serverlog (gzipped) file to the archived folder
func moveServerlogsToArchives(archivePath, filename, outputPath string) error {

	// Move the original serverlogs to an archive folder
	// The host of the original file is the directory name of the output path.
	archiveHost := filepath.Base(filepath.Dir(outputPath))
	// The archive path structure is: archives/<DATE>/<HOST>/filename
	archiveOutputPath := filepath.Join(
		archivePath,
		time.Now().UTC().Format(archiveDirectoryDateFormatString),
		archiveHost,
		filepath.Base(outputPath),
	)

	// create the archive directory
	archiveFolderPath := filepath.Dir(archiveOutputPath)
	if err := CreateDirectoryIfNotExists(archiveFolderPath); err != nil {
		return fmt.Errorf("Error creating archives directory '%s': %v", archiveFolderPath, err)
	}

	if err := os.Rename(filename, archiveOutputPath); err != nil {
		return fmt.Errorf("Error while moving '%s' to '%s': %v", filename, archiveOutputPath, err)

	}

	log.Printf("[serverlogs] Moved uploaded serverlogs to archives as '%s'", archiveOutputPath)
	// try to move the file there
	return nil
}

func parseServerlogFile(archivePath string, serverlog ServerlogToParse) (errorOut error) {

	filename := serverlog.SourceFile
	outputPath := serverlog.OutputFile

	// open the log file as a file stream
	rawReader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer rawReader.Close()

	// Create a gzip reader on top
	gzipReader, err := gzip.NewReader(rawReader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	// If we have to exit this function, we have to move the serverlog file.
	// As defered function are executed in a LIFO order, we have to close
	// the underlying readers before moving the file.
	defer func() {
		// Close both readers before moving the file to the archives:
		// After we are done, remove the original serverlogs file and the gzip
		// reader on top.
		gzipReader.Close()
		rawReader.Close()
		// re-assign the output
		errorOut = moveServerlogsToArchives(archivePath, filename, outputPath)
	}()

	serverlogs, errorRows, err := ParseServerlogs(gzipReader, serverlog.Timezone)
	if err != nil {
		return err
	}

	log.Printf("[serverlogs] Parsed %d lines with %d error lines from '%s'", len(serverlogs), len(errorRows), filename)

	tmpDir := serverlog.TmpDir

	// Write normal output
	if err := WriteServerlogsCsv(tmpDir, outputPath, serverlogs); err != nil {
		return fmt.Errorf("Error writing serverlogs CSV: %v", err)
	}
	// Write error output
	if err := WriteServerlogErrorsCsv(tmpDir, outputPath, errorRows); err != nil {
		return fmt.Errorf("Error writing errors CSV: %v", err)
	}

	return nil
}

var serverlogsCsvHeader []string = []string{
	"filename", "host_name", "ts", "pid", "tid",
	"sev", "req", "sess", "site", "user",
	"k", "v",
}

// Writes out the serverlogs CSV file
func WriteServerlogsCsv(tmpDir, outputPath string, serverlogs []ServerlogOutputRow) error {

	// Write normal output
	if len(serverlogs) > 0 {
		// make csv-compatible output
		serverlogRowsAsStr := make([][]string, len(serverlogs))
		for i, row := range serverlogs {
			o := &row.Outer
			// re-escape the output
			serverlogRowsAsStr[i] = EscapeRowForGreenPlum([]string{
				row.Filename, row.Hostname, o.Ts, fmt.Sprint(o.Pid), o.Tid,
				o.Sev, o.Req, o.Sess, o.Site, o.User,
				o.K,
				row.Inner,
			})
		}
		outputFile, err := WriteAsCsv(tmpDir, outputPath, "", serverlogsCsvHeader, serverlogRowsAsStr)
		if err != nil {
			return err
		}
		log.Printf("[serverlogs] written pre-parsed serverlogs to: '%s'", outputFile)
	}
	return nil
}

func WriteServerlogErrorsCsv(tmpDir, outputPath string, errorRows []ErrorRow) error {

	if len(errorRows) > 0 {
		// make csv-compatible output
		errorRowsAsStr := make([][]string, len(errorRows))
		for i, row := range errorRows {
			errorRowsAsStr[i] = EscapeRowForGreenPlum([]string{
				row.Error, row.Hostname, row.Filename, EscapeGreenPlumCSV(row.Json),
			})
		}
		// write it as csv
		errorsFile, err := WriteAsCsv(tmpDir, outputPath, "errors_", []string{"error", "hostname", "filename", "line"}, errorRowsAsStr)
		if err != nil {
			return err
		}
		log.Printf("[serverlogs] written pre-parsed serverlog error to: '%s'", errorsFile)
	}
	return nil
}

// The regex we'll use to remove the md5 from the end of the file
var md5RemoverRegexp = regexp.MustCompile("-[a-f0-9]{32}.csv.gz$")

// Computes the md5 of a file by path
func computeMd5ForFile(filePath string) ([]byte, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return result, err
	}

	return hash.Sum(result), nil
}

// Tries to write a bunch of rows as a greenplum-like CSV file (this function does not escape strings)
func WriteAsCsv(tmpDir, filename, prefix string, headers []string, rows [][]string) (string, error) {
	// The temporary output file which we'll move to its destination
	tmpFile, err := ioutil.TempFile(tmpDir, fmt.Sprintf("serverlogs-%s-output", prefix))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// The gzipped writer on top
	gzipWriter := gzip.NewWriter(tmpFile)
	defer gzipWriter.Close()

	// Output the csv
	csvWriter := MakeCsvWriter(gzipWriter)
	csvWriter.Write(headers)
	csvWriter.WriteAll(rows)

	// check for errors in flush
	err = csvWriter.Error()
	if err != nil {
		return "", err
	}

	// Get the temp file name before closing it
	tempFilePath := tmpFile.Name()

	// Close the gzip stream then close the temp file, so writes get flushed.
	gzipWriter.Close()
	tmpFile.Close()

	// re-calculate the hash of the file, so we dont have conflicts
	outputMd5, err := computeMd5ForFile(tempFilePath)
	if err != nil {
		// save the file even if the md5 is crap
		log.Printf("[serverlogs] error while computing md5 of csv '%s': %v", tmpFile.Name(), err)
		// generate 32 bytes of bullshit as md5
		outputMd5 = RandStringBytesMaskImprSrc(32)
		log.Printf("[serverlogs] using '%s' instead of md5", string(outputMd5))
	}

	// generate the output path of the file
	outputPath := fmt.Sprintf("%s/%s%s-%32x.csv.gz",
		filepath.Dir(filename),
		prefix,
		// the filename without the md5 part
		md5RemoverRegexp.ReplaceAllString(filepath.Base(filename), ""),
		// the new md5
		outputMd5,
	)

	// move the output file to the new path with the new name
	err = os.Rename(tempFilePath, outputPath)
	if err != nil {
		return "", err
	}

	log.Printf("[csv] Moved '%v' to '%v'\n", tempFilePath, outputPath)
	return outputPath, nil
}

const jsonDateFormat = "2006-01-02T15:04:05.999"

// Tries to parse the outer JSON from a serverlog row
func ParseOuterJson(logRow string, sourceTimezone *time.Location) (*ServerlogOuterJson, []byte, error) {

	// try to parse the low row
	outerJson := ServerlogOuterJson{}
	err := json.NewDecoder(strings.NewReader(logRow)).Decode(&outerJson)
	if err != nil {
		return nil, nil, fmt.Errorf("JSON parse error: %v", err)
	}

	// convert the tid
	if outerJson.Tid, err = hexToDecimal(outerJson.Tid); err != nil {
		return nil, nil, fmt.Errorf("Tid Parse error: %v", err)
	}

	// Parse the timestamp with the proper time zone
	transcodedTs, err := time.ParseInLocation(jsonDateFormat, outerJson.Ts, sourceTimezone)
	if err != nil {
		return nil, nil, fmt.Errorf("Timestamp parse error: %v", err)
	}

	// Convert the timestamp to utc
	outerJson.Ts = transcodedTs.UTC().Format(jsonDateFormat)

	// since the inner JSON can be anything, we unmarshal it into
	// a string, so the json marshaler can do his thing and we
	// dont have to care about what data is inside
	innerStr, err := json.Marshal(outerJson.V)
	if err != nil {
		return nil, nil, fmt.Errorf("Inner JSON remarshaling error: %v", err)
	}

	return &outerJson, innerStr, nil
}

// A function type that takes a a list of strings and returns the hostname, the filename and the log row (or an error)
type RecordParserFunction func(record []string) (hostName, fileName, logRow string, err error)

// The default function for getting the hostname, filename and logrow from regular serverlog files
func NormalServerlogsParserFn(record []string) (hostName, fileName, logRow string, err error) {
	if len(record) != 3 {
		return "", "", "", fmt.Errorf("Not enough columns read: %d instead of 3 from: %v", len(record), record)
	}
	return record[1], record[0], record[2], nil
}

// Wrapper function for parsing a regular serverlog
func ParseServerlogs(r io.Reader, timezoneName string) (rows []ServerlogOutputRow, errorRows []ErrorRow, err error) {
	return ParseServerlogsWithFn(r, timezoneName, NormalServerlogsParserFn)
}

// Parses a serverlogs file by parsing the outer json level and re-marshaling
// the inner json back into a string so talend can do its own parsing later.
func ParseServerlogsWithFn(r io.Reader, timezoneName string, parserFn RecordParserFunction) (rows []ServerlogOutputRow, errorRows []ErrorRow, err error) {

	// try to parse the timezone name
	sourceTimezone, err := time.LoadLocation(timezoneName)
	if err != nil {
		return nil, nil, err
	}

	csvReader := MakeCsvReader(r)

	isHeader := true

	for {
		record, err := csvReader.Read()
		// in case of EOF we have finished
		if err == io.EOF {
			return rows, errorRows, nil
		}
		// if the CSV has errors, skip the whole file as we dont know
		// how to parse it
		if err != nil {
			return nil, nil, fmt.Errorf("Error during CSV parsing: %v", err)
		}
		// skip the header row
		if isHeader {
			isHeader = false
			continue
		}

		// try to get the elements from the record
		hostName, fileName, logRow, err := parserFn(record)
		if err != nil {
			log.Println("[serverlogs.json] Error while parsing serverlog row: %v ", err)
			// put this row into the problematic ones
			errorRows = append(errorRows, ErrorRow{
				Json:     logRow,
				Hostname: hostName,
				Filename: fileName,
				Error:    fmt.Sprintf("%v", err),
			})
			// skip this row from processing
			continue
		}

		outerJson, innerStr, err := ParseOuterJson(UnescapeGreenPlumCSV(logRow), sourceTimezone)
		if err != nil {

			log.Println("[serverlogs.json] Error while parsing serverlog row: %v ", err)
			// put this row into the problematic ones
			errorRows = append(errorRows, ErrorRow{
				Json:     logRow,
				Hostname: hostName,
				Filename: fileName,
				Error:    fmt.Sprintf("%v", err),
			})
			// skip this row from processing
			continue
		}

		rows = append(rows, ServerlogOutputRow{
			Filename: fileName, // record[0],
			Hostname: hostName, // record[1],
			Outer:    *outerJson,
			Inner:    string(innerStr),
		})
	}

}
