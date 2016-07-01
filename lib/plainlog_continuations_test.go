package insight_server

import (
	"testing"

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
