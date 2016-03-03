package insight_server

import (
	"fmt"
	"path/filepath"
	"log"
	"os"

	"net/http"
	"bytes"
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

// a handler function taking a tenant
type HandlerFuncWithTenant func(w http.ResponseWriter, req *http.Request, user User)

// A possible implementation of a user
type User interface {
	GetUsername() string
	GetToken() []byte
}

// An interface that authenticates
type Authenticator interface {
	authenticate(username string, token []byte) (User, error)
}



// Generates a http.HandlerFunc that calls innerHandler with with the additional parameter of
// the authenticated user.
//
// The users are gotten from the licenses list which is filled on startup.
//
// Checks the auth information from the request, and fails if it isnt there or the auth
// info does not correspond to the
func MakeUserAuthHandler(authenticator Authenticator, internalHandler HandlerFuncWithTenant) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username, password, authOk := r.BasicAuth()
		if !authOk {
			logError(w, http.StatusForbidden, "[auth] No auth information provided in request")
			return
		}

		// check password / username and get the tenant
		tenant, err := authenticator.authenticate(username, []byte(password))
		if err != nil {
			logError(w, http.StatusForbidden, fmt.Sprintf("[auth] not a valid user: %v -- %v", username, err))
			return
		}

		internalHandler(w, r, tenant)
		return
	})
}

// LICENSES
// ========

type Licenses map[string]*License

//// The map of licenseId -> License which we use for authentication.
//var licenses Licenses

// Loads all licenses from the license directory
func loadAllLicenses(licensesRoot string) Licenses {
	licenses := Licenses{}

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
		log.Printf("[license] user: '%v' username: '%v'", tenant.Owner, tenant.LicenseId)
	}

	return licenses
}


// LICENSE AUTHENTICATOR
// =====================

// add User methods to License

func (l *License) GetUsername() string {
	return l.LicenseId;
}

func (l *License) GetToken() []byte {
	return l.Token;
}

type LicenseAuthenticator struct {
	licenses map[string]*License
}

// Creates a new license authenticator base on licenses from the given directory
func NewLicenseAuthenticator(licensesRoot string) Authenticator {
	return &LicenseAuthenticator{
		licenses: loadAllLicenses(licensesRoot),
	}
}


// Implements the authentication based on the licenses
func (a *LicenseAuthenticator) authenticate(username string, password []byte) (User, error) {

	// try to load the user by username from the db
	tenant, haveTenant := a.licenses[username]
	if !haveTenant {
		return nil, fmt.Errorf("Cannot find tenant named '%v'", username)
	}
	// check the password
	if !bytes.Equal(tenant.GetToken(), password) {
		return nil, fmt.Errorf("Cannot authenticate tenant '%v'", username)
	}
	return tenant, nil
}