package insight_server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"
)

const licensingUrl = "https://licensing.palette-software.com/license"
const otherServerForm = "2006-01-02 15:04:05"
const serverForm = "2006-01-02 15:04:05.000000"

var lastUpdatedAt = time.Now().AddDate(-1, 0, 0)
var cachedLicense *LicenseData

type LicenseData struct {
	Trial          bool   `json:"trial"`
	ExpirationTime string `json:"expiration-time"`
	Id             int64  `json:"id"`
	Stage          string `json:"stage"`
	Owner          string `json:"owner"`
	Valid          bool   `json:"valid"`
}

func UpdateLicense(licenseKey string) *LicenseData {
	data := url.Values{}
	// System-id is a required but not used parameter in licensing server
	data.Add("system-id", "1")
	data.Set("license-key", licenseKey)

	response, err := http.PostForm(licensingUrl, data)
	if err != nil || response.StatusCode != http.StatusOK {
		return nil
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	var license LicenseData
	err = json.Unmarshal(buf.Bytes(), &license)
	if err != nil {
		return nil
	}

	return &license
}

func CheckLicense(licenseKey string, license *LicenseData) (string, error) {
	expirationTime, err := time.Parse(serverForm, license.ExpirationTime)
	if err != nil {
		expirationTime, err = time.Parse(otherServerForm, license.ExpirationTime)
		if err != nil {
			return "", err
		}
	}

	license.Owner, err = os.Hostname()
	if err != nil {
		return "", err
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

		license, err := CheckLicense(licenseKey, cachedLicense)
		if err != nil {
			WriteResponse(w, http.StatusNotFound, "")
			return
		}

		WriteResponse(w, http.StatusOK, license)
	}
}
