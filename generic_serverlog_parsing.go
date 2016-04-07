package insight_server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// Plain logs
// ----------

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
	tsParsed, err := time.ParseInLocation(plainServerlogsTimestampFormat, ts, src.Timezone)
	if err != nil {
		return fmt.Errorf("Parsing timestamp '%s': %v", ts, err)
	}

	//
	tsParsed = tsParsed.UTC()

	// parse the pid (so we can check if is a valid number)
	if _, err := strconv.Atoi(pid); err != nil {
		return fmt.Errorf("Parsing pid '%s': %v", pid, err)
	}

	// Write the parsed line out (make sure its in the right order)
	w.WriteParsed(src, []string{
		tsParsed.UTC().Format(jsonDateFormat),
		pid,
		line,
	})

	return nil
}

// JSON Logs
// ---------

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

	// Parse the timestamp with the proper time zone
	transcodedTs, err := time.ParseInLocation(jsonDateFormat, outerJson.Ts, src.Timezone)
	if err != nil {
		return fmt.Errorf("Timestamp parse error: %v", err)
	}

	// Convert the timestamp to utc
	outerJson.Ts = transcodedTs.UTC().Format(jsonDateFormat)

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
			log.Printf("[serverlogs] Got '%s' for parsing", meta.OriginalFilename)
			if err := processServerlogRequest(tmpDir, baseDir, archivesDir, serverLog, parserMap[serverLog.Format]); err != nil {
				log.Printf("[serverlogs] ERROR: during parsing of serverlog '%s': %v", meta.OriginalFilename, err)
			}
		}
	}()

	return inputChan
}

func processServerlogRequest(tmpDir, baseDir, archivesDir string, serverLog ServerlogInput, parser ServerlogsParser) error {
	//inputFn := serverLog.InputFilename
	meta := serverLog.Meta

	// The input file is in the archives folder
	inputFn := serverLog.ArchivedFile
	// if we have a nil parser that means the input format is not ok
	if parser == nil {
		return fmt.Errorf("Unknown input format for '%s'", inputFn)
	}

	//// try to parse the timezone name
	//sourceTimezone, err := time.LoadLocation(serverLog.Timezone)
	//if err != nil {
	//	return fmt.Errorf("Unknown time zone for agent  '%s': %v", serverLog.Timezone, err)
	//}

	//// copy the file to the archives
	//archivePath := filepath.Join(archivesDir, filepath.Base(serverLog.InputFilename))
	//if err := CopyFileRaw(serverLog.InputFilename, archivePath); err != nil {
	//	return fmt.Errorf("Error copying file '%s' to '%s': %v", serverLog.InputFilename, archivePath, err)
	//}

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

	log.Printf("[serverlogs] Done parsing for '%s': %d parsed, %d error rows", inputFn, logWriter.ParsedRowCount(), logWriter.ErrorRowCount())

	return nil
}
