package migrate

import (
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/pkg/orm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestEmbeddedMigrationVersionsAreUnique(t *testing.T) {
	if _, err := iofs.New(sqlFiles, "database"); err != nil {
		t.Fatalf("embedded migrations are invalid: %v", err)
	}
}

func TestRegistrationMethodMigrationUsesExistingAuditTable(t *testing.T) {
	data, err := sqlFiles.ReadFile("database/02221_user_registration_method.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `FROM "`+(modelLog.SystemLog{}).TableName()+`"`) {
		t.Fatal("registration backfill must use the existing audit table")
	}
}

func getDSN() string {

	cfg := orm.Config{
		Addr:     "127.0.0.1:5432",
		Username: "postgres",
		Password: "postgres",
		Dbname:   "alpha",
	}
	mc := orm.Postgres{
		Config: cfg,
	}
	return mc.Dsn()
}

func TestMigrate(t *testing.T) {
	t.Skipf("skip test")
	m := Migrate(getDSN())
	err := m.Migrate(2004)
	if err != nil {
		t.Errorf("failed to migrate: %v", err)
	} else {
		t.Log("migrate success")
	}
}
func TestPostgres(t *testing.T) {
	t.Skip("requires a local PostgreSQL instance")
	db, err := gorm.Open(postgres.Open("postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"))
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	err = db.Migrator().AutoMigrate(&node.Node{})
	if err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
		return
	}
	t.Log("PostgreSQL connection and migration successful")
}
