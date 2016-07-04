package insight_server

import (
	"testing"

	"io/ioutil"
	"os"
	"time"

	tassert "github.com/stretchr/testify/assert"
)

func TestIsLineContinuation(t *testing.T) {

	tassert.False(t, IsLineContinuation("Session49: StatementExecute: OK rowset_guid=15 n_columns=4, Elapsed time:0.000s, Compilation time:0.000s, Execution time:0.000s"))
	tassert.False(t, IsLineContinuation("Session49: StatementClose: sess_guid=49 stmt_guid=14"))
	tassert.False(t, IsLineContinuation("logfile_rotation: closing this log"))

	tassert.True(t, IsLineContinuation("logfile_rotation: opening new log "))
	tassert.True(t, IsLineContinuation("logfile_rotation: opening new log"))

}

func TestLineWillHaveContinuation(t *testing.T) {

	tassert.False(t, LineWillHaveContinuation("Session49: StatementExecute: OK rowset_guid=15 n_columns=4, Elapsed time:0.000s, Compilation time:0.000s, Execution time:0.000s"))
	tassert.False(t, LineWillHaveContinuation("Session49: StatementClose: sess_guid=49 stmt_guid=14"))
	tassert.False(t, LineWillHaveContinuation("logfile_rotation: opening new log"))

	tassert.True(t, LineWillHaveContinuation("logfile_rotation: closing this log "))
	tassert.True(t, LineWillHaveContinuation("logfile_rotation: closing this log"))

}
func TestLineHasPid(t *testing.T) {

	tassert.False(t, LineHasPid("Session49: StatementExecute: OK rowset_guid=15 n_columns=4, Elapsed time:0.000s, Compilation time:0.000s, Execution time:0.000s"))
	tassert.False(t, LineHasPid("Session49: StatementClose: sess_guid=49 stmt_guid=14 pid=921"))
	tassert.False(t, LineHasPid("logfile_rotation: closing this log"))

	tassert.True(t, LineHasPid("pid=9544"))
	tassert.False(t, LineHasPid("we have the pid=9544"))

}

func TestSavingOfLogContinuations(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "logcontinuationtest")

	tassert.Nil(t, err)

	db, err := MakeBoltDbLogContinuationDb(dir)
	tassert.Nil(t, err)

	k := []byte("Hello")
	v := []byte("World")

	tassert.Nil(t, db.SetHeaderFor(k, v))

	existingVal, hasValue, err := db.HeaderLineFor(k)

	tassert.Nil(t, err)
	tassert.True(t, hasValue)
	tassert.Equal(t, v, existingVal)
}

func TestLogContinuationTTL(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "logcontinuationtest")

	tassert.Nil(t, err)

	db, err := MakeBoltDbLogContinuationDb(dir)
	tassert.Nil(t, err)

	k := []byte("Hello")
	v := []byte("World")

	tassert.Nil(t, db.SetHeaderFor(k, v))

	// sllep a second so the previous one will be over ttl
	time.Sleep(time.Second * 2)

	k2 := []byte("Foo")
	v2 := []byte("Bar")
	tassert.Nil(t, db.SetHeaderFor(k2, v2))

	// clean the db
	db.VacuumOld(time.Second)

	// check if the second entry is still there
	val, hasValue, err := db.HeaderLineFor(k2)
	tassert.Nil(t, err)
	tassert.True(t, hasValue)
	tassert.Equal(t, v2, val)

	// check if the second entry is still there
	_, hasValue, err = db.HeaderLineFor(k)
	tassert.Nil(t, err)
	tassert.False(t, hasValue)
}
