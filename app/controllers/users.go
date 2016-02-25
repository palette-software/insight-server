package controllers

import (
	"github.com/palette-software/insight-server/app/models"
	"github.com/revel/revel"
	//"io/ioutil"
	//"os"
	//"path"
	//"path/filepath"
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
