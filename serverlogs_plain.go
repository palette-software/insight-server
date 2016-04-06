package insight_server

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"
)

type PlainServerlogsSource struct {
	Host     string
	Filename string
	Timezone string
}

type PlainServerlogsInput struct {
	// Location data
	Source PlainServerlogsSource

	Reader io.Reader
}

type PlainServerlogsErrorRow struct {
	Source PlainServerlogsSource
	Line   string
	Error  error
}

type PlainServerlogsRow struct {
	Source PlainServerlogsSource
	Pid    int

	AtUTC time.Time
	// the log line itself
	Line string
}

var plainLineParserRegexp = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{3}) \(([0-9]+)\): (.*)$`)

//2016-04-02 23:57:44.216
const plainServerlogsTimestampFormat = "2006-01-02 15:04:05.999"

func parsePlainServerlog(input *PlainServerlogsInput) ([]PlainServerlogsRow, []PlainServerlogsErrorRow, error) {
	// try to parse the timezone name
	sourceTimezone, err := time.LoadLocation(input.Source.Timezone)
	if err != nil {
		return nil, nil, err
	}

	// create a buffered reader for ReadLine
	bufReader := bufio.NewReader(input.Reader)

	// the errors and the successful rows
	errors := []PlainServerlogsErrorRow{}
	parsed := []PlainServerlogsRow{}

	// helper that appends to the errors list
	appendToErrors := func(err error, line string) {
		errors = append(errors, PlainServerlogsErrorRow{
			Source: input.Source,
			Line:   line,
			Error:  err,
		})
	}

	// Read line-by-line
	for {
		newLineBytes, isPrefix, err := bufReader.ReadLine()

		// if its a prefix, we are out of memory
		// TODO: fix this by concatenating here
		if isPrefix {
			log.Printf("ERROR: too long line from %s in %s", input.Source.Host, input.Source.Filename)
		}

		// if we have reached the end, break the loop
		if err == io.EOF {
			return parsed, errors, nil
		}

		// Add the rows to the parse errors list if we cannot read it,
		// so skip this line
		if err != nil {
			appendToErrors(err, "")
			return parsed, errors, err
		}

		// make a string out of it
		newLine := string(newLineBytes)

		matches := plainLineParserRegexp.FindAllStringSubmatch(newLine, -1)
		if len(matches) != 1 {
			appendToErrors(fmt.Errorf("Error in regex matching log line: %v", err), newLine)
			continue
		}

		// get the parts
		ts, pid, line := matches[0][1], matches[0][2], matches[0][3]

		// parse the timestamp
		tsParsed, err := time.ParseInLocation(plainServerlogsTimestampFormat, ts, sourceTimezone)
		if err != nil {
			appendToErrors(fmt.Errorf("Parsing timestamp '%s': %v", ts, err), newLine)
			continue
		}

		//
		tsParsed = tsParsed.UTC()

		// parse the pid
		pidParsed, err := strconv.Atoi(pid)
		if err != nil {
			appendToErrors(fmt.Errorf("Parsing pid '%s': %v", pid, err), newLine)
			continue
		}

		// append to the successful ones
		parsed = append(parsed, PlainServerlogsRow{
			Source: input.Source,
			AtUTC:  tsParsed,
			Line:   line,
			Pid:    pidParsed,
		})

	}

	return parsed, errors, fmt.Errorf("Unreachable code reached")
}

// Creates a new parser that accepts filenames on the channel returned.
// Any passed files are stored in the directory pointed to by archivePath.
func MakePlainServerlogParser(bufferSize int, archivePath string) chan ServerlogToParse {
	input := make(chan ServerlogToParse, bufferSize)
	log.Printf("[serverlogs.plain] Using %d buffer slots on input channel", bufferSize)
	go func() {
		for {
			// Read a file for parsing
			serverlog := <-input

			// Try to parse it
			if err := parsePlainServerlogFile(archivePath, serverlog); err != nil {
				// log the error but keep on spinning
				log.Printf("[serverlogs.plain] Error during parsing of '%s': %v", serverlog.OutputFile, err)
			}

			// Move to the archives after parsing
			if err := moveServerlogsToArchives(archivePath, serverlog.SourceFile, serverlog.OutputFile); err != nil {
				log.Printf("[serverlogs.plain] Error during moving '%s' to archives: %v", serverlog.SourceFile, err)
			}

		}
	}()
	return input
}

// TODO: add timezone support
func parsePlainServerlogFile(archivePath string, serverlog ServerlogToParse) (errorOut error) {

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

	logInput := &PlainServerlogsInput{
		Source: PlainServerlogsSource{
			Host:     serverlog.Host,
			Filename: serverlog.OriginalFileName,
		},
		Reader: gzipReader,
	}

	serverlogs, errorRows, err := parsePlainServerlog(logInput)
	if err != nil {
		return err
	}

	log.Printf("[serverlogs.json] Parsed %d lines with %d error lines from '%s'", len(serverlogs), len(errorRows), filename)

	tmpDir := serverlog.TmpDir

	// Write normal output
	if err := WritePlainServerlogsCsv(tmpDir, outputPath, serverlogs); err != nil {
		return fmt.Errorf("Error writing serverlogs CSV: %v", err)
	}
	// Write error output
	if err := WritePlainServerlogErrorsCsv(tmpDir, outputPath, errorRows); err != nil {
		return fmt.Errorf("Error writing errors CSV: %v", err)
	}

	return nil
}

//

// Writes out the plain serverlogs CSV file
func WritePlainServerlogsCsv(tmpDir, outputPath string, serverlogs []PlainServerlogsRow) error {
	if len(serverlogs) == 0 {
		return nil
	}

	// Write normal output
	// make csv-compatible output
	serverlogRowsAsStr := make([][]string, len(serverlogs))
	for i, row := range serverlogs {
		serverlogRowsAsStr[i] = []string{
			row.Source.Host, row.Source.Filename,
			row.AtUTC.Format(jsonDateFormat),
			fmt.Sprint(row.Pid),
			row.Line,
		}
	}
	outputFile, err := WriteAsCsv(tmpDir, outputPath, "", []string{
		"hostname", "filename", "ts", "pid", "line",
	}, serverlogRowsAsStr)

	if err != nil {
		return err
	}
	log.Printf("[serverlogs.plain] written pre-parsed serverlogs to: '%s'", outputFile)
	return nil
}

// Writes out the serverlog parse errors file
func WritePlainServerlogErrorsCsv(tmpDir, outputPath string, errorRows []PlainServerlogsErrorRow) error {

	// skip empty tables
	if len(errorRows) == 0 {
		return nil
	}

	// make csv-compatible output
	errorRowsAsStr := make([][]string, len(errorRows))
	for i, row := range errorRows {
		errorRowsAsStr[i] = []string{
			fmt.Sprint(row.Error), row.Source.Host, row.Source.Filename, row.Line,
		}
	}
	// write it as csv
	errorsFile, err := WriteAsCsv(tmpDir, outputPath, "errors_", []string{
		"error", "hostname", "filename", "line",
	}, errorRowsAsStr)

	if err != nil {
		return err
	}
	log.Printf("[serverlogs.plain] written pre-parsed serverlog error to: '%s'", errorsFile)
	return nil
}
