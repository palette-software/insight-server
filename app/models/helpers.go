package models

import (
	"github.com/revel/revel"
	"regexp"
	"time"
)

const (
	defaultOutputDirectory   = "/tmp"
	outputDirectoryConfigKey = "tenants.basedir"
)

// The regexp we use for sanitizing any strings to a file name that is valid on all systems
var sanitizeRegexp = regexp.MustCompile("[^A-Za-z0-9]+")

// Returns a sanitized filename with all non-alphanumeric characters replaced by dashes
func SanitizeName(name string) string {
	return sanitizeRegexp.ReplaceAllString(name, "-")
}

// Returns the output directory for the uploaded CSV files.
func GetOutputDirectory() string {
	return revel.Config.StringDefault(outputDirectoryConfigKey, defaultOutputDirectory)
}

// make a nonsensical time for marking invalid responses
var NonsenseTime time.Time = time.Unix(int64(0), 0)
