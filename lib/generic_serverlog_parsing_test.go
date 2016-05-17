package insight_server

import (
	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// Fake writer, something that we can set expectations on.
type MockWriter struct {
	mock.Mock
}

// NOTE: This method is not being tested here, code that uses this object is.
func (m MockWriter) WriteParsed(source *ServerlogsSource, fields []string) error {
	args := m.Called(source, fields)
	return args.Error(0)
}
func (m MockWriter) Close() error {
	args := m.Called()
	return args.Error(0)
}
func (m MockWriter) WriteError(source *ServerlogsSource, err error, line string) error {
	args := m.Called(source, err, line)
	return args.Error(0)
}
func (m MockWriter) ParsedRowCount() int {
	args := m.Called()
	return args.Int(0)
}
func (m MockWriter) ErrorRowCount() int {
	args := m.Called()
	return args.Int(0)
}

func TestJsonParseElapsed_ShouldParseElapsedWhenAvailable(t *testing.T) {
	testLogLine := `{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query": "asd", "elapsed":0.039}}`
	tz, _ := time.LoadLocation("Europe/Berlin")
	src := ServerlogsSource{Timezone: tz}
	w := new(MockWriter)
	var p JsonLogParser
	w.On("WriteParsed", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fields := args.Get(1).([]string)
		tassert.Equal(t, "39", fields[10])
	})
	err := p.Parse(&src, testLogLine, *w)
	tassert.Nil(t, err)
}

func TestJsonParseElapsed_ShouldInsertNullWhenUnAvailable(t *testing.T) {
	testLogLine := `{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query": "asd"}}`
	tz, _ := time.LoadLocation("Europe/Berlin")
	src := ServerlogsSource{Timezone: tz}
	w := new(MockWriter)
	var p JsonLogParser
	w.On("WriteParsed", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fields := args.Get(1).([]string)
		tassert.Equal(t, "\\N", fields[10])
	})
	err := p.Parse(&src, testLogLine, w)
	tassert.Nil(t, err)
}
