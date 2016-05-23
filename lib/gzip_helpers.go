package insight_server

import (
	"compress/gzip"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
)

type gzippedFileReader struct {
	baseFile   *os.File
	gzipReader *gzip.Reader
}

// Creates a new gzipped file reader
func NewGzippedFileReader(fileName string) (io.ReadCloser, error) {
	baseFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	gzipReader, err := gzip.NewReader(baseFile)
	if err != nil {
		return nil, err
	}

	return &gzippedFileReader{
		baseFile:   baseFile,
		gzipReader: gzipReader,
	}, nil
}

// Closes the underlying file
func (g *gzippedFileReader) Close() error {
	return g.baseFile.Close()
}

// Forwards reading to the underlying gzipped stream
func (g *gzippedFileReader) Read(p []byte) (n int, err error) {
	return g.gzipReader.Read(p)
}

// Writer
// ------

// Encapsulates a writer that writes to a gzipped temp file
// that is moved to its final destination after Close() is called
type GzippedFileWriterWithTemp struct {
	filePath string

	// The directory where the tempfile will be located
	tmpDir string

	tmpFile    *os.File
	gzipWriter *gzip.Writer

	hasher hash.Hash

	isClosed bool

	BytesWritten int
}

func NewGzippedFileWriterWithTemp(file string, tmpDir string) (*GzippedFileWriterWithTemp, error) {
	tmpFile, err := ioutil.TempFile(tmpDir, fmt.Sprintf("gzipped-preprocess-%s", SanitizeName(filepath.Base(file))))
	if err != nil {
		return nil, fmt.Errorf("Cannot open temp file: %v", err)
	}

	gzipWriter := gzip.NewWriter(tmpFile)
	return &GzippedFileWriterWithTemp{
		filePath:     file,
		tmpFile:      tmpFile,
		gzipWriter:   gzipWriter,
		hasher:       md5.New(),
		isClosed:     false,
		BytesWritten: 0,
	}, nil
}

// Deletes the temporary file
func (g *GzippedFileWriterWithTemp) Drop() error {
	defer func() { g.isClosed = true }()
	g.tmpFile.Close()
	os.Remove(g.tmpFile.Name())
	return nil
}

// Returns the md5 of the already written data
func (g *GzippedFileWriterWithTemp) Md5() []byte {
	return g.hasher.Sum(nil)
}

// Returns the output filename by using the hash and the original output filename
func (g *GzippedFileWriterWithTemp) GetFileName() string {
	// get the file hash
	return g.GetFileNameForMd5(fmt.Sprintf("%032x", g.Md5()))
}

// Returns the output filename by using the supplied md5 string and the original output filename
func (g *GzippedFileWriterWithTemp) GetFileNameForMd5(fileMd5 string) string {
	// replace the {{md5}} token with the actual md5
	baseFilename := strings.Replace(filepath.Base(g.filePath), "{{md5}}", fileMd5, -1)
	baseFileExt := filepath.Ext(baseFilename)

	// the output is in the originally intended directory,
	// but with our new filename containing the hash of the file
	return filepath.ToSlash(path.Join(
		filepath.Dir(g.filePath),
		fmt.Sprintf("%s.%s.gz",
			SanitizeName(strings.TrimSuffix(baseFilename, baseFileExt)),
			SanitizeName(baseFileExt[1:]),
		),
	))
}

func (g *GzippedFileWriterWithTemp) Close() error {
	// if we are already closed
	if g.isClosed {
		return nil
	}

	return g.CloseWithFileName(g.GetFileName())
}

func (g *GzippedFileWriterWithTemp) CloseWithFileName(outFileName string) error {
	// if we are already closed
	if g.isClosed {
		return nil
	}

	// Create the files directory
	if err := CreateDirectoryIfNotExists(filepath.Dir(outFileName)); err != nil {
		return err
	}

	defer func() { g.isClosed = true }()
	// make sure we close the temp file even in case of error
	defer g.tmpFile.Close()

	// flush the stream
	if err := g.gzipWriter.Flush(); err != nil {
		return fmt.Errorf("Error while flushing gzip writer: %v", err)
	}

	// close the gzip stream
	if err := g.gzipWriter.Close(); err != nil {
		return fmt.Errorf("Error while closing gzip writer: %v", err)
	}

	// close the underlying file
	if err := g.tmpFile.Close(); err != nil {
		return fmt.Errorf("Error closing temporary file '%s': %v", g.tmpFile.Name(), err)
	}

	// Add the MD5 to the filename
	logrus.WithFields(logrus.Fields{
		"component":  "gzip-out",
		"sourceFile": g.tmpFile.Name(),
		"outputFile": outFileName,
	}).Debug("Moving output")
	// now we can move it to its final destination
	return os.Rename(g.tmpFile.Name(), outFileName)
}

// Forward writes to the gzip stream
func (g *GzippedFileWriterWithTemp) Write(p []byte) (n int, err error) {
	// append the bytes to the hasher
	g.hasher.Write(p)
	g.BytesWritten += len(p)
	// write the output
	return g.gzipWriter.Write(p)
}
