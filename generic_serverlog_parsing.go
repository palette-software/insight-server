package insight_server

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
)

// Identifies the source of a serverlog
type ServerlogsSource struct {
	Host     string
	Filename string
	Timezone *time.Location
}

// Reads serverlogs (the implementation determines the format)
type ServerlogsParser interface {
	// Gets the header for this parser
	Header() []string
	// Parses the lines in the reader
	Parse(src *ServerlogsSource, line string, w ServerlogWriter) error
}

// A generic log parser that takes a reader and a timezone
func ParseServerlogsWith(r io.Reader, parser ServerlogsParser, w ServerlogWriter, tz *time.Location) error {

	// All serverlogs arrive in CSV format with the source info in the first line
	csvReader := MakeCsvReader(r)

	isHeader := true

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

		// try to parse it and log errors
		if err := parser.Parse(rowSrc, unescapedRow, w); err != nil {
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

// Plain logs
// ----------

var plainLineParserRegexp = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{3}) \(([0-9]+)\): (.*)$`)

const plainServerlogsTimestampFormat = "2006-01-02 15:04:05.999"

type PlainLogParser struct {
}

// Headers for the plain serverlog files
func (p *PlainLogParser) Header() []string {
	return []string{
		"ts", "pid", "line",
	}
}

// Parses a plaintext log line
func (p *PlainLogParser) Parse(src *ServerlogsSource, line string, w ServerlogWriter) error {

	matches := plainLineParserRegexp.FindAllStringSubmatch(line, -1)
	if len(matches) != 1 {
		return fmt.Errorf("Error in regex matching log line: got %d row instead of 1", len(matches))
	}

	// get the parts
	ts, pid, line := matches[0][1], matches[0][2], matches[0][3]

	// parse the timestamp
	tsUtc, err := convertTimestringToUTC(plainServerlogsTimestampFormat, ts, src.Timezone)
	if err != nil {
		return fmt.Errorf("Parsing log timestamp: %v", err)
	}

	// parse the pid (so we can check if is a valid number)
	if _, err := strconv.Atoi(pid); err != nil {
		return fmt.Errorf("Parsing pid '%s': %v", pid, err)
	}

	// Write the parsed line out (make sure its in the right order)
	w.WriteParsed(src, []string{
		tsUtc,
		pid,
		line,
	})

	return nil
}

// JSON Logs
// ---------

// The outer Json wrapper
type ServerlogOuterJson struct {
	Ts, Sev, Req, Sess, Site, User, K string
	V                                 interface{}
	Pid                               int
	Tid                               string
}

type JsonLogParser struct{}

func (j *JsonLogParser) Header() []string {
	return []string{
		"pid", "tid",
		"sev", "req", "sess", "site", "user",
		"k", "v",
	}
}

// parses a server log in JSON format
func (j *JsonLogParser) Parse(src *ServerlogsSource, line string, w ServerlogWriter) error {

	// try to parse the low row
	outerJson := ServerlogOuterJson{}
	err := json.NewDecoder(strings.NewReader(line)).Decode(&outerJson)
	if err != nil {
		return fmt.Errorf("JSON parse error in '%s': %v", line, err)
	}

	// convert the tid
	if outerJson.Tid, err = hexToDecimal(outerJson.Tid); err != nil {
		return fmt.Errorf("Tid Parse error: %v", err)
	}

	tsUtc, err := convertTimestringToUTC(jsonDateFormat, outerJson.Ts, src.Timezone)
	if err != nil {
		return fmt.Errorf("Parsing log timestamp: %v", err)
	}

	// Re-assign the converted timestamp
	outerJson.Ts = tsUtc

	// since the inner JSON can be anything, we unmarshal it into
	// a string, so the json marshaler can do his thing and we
	// dont have to care about what data is inside
	innerStr, err := json.Marshal(outerJson.V)
	if err != nil {
		return fmt.Errorf("Inner JSON remarshaling error: %v", err)
	}

	//"pid", "tid",
	//"sev", "req", "sess", "site", "user",
	//"k", "v",
	w.WriteParsed(src, []string{
		strconv.Itoa(outerJson.Pid), outerJson.Tid, // the tid is already a string
		outerJson.Sev, outerJson.Req, outerJson.Sess, outerJson.Site, outerJson.User,
		outerJson.K, string(innerStr),
	})

	return nil

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

func MakeServerlogsParser(tmpDir, baseDir, archivesDir string, bufferSize int) chan ServerlogInput {
	parserMap := map[LogFormat]ServerlogsParser{
		LogFormatJson:  &JsonLogParser{},
		LogFormatPlain: &PlainLogParser{},
	}

	inputChan := make(chan ServerlogInput, bufferSize)
	go func() {
		for serverLog := range inputChan {
			meta := serverLog.Meta
			logrus.WithFields(logrus.Fields{
				"component": "serverlogs",
				"file":      meta.OriginalFilename,
			}).Info("Received parse request")
			if err := processServerlogRequest(tmpDir, baseDir, archivesDir, serverLog, parserMap[serverLog.Format]); err != nil {
				logrus.WithFields(logrus.Fields{
					"component": "serverlogs",
					"file":      meta.OriginalFilename,
				}).WithError(err).Error("Error during parsing of serverlog")
			}
		}
	}()

	return inputChan
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

	logrus.WithFields(logrus.Fields{
		"component":  "serverlogs",
		"file":       meta.OriginalFilename,
		"parsedRows": logWriter.ParsedRowCount(),
		"errorRows":  logWriter.ErrorRowCount(),
	}).Info("Done parsing")

	return nil
}
