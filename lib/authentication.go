package insight_server

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"

	"bytes"
	"net/http"
)

const (
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
	// returns a list of users available
	GetAllUsers() []User
}

// Generates a http.HandlerFunc that calls innerHandler with with the additional parameter of
// the authenticated user.
//
// The users are gotten from the license files loaded on startup.
func MakeUserAuthHandler(authenticator Authenticator, internalHandler HandlerFuncWithTenant) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		username, password, authOk := r.BasicAuth()
		if !authOk {
			writeResponse(w, http.StatusForbidden, "[auth] No auth information provided in request")
			return
		}

		// check password / username and get the tenant
		tenant, err := authenticator.authenticate(username, []byte(password))
		if err != nil {
			writeResponse(w, http.StatusForbidden, fmt.Sprintf("[auth] not a valid user: %v -- %v", username, err))
			return
		}

		internalHandler(w, r, tenant)
		return
	})
}

// LICENSES
// ========

type Licenses map[string]*License

//  Tries to load a license from a file
func loadLicenseFromFile(filename string) (*License, error) {

	licenseFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer licenseFile.Close()

	license, err := ReadLicense(licenseFile)
	if err != nil {
		return nil, err
	}
	return license, nil
}

// Loads all licenses from the license directory
func loadAllLicenses(licensesRoot string) Licenses {
	licenses := Licenses{}

	// get a list of all files inside licenseRoot
	glob := filepath.Join(licensesRoot, LicenseGlob)
	files, _ := filepath.Glob(glob)

	for _, f := range files {
		license, err := loadLicenseFromFile(f)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"component": "license",
				"file":      f,
				"error":     err,
			}).Warn("Error reading license")
			break
		}
		licenses[license.LicenseId] = license
	}

	logrus.WithFields(logrus.Fields{
		"component":     "license",
		"licensesCount": len(licenses),
		"glob":          glob,
	}).Info("Loaded licenses")

	for _, tenant := range licenses {
		logrus.WithFields(logrus.Fields{
			"component":  "license",
			"tenant":     tenant.Owner,
			"username":   tenant.LicenseId,
			"validUntil": tenant.ValidUntilUTC,
		}).Info("license available")
	}

	return licenses
}

// LICENSE AUTHENTICATOR
// =====================

// add User methods to License

func (l *License) GetUsername() string {
	return l.LicenseId
}

func (l *License) GetToken() []byte {
	return l.Token
}

type LicenseAuthenticator struct {
	licenses Licenses
}

// Creates a new license authenticator base on licenses from the given directory
func NewLicenseAuthenticator(licensesRoot string) Authenticator {
	return &LicenseAuthenticator{
		licenses: loadAllLicenses(licensesRoot),
	}
}

// Implements the authentication based on the licenses loaded from licensesRoot
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

// Returns a list of all users. Useful for figuring out the folders we care about
func (a *LicenseAuthenticator) GetAllUsers() []User {
	users := []User{}
	// since []User and []*License are not compatible, we re-create it here
	for _, license := range a.licenses {
		users = append(users, User(license))
	}

	return users
}