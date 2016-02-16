package models

import (
	"github.com/revel/revel"
	"regexp"
)

// Returns a sanitized filename with all non-alphanumeric characters replaced by dashes
func SanitizeName(name string) string {
	// sanitize the filename
	// TODO: refactor this to a static if golang regexp is thread-safe / re-enterant
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		revel.ERROR.Printf("Error compiling regexp: %v", err)
	}

	return reg.ReplaceAllString(name, "-")
}
