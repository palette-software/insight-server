package insight_server

import "testing"

var expectedCsvUnescapeResults = map[string]string{

	// vertical tabs
	`Hello\013World`: "Hello\vWorld",

	// newlines
	// TODO: do these matter at all for us?
	`I'm a\nnewline\rOnce Again`: "I'm a\nnewline\rOnce Again",

	// Test unescaping the vertical tab (\013)
	`e:\\\\tableau_server\\\\data\\\\tabsvc\\\\temp\\\\TableauTemp\\\\013d6h11b4x6w017h8gkh05dwnre\\\\Portfolio_Dashboard _ ICAM _ Cos.tde`: `e:\\tableau_server\\data\\tabsvc\\temp\\TableauTemp\\013d6h11b4x6w017h8gkh05dwnre\\Portfolio_Dashboard _ ICAM _ Cos.tde`,

	// a random line from GE
	`{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query":"(alter column [Extract].[Extract].[updated_dt] ( ( \\"comparable\\" \\"comparable\\" ) ( \\"compression\\" \\"array\\" ) ( \\"storagewidth\\" \\"0\\" ) ( \\"encoding\\" \\"encoding\\" ) ) )","cols":0,"protocol-id":11575,"rows":0,"elapsed":0.034,"query-hash":867325541}}`: `{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query":"(alter column [Extract].[Extract].[updated_dt] ( ( \"comparable\" \"comparable\" ) ( \"compression\" \"array\" ) ( \"storagewidth\" \"0\" ) ( \"encoding\" \"encoding\" ) ) )","cols":0,"protocol-id":11575,"rows":0,"elapsed":0.034,"query-hash":867325541}}`,

	// a line from GE with a \n error (\012)
	`"{"ts":"2016-03-24T08:15:16.687","pid":11540,"tid":"3f7c","sev":"info","req":"-","sess":"A18E57C9FAB842F2AC2A195FA99CF01D-1:0","site":"PGS","user":"pg_extractm","k":"ds-connect","v":{"name":"002 Process Compliance Report","attr":{"create-extracts-locally":"true","name":"002 Process Compliance Report","update-time":"03/24/2016 12:15:16 PM","schema":"Extract","filename":"e:\\\\tableau_server\\\\data\\\\tabsvc\\\\file_uploads\\\\uploads_3101\\\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","tablename":"Extract","port":"27042","class":"dataengine","server":"","extract-engine":"true","dbname":"e:\\\\tableau_server\\\\data\\\\tabsvc\\\\file_uploads\\\\uploads_3101\\\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","management-tablespace":"extracts"}}}"`: `"{"ts":"2016-03-24T08:15:16.687","pid":11540,"tid":"3f7c","sev":"info","req":"-","sess":"A18E57C9FAB842F2AC2A195FA99CF01D-1:0","site":"PGS","user":"pg_extractm","k":"ds-connect","v":{"name":"002 Process Compliance Report","attr":{"create-extracts-locally":"true","name":"002 Process Compliance Report","update-time":"03/24/2016 12:15:16 PM","schema":"Extract","filename":"e:\\tableau_server\\data\\tabsvc\\file_uploads\\uploads_3101\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","tablename":"Extract","port":"27042","class":"dataengine","server":"","extract-engine":"true","dbname":"e:\\tableau_server\\data\\tabsvc\\file_uploads\\uploads_3101\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","management-tablespace":"extracts"}}}"`,

	`C:\\Program Files\\ \015\012\\r\\n\013\\v`: "C:\\Program Files\\ \r\n\\r\\n\v\\v",

	`A\vB`: "A\vB",
}

func TestCsvUnEscape(t *testing.T) {
	for src, expected := range expectedCsvUnescapeResults {
		unescapedStr, err := UnescapeGPCsvString(src)
		if err != nil {
			panic(err)
		}
		assertString(t, expected, unescapedStr, "Escape mismatch")
	}

}

var expectedCsvEscapeResults = map[string]string{

	// vertical tabs
	"Hello\vWorld": `Hello\013World`,

	// newlines
	// TODO: do these matter at all for us?
	"I'm a\nnewline\rOnce Again": `I'm a\012newline\015Once Again`,

	// Test unescaping the vertical tab (\013)
	`e:\\tableau_server\\data\\tabsvc\\temp\\TableauTemp\\013d6h11b4x6w017h8gkh05dwnre\\Portfolio_Dashboard _ ICAM _ Cos.tde`: `e:\\\\tableau_server\\\\data\\\\tabsvc\\\\temp\\\\TableauTemp\\\\013d6h11b4x6w017h8gkh05dwnre\\\\Portfolio_Dashboard _ ICAM _ Cos.tde`,

	// a random line from GE
	`{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query":"(alter column [Extract].[Extract].[updated_dt] ( ( \"comparable\" \"comparable\" ) ( \"compression\" \"array\" ) ( \"storagewidth\" \"0\" ) ( \"encoding\" \"encoding\" ) ) )","cols":0,"protocol-id":11575,"rows":0,"elapsed":0.034,"query-hash":867325541}}`: `{"ts":"2016-03-25T00:59:10.599","pid":11540,"tid":"5640","sev":"info","req":"-","sess":"58F8C1074C3D496EB9B38B46ED14DCAE-1:0","site":"PGS","user":"pg_extractm","k":"end-query","v":{"query":"(alter column [Extract].[Extract].[updated_dt] ( ( \\"comparable\\" \\"comparable\\" ) ( \\"compression\\" \\"array\\" ) ( \\"storagewidth\\" \\"0\\" ) ( \\"encoding\\" \\"encoding\\" ) ) )","cols":0,"protocol-id":11575,"rows":0,"elapsed":0.034,"query-hash":867325541}}`,

	// a line from GE with a \n error (\012)
	`"{"ts":"2016-03-24T08:15:16.687","pid":11540,"tid":"3f7c","sev":"info","req":"-","sess":"A18E57C9FAB842F2AC2A195FA99CF01D-1:0","site":"PGS","user":"pg_extractm","k":"ds-connect","v":{"name":"002 Process Compliance Report","attr":{"create-extracts-locally":"true","name":"002 Process Compliance Report","update-time":"03/24/2016 12:15:16 PM","schema":"Extract","filename":"e:\\tableau_server\\data\\tabsvc\\file_uploads\\uploads_3101\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","tablename":"Extract","port":"27042","class":"dataengine","server":"","extract-engine":"true","dbname":"e:\\tableau_server\\data\\tabsvc\\file_uploads\\uploads_3101\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","management-tablespace":"extracts"}}}"`: `"{"ts":"2016-03-24T08:15:16.687","pid":11540,"tid":"3f7c","sev":"info","req":"-","sess":"A18E57C9FAB842F2AC2A195FA99CF01D-1:0","site":"PGS","user":"pg_extractm","k":"ds-connect","v":{"name":"002 Process Compliance Report","attr":{"create-extracts-locally":"true","name":"002 Process Compliance Report","update-time":"03/24/2016 12:15:16 PM","schema":"Extract","filename":"e:\\\\tableau_server\\\\data\\\\tabsvc\\\\file_uploads\\\\uploads_3101\\\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","tablename":"Extract","port":"27042","class":"dataengine","server":"","extract-engine":"true","dbname":"e:\\\\tableau_server\\\\data\\\\tabsvc\\\\file_uploads\\\\uploads_3101\\\\0124A6A59A2A49AEA1CE605DAEA3C476.tmp","management-tablespace":"extracts"}}}"`,

	"C:\\Program Files\\ \r\n\\r\\n\v\\v": `C:\\Program Files\\ \015\012\\r\\n\013\\v`,

	"A\vB": `A\013B`,
}

func TestCsvEscape(t *testing.T) {
	for src, expected := range expectedCsvEscapeResults {
		escapedStr, err := EscapeGPCsvString(src)
		if err != nil {
			panic(err)
		}
		assertString(t, expected, escapedStr, "Escape mismatch")
	}

}
