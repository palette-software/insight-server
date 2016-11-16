package insight_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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
	Valid          bool   `json:"valid"`
}

func UpdateLicense(licenseKey string) *LicenseData {
	data := url.Values{}
	// System-id is a required but not used parameter in licensing server
	data.Add("system-id", "1")
	data.Set("license-key", licenseKey)

	response, err := http.PostForm(licensingUrl, data)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"license": licenseKey,
		}).Error("Posting license failed!")
		return nil
	}

	if response.StatusCode != http.StatusOK {
		logrus.WithError(err).WithFields(logrus.Fields{
			"license": licenseKey,
			"status code": response.StatusCode,
		}).Error("Updating license failed!")
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	var license LicenseData
	err = json.Unmarshal(buf.Bytes(), &license)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"license": licenseKey,
		}).Error("License update response is not a valid JSON!")
		return nil
	}

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

	license.Owner, err = GetLicenseOwner()
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

		license, err := CheckLicense(cachedLicense)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"version": GetVersion(),
				"license": licenseKey,
			}).Error("Invalid or expired license, exiting.")

			WriteResponse(w, http.StatusNotFound, "")
			return
		}

		WriteResponse(w, http.StatusOK, license)
	}
}

func GetLicenseOwner() (string, error) {
	owner, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("Failed to get hostname to prepare license owner name! Error: %v", err)
	}

	owner = strings.TrimSuffix(owner, "-insight")
	return strings.ToUpper(owner), nil
}
