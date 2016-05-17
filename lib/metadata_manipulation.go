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

var serverlogsTable = metaTable{"public", "serverlogs"}
var serverlogsTableAlt = metaTable{"public", "jsonlogs"}
var plainServerlogsTable = metaTable{"public", "plainlogs"}

var serverlogsErrorTable = metaTable{"public", "error_jsonlogs"}
var serverlogsAltErrorTable = metaTable{"public", "error_serverlogs"}
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
		metaColumn{serverlogsTable, "elapsed", "double"},
		metaColumn{serverlogsTable, "start_ts", "timestamp without time zone"},
	},
	{
		metaColumn{serverlogsTableAlt, "filename", "text"},
		metaColumn{serverlogsTableAlt, "host_name", "text"},
		metaColumn{serverlogsTableAlt, "ts", "timestamp without time zone"},
		metaColumn{serverlogsTableAlt, "pid", "integer"},
		metaColumn{serverlogsTableAlt, "tid", "integer"},
		metaColumn{serverlogsTableAlt, "sev", "text"},
		metaColumn{serverlogsTableAlt, "req", "text"},
		metaColumn{serverlogsTableAlt, "sess", "text"},
		metaColumn{serverlogsTableAlt, "site", "text"},
		metaColumn{serverlogsTableAlt, "user", "text"},
		metaColumn{serverlogsTableAlt, "k", "text"},
		metaColumn{serverlogsTableAlt, "v", "text"},
		metaColumn{serverlogsTableAlt, "elapsed", "double"},
		metaColumn{serverlogsTableAlt, "start_ts", "timestamp without time zone"},
	},
	{
		metaColumn{plainServerlogsTable, "filename", "text"},
		metaColumn{plainServerlogsTable, "host_name", "text"},
		metaColumn{plainServerlogsTable, "ts", "timestamp without time zone"},
		metaColumn{plainServerlogsTable, "pid", "integer"},
		metaColumn{plainServerlogsTable, "line", "text"},
		metaColumn{plainServerlogsTable, "elapsed", "double"},
		metaColumn{plainServerlogsTable, "start_ts", "timestamp without time zone"},
	},
	{
		metaColumn{serverlogsAltErrorTable, "error", "text"},
		metaColumn{serverlogsAltErrorTable, "host_name", "text"},
		metaColumn{serverlogsAltErrorTable, "filename", "text"},
		metaColumn{serverlogsAltErrorTable, "line", "text"},
	},
	{
		metaColumn{serverlogsErrorTable, "error", "text"},
		metaColumn{serverlogsErrorTable, "host_name", "text"},
		metaColumn{serverlogsErrorTable, "filename", "text"},
		metaColumn{serverlogsErrorTable, "line", "text"},
	},
	{
		metaColumn{plainServerlogsErrorTable, "error", "text"},
		metaColumn{plainServerlogsErrorTable, "host_name", "text"},
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
var plainlogsRegexp *regexp.Regexp = regexp.MustCompile("plainlogs")

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

		// skip any lines from the serverlogs or plainlogs table
		if !serverlogsRegexp.Match(line) && !plainlogsRegexp.Match(line) {
			outWriter.Write(line)
			outWriter.Write(eolChars)
		}
	}

	logrus.WithFields(logrus.Fields{
		"component": "metadata",
		"file":      meta.OriginalFilename,
	}).Info("Adding metadata")

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
	return outWriter.Flush()
}
