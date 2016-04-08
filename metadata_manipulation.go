package insight_server

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

type metaTable struct {
	schema, name string
}

type metaColumn struct {
	table              metaTable
	column, formatType string
}

var serverlogsTable = metaTable{"public", "jsonlogs"}
var plainServerlogsTable = metaTable{"public", "plainlogs"}

var serverlogsErrorTable = metaTable{"public", "error_jsonlogs"}
var plainServerlogsErrorTable = metaTable{"public", "error_plainlogs"}

var preparsedServerlogsColumns = [][]metaColumn{
	{
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
	},
	{

		metaColumn{plainServerlogsTable, "filename", "text"},
		metaColumn{plainServerlogsTable, "host_name", "text"},
		metaColumn{plainServerlogsTable, "ts", "timestamp without time zone"},
		metaColumn{plainServerlogsTable, "pid", "integer"},
		metaColumn{plainServerlogsTable, "line", "text"},
	},
	// Hostname and host_name are incosistent because they were used
	// this way, and they might be hardcoded into SQL
	// TODO: fix this situation
	{

		metaColumn{serverlogsErrorTable, "error", "text"},
		metaColumn{serverlogsErrorTable, "hostname", "text"},
		metaColumn{serverlogsErrorTable, "filename", "text"},
		metaColumn{serverlogsErrorTable, "line", "text"},
	},
	{
		metaColumn{plainServerlogsErrorTable, "error", "text"},
		metaColumn{plainServerlogsErrorTable, "hostname", "text"},
		metaColumn{plainServerlogsErrorTable, "filename", "text"},
		metaColumn{plainServerlogsErrorTable, "line", "text"},
	},
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

// The regexp we use to filter out the serverlogs rows
var serverlogsRegexp *regexp.Regexp = regexp.MustCompile("serverlogs")

// The EOL characters used in the output csv file
var eolChars []byte = []byte{'\r', '\n'}

// Handler updating metadata
func MetadataUploadHandler(meta *UploadMeta, tmpDir, baseDir, archivedFile string) error {

	outFileWriter, err := meta.GetOutputGzippedWriter(baseDir, tmpDir)
	if err != nil {
		return fmt.Errorf("Error opening metadata output: %v", err)
	}

	defer outFileWriter.Close()

	inFileReader, err := NewGzippedFileReader(archivedFile)
	defer inFileReader.Close()

	// Create a pair of buffered readers on top of the
	// gzip stream.
	inReader := bufio.NewReader(inFileReader)
	outWriter := bufio.NewWriter(outFileWriter)

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

	logrus.Printf("[metadata] adding metadata to: '%s'", meta.OriginalFilename)

	metadata := make([]string, len(preparsedServerlogsColumns))
	for i, table := range preparsedServerlogsColumns {
		metadata[i] = makeMetaString(table)
	}

	metadataString := strings.Join(metadata, "\r\n")

	// Append the prepared serverlogs data
	_, err = outWriter.WriteString(metadataString)
	if err != nil {
		return err
	}

	// Flush the outputs buffer
	err = outWriter.Flush()
	if err != nil {
		return err
	}
	return nil

}
