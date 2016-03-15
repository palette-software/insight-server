package insight_server

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

type metaTable struct {
	schema, name string
}

type metaColumn struct {
	table              metaTable
	column, formatType string
}

var serverlogsTable metaTable = metaTable{
	"public", "serverlogs",
}

var serverlogsErrorTable metaTable = metaTable{
	"public", "error_serverlogs",
}
var preparsedServerlogsColumns []metaColumn = []metaColumn{
	metaColumn{serverlogsTable, "filename", "text"},
	metaColumn{serverlogsTable, "host_name", "text"},
	metaColumn{serverlogsTable, "ts", "timestamp without time zone"},
	metaColumn{serverlogsTable, "pid", "integer"},
	metaColumn{serverlogsTable, "tid", "integer"},
	metaColumn{serverlogsTable, "sev", "text"},
	metaColumn{serverlogsTable, "req", "text"},
	metaColumn{serverlogsTable, "sess", "text"},
	metaColumn{serverlogsTable, "site", "text"},
	metaColumn{serverlogsTable, "user", "text"},
	metaColumn{serverlogsTable, "k", "text"},
	metaColumn{serverlogsTable, "v", "text"},

	metaColumn{serverlogsErrorTable, "error", "text"},
	metaColumn{serverlogsErrorTable, "line", "text"},
}

func makeMetaString(cols []metaColumn) string {
	o := make([]string, len(cols))
	for i, col := range cols {
		o[i] = fmt.Sprintf("%s\v%s\v%s\v%s\v%d",
			col.table.schema,
			col.table.name,
			col.column,
			col.formatType,
			i+1)
	}
	return strings.Join(o, "\r\n")
}

var ServerlogsMetaString string = makeMetaString(preparsedServerlogsColumns)

// The regexp we use to filter out the serverlogs rows
var serverlogsRegexp *regexp.Regexp = regexp.MustCompile("serverlogs")

// The EOL characters used in the output csv file
var eolChars []byte = []byte{'\r', '\n'}

// Handler updating metadata
func MetadataUploadHandler(c *UploadCallbackCtx) error {

	// open the output
	outf, err := ioutil.TempFile(c.Basedir, "metadata-rewrite")
	if err != nil {
		return err
	}
	defer outf.Close()

	// put a gzipped writer on top
	gzipWriter := gzip.NewWriter(outf)
	defer gzipWriter.Close()

	// open the input
	inf, err := os.Open(c.SourceFile)
	if err != nil {
		return err
	}
	defer inf.Close()

	// open a gzipped reader
	gzipReader, err := gzip.NewReader(inf)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	// Create a pair of buffered readers on top of the
	// gzip stream.
	inReader := bufio.NewReader(gzipReader)
	outWriter := bufio.NewWriter(gzipWriter)

	// Filter out
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
			outWriter.Write(eolChars)
		}
	}

	log.Printf("[metadata] adding metadata to: '%s'", c.OutputFile)

	// Append the prepared serverlogs data
	_, err = outWriter.WriteString(ServerlogsMetaString)
	if err != nil {
		return err
	}

	// Flush the outputs buffer
	err = outWriter.Flush()
	if err != nil {
		return err
	}

	// close the output gzip writer
	gzipWriter.Close()
	err = outf.Close()
	if err != nil {
		return err
	}

	// close and delete the input file
	log.Printf("[metadata] removing temporary '%s'", inf.Name())
	gzipReader.Close()
	inf.Close()
	err = os.Remove(inf.Name())
	if err != nil {
		return err
	}

	// defer moving to the default move handler
	return MoveHandler(&UploadCallbackCtx{
		SourceFile: outf.Name(),
		OutputDir:  c.OutputDir,
		OutputFile: c.OutputFile,
	})
}
