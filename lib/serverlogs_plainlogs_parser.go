package insight_server

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/Sirupsen/logrus"
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

	pidHeaderKey = "pid-header"
)

// The key of the pid header in the parser state

type PlainLogParser struct {
	// The DB housing our continuations
	ContinuationDb LogContinuation
}

func MakePlainlogParser(dbDirectory string) (*PlainLogParser, error) {
	logDb, err := MakeBoltDbLogContinuationDb(dbDirectory)
	if err != nil {
		return nil, fmt.Errorf("Cannot open log continuation db: '%v'", err)
	}

	return &PlainLogParser{
		ContinuationDb: logDb,
	}, nil
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
		return fmt.Errorf("Parsing log timestamp: %v", err)
	}

	// parse the pid (so we can check if is a valid number)
	if _, err := strconv.Atoi(pid); err != nil {
		return fmt.Errorf("Parsing pid '%s': %v", pid, err)
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

	// ==================== Continuations ====================

	switch {
	// check if this line looks like a continuation
	case IsLineContinuation(line):
		// Create the continuation key from the line contents
		continuationKey := MakeContinuationKey(src.Host, tsUtc, pid)

		//check if we have a header for this continuation key
		pidHeader, hasPidHeader, err := p.ContinuationDb.HeaderLineFor(continuationKey)

		// handle errors
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"component": "serverlogs",
				"key":       string(continuationKey),
			}).Error("Error getting PID header from LogContinuationDb")
		}

		// handle good things (we have the pid header)
		if hasPidHeader {
			// if we have, emit it before the current line, and
			// use pidHeader instead of line
			w.WriteParsed(src, []string{tsUtc, pid, string(pidHeader), elapsed, start_ts})
			// Store the looked up pid header for the file
			// (so we can save it when the log file is rotated
			state.Set(pidHeaderKey, pidHeader)
			logrus.WithFields(logrus.Fields{
				"component": "serverlogs",
				"line":      line,
				"pidHeader": pidHeader,
			}).Info("Emitted pid header for continuation, updated continuation pid in state")
		}

	// Check if this line looks like a log file end
	case LineWillHaveContinuation(line):
		continuationKey := MakeContinuationKey(src.Host, tsUtc, pid)

		// Try to get the pid header from the state
		if pidHeader, hasPidHeader := state.Get(pidHeaderKey); hasPidHeader {
			// if we have the pid header, store it in the DB for this
			// continuation key
			p.ContinuationDb.SetHeaderFor(continuationKey, pidHeader)

			logrus.WithFields(logrus.Fields{
				"component":       "serverlogs",
				"line":            line,
				"continuationKey": continuationKey,
				"pidHeader":       pidHeader,
			}).Info("Storing continuation pid in db")
		}

	// Check if this line is a pid header
	case LineHasPid(line):
		// store the current line as the pid header in this case
		state.Set(pidHeaderKey, []byte(line))

		logrus.WithFields(logrus.Fields{
			"component": "serverlogs",
			"line":      line,
		}).Info("Saving pid header in state")
	}

	// ==================== Emitting the line ====================

	// Write the parsed line out (make sure its in the right order)
	w.WriteParsed(src, []string{tsUtc, pid, line, elapsed, start_ts})

	return nil
}
