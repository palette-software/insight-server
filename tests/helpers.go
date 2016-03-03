package tests

import (
	"github.com/go-gorp/gorp"
	"github.com/palette-software/insight-server/app/models"
)

const (
	testPkg      = "testPkg"

	testTenantUsername = "testTenant"
	testTenantPassword = "testTenantPw"

)



// Creates a test tenant with the username testTenantUsername and password/token of testTenantPassword
func makeTestTenant(Dbm *gorp.DbMap) *models.Tenant {
	// create a tenant for the test run
	tenant, err := models.CreateTenant(Dbm, testTenantUsername, testTenantPassword, "Test User", testTenantUsername)

	if err != nil {
		panic(err)
	}
	return tenant
}

// Deletes a tenant by id
func deleteTestTenant(Dbm *gorp.DbMap, tenant *models.Tenant) {
	// delete the existing test tenant, so the DB stays relatively clean
	models.DeleteTenant(Dbm, tenant)
}
