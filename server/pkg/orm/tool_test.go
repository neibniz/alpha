package orm

import (
	"testing"

	"github.com/neibniz/alpha/server/internal/model/auth"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestParseDSN(t *testing.T) {
	dsn := "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	config := ParseDSN(dsn)
	if config == nil {
		t.Fatal("config is nil")
	}
	t.Log(config)
}

func TestPing(t *testing.T) {
	t.Skip("requires a local PostgreSQL instance")
	dsn := "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	status := Ping(dsn)
	t.Log(status)
}

func TestPostgres(t *testing.T) {
	t.Skip("requires a local PostgreSQL instance")
	db, err := gorm.Open(postgres.Open("postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"))
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	err = db.Migrator().AutoMigrate(&auth.Auth{})
	if err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
		return
	}
	t.Log("PostgreSQL connection and migration successful")
}
