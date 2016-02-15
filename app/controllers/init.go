package controllers

import "github.com/revel/revel"

func init() {
	revel.OnAppStart(InitDB)
	revel.InterceptMethod((*MigrationsController).Begin, revel.BEFORE)
	//revel.InterceptMethod(Application.AddUser, revel.BEFORE)
	//revel.InterceptMethod(Hotels.checkUser, revel.BEFORE)
	revel.InterceptMethod((*MigrationsController).Commit, revel.AFTER)
	revel.InterceptMethod((*MigrationsController).Rollback, revel.FINALLY)

	revel.InterceptMethod((*App).CheckUserAuth, revel.BEFORE)
}
