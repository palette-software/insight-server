package insight_server

import (
	"io"
	"encoding/csv"
	"fmt"
	"encoding/json"
	"strings"
	"log"
)

type ServerlogOuterJson struct {
	// dont bother with the timestamp, keep to original format
	Ts, Sev, Req, Sess, Site, User, K string
	V                                 interface{}
	Pid                               int
	Tid                               string
}

type ServerlogOutputRow struct {
	Filename, Hostname string

	Outer              ServerlogOuterJson
	Inner              string
}

type ErrorRow struct {
	Json  string
	Error string
}

func ParseServerlogs(r io.Reader) (rows []ServerlogOutputRow, errorRows []ErrorRow, err error) {

	csvReader := makeCsvReader(r)

	isHeader := true

	for {
		record, err := csvReader.Read()
		// in case of EOF we have finished
		if err == io.EOF {
			return rows, errorRows, nil
		}
		// if the CSV has errors, skip the whole file as we dont know
		// how to parse it
		if err != nil {
			return nil, nil, fmt.Errorf("Error during CSV parsing: %v", err)
		}
		// skip the header row
		if isHeader {
			isHeader = false
			continue
		}

		logRow := record[2]

		// try to parse the low row
		jsonDecoder := json.NewDecoder(strings.NewReader(logRow))
		outerJson := ServerlogOuterJson{}

		if err = jsonDecoder.Decode(&outerJson); err != nil {
			log.Println("[serverlogs.json] Parse error: %v", err)
			// put this row into the problematic ones
			errorRows = append(errorRows, ErrorRow{
				Json: logRow,
				Error: fmt.Sprintf("%v", err),
			})
			// skip this row from processing
			continue
		}

		// since the inner JSON can be anything, we unmarshal it into
		// a string, so the json marshaler can do his thing and we
		// dont have to care about what data is inside
		innerStr, err := json.Marshal(outerJson.V)
		if err != nil {
			log.Println("[serverlogs.json] Inner JSON remarshaling error: %v", err)
			// put this row into the problematic ones
			errorRows = append(errorRows, ErrorRow{
				Json: logRow,
				Error: fmt.Sprintf("%v", err),
			})
			// skip this row from processing
			continue
		}

		rows = append(rows, ServerlogOutputRow{
			Filename:  record[0],
			Hostname: record[1],
			Outer: outerJson,
			Inner: string(innerStr),
		})
	}

}


///////////////////////////////////

func makeCsvReader(r io.Reader) *csv.Reader {
	reader := csv.NewReader(r)
	reader.Comma = '\v'
	reader.FieldsPerRecord = 3
	reader.LazyQuotes = true
	return reader
}
