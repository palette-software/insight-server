package insight_server

import (
	"fmt"
	"regexp"
)

type LogContinuation interface {

	// Returns the stored line to emit for a certain key.
	// Returns the line and a boolean indicating if there
	// was such a line in the DB
	HeaderLineFor(key string) (string, bool)

	// Save the header for a certain key
	SetHeaderFor(key, value string) error
}

// Helper that returns a continuation key
func MakeContinuationKey(tsUtc, pid string) string {
	return fmt.Sprintf("%s||%s", tsUtc, pid)
}

// ==================== Line type checks ====================

var lineContinuationRx = regexp.MustCompile("logfile_rotation: opening new log")
var lineWillHaveContinuationRx = regexp.MustCompile("logfile_rotation: closing this log")
var lineHasPidRx = regexp.MustCompile("^pid=([0-9]+)$")

// Returns true if the line string given looks like a continuation string
func IsLineContinuation(line string) bool {
	return lineContinuationRx.MatchString(line)
}

// Returns true if the line string given looks like a continuation string
func LineWillHaveContinuation(line string) bool {
	return lineWillHaveContinuationRx.MatchString(line)
}

// Returns true if the line is a PID-header line for plainlogs
func LineHasPid(line string) bool {
	return lineHasPidRx.MatchString(line)
}
