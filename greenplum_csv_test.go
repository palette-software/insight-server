package insight_server

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

type gpQuoteTestEntry struct {
	row    []string
	output string
}

var gpquotesTest []gpQuoteTestEntry = []gpQuoteTestEntry{
	{[]string{`hello`, `world`}, "hello\vworld"},
	{[]string{`"hello"`, `world`}, "\\\"hello\\\"\vworld"},
}

func TestGreenplumQuotes(t *testing.T) {
	for _, test := range gpquotesTest {
		b := bytes.NewBuffer([]byte{})
		w := MakeCsvWriter(b)
		//w := NewGpCsvWriter(b)

		w.WriteAll([][]string{test.row})

		// remove the ending line breaks
		s := strings.TrimRight(string(b.Bytes()), " \n\r")

		w.Flush()

		log.Printf(" %v -> '%s'", test.row, s)

		assertString(t, test.output, s, "mismatch")
	}
}
