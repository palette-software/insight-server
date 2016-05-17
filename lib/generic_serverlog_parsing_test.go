package insight_server

import (
	"fmt"
	tassert "github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetElapsed(t *testing.T) {
	testValue := `{"elapsed":0.215}`
	elapsedTime, err := getElapsed(testValue)
	tassert.Nil(t, err)
	tassert.Equal(t, int64(215), elapsedTime)
}

func TestGetElapsed_WithIgnoredValues(t *testing.T) {
	testValue := `{"just":"ignore", "elapsed":0.215, "this":66}`
	elapsedTime, err := getElapsed(testValue)
	tassert.Nil(t, err)
	tassert.Equal(t, int64(215), elapsedTime)
}

func TestGetElapsed_MissingElapsed(t *testing.T) {
	testValue := `{"just":"ignore", "this":66}`
	_, err := getElapsed(testValue)
	tassert.NotNil(t, err)
}

func TestGetElapsed_InvalidElapsed(t *testing.T) {
	testValue := `{"just":"ignore", "elapsed":"should_not_be_string", "this":66}`
	_, err := getElapsed(testValue)
	tassert.NotNil(t, err)
}

func TestGetElapsedMs(t *testing.T) {
	testValue := `{"elapsed-ms":44}`
	elapsedTime, err := getElapsed(testValue)
	tassert.Nil(t, err)
	tassert.Equal(t, int64(44), elapsedTime)
}

func TestGetElapsedMs_IgnoreFractionOfMilliseconds(t *testing.T) {
	testValue := `{"elapsed-ms":23.215}`
	elapsedTime, err := getElapsed(testValue)
	tassert.Nil(t, err)
	tassert.Equal(t, int64(23), elapsedTime)
}

func TestGetElapsedMs_HandleInvalidValue(t *testing.T) {
	testValue := `{"elapsed-ms":true}`
	_, err := getElapsed(testValue)
	tassert.NotNil(t, err)
}

func TestGetElapsed_BothElapsedAndElapsedMs(t *testing.T) {
	testValue := `{"just":"ignore", "elapsed":4.876, "elapsed-ms":3, "this":66}`
	elapsedTime, err := getElapsed(testValue)
	tassert.Nil(t, err)
	tassert.Equal(t, int64(4876), elapsedTime, "In such situations we currently expect 'elapsed' to win.")
}

type TestFunc func(fields []string)
type DummyServerlogWriter struct {
	tests TestFunc
}

func (w DummyServerlogWriter) WriteParsed(source *ServerlogsSource, fields []string) error {
	w.tests(fields)
	return nil
}

func (w DummyServerlogWriter) WriteError(source *ServerlogsSource, err error, line string) error {
	return nil
}

func (w DummyServerlogWriter) ParsedRowCount() int {
	return 0
}

func (w DummyServerlogWriter) ErrorRowCount() int {
	return 0
}

func (w DummyServerlogWriter) Close() error {
	return nil
}

func TestJsonParseElapsed_ShouldParseElapsedWhenAvailable(t *testing.T) {
	testLogLine := `{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query": "asd", "elapsed":0.034}}`
	tz, _ := time.LoadLocation("Europe/Berlin")
	src := ServerlogsSource{Timezone: tz}
	actualTests := func(fields []string) {
		tassert.NotEqual(t, "\\N", fields[10])
	}
	w := DummyServerlogWriter{tests: actualTests}
	var p JsonLogParser
	err := p.Parse(&src, testLogLine, w)
	tassert.Nil(t, err)
}

func TestJsonParseElapsed_ShouldInsertNullWhenUnAvailable(t *testing.T) {
	testLogLine := `{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query": "asd"}}`
	tz, _ := time.LoadLocation("Europe/Berlin")
	src := ServerlogsSource{Timezone: tz}
	actualTests := func(fields []string) {
		tassert.Equal(t, "\\N", fields[10])
	}
	w := DummyServerlogWriter{tests: actualTests}
	var p JsonLogParser
	err := p.Parse(&src, testLogLine, w)
	tassert.Nil(t, err)
}
