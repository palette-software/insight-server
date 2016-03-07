package insight_server

import (
	"strings"
	"fmt"
	"io/ioutil"
	"bufio"
	"regexp"
	"os"
	"io"
	"log"
)

type metaTable struct {
	schema, name string
}

type metaColumn struct {
	table              metaTable
	column, formatType string
}

var serverlogsTable metaTable = metaTable{
	"public", "preparsed_serverlogs",
}

var serverlogsErrorTable metaTable = metaTable{
	"public", "error_serverlogs",
}
var preparsedServerlogsColumns []metaColumn = []metaColumn{
	metaColumn{serverlogsTable, "filename", "text" },
	metaColumn{serverlogsTable, "host_name", "text" },
	metaColumn{serverlogsTable, "ts", "text" },
	metaColumn{serverlogsTable, "pid", "integer" },
	metaColumn{serverlogsTable, "tid", "integer" },
	metaColumn{serverlogsTable, "sev", "text" },
	metaColumn{serverlogsTable, "req", "text" },
	metaColumn{serverlogsTable, "sess", "text" },
	metaColumn{serverlogsTable, "site", "text" },
	metaColumn{serverlogsTable, "user", "text" },
	metaColumn{serverlogsTable, "k", "text" },
	metaColumn{serverlogsTable, "v", "text" },

	metaColumn{serverlogsErrorTable, "error", "text" },
	metaColumn{serverlogsErrorTable, "line", "text" },
}

func makeMetaString(cols []metaColumn) string {
	o := make([]string, len(cols))
	for i, col := range cols {
		o[i] = fmt.Sprintf("%s\v%s\v%s\v%s\v%d",
			col.table.schema,
			col.table.name,
			col.column,
			col.formatType,
			i + 1)
	}
	return strings.Join(o, "\r\n")
}

var ServerlogsMetaString string = makeMetaString(preparsedServerlogsColumns)


// Handler updating metadata
func MetadataUploadHandler(c *UploadCallbackCtx) error {

	// open the output
	outf, err := ioutil.TempFile(c.Basedir, "metadata-rewrite")
	if err != nil {
		return err
	}
	defer outf.Close()

	// open the input
	inf, err := os.Open(c.SourceFile)
	if err != nil {
		return err
	}
	defer inf.Close()

	inReader := bufio.NewReader(inf)
	outWriter := bufio.NewWriter(outf)

	serverlogsRegexp := regexp.MustCompile("serverlogs")
	eol := []byte{'\r', '\n'}

	for {
		line, _, err := inReader.ReadLine()
		// if eof, we are done
		if err == io.EOF {
			break
		}
		// otherwise we arent ok
		if err != nil {
			return err
		}

		// skip any lines from the serverlogs table
		if !serverlogsRegexp.Match(line) {
			outWriter.Write(line)
			outWriter.Write(eol)
		}
	}

	log.Printf("[metadata] adding metadata to: '%s'", c.OutputFile)

	_, err = outWriter.WriteString(ServerlogsMetaString)
	if err != nil {
		return err
	}

	err = outWriter.Flush()
	if err != nil {
		return err
	}

	err = outf.Close()
	if err != nil {
		return err
	}

	// close and delete the input file
	log.Printf("[metadata] removing temporary '%s'", inf.Name())
	inf.Close()
	err = os.Remove(inf.Name())
	if err != nil {
		return err
	}


	// defer moving to the default move handler
	return MoveHandler(&UploadCallbackCtx{
		SourceFile: outf.Name(),
		OutputDir: c.OutputDir,
		OutputFile: c.OutputFile,
	})
}
