package insight_server

import (
	"fmt"
	"path/filepath"
	"log"
	"os"

	"net/http"
	"bytes"
	"path"
)

const (
	// The key in ENV where the license files are looked up.
	// If this key isnt provided, the 'licenses' subdirectory of the working directory is used
	LicenseDirectoryKey = "INSIGHT_LICENSES_PATH"
	// The glob to use for getting the license files (relative to the path of the working directory)
	LicenseGlob = "*.license"
)

// PUBLIC API
// ==========

// Generates a http.HandlerFunc that calls innerHandler with with the additional parameter of
// the authenticated user.
//
// The users are gotten from the licenses list which is filled on startup.
//
// Checks the auth information from the request, and fails if it isnt there or the auth
// info does not correspond to the
func CheckUserAuth(internalHandler func(w http.ResponseWriter, req *http.Request, tenant *License)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		username, password, authOk := req.BasicAuth()
		if !authOk {
			logError(w, http.StatusForbidden, "[auth] No auth information provided in request")
			return
		}

		// check password / username and get the tenant
		tenant, err := authenticateTenant(username, []byte(password))
		if err != nil {
			logError(w, http.StatusForbidden, fmt.Sprintf("[auth] not a valid user: %v -- %v", username, err))
			return
		}

		internalHandler(w, req, tenant)
		return
	}
}

// LICENSES
// ========

type Licenses map[string]*License

// The map of licenseId -> License which we use for authentication.
var licenses Licenses

// Load all licenses from the license directory
func initLicenses() {
	licenses = loadAllLicenses()
}

// Loads all licenses from the license directory
func loadAllLicenses() Licenses {
	licenses := Licenses{}

	// get the licenses root directory from the env if possible
	licensesRoot := os.Getenv(LicenseDirectoryKey)
	if licensesRoot == "" {
		licensesRoot = path.Join(getCurrentPath(), "licenses")
	}

	// get a list of all files inside licenseRoot
	glob := filepath.Join(licensesRoot, LicenseGlob)
	files, _ := filepath.Glob(glob)
	fmt.Println(files) // contains a list of all files in the current directory

	for _, f := range files {
		licenseFile, err := os.Open(f)
		if err != nil {
			break
		}

		license, err := ReadLicense(licenseFile)
		if err != nil {
			log.Printf("[license] Error reading %v: %v", f, err)
		}
		licenses[license.LicenseId] = license
	}

	log.Printf("[license] Loaded %v licenses from %v.", len(licenses), glob)

	for _, tenant := range licenses {
		log.Printf("[license] user: '%v' username: '%v' home: '%v'", tenant.Owner, tenant.LicenseId, getUploadBasePath(tenant.LicenseId, ""))
	}

	return licenses
}

// Returns the authenticated tenant if the username and password are ok, or nil if not.
func authenticateTenant(username string, password []byte) (*License, error){
	// try to load the user by username from the db
	tenant, haveTenant := licenses[username]
	if !haveTenant {
		return nil, fmt.Errorf("Cannot find tenant named '%v'", username)
	}
	// check the password
	if !bytes.Equal(tenant.Token, password) {
		return nil, fmt.Errorf("Cannot authenticate tenant '%v'", username)
	}
	return tenant, nil
}

