package controllers

import (
	"github.com/palette-software/insight-server/app/models"
	"github.com/revel/revel"

	"strings"
)

// The users controller
type Users struct {
	*revel.Controller
}

// Creates a test user (deprecated)
func (c *Users) CreateTest() revel.Result {
	tenant, err := models.CreateTenant(Dbm, "test", "test", "Test User", "test")

	if err != nil {
		panic(err)
	}

	return c.RenderJson(tenant)
}

// Displays the create user from license form
func (c *Users) New() revel.Result {
	return c.Render()
}

// Creates a new user from a license.
func (c *Users) CreateFromLicense() revel.Result {
	var licenseText string
	c.Params.Bind(&licenseText, "license")

	// get the license string
	r := strings.NewReader(licenseText)
	license, err := models.ReadLicense(r)
	if err != nil {
		panic(err)
	}

	user, err := models.CreateTenantWithToken(Dbm, license.LicenseId, license.Token, license.Owner)

	if err != nil {
		panic(err)
	}

	// if the license is valid, save a new user
	return c.RenderJson(user)
}
