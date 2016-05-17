package insight_server

import (
	tassert "github.com/stretchr/testify/assert"
	"testing"
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
