package models

import (
	"fmt"
	"github.com/go-gorp/gorp"
	"github.com/revel/revel"
	"golang.org/x/crypto/bcrypt"
	"regexp"
)

// Represents a client that -after successful authentication-
// can upload files to HomeDirectory
type Tenant struct {
	// The id in the database
	TenantId int
	// The full name of the tenant
	Name string
	// The username and password for the tenant to log in
	Username, Password string
	// The name of the directory where we'll save the files.
	// This allows us to use multiple tenant names working into
	// the same output directory if necessary
	HomeDirectory string
	// A BCrypted hash of the password for the tenant
	HashedPassword []byte
}

// Formats a tenant to a string for debuging
func (u *Tenant) String() string {
	return fmt.Sprintf("Tenant{Username:%s, Directory: %v}", u.Username, u.HomeDirectory)
}

// A regex for matching any non-whitespace character
var userRegex = regexp.MustCompile("^\\w*$")

// validator regex for the home directory name
var directoryRegex = regexp.MustCompile("^[a-z-_]+$")

// Validator function for a Tenant
// TODO: validate uniqieness of username
func (user *Tenant) Validate(v *revel.Validation) {
	v.Check(user.Username,
		revel.Required{},
		revel.MaxSize{15},
		revel.MinSize{4},
		revel.Match{userRegex},
	)

	ValidatePassword(v, user.Password).Key("user.Password")

	v.Check(user.Name,
		revel.Required{},
		revel.MaxSize{100},
	)

	v.Check(user.HomeDirectory,
		revel.Required{},
		revel.Match{directoryRegex},
	)
}

// Helper function to validate a password
func ValidatePassword(v *revel.Validation, password string) *revel.ValidationResult {
	return v.Check(password,
		revel.Required{},
		revel.MaxSize{25},
		revel.MinSize{5},
	)
}

// Creating tenants
///////////////////

//  Helper functino to create a new Tenant
func NewTenant(username, password, fullName string) *Tenant {
	return NewTenantWithHome(username, password, fullName, username)
}

// Creating tenants with passwords (deprecated)
///////////////////////////////////////////////

//  Helper functino to create a new Tenant
func NewTenantWithHome(username, password, fullName, homeDir string) *Tenant {
	bcryptPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	// make a valid homedir name out of the homeDir string
	homeDirName := SanitizeName(homeDir)

	return &Tenant{
		Name:           fullName,
		Username:       username,
		Password:       password,
		HomeDirectory:  homeDirName,
		HashedPassword: bcryptPassword,
	}
}

// Creates and saves a new tenant into the database
func CreateTenant(Dbm *gorp.DbMap, username, password, fullName, homeDir string) (*Tenant, error) {

	demoUser := NewTenantWithHome(username, password, fullName, homeDir)

	if err := Dbm.Insert(demoUser); err != nil {
		return nil, err
	}
	return demoUser, nil
}

// Creating tenants from license
////////////////////////////////

//  Helper functino to create a new Tenant with a token
func NewTenantWithHomeAndToken(username string, password []byte, fullName, homeDir string) *Tenant {
	bcryptPassword, _ := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	// make a valid homedir name out of the homeDir string
	homeDirName := SanitizeName(homeDir)

	return &Tenant{
		Name:           fullName,
		Username:       username,
		HomeDirectory:  homeDirName,
		HashedPassword: bcryptPassword,
	}
}

// Creates and saves a new tenant into the database
func CreateTenantWithToken(Dbm *gorp.DbMap, username string, password []byte, fullName string) (*Tenant, error) {

	// use the username as home directory for now
	demoUser := NewTenantWithHomeAndToken(username, password, fullName, username)

	if err := Dbm.Insert(demoUser); err != nil {
		return nil, err
	}
	return demoUser, nil
}

// Deletes a tenant from the database
func DeleteTenant(Dbm *gorp.DbMap, tenant *Tenant) {
	_, err := Dbm.Delete(tenant)
	if err != nil {
		panic(err)
	}
}

// Getting a tenant from pieces of information
//////////////////////////////////////////////

// Returns true if there is a tenant registered in the database with the given password
//func IsValidTenant(dbmap *gorp.DbMap, username, password string) (*Tenant, bool) {
func TenantFromAuthentication(dbmap *gorp.DbMap, username, password string) *Tenant {

	// try to load the user by username from the db
	tenant := Tenant{}
	// load the latest user if multiple users are present with the same username
	err := dbmap.SelectOne(&tenant, "select * from Tenant where username=? order by TenantId desc limit 1", username)

	if err != nil {
		revel.INFO.Printf("cannot load user: %v", err)
		return nil
	}

	// check the password hash
	checkResult := bcrypt.CompareHashAndPassword(tenant.HashedPassword, []byte(password))
	if checkResult == nil {
		return &tenant
	} else {
		return nil
	}
}

// Tries to find a tenant from the database via the username
func TenantFromHomeDirectory(dbmap *gorp.DbMap, homeDirName string) (*Tenant, error) {
	tenant := Tenant{}
	// find the tenant (limit the select results to 1, so selectOne wont fail)
	err := dbmap.SelectOne(&tenant, "select * from Tenant where HomeDirectory= order by TenantId desc limit 1", homeDirName)
	if err != nil {
		return nil, err
	}

	return &tenant, nil
}
