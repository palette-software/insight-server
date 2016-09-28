package insight_server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const licensingUrl = "https://licensing.palette-software.com/license"
const serverForm = "2006-01-02 15:04:05.000000"

type LicenseData struct {
	Trial          bool   `json:"trial"`
	ExpirationTime string `json:"expiration-time"`
	Id             int64  `json:"id"`
	Stage          string `json:"stage"`
	Owner          string `json:"owner"`
	Valid          bool   `json:"valid"`
}

func CheckLicense(licenseKey string) (bool, string) {
	data := url.Values{}
	// System-id is a required but not used parameter in licensing server
	data.Add("system-id", "1")
	data.Set("license-key", licenseKey)

	response, err := http.PostForm(licensingUrl, data)
	if err != nil || response.StatusCode != http.StatusOK {
		fmt.Printf("Err: %s\n", err)
		fmt.Printf("StatusCode: %s\n", response.StatusCode)
		return false, ""
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	var license LicenseData
	err = json.Unmarshal(buf.Bytes(), &license)
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		return false, ""
	}

	expirationTime, err := time.Parse(serverForm, license.ExpirationTime)
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		return false, ""
	}

	license.Owner, err = os.Hostname()
	fmt.Printf("Hostname: %s\n", license.Owner)
	if err != nil {
		fmt.Printf("Err: %s\n", err)
		return false, ""
	}

	license.Valid = time.Now().Before(expirationTime)
	fmt.Printf("Expiration time: %s\n", expirationTime)
	jsonResponse, err := json.Marshal(license)
	if err != nil {
		return false, ""
	}

	return license.Valid, string(jsonResponse)
}

func LicenseHandler(licenseKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		agentKey, err := getUrlParam(req.URL, "key")
		if err != nil || licenseKey != strings.ToLower(agentKey) {
			writeResponse(w, http.StatusNotFound, "")
			return
		}

		valid, license := CheckLicense(licenseKey)
		if !valid {
			writeResponse(w, http.StatusNotFound, "")
			return
		}

		writeResponse(w, http.StatusOK, license)
	}
}
