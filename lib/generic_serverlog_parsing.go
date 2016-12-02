package insight_server

import (
	"fmt"
	"io"
	"path/filepath"
	"time"

	log "github.com/palette-software/insight-tester/common/logging"
)

// Identifies the source of a serverlog
type ServerlogsSource struct {
	Host     string
	Filename string
	Timezone *time.Location
}

// ==================== Serverlog Parser State ====================

// The state for files (this state gets passed to
// each call of ServerlogParser.Parse(), and is persistent
// for a file)

type ServerlogParserState interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
}

type baseServerlogParserState struct {
	data map[string][]byte
}

// Creates a new state for the parser
func MakeServerlogParserState() ServerlogParserState {
	return &baseServerlogParserState{
		data: map[string][]byte{},
	}
}

func (p *baseServerlogParserState) Get(key string) ([]byte, bool) {
	v, hasValue := p.data[key]
	return v, hasValue
}

func (p *baseServerlogParserState) Set(key string, value []byte) { p.data[key] = value }

// ==================== Serverlog Parser ====================

// Reads serverlogs (the implementation determines the format)
type ServerlogsParser interface {
	// Gets the header for this parser
	Header() []string
	// Parses the lines in the reader
	Parse(state ServerlogParserState, src *ServerlogsSource, line string, w ServerlogWriter) error
}

// A generic log parser that takes a reader and a timezone
func ParseServerlogsWith(r io.Reader, parser ServerlogsParser, w ServerlogWriter, tz *time.Location) error {

	// All serverlogs arrive in CSV format with the source info in the first line
	csvReader := MakeCsvReader(r)

	isHeader := true
	parserState := MakeServerlogParserState()

	for {
		record, err := csvReader.Read()
		// in case of EOF we have finished
		if err == io.EOF {
			return nil
		}

		// if the CSV has errors, skip the whole file as we dont know
		// how to parse it
		if err != nil {
			return fmt.Errorf("Error during CSV parsing: %v", err)
		}

		// skip the header row
		if isHeader {
			isHeader = false
			continue
		}

		// if not enough fields in this row, signal it
		if len(record) != 3 {
			w.WriteError(&ServerlogsSource{}, fmt.Errorf("Not enough columns read: %d instead of 3 from: %v", len(record), record), "")
			continue
		}

		// split the row
		fileName, hostName, logRow := record[0], record[1], record[2]

		// handle the case where src isnt yet set up
		rowSrc := &ServerlogsSource{
			Host:     hostName,
			Filename: fileName,
			Timezone: tz,
		}

		// try to un-escape the csv
		unescapedRow, err := UnescapeGPCsvString(logRow)
		if err != nil {
			w.WriteError(rowSrc, fmt.Errorf("[serverlogs.json] Error while unescaping serverlog row: %v ", err), logRow)
			// skip this row from processing
			continue
		}

		// if the line is empty, dont try to parse it
		if len(unescapedRow) == 0 {
			continue
		}

		// try to parse it and log errors
		if err := parser.Parse(parserState, rowSrc, unescapedRow, w); err != nil {
			w.WriteError(rowSrc, err, unescapedRow)
			continue
		}

	}

	return fmt.Errorf("Unreachable code reached")
}

// Shared helpers
// --------------

// Tries to parse a timestamp in the given timezone using the provided format.
// Returns the parsed timestamp in a JSON timestamp format.
func convertTimestringToUTC(format, timeString string, tz *time.Location) (string, error) {

	// parse the timestamp
	tsParsed, err := time.ParseInLocation(format, timeString, tz)
	if err != nil {
		return "", fmt.Errorf("Parsing timestamp '%s' with format '%s': %v", timeString, format, err)
	}

	// unify the output format
	return tsParsed.UTC().Format(jsonDateFormat), nil
}

// Checker channel
// ---------------

type LogFormat int

const (
	LogFormatJson  = LogFormat(0)
	LogFormatPlain = LogFormat(1)
)

type ServerlogInput struct {
	// the upload metadata
	Meta *UploadMeta

	// The actual path in the archives
	ArchivedFile string

	// The format of these logs
	Format LogFormat
}

func MakeServerlogsParser(tmpDir, baseDir, archivesDir string, bufferSize int) (chan ServerlogInput, error) {
	plainlogParser, err := MakePlainlogParser(tmpDir)

	if err != nil {
		return nil, fmt.Errorf("Error creating plainlog parser: %v", err)
	}

	parserMap := map[LogFormat]ServerlogsParser{
		LogFormatJson:  &JsonLogParser{},
		LogFormatPlain: plainlogParser,
	}

	inputChan := make(chan ServerlogInput, bufferSize)
	go func() {
		for serverLog := range inputChan {
			meta := serverLog.Meta
			log.Infof("Received parse request: host=%s file=%s", meta.Host, meta.OriginalFilename)
			if err := processServerlogRequest(tmpDir, baseDir, archivesDir, serverLog, parserMap[serverLog.Format]); err != nil {
				log.Errorf("Error during parsing of serverlog host=%s file=%s", meta.Host, meta.OriginalFilename)
			}
		}
	}()

	return inputChan, nil
}

func processServerlogRequest(tmpDir, baseDir, archivesDir string, serverLog ServerlogInput, parser ServerlogsParser) error {
	meta := serverLog.Meta

	// The input file is in the archives folder
	inputFn := serverLog.ArchivedFile
	// if we have a nil parser that means the input format is not ok
	if parser == nil {
		return fmt.Errorf("Unknown input format for '%s'", inputFn)
	}

	// open the file we have been sent as a gzipped file
	inputF, err := NewGzippedFileReader(inputFn)
	if err != nil {
		return fmt.Errorf("Error opening serverlog file '%s' for parsing: %v", inputFn, err)
	}
	defer inputF.Close()

	// find out where we are planning to output the parsed data
	targetFile := meta.GetOutputFilename(baseDir)

	// create the log writer
	logWriter := NewServerlogsWriter(
		filepath.Dir(targetFile),
		tmpDir,
		filepath.Base(targetFile),
		parser.Header(),
	)
	defer logWriter.Close()

	// try to parse the logs using this parser
	if err := ParseServerlogsWith(inputF, parser, logWriter, meta.Timezone); err != nil {
		return fmt.Errorf("Error during parsing serverlog file '%s': %v", inputFn, err)
	}

	log.Infof("Done parsing host=%s file=%s count=%d errorCount=%d", meta.Host,
		meta.OriginalFilename, logWriter.ParsedRowCount(), logWriter.ErrorRowCount())

	return nil
}
