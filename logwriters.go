package insight_server

import (
	"fmt"
	"io"
	"path/filepath"
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
type possibleFileWriter struct {
	tmpDir string

	baseFileName string
	hasFile      bool

	headers []string

	file   io.WriteCloser
	writer *GpCsvWriter

	isClosed bool
}

func NewPossibleFileWriter(tmpDir, baseFileName string, headers []string) *possibleFileWriter {
	return &possibleFileWriter{
		tmpDir:       tmpDir,
		baseFileName: baseFileName,
		hasFile:      false,
		headers:      headers,
		isClosed:     false,
	}
}

// Writes a row to the csv writer
func (w *possibleFileWriter) WriteRow(row []string) error {
	if !w.hasFile {

		f, err := NewGzippedFileWriterWithTemp(w.baseFileName, w.tmpDir)
		if err != nil {
			return fmt.Errorf("Error while creating parse error output file for '%s': %v", w.baseFileName, err)
		}

		w.file = f
		w.writer = MakeCsvWriter(f)

		// escape each header
		escapedHeaders, err := EscapeRowForGreenPlum(w.headers)
		if err != nil {
			return fmt.Errorf("Error escaping header fields for greenplum >>%v<<: %v", w.headers, err)
		}

		// write the error header to the file
		if err := w.writer.Write(escapedHeaders); err != nil {
			return fmt.Errorf("Error writing parse error CSV file '%s' header: %v", w.baseFileName, err)
		}

		// mark that we have the file created
		w.hasFile = true
	}

	// escape each field
	escaped, err := EscapeRowForGreenPlum(row)
	if err != nil {
		return fmt.Errorf("Error escaping fields for greenplum >>%v<<: %v", row, err)
	}

	// write it out
	if err := w.writer.Write(escaped); err != nil {
		return fmt.Errorf("Error writing parse error row: %v", err)
	}

	return nil
}

// Closes the file if it is open
func (w *possibleFileWriter) Close() error {
	// if we are already closed, return
	if w.isClosed {
		return nil
	}
	defer func() { w.isClosed = true }()

	if !w.hasFile {
		return nil
	}

	// make sure we close (and move) the error file even if we have errors
	defer w.file.Close()

	// flush the output
	w.writer.Flush()
	if err := w.writer.Error(); err != nil {
		return fmt.Errorf("Error flushing CSV output: %v", err)
	}

	// try to close the output
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("Error closing CSV output: %v", err)
	}

	return nil
}

// The combined writer for errors and parsed
// -----------------------------------------

type serverlogsWriter struct {
	parsedWriter, errorsWriter *possibleFileWriter

	parsedCount, errorCount int
	isClosed                bool
}

func NewServerlogsWriter(outputDir, tmpDir, fileBaseName string, parsedHeaders []string) ServerlogWriter {
	// the output path for the logs
	parsedOutputPath := filepath.Join(outputDir, fileBaseName)
	// error files are in the same directory but have a prefix
	errorsOutputPath := filepath.Join(outputDir, fmt.Sprintf("errors_%s", fileBaseName))

	return &serverlogsWriter{
		parsedWriter: NewPossibleFileWriter(
			tmpDir,
			parsedOutputPath,
			append([]string{"filename", "host_name"}, parsedHeaders...),
		),
		errorsWriter: NewPossibleFileWriter(
			tmpDir,
			errorsOutputPath,
			[]string{"error", "hostname", "filename", "line"},
		),
		isClosed: false,
	}
}

func (w *serverlogsWriter) WriteError(source *ServerlogsSource, parseErr error, line string) error {
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
