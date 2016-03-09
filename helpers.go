package insight_server

import (
	"log"
	"regexp"
	"net/http"
	"mime/multipart"
	"fmt"
	"net/url"
	"os"
)


// HELPERS
// =======

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
	log.Printf("[http] {%v}: %s", status, err)
	http.Error(w, err, status)
	return
}


// Returns whether the given file or directory exists or not
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return true, err
}


// HTTP package helpers
// ===================


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

