package insight_server

import (
	"testing"
	tassert "github.com/stretchr/testify/assert"
)

func TestGetElapsed(t *testing.T) {
	testValue := `{"elapsed":0.215}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Equal(t, int64(215), *elapsedTimePtr)
}

func TestGetElapsed_WithIgnoredValues(t *testing.T) {
	testValue := `{"just":"ignore", "elapsed":0.215, "this":66}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Equal(t, int64(215), *elapsedTimePtr)
}

func TestGetElapsed_MissingElapsed(t *testing.T) {
	testValue := `{"just":"ignore", "this":66}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Nil(t, elapsedTimePtr)
}

func TestGetElapsed_InvalidElapsed(t *testing.T) {
	testValue := `{"just":"ignore", "elapsed":"should_not_be_string", "this":66}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Nil(t, elapsedTimePtr)
}

func TestGetElapsedMs(t *testing.T) {
	testValue := `{"elapsed-ms":44}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Equal(t, int64(44), *elapsedTimePtr)
}

func TestGetElapsedMs_IgnoreFractionOfMilliseconds(t *testing.T) {
	testValue := `{"elapsed-ms":23.215}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Equal(t, int64(23), *elapsedTimePtr)
}

func TestGetElapsedMs_HandleInvalidValue(t *testing.T) {
	testValue := `{"elapsed-ms":true}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Nil(t, elapsedTimePtr)
}

func TestGetElapsed_BothElapsedAndElapsedMs(t *testing.T) {
	testValue := `{"just":"ignore", "elapsed":4.876, "elapsed-ms":3, "this":66}`
	elapsedTimePtr := getElapsed(testValue)
	tassert.Equal(t, int64(4876), *elapsedTimePtr, "In such situations we currently expect 'elapsed' to win.")
}
