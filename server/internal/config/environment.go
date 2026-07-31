package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type secretEnvironmentBinding struct {
	name   string
	target *string
}

// ApplyEnvironment overlays runtime values without requiring them in YAML.
// A NAME_FILE secret takes precedence by design, and using both forms is rejected.
func ApplyEnvironment(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := applyGeoIPEnvironment(cfg); err != nil {
		return err
	}
	bindings := []secretEnvironmentBinding{
		{name: "ALPHA_JWT_ACCESS_SECRET", target: &cfg.JwtAuth.AccessSecret},
		{name: "ALPHA_POSTGRES_PASSWORD", target: &cfg.Postgres.Password},
		{name: "ALPHA_REDIS_PASSWORD", target: &cfg.Redis.Pass},
		{name: "ALPHA_CLICKHOUSE_PASSWORD", target: &cfg.ClickHouse.Password},
		{name: "ALPHA_NODE_SECRET", target: &cfg.Node.NodeSecret},
		{name: "ALPHA_NODE_DEPLOY_SECRET", target: &cfg.Node.NodeDeploySecret},
		{name: "ALPHA_FORWARDER_BOOTSTRAP_SECRET", target: &cfg.Forwarder.BootstrapSecret},
		{name: "ALPHA_ADMIN_PASSWORD", target: &cfg.Administrator.Password},
	}
	for _, binding := range bindings {
		value, found, err := environmentSecret(binding.name)
		if err != nil {
			return err
		}
		if found {
			*binding.target = value
		}
	}
	return nil
}

func applyGeoIPEnvironment(cfg *Config) error {
	if value, found := os.LookupEnv("ALPHA_GEOIP_ENABLED"); found && strings.TrimSpace(value) != "" {
		enabled, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return fmt.Errorf("ALPHA_GEOIP_ENABLED must be a boolean: %w", err)
		}
		cfg.GeoIP.Enable = enabled
	}
	if value := strings.TrimSpace(os.Getenv("ALPHA_GEOIP_DATABASE_PATH")); value != "" {
		cfg.GeoIP.DatabasePath = value
	}
	return nil
}

func environmentSecret(name string) (string, bool, error) {
	direct, hasDirect := os.LookupEnv(name)
	filePath, hasFile := os.LookupEnv(name + "_FILE")
	direct = strings.TrimRight(direct, "\r\n")
	filePath = strings.TrimSpace(filePath)
	if hasDirect && direct != "" && hasFile && filePath != "" {
		return "", false, fmt.Errorf("%s and %s_FILE cannot both be set", name, name)
	}
	if hasFile && filePath != "" {
		content, err := os.ReadFile(filePath) // #nosec G304 -- path is supplied by the trusted process environment
		if err != nil {
			return "", false, fmt.Errorf("read %s_FILE: %w", name, err)
		}
		value := strings.TrimRight(string(content), "\r\n")
		if value == "" {
			return "", false, fmt.Errorf("%s_FILE is empty", name)
		}
		return value, true, nil
	}
	if hasDirect && direct != "" {
		return direct, true, nil
	}
	return "", false, nil
}
