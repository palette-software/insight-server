package insight_server

import (
	"testing"
	"os"
)

const SAMPLELOGS_FILE = "testdata/serverlogs.csv"

func assertInt(t *testing.T, a, b int, msg string) {
	if a != b {
		t.Fatalf("%s: %v vs %v", msg, a, b)
	}
}

func assertString(t *testing.T, a, b string, msg string) {
	if a != b {
		t.Fatalf("%s: %v vs %v", msg, a, b)
	}
}

func TestServerlogsImport(t *testing.T) {

	f, err := os.Open(SAMPLELOGS_FILE)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	serverlogs, errorRows, err := ParseServerlogs(f)

	if err != nil {
		panic(err)
	}

	assertInt(t, len(errorRows), 0, "Error row miscount")
	assertInt(t, len(serverlogs), 4230, "Row miscount")

	firstRow := serverlogs[0]

	assertInt(t, firstRow.Outer.Pid, 5556, "1st row Pid mismatch")
	assertString(t, firstRow.Outer.Ts, "2015-12-29T16:34:30.585", "1st row Date mismatch")
	assertString(t, firstRow.Outer.Tid, "15b8", "1st row Tid mismatch")
	assertString(t, firstRow.Outer.Sev, "info", "1st row sev mismatch")
	assertString(t, firstRow.Outer.Sess, "-", "1st row sess mismatch")
	assertString(t, firstRow.Outer.Req, "-", "1st row req mismatch")
	assertString(t, firstRow.Outer.Site, "{B0EEC83D-2AC7-4CA7-839D-C3B5F04D85E5}", "1st row site mismatch")
	assertString(t, firstRow.Outer.User, "-", "1st row user mismatch")
	assertString(t, firstRow.Outer.K, "open-log", "1st row k mismatch")
	assertString(t, firstRow.Inner, `{"path":"E:\\Tableau\\Tableau Server\\data\\tabsvc\\logs\\vizqlserver\\tabprotosrv_2.txt"}`, "1st row k mismatch")

}