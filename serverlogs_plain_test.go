package insight_server

import (
	"strings"
	"testing"
)

const plainTextTestLines = `2016-04-02 23:57:38.215 (15992): logfile_rotation: opening new log
2016-04-02 23:57:38.215 (7136): tdeserver: accepting new connection
2016-04-02 23:57:38.215 (7136): tdeserver: new connection, connection=::ffff:127.0.0.1:27042->::ffff:127.0.0.1:57174
2016-04-02 23:57:38.216 (12220): beginthread: recv, connection=::ffff:127.0.0.1:27042->::ffff:127.0.0.1:57174
2016-04-02 23:57:38.216 (12220): tdeserver: client disconnected
2016-04-02 23:57:38.216 (12220): tdeserver: closing connection, connection=::ffff:127.0.0.1:27042->::ffff:127.0.0.1:57174
2016-04-02 23:57:38.216 (12220): endthread: recv, connection=::ffff:127.0.0.1:27042->::ffff:127.0.0.1:57174
2016-04-02 23:57:38.216 (15396): beginthread: send, connection=::ffff:127.0.0.1:27042->::ffff:127.0.0.1:57174
2016-04-02 23:57:38.216 (15396): endthread: send, connection=::ffff:127.0.0.1:27042->::ffff:127.0.0.1:57174
2016-04-02 23:57:44.216 (7136): tdeserver: accepting new connection`

func TestPlainServerlogParse(t *testing.T) {
	parsed, errors, err := parsePlainServerlog(&PlainServerlogsInput{
		Source: PlainServerlogsSource{
			Host:     "Miles-PC",
			Filename: "tdeserver0_2016_04_02_23_57_38.log",
		},
		Reader: strings.NewReader(plainTextTestLines),
	})

	if err != nil {
		t.Error("Error parsing serverlogs: ", err)
		return
	}

	assertInt(t, 0, len(errors), "Mismatching error line count")
	assertInt(t, 10, len(parsed), "Mismatching line count")

	assertInt(t, 7136, parsed[1].Pid, "Pid mismatch")
	assertInt(t, 15396, parsed[8].Pid, "Pid mismatch")

	assertString(t, "endthread: send, connection=::ffff:127.0.0.1:27042->::ffff:127.0.0.1:57174", parsed[8].Line, "Line mismatch")
	assertString(t, "tdeserver: accepting new connection", parsed[1].Line, "Line mismatch")
}
