package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyEnvironmentLoadsDirectAndFileSecrets(t *testing.T) {
	dir := t.TempDir()
	jwtFile := filepath.Join(dir, "jwt")
	if err := os.WriteFile(jwtFile, []byte("jwt-from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALPHA_JWT_ACCESS_SECRET_FILE", jwtFile)
	t.Setenv("ALPHA_POSTGRES_PASSWORD", "postgres-from-env")
	t.Setenv("ALPHA_REDIS_PASSWORD", "redis-from-env\n")
	t.Setenv("ALPHA_CLICKHOUSE_PASSWORD", "clickhouse-from-env")
	t.Setenv("ALPHA_NODE_SECRET", "node-from-env")
	t.Setenv("ALPHA_NODE_DEPLOY_SECRET", "deploy-from-env")
	t.Setenv("ALPHA_ADMIN_PASSWORD", "admin-from-env")

	cfg := Config{}
	if err := ApplyEnvironment(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.JwtAuth.AccessSecret != "jwt-from-file" ||
		cfg.Postgres.Password != "postgres-from-env" ||
		cfg.Redis.Pass != "redis-from-env" ||
		cfg.ClickHouse.Password != "clickhouse-from-env" ||
		cfg.Node.NodeSecret != "node-from-env" ||
		cfg.Node.NodeDeploySecret != "deploy-from-env" ||
		cfg.Administrator.Password != "admin-from-env" {
		t.Fatalf("environment secrets were not applied: %#v", cfg)
	}
}

func TestApplyEnvironmentPreservesYAMLValuesWithoutOverrides(t *testing.T) {
	cfg := Config{}
	cfg.JwtAuth.AccessSecret = "yaml-secret"
	cfg.Postgres.Password = "yaml-password"
	if err := ApplyEnvironment(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.JwtAuth.AccessSecret != "yaml-secret" || cfg.Postgres.Password != "yaml-password" {
		t.Fatalf("YAML values changed without overrides: %#v", cfg)
	}
}

func TestApplyEnvironmentOverridesGeoIPSettings(t *testing.T) {
	t.Setenv("ALPHA_GEOIP_ENABLED", "true")
	t.Setenv("ALPHA_GEOIP_DATABASE_PATH", "/usr/share/alpha/geoip/dbip-city-lite.mmdb")

	cfg := Config{}
	if err := ApplyEnvironment(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.GeoIP.Enable || cfg.GeoIP.DatabasePath != "/usr/share/alpha/geoip/dbip-city-lite.mmdb" {
		t.Fatalf("GeoIP environment was not applied: %#v", cfg.GeoIP)
	}

	t.Setenv("ALPHA_GEOIP_ENABLED", "not-a-boolean")
	if err := ApplyEnvironment(&Config{}); err == nil || !strings.Contains(err.Error(), "must be a boolean") {
		t.Fatalf("invalid GeoIP boolean error = %v", err)
	}
}

func TestApplyEnvironmentRejectsAmbiguousOrEmptyFiles(t *testing.T) {
	t.Setenv("ALPHA_JWT_ACCESS_SECRET", "direct")
	t.Setenv("ALPHA_JWT_ACCESS_SECRET_FILE", filepath.Join(t.TempDir(), "jwt"))
	if err := ApplyEnvironment(&Config{}); err == nil || !strings.Contains(err.Error(), "cannot both be set") {
		t.Fatalf("ambiguous secret error = %v", err)
	}

	t.Setenv("ALPHA_JWT_ACCESS_SECRET", "")
	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ALPHA_JWT_ACCESS_SECRET_FILE", empty)
	if err := ApplyEnvironment(&Config{}); err == nil || !strings.Contains(err.Error(), "is empty") {
		t.Fatalf("empty secret error = %v", err)
	}
}
