package insight_server

import (
	"bytes"
	"strings"
	"testing"
)

var extendAndCopyByLinesTest1In = `original_header
hello world
foo bar`

var extendAndCopyByLinesTest1Out = "prefix_header,original_header,postfix_header\nprefix,hello world,postfix\nprefix,foo bar,postfix\n"

var extendAndCopyByLinesTest2In = `original_header
hello world
foo bar
`

var extendAndCopyByLinesTest2Out = "prefix_header,original_header,postfix_header\nprefix,hello world,postfix\nprefix,foo bar,postfix\n"

func runTestExtendAndCopyByLines(t *testing.T, in, out, prefix, postfix string) {
	inR := strings.NewReader(in)
	outW := bytes.NewBuffer([]byte{})

	if err := extendAndCopyByLines(inR, outW, []byte(prefix), []byte("prefix_header,"), []byte(postfix), []byte(",postfix_header")); err != nil {
		t.Fatalf("Error in extendAndCopyByLines: %v", err)
	}
	transformed := string(outW.Bytes())
	assertString(t, out, transformed, "extendAndCopyByLines fail")
}

func TestExtendAndCopyByLines(t *testing.T) {
	runTestExtendAndCopyByLines(
		t,
		extendAndCopyByLinesTest1In,
		extendAndCopyByLinesTest1Out,
		"prefix,",
		",postfix",
	)
	runTestExtendAndCopyByLines(
		t,
		extendAndCopyByLinesTest2In,
		extendAndCopyByLinesTest2Out,
		"prefix,",
		",postfix",
	)
}
