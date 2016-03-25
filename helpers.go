package insight_server

//go:generate go-bindata -pkg $GOPACKAGE -o assets.go assets/

import (
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"hash"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GENERIC HELPERS
// ===============

// simple helper that logs an error then panics
func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

// The regexp we use for sanitizing any strings to a file name that is valid on all systems
var sanitizeRegexp = regexp.MustCompile("[^_A-Za-z0-9]")

// Returns a sanitized filename with all non-alphanumeric characters replaced by dashes
func SanitizeName(name string) string {
	return sanitizeRegexp.ReplaceAllString(name, "-")
}

// Writes the error message to the log then responds with an error message
func writeResponse(w http.ResponseWriter, status int, err string) {
	log.Printf("[http] <=== {%v}: %s", status, err)
	http.Error(w, err, status)
	return
}

// FS HELPERS
// ==========

// Returns whether the given file or directory exists or not
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// Returns true if path is a directory. If it does not exist err is returned
func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// Returns true if path is a directory. Otherwise (even if there was an error) returns false.
func isDirectoryNoFail(path string) bool {
	isDir, err := isDirectory(path)
	return (err == nil && isDir)
}

// / Helper that creates a directory if it does not exist
func CreateDirectoryIfNotExists(path string) error {
	exists, err := fileExists(path)
	// forward errors
	if err != nil {
		return err
	}
	// if it already exists, dont create it
	if exists {
		return nil
	}

	// create the directory
	log.Printf("[storage] Creating directory: '%s'", path)
	if err := os.MkdirAll(path, OUTPUT_DEFAULT_DIRMODE); err != nil {
		return err
	}

	return nil
}

// HTTP PACKAGE HELPERS
// ====================

// Helper to get a part from a multipart message
func getMultipartFile(form *multipart.Form, fieldName string) (file multipart.File, fileName string, err error) {

	// get the file from the form
	fn := form.File[fieldName]
	if len(fn) != 1 {
		err = fmt.Errorf("The request must have exactly 1 '%v' field (has %v).", fieldName, len(fn))
		return
	}

	// take the first one
	uploadedFile := fn[0]

	// set the filename
	fileName = uploadedFile.Filename

	// get the file reader
	file, err = uploadedFile.Open()
	if err != nil {
		return
	}

	return
}

// Helper to get a part from a multipart message
func getMultipartParam(form *multipart.Form, fieldName string) (value string, err error) {

	// get the file from the form
	fn := form.Value[fieldName]
	if len(fn) != 1 {
		err = fmt.Errorf("The request must have exactly 1 '%v' field (has %v).", fieldName, len(fn))
		return "", err
	}

	return fn[0], nil
}

// Returns an url param, or an error if no such param is available
func getUrlParam(reqUrl *url.URL, paramName string) (string, error) {

	// parse the url params
	urlParams, err := url.ParseQuery(reqUrl.RawQuery)
	if err != nil {
		return "", err
	}

	// get the package
	paramVals := urlParams[paramName]
	if len(paramVals) != 1 {
		return "", fmt.Errorf("BAD REQUEST: No '%v' parameter provided", paramName)
	}

	return paramVals[0], nil
}

// Returns a new handler that simply responds with an asset from the precompiled assets
func AssetPageHandler(assetName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := Asset(assetName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write(page)
	}
}

// Gets the version of the server from the VERSION file in the assets directory
func GetVersion() string {
	version, err := Asset("assets/VERSION")
	if err != nil {
		return "v1.0.0"
	}
	return string(version)
}

// MD5 hashing TeeReader helper
// ----------------------------

type Md5Hasher struct {
	Md5    hash.Hash
	Reader io.Reader
}

func makeMd5Hasher(r io.Reader) *Md5Hasher {

	hash := md5.New()

	// create a TeeReader that automatically forwards bytes read from the file to
	// the md5 hasher's reader
	readerWithMd5 := io.TeeReader(r, hash)

	return &Md5Hasher{hash, readerWithMd5}
}

// Returns the hash of the tree
func (m *Md5Hasher) GetHash() []byte {
	return m.Md5.Sum(nil)
}

// Returns the (lowercased) hex string of the Md5
func (m *Md5Hasher) GetHashString() string {
	return fmt.Sprintf("%32x", m.GetHash())
}

// Random string generation
// ========================

var randomStringSrc = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// Generates a bunch of random bytes for a string. Is pretty fast...
// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func RandStringBytesMaskImprSrc(n int) []byte {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, randomStringSrc.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randomStringSrc.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}

// CSV Reading/writing for GP
// ==========================

func MakeCsvReader(r io.Reader) *csv.Reader {
	reader := csv.NewReader(r)
	reader.Comma = '\v'
	reader.LazyQuotes = true
	return reader
}

func MakeCsvWriter(w io.Writer) *csv.Writer {
	writer := csv.NewWriter(w)
	writer.Comma = '\v'
	writer.UseCRLF = true
	return writer
}

// Escapes all strings in a slice for greenplum
func EscapeRowForGreenPlum(row []string) ([]string, error) {
	output := make([]string, len(row))
	// Escape each column
	for i, column := range row {
		outputStr, err := EscapeGPCsvString(column)
		if err != nil {
			return nil, err
		}
		output[i] = outputStr
	}
	return output, nil
}

///////////////////////////////////

func hexToDecimal(tidHexa string) (string, error) {
	decimal, err := strconv.ParseInt(tidHexa, 16, 32)
	decimalString := strconv.FormatInt(decimal, 10)
	return decimalString, err
}

const (
	backslash   = '\\'
	octalPrefix = '0'

	escapeStateNormal = 0
	// when we are in after a normal backslash
	escapeStateBackslashed = 1
)

///////////////////////////////////

// Unescapes a string escaped for greenplum CSV in a linear fashion
// This version keeps a state, so unescapes should be safe even for octal codes.
func UnescapeGPCsvString(field string) (string, error) {
	r := strings.NewReader(field)
	w := bytes.NewBuffer([]byte{})

	b := make([]byte, 1)
	octalBuffer := []byte{0, 0}
	state := escapeStateNormal

	for {
		_, err := r.Read(b)

		// if end of string, return the built string
		if err == io.EOF {
			return string(w.Bytes()), nil
		}

		// otherwise return the error
		if err != nil {
			return "", err
		}

		char := b[0]

		switch state {

		// if we arent escaped, write the character to the normal buffer
		case escapeStateNormal:
			if char == backslash {
				state = escapeStateBackslashed
			} else {
				w.WriteByte(b[0])
			}

		case escapeStateBackslashed:
			// move the state back after this read
			state = escapeStateNormal
			switch char {

			// if its a backslash, write it out
			case backslash:
				w.WriteByte(backslash)

			// TODO: are these cases necessary? They work OK, but are they already be handled by the CSV reader
			case 'n':
				w.WriteByte('\n')
			case 'r':
				w.WriteByte('\r')
			case 't':
				w.WriteByte('\t')
			case 'v':
				w.WriteByte('\v')
			case 'b':
				w.WriteByte('\b')
			case 'f':
				w.WriteByte('\f')

			// if its the octal prefix, move to octal mode
			case octalPrefix:

				// try to read two bytes for octal
				bytesRead, err := r.Read(octalBuffer)
				if err != nil {
					return "", fmt.Errorf("Error while reading octal escape sequence from '%s': %v", field, err)
				}
				if bytesRead != 2 {
					return "", fmt.Errorf("Premature end of string '%s' during octal escape.", field)
				}

				// parse the octal code
				charCode, err := strconv.ParseInt(string(octalBuffer), 8, 8)
				if err != nil {
					return "", fmt.Errorf("Error while parsing octal escape '%s': %v", octalBuffer, err)
				}

				w.WriteByte(byte(charCode))

				state = escapeStateNormal

			default:
				return "", fmt.Errorf("Invalid backslashed character in '%s' @ %d: %d", field, r.Len(), char)
			}

		}

	}

	return "", fmt.Errorf("Unreachable code reached.")
}

// Escapes a string escaped for greenplum CSV in a linear fashion
func EscapeGPCsvString(field string) (string, error) {
	r := strings.NewReader(field)
	w := bytes.NewBuffer([]byte{})
	b := make([]byte, 1)

	for {
		_, err := r.Read(b)

		// if end of string, return the built string
		if err == io.EOF {
			return string(w.Bytes()), nil
		}

		// otherwise return the error (this should never be called, but who knows...)
		if err != nil {
			return "", err
		}

		char := b[0]

		switch char {
		case '\r':
			w.WriteString("\\015")
		case '\n':
			w.WriteString("\\012")
		case '\\':
			w.WriteString("\\\\")
		case '\v':
			w.WriteString("\\013")
		default:
			w.WriteByte(char)
		}

	}

	return "", fmt.Errorf("Unreachable code reached.")
}
