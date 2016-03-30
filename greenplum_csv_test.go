package insight_server

import (
	"bytes"
	"fmt"
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

		w.Flush()

		assertString(t, fmt.Sprint(test.output, "\r\n"), string(b.Bytes()), "mismatch")
	}
}
