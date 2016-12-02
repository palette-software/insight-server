package insight_server

import (
	"fmt"
	"io"
	"path/filepath"

	"time"

	log "github.com/palette-software/insight-tester/common/logging"
)

type ServerlogWriter interface {
	io.Closer

	WriteParsed(source *ServerlogsSource, fields []string) error
	WriteError(source *ServerlogsSource, err error, line string) error

	ParsedRowCount() int
	ErrorRowCount() int
}

// Log Writer
// ----------

// A writer that creates output files on-demand
type csvFileWriter struct {
	tmpDir string

	baseFileName string
	hasFile      bool

	headers []string

	file   *GzippedFileWriterWithTemp
	writer *GpCsvWriter

	isClosed bool

	tsColumn    string
	outFileName string
}

func NewCsvFileWriter(tmpDir, baseFileName string, headers []string) *csvFileWriter {
	return &csvFileWriter{
		tmpDir:       tmpDir,
		baseFileName: baseFileName,
		hasFile:      false,
		headers:      headers,
		isClosed:     false,
		// the timestamp we'll be using
		tsColumn:    time.Now().Format(GpfdistPostfixTsFormat),
		outFileName: "",
	}
}

func (w *csvFileWriter) extendHeaders(headers []string) []string {
	return surroundWith(headers, "p_filepath", "p_cre_date")
}

func (w *csvFileWriter) extendRow(headers []string) []string {
	return surroundWith(headers, w.outFileName, w.tsColumn)
}

// Creates the output file for this writer
func (w *csvFileWriter) CreateFile() error {
	// if the file already exists, dont continue
	if w.hasFile {
		return nil
	}

	f, err := NewGzippedFileWriterWithTemp(w.baseFileName, w.tmpDir)
	if err != nil {
		return fmt.Errorf("Error while creating parse error output file for '%s': %v", w.baseFileName, err)
	}

	w.file = f
	w.outFileName = f.GetRandomFileName()
	w.writer = MakeCsvWriter(f)

	// write the headers
	if err := w.writeInternal(w.extendHeaders(w.headers)); err != nil {
		return err
	}

	// mark that we have the file created
	w.hasFile = true
	return nil
}

// Writes a row to the csv writer
func (w *csvFileWriter) WriteRow(row []string) error {
	if !w.hasFile {
		w.CreateFile()
	}

	return w.writeInternal(w.extendRow(row))

}

// code shared between CreateFile() and WriteRow()
func (w *csvFileWriter) writeInternal(row []string) error {
	// escape each field
	escaped, err := EscapeRowForGreenPlum(row)
	if err != nil {
		return fmt.Errorf("Error escaping fields for greenplum >>%v<<: %v", row, err)
	}

	// write it out
	if err := w.writer.Write(escaped); err != nil {
		return fmt.Errorf("Error writing CSV output row: %v", err)
	}

	return nil
}

// Closes the file if it is open
func (w *csvFileWriter) Close() error {
	// if we are already closed, return
	if w.isClosed {
		return nil
	}
	defer func() { w.isClosed = true }()

	if !w.hasFile {
		return nil
	}

	// make sure we close (and move) the error file even if we have errors
	defer w.file.CloseWithFileName(w.outFileName)

	// flush the output
	w.writer.Flush()
	if err := w.writer.Error(); err != nil {
		return fmt.Errorf("Error flushing CSV output: %v", err)
	}

	// try to close the output
	if err := w.file.CloseWithFileName(w.outFileName); err != nil {
		return fmt.Errorf("Error closing CSV output: %v", err)
	}

	return nil
}

// The combined writer for errors and parsed
// -----------------------------------------

type serverlogsWriter struct {
	parsedWriter, errorsWriter *csvFileWriter

	parsedCount, errorCount int
	isClosed                bool
}

func NewServerlogsWriter(outputDir, tmpDir, fileBaseName string, parsedHeaders []string) ServerlogWriter {
	// the output path for the logs
	parsedOutputPath := filepath.Join(outputDir, fileBaseName)
	// error files are in the same directory but have a prefix
	errorsOutputPath := filepath.Join(outputDir, fmt.Sprintf("errors_%s", fileBaseName))

	return &serverlogsWriter{
		parsedWriter: NewCsvFileWriter(
			tmpDir,
			parsedOutputPath,
			append([]string{"filename", "host_name"}, parsedHeaders...),
		),
		errorsWriter: NewCsvFileWriter(
			tmpDir,
			errorsOutputPath,
			[]string{"error", "hostname", "filename", "line"},
		),
		isClosed: false,
	}
}

func (w *serverlogsWriter) WriteError(source *ServerlogsSource, parseErr error, line string) error {
	// log errors so splunk can pick them up
	log.Errorf("Error during serverlog parsing. host=%s file=%s line=%s err=%s", source.Host, source.Filename, line, parseErr)

	err := w.errorsWriter.WriteRow([]string{fmt.Sprint(parseErr), source.Host, source.Filename, line})
	if err == nil {
		w.errorCount++
	}
	return err
}

func (w *serverlogsWriter) WriteParsed(source *ServerlogsSource, fields []string) error {
	// add the filename and host to the fields
	err := w.parsedWriter.WriteRow(append([]string{source.Filename, source.Host}, fields...))
	if err == nil {
		w.parsedCount++
	}
	return err
}

func (w *serverlogsWriter) Close() error {
	if w.isClosed {
		return nil
	}
	// update the isClosed flag
	defer func() { w.isClosed = true }()
	// close the errors file
	// TODO: merge the possible error from here with the possible error from parsedwriter's close()
	defer w.errorsWriter.Close()
	// this should be the more important file
	return w.parsedWriter.Close()
}

func (w *serverlogsWriter) ParsedRowCount() int {
	return w.parsedCount
}
func (w *serverlogsWriter) ErrorRowCount() int {
	return w.errorCount
}
