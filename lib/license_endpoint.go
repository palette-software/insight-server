package insight_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/palette-software/insight-tester/common/logging"
)

const licensingUrl = "https://licensing.palette-software.com/license"
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

func UpdateLicense(licenseKey string) *LicenseData {
	data := url.Values{}
	// System-id is a required but not used parameter in licensing server
	data.Add("system-id", "1")
	data.Set("license-key", licenseKey)

	response, err := http.PostForm(licensingUrl, data)
	if err != nil {
		log.Errorf("Posting license failed: license=%s err=%s", licenseKey, err)
		return nil
	}

	if response.StatusCode != http.StatusOK {
		log.Errorf("Updating license failed: license=%s status=%d err=%s", licenseKey, response.StatusCode, err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	var license LicenseData
	err = json.Unmarshal(buf.Bytes(), &license)
	if err != nil {
		log.Errorf("License update response is not a valid JSON: license=%s err=%s", licenseKey, err)
		return nil
	}

	// The licensing server response does not contain the license owner, but currently it should be
	// the same as the license name
	license.Owner = license.Name

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
