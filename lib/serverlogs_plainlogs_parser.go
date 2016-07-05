package insight_server

import (
	"fmt"
	"regexp"
	"strconv"
)

// Plain logs
// ----------

var (
	plainLineParserRegexp  = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{3}) \(([0-9]+)\): (.*)$`)
	plainLineElapsedRegexp = regexp.MustCompile(`^.*Elapsed time:(\d+\.\d+)s.*`)
)

const (
	plainServerlogsTimestampFormat = "2006-01-02 15:04:05.999"
	jsonDateFormat                 = "2006-01-02T15:04:05.999"
)

// The key of the pid header in the parser state

type PlainLogParser struct {
}

func MakePlainlogParser(dbDirectory string) (*PlainLogParser, error) {
	return &PlainLogParser{}, nil
}

// Headers for the plain serverlog files
func (p *PlainLogParser) Header() []string {
	return []string{
		"ts", "pid", "line", "elapsed_ms", "start_ts",
	}
}

// Parses a plaintext log line
func (p *PlainLogParser) Parse(state ServerlogParserState, src *ServerlogsSource, line string, w ServerlogWriter) error {

	// try to extract the timestamp
	matches := plainLineParserRegexp.FindAllStringSubmatch(line, -1)
	if len(matches) != 1 {
		return fmt.Errorf("Error in regex matching log line: got %d row instead of 1", len(matches))
	}

	// get the parts
	ts, pid, line := matches[0][1], matches[0][2], matches[0][3]

	// ==================== TS + PID ====================

	// parse the timestamp
	tsUtc, err := convertTimestringToUTC(plainServerlogsTimestampFormat, ts, src.Timezone)
	if err != nil {
		return fmt.Errorf("Error parsing log timestamp: %v", err)
	}

	// parse the pid (so we can check if is a valid number)
	if _, err := strconv.Atoi(pid); err != nil {
		return fmt.Errorf("Error parsing pid '%s': %v", pid, err)
	}

	// ==================== Elapsed ====================

	// Get the elapsed time
	elapsedMs, err := getElapsedFromPlainlogs(line)
	var elapsed, start_ts string
	if err == nil {
		elapsed = strconv.FormatInt(elapsedMs, 10)
		start_ts = getStartTime(tsUtc, elapsedMs)
	} else {
		elapsed = "0"
		start_ts = tsUtc
	}

	// ==================== Emitting the line ====================

	// Write the parsed line out (make sure its in the right order)
	w.WriteParsed(src, []string{tsUtc, pid, line, elapsed, start_ts})

	return nil
}
