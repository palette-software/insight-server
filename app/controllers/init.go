package controllers

import "github.com/revel/revel"

func init() {
	revel.OnAppStart(InitDB)
	revel.InterceptMethod((*MigrationsController).Begin, revel.BEFORE)
	revel.InterceptMethod((*MigrationsController).Commit, revel.AFTER)
	revel.InterceptMethod((*MigrationsController).Rollback, revel.FINALLY)

	revel.InterceptMethod((*CsvUpload).CheckUserAuth, revel.BEFORE)
}
