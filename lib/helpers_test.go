package insight_server

import (
	"bytes"
	"strings"
	"testing"
)

func assert(t *testing.T, val bool, msg string) {
	if !val {
		t.Fatalf("%s", msg)
	}
}

func assertInt(t *testing.T, a, b int, msg string) {
	if a != b {
		t.Fatalf("%s: %v vs %v", msg, a, b)
	}
}

func assertString(t *testing.T, a, b string, msg string) {
	if a != b {
		t.Fatalf("%s:\n\tExpected:'%s'\n\tActual:  '%s'", msg, a, b)
	}
}

func TestSanitizeName(t *testing.T) {
	assertString(t, SanitizeName("asd"), "asd", "Regular version escaped badly")
	assertString(t, SanitizeName("a_sd"), "a_sd", "Underscored version escaped badly")
	assertString(t, SanitizeName("a_s d"), "a_s-d", "Underscored-spaced version escaped badly")
	assertString(t, SanitizeName("a_s d*&:%"), "a_s-d----", "Underscored-spaced and misc characters version escaped badly")
}

func TestUnicodeUnescape(t *testing.T) {
	inString := `(\"CalQtrOffset\" \u003c= 0) AND (\"CalQtrOffset\" \u003e= -4)`

	outStr := &bytes.Buffer{}

	if err := unescapeUnicodePoints(strings.NewReader(inString), outStr); err != nil {
		panic(err)
	}

	assertString(t, `(\"CalQtrOffset\" <= 0) AND (\"CalQtrOffset\" >= -4)`, string(outStr.Bytes()), "Mismattched strign")
}
