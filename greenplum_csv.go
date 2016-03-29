package insight_server

// THIS FILE CONTAINS A MODIFIED VERSION OF THE GO CSV WRITER TO HANDLE
// THE NON-STANDARD ASPECTS OF THE GREENPLUM/POSTGRES CSV SPECIFICATION

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"bufio"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A Writer writes records to a CSV encoded file.
//
// As returned by NewWriter, a Writer writes records terminated by a
// newline and uses ',' as the field delimiter.  The exported fields can be
// changed to customize the details before the first call to Write or WriteAll.
//
// Comma is the field delimiter.
//
// If UseCRLF is true, the Writer ends each record with \r\n instead of \n.
type GpCsvWriter struct {
	Comma   rune // Field delimiter (set to ',' by NewWriter)
	UseCRLF bool // True to use \r\n as the line terminator

	// NOTE: Fields added by starschema
	// The character sequence to output when seeing a quote in a field
	QuoteEscaped string
	ForceQuotes  bool

	w *bufio.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewGpCsvWriter(w io.Writer) *GpCsvWriter {
	return &GpCsvWriter{
		Comma:        ',',
		QuoteEscaped: `\"`,
		ForceQuotes:  false,
		w:            bufio.NewWriter(w),
	}
}

// Writer writes a single CSV record to w along with any necessary quoting.
// A record is a slice of strings with each string being one field.
func (w *GpCsvWriter) Write(record []string) (err error) {
	for n, field := range record {
		if n > 0 {
			if _, err = w.w.WriteRune(w.Comma); err != nil {
				return
			}
		}

		// If we don't have to have a quoted field then just
		// write out the field and continue to the next field.
		if !w.fieldNeedsQuotes(field) {
			if _, err = w.w.WriteString(field); err != nil {
				return
			}
			continue
		}
		// NOTE: forceQuotes check is added by starschema
		if w.ForceQuotes {
			if err = w.w.WriteByte('"'); err != nil {
				return
			}
		}

		for _, r1 := range field {
			switch r1 {
			case '"':
				// NOTE: QuoteEscapeds is added by starschema
				_, err = w.w.WriteString(w.QuoteEscaped)
			case '\r':
				if !w.UseCRLF {
					err = w.w.WriteByte('\r')
				}
			case '\n':
				if w.UseCRLF {
					_, err = w.w.WriteString("\r\n")
				} else {
					err = w.w.WriteByte('\n')
				}
			default:
				_, err = w.w.WriteRune(r1)
			}
			if err != nil {
				return
			}
		}

		// NOTE: forceQuotes check is added by starschema
		if w.ForceQuotes {
			if err = w.w.WriteByte('"'); err != nil {
				return
			}
		}
	}
	if w.UseCRLF {
		_, err = w.w.WriteString("\r\n")
	} else {
		err = w.w.WriteByte('\n')
	}
	return
}

// Flush writes any buffered data to the underlying io.Writer.
// To check if an error occurred during the Flush, call Error.
func (w *GpCsvWriter) Flush() {
	w.w.Flush()
}

// Error reports any error that has occurred during a previous Write or Flush.
func (w *GpCsvWriter) Error() error {
	_, err := w.w.Write(nil)
	return err
}

// WriteAll writes multiple CSV records to w using Write and then calls Flush.
func (w *GpCsvWriter) WriteAll(records [][]string) (err error) {
	for _, record := range records {
		err = w.Write(record)
		if err != nil {
			return err
		}
	}
	return w.w.Flush()
}

// fieldNeedsQuotes reports whether our field must be enclosed in quotes.
// Fields with a Comma, fields with a quote or newline, and
// fields which start with a space must be enclosed in quotes.
// We used to quote empty strings, but we do not anymore (as of Go 1.4).
// The two representations should be equivalent, but Postgres distinguishes
// quoted vs non-quoted empty string during database imports, and it has
// an option to force the quoted behavior for non-quoted CSV but it has
// no option to force the non-quoted behavior for quoted CSV, making
// CSV with quoted empty strings strictly less useful.
// Not quoting the empty string also makes this package match the behavior
// of Microsoft Excel and Google Drive.
// For Postgres, quote the data terminating string `\.`.
func (w *GpCsvWriter) fieldNeedsQuotes(field string) bool {
	if field == "" {
		return false
	}
	if field == `\.` || strings.IndexRune(field, w.Comma) >= 0 || strings.IndexAny(field, "\"\r\n") >= 0 {
		return true
	}

	r1, _ := utf8.DecodeRuneInString(field)
	return unicode.IsSpace(r1)
}
