package insight_server

import (
	"bytes"
	"net/http"
	"fmt"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"time"
)

type LicenseCheckResponse struct {
	Valid bool
	OwnerName string
}

func LicenseCheckHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//err := r.ParseMultipartForm(multipartMaxSize)
		//if err != nil {
		//	writeResponse(w, http.StatusBadRequest, fmt.Sprintf("Error while parsing multipart form: %v", err))
		//	return
		//}

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

		logrus.WithFields(logrus.Fields{
			"component":       "license check endpoint",
		}).Infof("Retrieved part's file name: %v", part.FileName())

		license, err := ReadLicense(part)
		if err != nil {
			writeResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to read license from multipart: %v! Error message: %v", part, err))
			return
		}

		checkResponse := LicenseCheckResponse{
			Valid: time.Now().Before(license.ValidUntilUTC),
			OwnerName: license.Owner,
		}

		wb := &bytes.Buffer{}
		if err := json.NewEncoder(wb).Encode(checkResponse); err != nil {
			writeResponse(w, http.StatusInternalServerError,
				fmt.Sprintf("Failed to encode license check response: %v", err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(wb.Bytes())
	}
}

