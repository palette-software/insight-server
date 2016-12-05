package insight_server

import (
	tassert "github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMakeMetaFromRequest(t *testing.T) {
	postData :=
		`--xxx
Content-Disposition: form-data; name="field1"

value1
--xxx
Content-Disposition: form-data; name="_md5"

M2Y3ZWEwYTg2ZjkxOTUyZmU3Y2Y3ZGJhNjViMDcxYjM=
--xxx
Content-Disposition: form-data; name="_file"; filename="threadinfo-2016-10-07--15-48-25--seq0000--part0000.csv.gz"
Content-Type: application/octet-stream
Content-Transfer-Encoding: binary

binary data
--xxx--
`
	req, _ := http.NewRequest("PUT", "/upload?pkg=testpkg&host=local&tz=UTC&compression=gzip", ioutil.NopCloser(strings.NewReader(postData)))
	req.Header.Add("Content-Type", "multipart/form-data; boundary=xxx")
	u, _, err := MakeMetaFromRequest(req)
	tassert.NotNil(t, u)
	if u != nil {
		tassert.Equal(t, u.Date, time.Date(2016, time.October, 7, 15, 48, 25, 0, time.UTC))
		tassert.Nil(t, err)
	}
}
