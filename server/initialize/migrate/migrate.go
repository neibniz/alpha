package migrate

import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/neibniz/alpha/server/pkg/logger"
)

//go:embed database/*.sql
var sqlFiles embed.FS
var NoChange = migrate.ErrNoChange

func Migrate(dsn string) *migrate.Migrate {
	d, err := iofs.New(sqlFiles, "database")
	if err != nil {
		logger.Errorf("[Migrate] iofs.New error: %v", err.Error())
		panic(err)
	}
	client, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		logger.Errorf("[Migrate] NewWithSourceInstance error: %v", err.Error())
		panic(err)
	}
	return client
}
