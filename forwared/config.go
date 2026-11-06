package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	DataDir                   string `yaml:"data_dir"`
	GeoIPDownloadURL          string `yaml:"geoip_download_url"`
	GeoIPDownloadToken        string `yaml:"geoip_download_token"`
	GeoIPDownloadCode         string `yaml:"geoip_download_code"`
	GeoIPUpdateInterval       uint   `yaml:"geoip_update_interval"`
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
	var config Config
	if path == "" {
		for _, candidate := range []string{"forwared.yaml", "/app/config/forwared.yaml"} {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("配置读取失败: %w", err)
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("YAML解析失败: %w", err)
		}
	}
	for name, target := range map[string]*string{
		"BASE_URL": &config.BaseURL, "SECRET_KEY": &config.SecretKey,
		"NAME": &config.Name, "PUBLIC_IP": &config.PublicIP, "DATA_DIR": &config.DataDir,
		"IP2LOCATION_TOKEN": &config.GeoIPDownloadToken, "IP2LOCATION_CODE": &config.GeoIPDownloadCode,
		"GEOIP_URL": &config.GeoIPDownloadURL,
	} {
		if value, ok := os.LookupEnv("ALPHA_FORWARED_" + name); ok {
			*target = value
		}
	}
	if value, ok := os.LookupEnv("ALPHA_FORWARED_GEOIP_UPDATE_INTERVAL"); ok {
		seconds, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("ALPHA_FORWARED_GEOIP_UPDATE_INTERVAL must be a positive integer")
		}
		config.GeoIPUpdateInterval = uint(seconds)
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
	if c.DataDir == "" {
		c.DataDir = "/app/data"
	}
	if c.GeoIPFile == "" {
		c.GeoIPFile = filepath.Join(c.DataDir, "geoip", "IP2LOCATION-LITE-DB3.BIN")
	}
	if c.GeoIPDownloadCode == "" {
		c.GeoIPDownloadCode = "DB3LITEBINIPV6"
	}
	if c.GeoIPUpdateInterval == 0 {
		c.GeoIPUpdateInterval = 86400
	}
	if c.GeoIPUpdateInterval < 3600 || c.GeoIPUpdateInterval > 30*86400 {
		return fmt.Errorf("geoip_update_interval must be between 3600 and 2592000 seconds")
	}
	if c.GeoIPDownloadURL != "" {
		u, err := url.Parse(c.GeoIPDownloadURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
			return fmt.Errorf("geoip_download_url must be an HTTPS URL without userinfo or a fragment")
		}
	}
	if strings.TrimSpace(c.StateFile) == "" {
		c.StateFile = filepath.Join(c.DataDir, "identity.json")
	}
	if strings.TrimSpace(c.SpoolFile) == "" {
		c.SpoolFile = filepath.Join(c.DataDir, "connection-events.jsonl")
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
