package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	BaseURL                   string `yaml:"base_url"`   // server base URL, for example https://panel.example.com
	SecretKey                 string `yaml:"secret_key"` // server node secret
	PublicIP                  string `yaml:"public_ip"`
	PublicIPURL               string `yaml:"public_ip_url"`
	FetchIPInterval           uint   `yaml:"fetch_ip_interval"`
	FetchForwardTableInterval uint   `yaml:"fetch_forward_table_interval"`
	GeoIPFile                 string `yaml:"geoipfile"`
	LogFile                   string `yaml:"logfile"`
	Name                      string `yaml:"name"`
	StateFile                 string `yaml:"state_file"`
	SpoolFile                 string `yaml:"spool_file"`
	StatusInterval            uint   `yaml:"status_interval"`
	EventBatchSize            int    `yaml:"event_batch_size"`
	EventFlushInterval        uint   `yaml:"event_flush_interval"`
	MaxSpoolBytes             int64  `yaml:"max_spool_bytes"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("配置读取失败: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("YAML解析失败: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("base_url is required")
	}
	if len(strings.TrimSpace(c.SecretKey)) < 32 {
		return fmt.Errorf("secret_key must contain at least 32 characters")
	}
	if strings.TrimSpace(c.PublicIP) != "" && net.ParseIP(strings.TrimSpace(c.PublicIP)) == nil {
		return fmt.Errorf("public_ip must be a valid IP address")
	}
	if strings.TrimSpace(c.GeoIPFile) == "" {
		return fmt.Errorf("geoipfile is required")
	}
	if strings.TrimSpace(c.LogFile) == "" {
		return fmt.Errorf("logfile is required")
	}
	if strings.TrimSpace(c.StateFile) == "" {
		c.StateFile = "/app/state/identity.json"
	}
	if strings.TrimSpace(c.SpoolFile) == "" {
		c.SpoolFile = "/app/state/connection-events.jsonl"
	}
	if c.StatusInterval == 0 {
		c.StatusInterval = 5
	}
	if c.EventBatchSize <= 0 || c.EventBatchSize > 1000 {
		c.EventBatchSize = 500
	}
	if c.EventFlushInterval == 0 {
		c.EventFlushInterval = 5
	}
	if c.MaxSpoolBytes <= 0 {
		c.MaxSpoolBytes = 64 << 20
	}
	return nil
}
