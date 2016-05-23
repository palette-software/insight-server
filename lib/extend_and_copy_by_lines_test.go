package insight_server

import (
	"bytes"
	"strings"
	"testing"
)

var extendAndCopyByLinesTest1In = `hello world
foo bar`

var extendAndCopyByLinesTest1Out = "prefix,hello world,postfix\nprefix,foo bar,postfix"

var extendAndCopyByLinesTest2In = `hello world
foo bar
`

var extendAndCopyByLinesTest2Out = "prefix,hello world,postfix\nprefix,foo bar,postfix"

func runTestExtendAndCopyByLines(t *testing.T, in, out, prefix, postfix string) {
	inR := strings.NewReader(in)
	outW := bytes.NewBuffer([]byte{})

	if err := extendAndCopyByLines(inR, outW, []byte(prefix), []byte(postfix)); err != nil {
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
