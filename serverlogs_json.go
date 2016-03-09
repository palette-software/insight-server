package insight_server

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The outer Json wrapper
type ServerlogOuterJson struct {
	// dont bother with the timestamp, keep to original format
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
	Json  string
	Error string
}

type ServerlogToParse struct {
	SourceFile, OutputFile, TmpDir string
}

// Creates a new parser that accepts filenames on the channel returned
func MakeServerlogParser(bufferSize int) chan ServerlogToParse {
	input := make(chan ServerlogToParse, bufferSize)
	go func() {
		for {
			serverlog := <-input
			err := parseServerlogFile(serverlog)
			if err != nil {
				// log the error but keep on spinning
				log.Printf("[serverlogs] ERROR: %s", err)
			}

		}
	}()
	return input
}

//////////////////////////////////

func parseServerlogFile(serverlog ServerlogToParse) error {

	filename := serverlog.SourceFile
	outputPath := serverlog.OutputFile

	// open the log file
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	serverlogs, errorRows, err := parseServerlogs(f)

	if err != nil {
		return err
	}

	log.Printf("[serverlogs] Parsed %d lines with %d error lines from '%s'", len(serverlogs), len(errorRows), filename)

	tmpDir := serverlog.TmpDir

	if len(serverlogs) > 0 {
		// make csv-compatible output
		serverlogRowsAsStr := make([][]string, len(serverlogs))
		for i, row := range serverlogs {
			o := &row.Outer
			serverlogRowsAsStr[i] = []string{
				row.Filename, row.Hostname, o.Ts, fmt.Sprint(o.Pid), o.Tid,
				o.Sev, o.Req, o.Sess, o.Site, o.User,
				o.K, row.Inner,
			}
		}
		outputFile, err := writeAsCsv(tmpDir, outputPath, "", serverlogsCsvHeader, serverlogRowsAsStr)
		if err != nil {
			return err
		}
		log.Printf("[serverlogs] written pre-parsed serverlogs to: '%s'", outputFile)
	}

	if len(errorRows) > 0 {
		// make csv-compatible output
		errorRowsAsStr := make([][]string, len(errorRows))
		for i, row := range errorRows {
			errorRowsAsStr[i] = []string{row.Error, row.Json}
		}
		// write it as csv
		errorsFile, err := writeAsCsv(tmpDir, outputPath, "errors_", []string{"error", "line"}, errorRowsAsStr)
		if err != nil {
			return err
		}
		log.Printf("[serverlogs] written pre-parsed serverlog error to: '%s'", errorsFile)
	}

	// after we are done, remove the original serverlogs file
	f.Close()
	log.Printf("[serverlogs] removing temporary '%s'", filename)
	return os.Remove(filename)
}

var serverlogsCsvHeader []string = []string{
	"filename", "host_name", "ts", "pid", "tid",
	"sev", "req", "sess", "site", "user",
	"k", "v",
}

func writeAsCsv(tmpDir, filename, prefix string, headers []string, rows [][]string) (string, error) {
	tmpFile, err := ioutil.TempFile(tmpDir, fmt.Sprintf("serverlogs-%s-output", prefix))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	csvWriter := makeCsvWriter(tmpFile)
	csvWriter.Write(headers)
	csvWriter.WriteAll(rows)

	// check for errors in flush
	err = csvWriter.Error()
	if err != nil {
		return "", err
	}

	outputPath := fmt.Sprintf("%s/%s%s", filepath.Dir(filename), prefix, filepath.Base(filename))

	// Get the temp file name before closing it
	tempFilePath := tmpFile.Name()

	// close the temp file, so writes get flushed
	tmpFile.Close()

	// move the output file to the new path with the new name
	err = os.Rename(tempFilePath, outputPath)
	if err != nil {
		return "", err
	}

	log.Printf("[csv] Moved '%v' to '%v'\n", tempFilePath, outputPath)
	return outputPath, nil
}

// Parses a serverlogs file by parsing the outer json level and re-marshaling
// the inner json back into a string so talend can do its own parsing later.
func parseServerlogs(r io.Reader) (rows []ServerlogOutputRow, errorRows []ErrorRow, err error) {

	csvReader := makeCsvReader(r)

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

		logRow := record[2]

		// try to parse the low row
		jsonDecoder := json.NewDecoder(strings.NewReader(logRow))
		outerJson := ServerlogOuterJson{}
		err = jsonDecoder.Decode(&outerJson)
		if err != nil {
			log.Println("[serverlogs.json] Parse error: ", err)
			// put this row into the problematic ones
			errorRows = append(errorRows, ErrorRow{
				Json:  logRow,
				Error: fmt.Sprintf("%v", err),
			})
			// skip this row from processing
			continue
		}

		outerJson.Tid, err = hexToDecimal(outerJson.Tid)

		// since the inner JSON can be anything, we unmarshal it into
		// a string, so the json marshaler can do his thing and we
		// dont have to care about what data is inside
		innerStr, err := json.Marshal(outerJson.V)
		if err != nil {
			log.Println("[serverlogs.json] Inner JSON remarshaling error: ", err)
			// put this row into the problematic ones
			errorRows = append(errorRows, ErrorRow{
				Json:  logRow,
				Error: fmt.Sprintf("%v", err),
			})
			// skip this row from processing
			continue
		}

		rows = append(rows, ServerlogOutputRow{
			Filename: record[0],
			Hostname: record[1],
			Outer:    outerJson,
			Inner:    string(innerStr),
		})
	}

}

///////////////////////////////////

func hexToDecimal(tidHexa string) (string, error) {
	decimal, err := strconv.ParseInt(tidHexa, 16, 32)
	decimalString := strconv.FormatInt(decimal, 10)
	return decimalString, err
}

func makeCsvReader(r io.Reader) *csv.Reader {
	reader := csv.NewReader(r)
	reader.Comma = '\v'
	reader.FieldsPerRecord = 3
	reader.LazyQuotes = true
	return reader
}

func makeCsvWriter(w io.Writer) *csv.Writer {
	writer := csv.NewWriter(w)
	writer.Comma = '\v'
	writer.UseCRLF = true
	return writer
}
