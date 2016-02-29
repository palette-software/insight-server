package controllers

import (
	"github.com/palette-software/insight-server/app/models"
	"github.com/revel/revel"

	"strings"
)

// The application controller itself
type Users struct {
	*revel.Controller
}

func (c *Users) CreateTest() revel.Result {
	tenant, err := models.CreateTenant(Dbm, "test", "test", "Test User", "test")

	if err != nil {
		panic(err)
	}

	return c.RenderJson(tenant)
}

func (c *Users) New() revel.Result {
	return c.Render()
}

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
