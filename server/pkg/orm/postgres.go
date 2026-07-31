package orm

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/neibniz/alpha/server/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type Config struct {
	Addr          string `yaml:"Addr"`
	Username      string `yaml:"Username"`
	Password      string `yaml:"Password"`
	Dbname        string `yaml:"Dbname"`
	Config        string `yaml:"Config" default:"sslmode=disable&TimeZone=Asia/Shanghai"`
	MaxIdleConns  int    `yaml:"MaxIdleConns" default:"10"`
	MaxOpenConns  int    `yaml:"MaxOpenConns" default:"10"`
	SlowThreshold int64  `yaml:"SlowThreshold" default:"1000"`
}

type Postgres struct {
	Config Config
}

func (p *Postgres) Dsn() string {
	values, _ := url.ParseQuery(p.Config.Config)
	if values.Get("sslmode") == "" {
		values.Set("sslmode", "disable")
	}
	if values.Get("TimeZone") == "" && values.Get("timezone") == "" {
		values.Set("TimeZone", "Asia/Shanghai")
	}
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.Config.Username, p.Config.Password),
		Host:     p.Config.Addr,
		Path:     "/" + strings.TrimPrefix(p.Config.Dbname, "/"),
		RawQuery: values.Encode(),
	}
	return u.String()
}

func (p *Postgres) GetSlowThreshold() time.Duration {
	return time.Duration(p.Config.SlowThreshold) * time.Millisecond
}

func (p *Postgres) GetColorful() bool {
	return true
}

func ConnectPostgres(p Postgres) (*gorm.DB, error) {
	if p.Config.Dbname == "" {
		return nil, errors.New("database name is empty")
	}

	db, err := openPostgres(p)
	if err == nil {
		return db, nil
	}
	if !isUndefinedDatabaseError(err) {
		return nil, err
	}
	if createErr := ensurePostgresDatabase(p); createErr != nil {
		return nil, fmt.Errorf("database %q does not exist and auto-create failed: %w; original connect error: %v", p.Config.Dbname, createErr, err)
	}
	return openPostgres(p)
}

func openPostgres(p Postgres) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(p.Dsn()), &gorm.Config{
		Logger: new(logger.GormLogger),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, err
	}
	sqldb, _ := db.DB()
	if err = sqldb.Ping(); err != nil {
		_ = sqldb.Close()
		return nil, err
	}
	sqldb.SetMaxIdleConns(p.Config.MaxIdleConns)
	sqldb.SetMaxOpenConns(p.Config.MaxOpenConns)
	return db, nil
}

func ensurePostgresDatabase(p Postgres) error {
	maintenance := p
	maintenance.Config.Dbname = "postgres"
	db, err := openPostgres(maintenance)
	if err != nil {
		return fmt.Errorf("connect maintenance database postgres failed: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	var exists bool
	if err = sqlDB.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, p.Config.Dbname).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err = sqlDB.Exec(fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, quoteIdentifier(p.Config.Dbname), quoteIdentifier(p.Config.Username))); err != nil {
		return err
	}
	return nil
}

func isUndefinedDatabaseError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "3D000"
	}
	return strings.Contains(err.Error(), "SQLSTATE 3D000")
}

func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func Ping(dsn string) bool {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Errorw("[Postgres] Connect failed", logger.Field("error", err.Error()))
		return false
	}
	sqlDB, _ := db.DB()
	return sqlDB.Ping() == nil
}
