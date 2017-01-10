package insight_server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/palette-software/insight-tester/common/logging"
)

const milliseclessServerForm = "2006-01-02 15:04:05"
const serverForm = "2006-01-02 15:04:05.000000"

var lastUpdatedAt = time.Now().AddDate(-1, 0, 0)
var cachedLicense *LicenseData

type LicenseData struct {
	Trial          bool   `json:"trial"`
	ExpirationTime string `json:"expiration-time"`
	Id             int64  `json:"id"`
	Stage          string `json:"stage"`
	Owner          string `json:"owner"`
	Name           string `json:"name"`
	Valid          bool   `json:"valid"`
}

// Cleared in way of open-sourcing
func UpdateLicense(licenseKey string) *LicenseData {
	var license LicenseData
	license.ExpirationTime = "9999-12-31 23:59:59.999999"
	license.Trial = false
	license.Id = 0
	license.Stage = "Free"
	license.Owner = "none"
	license.Name = "none"
	license.Valid = true

	return &license
}

func CheckLicense(license *LicenseData) (string, error) {
	if license == nil {
		return "", fmt.Errorf("Got nil license while checking license!")
	}

	expirationTime, err := time.Parse(serverForm, license.ExpirationTime)
	if err != nil {
		expirationTime, err = time.Parse(milliseclessServerForm, license.ExpirationTime)
		if err != nil {
			return "", err
		}
	}

	if license.Owner == "" {
		return "", fmt.Errorf("Owner of the license is empty!")
	}

	license.Valid = time.Now().Before(expirationTime)
	jsonResponse, err := json.Marshal(license)
	if err != nil {
		return "", err
	}

	return string(jsonResponse), nil
}

func LicenseHandler(licenseKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if cachedLicense == nil || time.Now().After(lastUpdatedAt.AddDate(0, 0, 1)) {
			cachedLicense = UpdateLicense(licenseKey)
			lastUpdatedAt = time.Now()
		}

		license, err := CheckLicense(cachedLicense)
		if err != nil {
			log.Errorf("Invalid or expired license, exiting: version=%s license=%s err=%s", GetVersion(), licenseKey, err)
			WriteResponse(w, http.StatusNotFound, "", req)
			return
		}

		WriteResponse(w, http.StatusOK, license, req)
	}
}
