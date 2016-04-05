package insight_server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"time"
)

type PlainServerlogsSource struct {
	Host     string
	Filename string
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

func parseServerlog(input *PlainServerlogsInput) ([]PlainServerlogsRow, []PlainServerlogsErrorRow, error) {

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
		tsParsed, err := time.Parse(plainServerlogsTimestampFormat, ts)
		if err != nil {
			appendToErrors(fmt.Errorf("Parsing timestamp '%s': %v", ts, err), newLine)
			continue
		}

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
