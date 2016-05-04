package insight_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"io"

	"github.com/Sirupsen/logrus"
)

type LicenseCheckResponse struct {
	Valid     bool
	OwnerName string
}

func LicenseCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// License file to be checked is expected as a multipart file
		mpr, err := r.MultipartReader()
		if err != nil {
			writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Error while getting multipart reader: %v", err))
			return
		}

		// We expect one and only one part in the multipart. And that part must be the license file.
		part, err := mpr.NextPart()
		if err != nil {
			writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to get next part: %v", err))
			return
		}

		// Create a buffer and a teeReader to read the raw license
		// into for logging
		licenseBuffer := &bytes.Buffer{}
		licenseReader := io.TeeReader(part, licenseBuffer)

		logrus.WithFields(logrus.Fields{
			"component": "licensecheck",
			"file":      part.FileName(),
		}).Debug("Retrieved part's file name")

		// Read the license from the teeReader
		license, err := ReadLicense(licenseReader)
		if err != nil {
			writeResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to read license from multipart: %v! Error message: %v", part, err))
			return
		}

		logrus.WithFields(logrus.Fields{
			"component":  "licensecheck",
			"file":       part.FileName(),
			"rawLicense": licenseBuffer,
		}).Debug("Read license from agent")

		checkResponse := LicenseCheckResponse{
			Valid:     time.Now().Before(license.ValidUntilUTC),
			OwnerName: license.Owner,
		}

		wb := &bytes.Buffer{}
		if err := json.NewEncoder(wb).Encode(checkResponse); err != nil {
			writeResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to encode license check response %v: %v", checkResponse, err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(wb.Bytes())
	}
}
