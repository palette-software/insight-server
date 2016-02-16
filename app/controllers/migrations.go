package controllers

import (
	"database/sql"
	"github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/palette-software/insight-webservice-go/app/models"
	"github.com/revel/modules/db/app"
	"github.com/revel/revel"
)

var (
	Dbm *gorp.DbMap
)

func InitDB() {
	db.Init()
	Dbm = &gorp.DbMap{Db: db.Db, Dialect: gorp.SqliteDialect{}}

	initTenantsTable(Dbm)

	// only show  query info when in dev mode
	if revel.DevMode {
		Dbm.TraceOn("[gorp]", revel.INFO)
	}
	Dbm.CreateTables()

}

// Table initializer scripts
// -------------------------

func setColumnSizes(t *gorp.TableMap, colSizes map[string]int) {
	for col, size := range colSizes {
		t.ColMap(col).MaxSize = size
	}
}

// initialilzes the tenants table
func initTenantsTable(Dbm *gorp.DbMap) {

	t := Dbm.AddTable(models.Tenant{}).SetKeys(true, "TenantId")

	// passwords are transient properties
	t.ColMap("Password").Transient = true

	// limit the data size
	setColumnSizes(t, map[string]int{
		"Username": 20,
		"Name":     100,
	})

}

// MIGRATIONS CONTROLLER USING GORP
// --------------------------------

type MigrationsController struct {
	*revel.Controller
	Txn *gorp.Transaction
}

func (c *MigrationsController) Begin() revel.Result {
	txn, err := Dbm.Begin()
	if err != nil {
		panic(err)
	}
	c.Txn = txn
	return nil
}

func (c *MigrationsController) Commit() revel.Result {
	if c.Txn == nil {
		return nil
	}
	if err := c.Txn.Commit(); err != nil && err != sql.ErrTxDone {
		panic(err)
	}
	c.Txn = nil
	return nil
}

func (c *MigrationsController) Rollback() revel.Result {
	if c.Txn == nil {
		return nil
	}
	if err := c.Txn.Rollback(); err != nil && err != sql.ErrTxDone {
		panic(err)
	}
	c.Txn = nil
	return nil
}
