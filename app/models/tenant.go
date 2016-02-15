package models

import (
	"fmt"
	"github.com/go-gorp/gorp"
	"github.com/revel/revel"
	"golang.org/x/crypto/bcrypt"
	"regexp"
)

type Tenant struct {
	TenantId           int
	Name               string
	Username, Password string
	HashedPassword     []byte
}

func (u *Tenant) String() string {
	return fmt.Sprintf("Tenant(%s)", u.Username)
}

var userRegex = regexp.MustCompile("^\\w*$")

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
}

func ValidatePassword(v *revel.Validation, password string) *revel.ValidationResult {
	return v.Check(password,
		revel.Required{},
		revel.MaxSize{15},
		revel.MinSize{5},
	)
}

func NewTenant(username string, password string, fullName string) *Tenant {
	bcryptPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	revel.INFO.Printf("Password is: %v crypt: %v", password, bcryptPassword)

	return &Tenant{
		Name:           fullName,
		Username:       username,
		Password:       password,
		HashedPassword: bcryptPassword,
	}
}

// Creates and saves a new tenant into the database
func CreateTenant(Dbm *gorp.DbMap, username string, password string, fullName string) *Tenant {

	demoUser := NewTenant(username, password, fullName)

	if err := Dbm.Insert(demoUser); err != nil {
		panic(err)
	}
	return demoUser
}

// Deletes a tenant from the database
func DeleteTenant(Dbm *gorp.DbMap, tenant *Tenant) {
	_, err := Dbm.Delete(tenant)
	if err != nil {
		panic(err)
	}
}

// Returns true if there is a tenant registered in the database with the given password
func IsValidTenant(dbmap *gorp.DbMap, username string, password string) bool {

	// try to load the user by username from the db
	tenant := Tenant{}
	err := dbmap.SelectOne(&tenant, "select * from Tenant where username=?", username)

	if err != nil {
		revel.TRACE.Printf("cannot load user: %v", err)
		return false
	}

	// check the password hash
	checkResult := bcrypt.CompareHashAndPassword(tenant.HashedPassword, []byte(password))
	return checkResult == nil
}
