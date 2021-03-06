package insight_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/palette-software/go-log-targets"
)

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
		"ts",
		"pid", "tid",
		"sev", "req", "sess", "site", "user",
		"k", "v", "elapsed_ms", "start_ts",
	}
}

// Returns the elapsed time, if the incoming string value is a JSON
// value and it contains an "elapsed" or an "elapsed-ms" key. The
// returned value is always given back in milliseconds.
//
// NOTE: "elapsed" key has its value in seconds, but "elapsed-ms"
// key has its in milliseconds.
//
// If the JSON value contains both keys, the value of the "elapsed"
// key is returned.
func getElapsed(key, line string) (int64, bool, error) {
	m := map[string]interface{}{}
	err := json.Unmarshal([]byte(line), &m)
	if err != nil {
		// If the incoming line is not a JSON, it does not
		// have elapsed key for sure. It is not an error.
		return 0, false, nil
	}
	if m["elapsed"] != nil {
		var floatValue float64
		value, ok := m["elapsed"].(string)
		if ok {
			floatValue, err = strconv.ParseFloat(value, 64)
			if err != nil {
				return 0, false, fmt.Errorf("Error while parsing elapsed value: '%v'. Error: %v", value, err)
			}
		} else {
			floatValue, ok = m["elapsed"].(float64)
			if !ok {
				return 0, false, fmt.Errorf("Can't parse elapsed to float64: '%v'", m["elapsed"])
			}
		}
		// Tableau is not consistent and logs ms instead of seconds with elapsed key for some keys
		switch key {
		case
			"read-foreign-keys",
			"read-statistics",
			"read-primary-keys":
			return int64(floatValue), true, nil
		}
		return int64(floatValue * 1000), true, nil
	}
	if m["elapsed-ms"] != nil {
		value, ok := m["elapsed-ms"].(string)
		if !ok {
			return 0, false, fmt.Errorf("Can't parse elapsed-ms to string '%v'", m["elapsed"])
		}
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false, err
		}
		return intValue, true, nil
	}
	// No elapsed or elapsed-ms in log line
	return 0, false, nil
}

// Returns the elapsed time, if the incoming string value is from a plaintext log file
// and it contains an "Elapsed time:x.xxxs" section. The
// returned value is given back in milliseconds.
func getElapsedFromPlainlogs(line string) (int64, error) {
	m := plainLineElapsedRegexp.FindStringSubmatch(line)
	if m == nil || len(m) < 2 {
		return 0, fmt.Errorf("No elapsed in log line.")
	}

	value, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("Can't parse elapsed time value to float")
	}
	return int64(value * 1000), nil
}

func getStartTime(end string, elapsed int64) string {
	end_ts, err := time.Parse(jsonDateFormat, end)
	if err != nil {
		log.Error("Unable to parse ts while calculating startTime.", err)
		return end
	}
	start_ts := end_ts.Add(-time.Duration(elapsed) * time.Millisecond)
	start := start_ts.Format(jsonDateFormat)
	if err != nil {
		log.Error("Unable to format start_ts while calculating it.", err)
		return end
	}
	return start
}

// parses a server log in JSON format
func (j *JsonLogParser) Parse(state ServerlogParserState, src *ServerlogsSource, line string, w ServerlogWriter) error {

	// try to parse the log row
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

	// ==================== JSON ====================

	// since the inner JSON can be anything, we unmarshal it into
	// a string, so the json marshaler can do his thing and we
	// dont have to care about what data is inside
	innerJsonStr, err := json.Marshal(outerJson.V)
	if err != nil {
		return fmt.Errorf("Inner JSON remarshaling error: %v", err)
	}

	unicodeUnescapeJsonBuffer := &bytes.Buffer{}
	// we need to do the unicode unescaping here in the inner JSON string
	// as '>' and  '<' appear frequently in their unicode escaped form
	if err := unescapeUnicodePoints(bytes.NewReader(innerJsonStr), unicodeUnescapeJsonBuffer); err != nil {
		return fmt.Errorf("Error during unicode unescape: %v", err)
	}

	// ==================== Elapsed ====================

	v := string(unicodeUnescapeJsonBuffer.Bytes())
	elapsedMs, hasElapsed, err := getElapsed(outerJson.K, v)

	// Log any errors we may have had
	if err != nil {
		// As the logging has been moved from getElapsed to
		// here, log the parse errors properly
		log.Errorf("Error parsing elapsed time file=%s host=%s k=%s v=%s err=%s", src.Filename, src.Host, outerJson.K, v, err)
	}

	// Get the elapsed values
	var elapsed, start_ts string

	// if we have a value, use it
	if hasElapsed {
		elapsed = strconv.FormatInt(elapsedMs, 10)
		start_ts = getStartTime(tsUtc, elapsedMs)
	} else {
		elapsed = "0"
		start_ts = tsUtc
	}

	// "ts"
	//"pid", "tid",
	//"sev", "req", "sess", "site", "user",
	//"k", "v", "elapsed_ms", "start_ts"
	w.WriteParsed(src, []string{
		outerJson.Ts,
		strconv.Itoa(outerJson.Pid), outerJson.Tid, // the tid is already a string
		outerJson.Sev, outerJson.Req, outerJson.Sess, outerJson.Site, outerJson.User,
		outerJson.K, v, elapsed, start_ts,
	})

	return nil

}
