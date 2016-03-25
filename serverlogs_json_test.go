package insight_server

import (
	"fmt"
	"os"
	"testing"
)

const SAMPLELOGS_FILE = "testdata/serverlogs.csv"

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

func TestServerlogsImport(t *testing.T) {

	f, err := os.Open(SAMPLELOGS_FILE)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	serverlogs, errorRows, err := ParseServerlogs(f, "Etc/UTC")

	if err != nil {
		panic(err)
	}

	assertInt(t, len(errorRows), 0, "Error row miscount")
	assertInt(t, len(serverlogs), 634, "Row miscount")

	rowToTest := serverlogs[3]

	assertInt(t, rowToTest.Outer.Pid, 6176, "1st row Pid mismatch")
	assertString(t, rowToTest.Outer.Ts, "2015-11-13T15:48:51.719", "1st row Date mismatch")
	assertString(t, rowToTest.Outer.Tid, fmt.Sprintf("%d", 0x540), "1st row Tid mismatch")
	assertString(t, rowToTest.Outer.Sev, "info", "1st row sev mismatch")
	assertString(t, rowToTest.Outer.Sess, "04432582338743DFAF121D586C6BC36D-0:1", "1st row sess mismatch")
	assertString(t, rowToTest.Outer.Req, "VkYGY6weAFgAACAEOUUAAAIt", "1st row req mismatch")
	assertString(t, rowToTest.Outer.Site, "Default", "1st row site mismatch")
	assertString(t, rowToTest.Outer.User, "palette", "1st row user mismatch")
	assertString(t, rowToTest.Outer.K, "lock-session", "1st row k mismatch")

	// for now skip this test, as the re-marshaling may change the order of the fields, so
	// a string comparison is not the best idea
	//assertString(t, rowToTest.Inner, `{"sess":"04432582338743DFAF121D586C6BC36D-0:1","user":"palette","workbook":"Book1","site":"Default"}`, "1st row k mismatch")

}
