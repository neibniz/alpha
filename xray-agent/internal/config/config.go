package config

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const Version = "0.1.0"

const defaultXrayAPIPort = 10085

type Config struct {
	Server ServerConfig `json:"server" yaml:"server"`
	Panel  ServerConfig `json:"panel" yaml:"panel"`
	Xray   XrayConfig   `json:"xray" yaml:"xray"`
	Agent  AgentConfig  `json:"agent" yaml:"agent"`
	Report ReportConfig `json:"report" yaml:"report"`
	Log    LogConfig    `json:"log" yaml:"log"`
}

type ServerConfig struct {
	BaseURL   string `json:"base_url" yaml:"base_url"`
	ServerID  int64  `json:"server_id" yaml:"server_id"`
	SecretKey string `json:"secret_key" yaml:"secret_key"`
	Token     string `json:"token" yaml:"token"`
}

type PanelConfig = ServerConfig

type XrayConfig struct {
	Executable      string `json:"executable" yaml:"executable"`
	BinPath         string `json:"bin_path" yaml:"bin_path"`
	WorkDir         string `json:"work_dir" yaml:"work_dir"`
	ConfigPath      string `json:"config_path" yaml:"config_path"`
	ConfigFile      string `json:"config_file" yaml:"config_file"`
	LastGoodPath    string `json:"last_good_path" yaml:"last_good_path"`
	LastGoodFile    string `json:"last_good_file" yaml:"last_good_file"`
	StatsCursorPath string `json:"stats_cursor_path" yaml:"stats_cursor_path"`
	StatsCursorFile string `json:"stats_cursor_file" yaml:"stats_cursor_file"`
	APIAddress      string `json:"api_address" yaml:"api_address"`
	APIPort         int    `json:"api_port" yaml:"api_port"`
	InjectAPI       bool   `json:"inject_api" yaml:"inject_api"`
	ValidateConfig  bool   `json:"validate_config" yaml:"validate_config"`
}

type AgentConfig struct {
	PullInterval     time.Duration `json:"-"`
	PullIntervalText string        `json:"pull_interval" yaml:"pull_interval"`
}

type ReportConfig struct {
	StatusInterval      time.Duration       `json:"-"`
	StatusIntervalText  string              `json:"status_interval" yaml:"status_interval"`
	TrafficInterval     time.Duration       `json:"-"`
	TrafficIntervalText string              `json:"traffic_interval" yaml:"traffic_interval"`
	InterfaceName       string              `json:"interface_name" yaml:"interface_name"`
	Traffic             TrafficReportConfig `json:"traffic" yaml:"traffic"`
	Access              AccessReportConfig  `json:"access" yaml:"access"`
}

type TrafficReportConfig struct {
	BatchSize         int           `json:"batch_size" yaml:"batch_size"`
	FlushInterval     time.Duration `json:"-"`
	FlushIntervalText string        `json:"flush_interval" yaml:"flush_interval"`
	MaxSpoolBytes     int64         `json:"max_spool_bytes" yaml:"max_spool_bytes"`
	SpoolPath         string        `json:"spool_path" yaml:"spool_path"`
	StatePath         string        `json:"state_path" yaml:"state_path"`
}

type AccessReportConfig struct {
	Enabled           bool          `json:"enabled" yaml:"enabled"`
	BatchSize         int           `json:"batch_size" yaml:"batch_size"`
	FlushInterval     time.Duration `json:"-"`
	FlushIntervalText string        `json:"flush_interval" yaml:"flush_interval"`
	PollInterval      time.Duration `json:"-"`
	PollIntervalText  string        `json:"poll_interval" yaml:"poll_interval"`
	MaxBatchBytes     int64         `json:"max_batch_bytes" yaml:"max_batch_bytes"`
	MaxSpoolBytes     int64         `json:"max_spool_bytes" yaml:"max_spool_bytes"`
	SpoolPath         string        `json:"spool_path" yaml:"spool_path"`
	StatePath         string        `json:"state_path" yaml:"state_path"`
}

type LogConfig struct {
	AgentPath      string `json:"agent_path" yaml:"agent_path"`
	XrayErrorPath  string `json:"xray_error_path" yaml:"xray_error_path"`
	XrayAccessPath string `json:"xray_access_path" yaml:"xray_access_path"`
	XrayLevel      string `json:"xray_level" yaml:"xray_level"`
}

func Load() (Config, error) {
	var configPath string
	flag.StringVar(&configPath, "config", getenv("XRAY_AGENT_CONFIG", ""), "config file path")
	flag.Parse()

	cfg := Default()
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return cfg, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
	} else if defaultPath := firstExistingConfig(); defaultPath != "" {
		data, err := os.ReadFile(defaultPath)
		if err != nil {
			return cfg, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
	}
	applyEnv(&cfg)
	return normalize(cfg)
}

func Default() Config {
	workDir := getenv("XRAY_AGENT_WORK_DIR", "/var/lib/alpha/xray-agent")
	return Config{
		Server: ServerConfig{
			BaseURL:  getenv("ALPHA_SERVER_URL", "http://127.0.0.1:27001"),
			ServerID: getenvInt64("ALPHA_SERVER_ID", 0),
			Token:    getenv("ALPHA_NODE_TOKEN", os.Getenv("ALPHA_NODE_SECRET")),
		},
		Xray: XrayConfig{
			Executable:      getenv("XRAY_BIN", "/usr/local/bin/xray"),
			WorkDir:         workDir,
			ConfigPath:      filepath.Join(workDir, "current.json"),
			LastGoodPath:    filepath.Join(workDir, "last_good.json"),
			StatsCursorPath: filepath.Join(workDir, "stats_cursor.json"),
			APIAddress:      getenv("XRAY_API_ADDRESS", "127.0.0.1:10085"),
			APIPort:         getenvInt("XRAY_API_PORT", defaultXrayAPIPort),
			InjectAPI:       getenvBool("XRAY_AGENT_INJECT_API", true),
			ValidateConfig:  getenvBool("XRAY_AGENT_VALIDATE_CONFIG", true),
		},
		Agent: AgentConfig{PullIntervalText: getenv("XRAY_AGENT_PULL_INTERVAL", "30s")},
		Report: ReportConfig{
			StatusIntervalText:  getenv("XRAY_AGENT_STATUS_INTERVAL", "5s"),
			TrafficIntervalText: getenv("XRAY_AGENT_TRAFFIC_INTERVAL", "5s"),
			InterfaceName:       os.Getenv("XRAY_AGENT_INTERFACE"),
			Traffic: TrafficReportConfig{
				BatchSize:         getenvInt("XRAY_TRAFFIC_BATCH_SIZE", 12),
				FlushIntervalText: getenv("XRAY_TRAFFIC_FLUSH_INTERVAL", "5s"),
				MaxSpoolBytes:     getenvInt64("XRAY_TRAFFIC_MAX_SPOOL_BYTES", 64<<20),
				SpoolPath:         os.Getenv("XRAY_TRAFFIC_SPOOL_PATH"),
				StatePath:         os.Getenv("XRAY_TRAFFIC_STATE_PATH"),
			},
			Access: AccessReportConfig{
				Enabled:           getenvBool("XRAY_ACCESS_REPORT_ENABLED", true),
				BatchSize:         getenvInt("XRAY_ACCESS_BATCH_SIZE", 1000),
				FlushIntervalText: getenv("XRAY_ACCESS_FLUSH_INTERVAL", "5s"),
				PollIntervalText:  getenv("XRAY_ACCESS_POLL_INTERVAL", "1s"),
				MaxBatchBytes:     getenvInt64("XRAY_ACCESS_MAX_BATCH_BYTES", 1<<20),
				MaxSpoolBytes:     getenvInt64("XRAY_ACCESS_MAX_SPOOL_BYTES", 256<<20),
				SpoolPath:         os.Getenv("XRAY_ACCESS_SPOOL_PATH"),
				StatePath:         os.Getenv("XRAY_ACCESS_STATE_PATH"),
			},
		},
		Log: LogConfig{
			AgentPath:      os.Getenv("XRAY_AGENT_LOG_PATH"),
			XrayErrorPath:  os.Getenv("XRAY_ERROR_LOG_PATH"),
			XrayAccessPath: os.Getenv("XRAY_ACCESS_LOG_PATH"),
			XrayLevel:      os.Getenv("XRAY_LOG_LEVEL"),
		},
	}
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("ALPHA_SERVER_URL"); value != "" {
		cfg.Server.BaseURL = value
	}
	if value := getenvInt64("ALPHA_SERVER_ID", 0); value > 0 {
		cfg.Server.ServerID = value
	}
	if value := getenv("ALPHA_NODE_TOKEN", os.Getenv("ALPHA_NODE_SECRET")); value != "" {
		cfg.Server.Token = value
	}
	if value := os.Getenv("XRAY_BIN"); value != "" {
		cfg.Xray.Executable = value
	}
	if value := os.Getenv("XRAY_AGENT_WORK_DIR"); value != "" {
		cfg.Xray.WorkDir = value
	}
	if value := os.Getenv("XRAY_API_ADDRESS"); value != "" {
		cfg.Xray.APIAddress = value
	}
	if value := getenvInt("XRAY_API_PORT", 0); value > 0 {
		cfg.Xray.APIPort = value
	}
	if value := os.Getenv("XRAY_AGENT_INJECT_API"); value != "" {
		cfg.Xray.InjectAPI = getenvBool("XRAY_AGENT_INJECT_API", cfg.Xray.InjectAPI)
	}
	if value := os.Getenv("XRAY_AGENT_STATUS_INTERVAL"); value != "" {
		cfg.Report.StatusIntervalText = value
	}
	if value := os.Getenv("XRAY_AGENT_TRAFFIC_INTERVAL"); value != "" {
		cfg.Report.TrafficIntervalText = value
	}
	if value := os.Getenv("XRAY_AGENT_LOG_PATH"); value != "" {
		cfg.Log.AgentPath = value
	}
	if value := os.Getenv("XRAY_ERROR_LOG_PATH"); value != "" {
		cfg.Log.XrayErrorPath = value
	}
	if value := os.Getenv("XRAY_ACCESS_LOG_PATH"); value != "" {
		cfg.Log.XrayAccessPath = value
	}
	if value := os.Getenv("XRAY_LOG_LEVEL"); value != "" {
		cfg.Log.XrayLevel = value
	}
	if value := getenvInt("XRAY_TRAFFIC_BATCH_SIZE", 0); value > 0 {
		cfg.Report.Traffic.BatchSize = value
	}
	if value := os.Getenv("XRAY_TRAFFIC_FLUSH_INTERVAL"); value != "" {
		cfg.Report.Traffic.FlushIntervalText = value
	}
	if value := getenvInt64("XRAY_TRAFFIC_MAX_SPOOL_BYTES", 0); value > 0 {
		cfg.Report.Traffic.MaxSpoolBytes = value
	}
	if value := os.Getenv("XRAY_TRAFFIC_SPOOL_PATH"); value != "" {
		cfg.Report.Traffic.SpoolPath = value
	}
	if value := os.Getenv("XRAY_TRAFFIC_STATE_PATH"); value != "" {
		cfg.Report.Traffic.StatePath = value
	}
	if value := os.Getenv("XRAY_ACCESS_REPORT_ENABLED"); value != "" {
		cfg.Report.Access.Enabled = getenvBool("XRAY_ACCESS_REPORT_ENABLED", cfg.Report.Access.Enabled)
	}
	if value := getenvInt("XRAY_ACCESS_BATCH_SIZE", 0); value > 0 {
		cfg.Report.Access.BatchSize = value
	}
	if value := os.Getenv("XRAY_ACCESS_FLUSH_INTERVAL"); value != "" {
		cfg.Report.Access.FlushIntervalText = value
	}
	if value := os.Getenv("XRAY_ACCESS_POLL_INTERVAL"); value != "" {
		cfg.Report.Access.PollIntervalText = value
	}
	if value := getenvInt64("XRAY_ACCESS_MAX_BATCH_BYTES", 0); value > 0 {
		cfg.Report.Access.MaxBatchBytes = value
	}
	if value := getenvInt64("XRAY_ACCESS_MAX_SPOOL_BYTES", 0); value > 0 {
		cfg.Report.Access.MaxSpoolBytes = value
	}
	if value := os.Getenv("XRAY_ACCESS_SPOOL_PATH"); value != "" {
		cfg.Report.Access.SpoolPath = value
	}
	if value := os.Getenv("XRAY_ACCESS_STATE_PATH"); value != "" {
		cfg.Report.Access.StatePath = value
	}
}

func normalize(cfg Config) (Config, error) {
	cfg = normalizeLegacyFields(cfg)
	if cfg.Server.ServerID <= 0 {
		return cfg, fmt.Errorf("server.server_id is required")
	}
	if cfg.Server.Token == "" {
		return cfg, fmt.Errorf("server.token is required")
	}
	if cfg.Xray.WorkDir == "" {
		cfg.Xray.WorkDir = "/var/lib/alpha/xray-agent"
	}
	if cfg.Xray.Executable == "" {
		cfg.Xray.Executable = "/usr/local/bin/xray"
	}
	if cfg.Xray.ConfigPath == "" {
		cfg.Xray.ConfigPath = filepath.Join(cfg.Xray.WorkDir, "current.json")
	}
	if cfg.Xray.LastGoodPath == "" {
		cfg.Xray.LastGoodPath = filepath.Join(cfg.Xray.WorkDir, "last_good.json")
	}
	if cfg.Xray.StatsCursorPath == "" {
		cfg.Xray.StatsCursorPath = filepath.Join(cfg.Xray.WorkDir, "stats_cursor.json")
	}
	if cfg.Xray.APIPort == 0 {
		cfg.Xray.APIPort = defaultXrayAPIPort
	}
	if cfg.Xray.APIAddress == "" ||
		(cfg.Xray.APIAddress == fmt.Sprintf("127.0.0.1:%d", defaultXrayAPIPort) &&
			cfg.Xray.APIPort != defaultXrayAPIPort) {
		cfg.Xray.APIAddress = fmt.Sprintf("127.0.0.1:%d", cfg.Xray.APIPort)
	}
	if cfg.Agent.PullIntervalText == "" {
		cfg.Agent.PullIntervalText = "30s"
	}
	if cfg.Report.StatusIntervalText == "" {
		cfg.Report.StatusIntervalText = "5s"
	}
	if cfg.Report.TrafficIntervalText == "" {
		cfg.Report.TrafficIntervalText = "5s"
	}
	if cfg.Report.Traffic.BatchSize <= 0 {
		cfg.Report.Traffic.BatchSize = 12
	}
	if cfg.Report.Traffic.BatchSize > 120 {
		cfg.Report.Traffic.BatchSize = 120
	}
	if cfg.Report.Traffic.FlushIntervalText == "" {
		cfg.Report.Traffic.FlushIntervalText = "5s"
	}
	if cfg.Report.Traffic.MaxSpoolBytes <= 0 {
		cfg.Report.Traffic.MaxSpoolBytes = 64 << 20
	}
	if cfg.Report.Traffic.SpoolPath == "" {
		cfg.Report.Traffic.SpoolPath = filepath.Join(cfg.Xray.WorkDir, "traffic_spool.jsonl")
	} else {
		cfg.Report.Traffic.SpoolPath = normalizeOptionalPath(cfg.Xray.WorkDir, cfg.Report.Traffic.SpoolPath)
	}
	if cfg.Report.Traffic.StatePath == "" {
		cfg.Report.Traffic.StatePath = filepath.Join(cfg.Xray.WorkDir, "traffic_state.json")
	} else {
		cfg.Report.Traffic.StatePath = normalizeOptionalPath(cfg.Xray.WorkDir, cfg.Report.Traffic.StatePath)
	}
	if cfg.Report.Access.BatchSize <= 0 {
		cfg.Report.Access.BatchSize = 1000
	}
	if cfg.Report.Access.BatchSize > 2000 {
		cfg.Report.Access.BatchSize = 2000
	}
	if cfg.Report.Access.FlushIntervalText == "" {
		cfg.Report.Access.FlushIntervalText = "5s"
	}
	if cfg.Report.Access.PollIntervalText == "" {
		cfg.Report.Access.PollIntervalText = "1s"
	}
	if cfg.Report.Access.MaxBatchBytes <= 0 {
		cfg.Report.Access.MaxBatchBytes = 1 << 20
	}
	if cfg.Report.Access.MaxSpoolBytes <= 0 {
		cfg.Report.Access.MaxSpoolBytes = 256 << 20
	}
	if cfg.Report.Access.SpoolPath == "" {
		cfg.Report.Access.SpoolPath = filepath.Join(cfg.Xray.WorkDir, "access_spool.jsonl")
	} else {
		cfg.Report.Access.SpoolPath = normalizeOptionalPath(cfg.Xray.WorkDir, cfg.Report.Access.SpoolPath)
	}
	if cfg.Report.Access.StatePath == "" {
		cfg.Report.Access.StatePath = filepath.Join(cfg.Xray.WorkDir, "access_state.json")
	} else {
		cfg.Report.Access.StatePath = normalizeOptionalPath(cfg.Xray.WorkDir, cfg.Report.Access.StatePath)
	}
	var err error
	cfg.Agent.PullInterval, err = time.ParseDuration(cfg.Agent.PullIntervalText)
	if err != nil {
		return cfg, fmt.Errorf("agent.pull_interval: %w", err)
	}
	cfg.Report.StatusInterval, err = time.ParseDuration(cfg.Report.StatusIntervalText)
	if err != nil {
		return cfg, fmt.Errorf("report.status_interval: %w", err)
	}
	cfg.Report.TrafficInterval, err = time.ParseDuration(cfg.Report.TrafficIntervalText)
	if err != nil {
		return cfg, fmt.Errorf("report.traffic_interval: %w", err)
	}
	if cfg.Report.TrafficInterval < time.Second || cfg.Report.TrafficInterval > time.Hour {
		return cfg, fmt.Errorf("report.traffic_interval must be between 1s and 1h")
	}
	cfg.Report.Traffic.FlushInterval, err = time.ParseDuration(cfg.Report.Traffic.FlushIntervalText)
	if err != nil {
		return cfg, fmt.Errorf("report.traffic.flush_interval: %w", err)
	}
	if cfg.Report.Traffic.FlushInterval <= 0 {
		return cfg, fmt.Errorf("report.traffic.flush_interval must be positive")
	}
	cfg.Report.Access.FlushInterval, err = time.ParseDuration(cfg.Report.Access.FlushIntervalText)
	if err != nil {
		return cfg, fmt.Errorf("report.access.flush_interval: %w", err)
	}
	if cfg.Report.Access.FlushInterval <= 0 {
		return cfg, fmt.Errorf("report.access.flush_interval must be positive")
	}
	cfg.Report.Access.PollInterval, err = time.ParseDuration(cfg.Report.Access.PollIntervalText)
	if err != nil {
		return cfg, fmt.Errorf("report.access.poll_interval: %w", err)
	}
	if cfg.Report.Access.PollInterval <= 0 {
		return cfg, fmt.Errorf("report.access.poll_interval must be positive")
	}
	if err := os.MkdirAll(cfg.Xray.WorkDir, 0o755); err != nil {
		return cfg, err
	}
	cfg.Log = normalizeLogConfig(cfg.Log, cfg.Xray.WorkDir)
	if err := ensureLogDirs(cfg.Log); err != nil {
		return cfg, err
	}
	for _, path := range []string{
		cfg.Report.Traffic.SpoolPath,
		cfg.Report.Traffic.StatePath,
		cfg.Report.Access.SpoolPath,
		cfg.Report.Access.StatePath,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return cfg, err
		}
	}
	cfg.Xray.BinPath = cfg.Xray.Executable
	cfg.Xray.ConfigFile = cfg.Xray.ConfigPath
	cfg.Xray.LastGoodFile = cfg.Xray.LastGoodPath
	cfg.Xray.StatsCursorFile = cfg.Xray.StatsCursorPath
	cfg.Panel = cfg.Server
	return cfg, nil
}

func normalizeLegacyFields(cfg Config) Config {
	if cfg.Server.BaseURL == "" && cfg.Panel.BaseURL != "" {
		cfg.Server.BaseURL = cfg.Panel.BaseURL
	}
	if cfg.Server.ServerID == 0 && cfg.Panel.ServerID != 0 {
		cfg.Server.ServerID = cfg.Panel.ServerID
	}
	if cfg.Server.Token == "" && cfg.Panel.Token != "" {
		cfg.Server.Token = cfg.Panel.Token
	}
	if cfg.Server.Token == "" && cfg.Server.SecretKey != "" {
		cfg.Server.Token = cfg.Server.SecretKey
	}
	if cfg.Server.Token == "" && cfg.Panel.SecretKey != "" {
		cfg.Server.Token = cfg.Panel.SecretKey
	}
	if cfg.Xray.Executable == "" && cfg.Xray.BinPath != "" {
		cfg.Xray.Executable = cfg.Xray.BinPath
	}
	if cfg.Xray.ConfigPath == "" && cfg.Xray.ConfigFile != "" {
		cfg.Xray.ConfigPath = cfg.Xray.ConfigFile
	}
	if cfg.Xray.LastGoodPath == "" && cfg.Xray.LastGoodFile != "" {
		cfg.Xray.LastGoodPath = cfg.Xray.LastGoodFile
	}
	if cfg.Xray.StatsCursorPath == "" && cfg.Xray.StatsCursorFile != "" {
		cfg.Xray.StatsCursorPath = cfg.Xray.StatsCursorFile
	}
	if port, ok := apiAddressPort(cfg.Xray.APIAddress); ok &&
		(cfg.Xray.APIPort == 0 ||
			(cfg.Xray.APIPort == defaultXrayAPIPort && port != defaultXrayAPIPort)) {
		cfg.Xray.APIPort = port
	}
	return cfg
}

func normalizeLogConfig(logCfg LogConfig, workDir string) LogConfig {
	logCfg.AgentPath = normalizeOptionalPath(workDir, logCfg.AgentPath)
	logCfg.XrayErrorPath = normalizeOptionalPath(workDir, logCfg.XrayErrorPath)
	logCfg.XrayAccessPath = normalizeOptionalPath(workDir, logCfg.XrayAccessPath)
	return logCfg
}

func normalizeOptionalPath(baseDir, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

func ensureLogDirs(logCfg LogConfig) error {
	for _, path := range []string{logCfg.AgentPath, logCfg.XrayErrorPath, logCfg.XrayAccessPath} {
		if path == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func apiAddressPort(address string) (int, bool) {
	if address == "" {
		return 0, false
	}
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		_, portText, _ = strings.Cut(address, ":")
		if portText == "" {
			portText = address
		}
	}
	port, err := strconv.Atoi(portText)
	return port, err == nil && port > 0
}

func firstExistingConfig() string {
	for _, path := range []string{
		"/etc/alpha/xray-agent.yaml",
		"/etc/alpha/xray-agent.yml",
		"/etc/alpha/xray-agent.json",
	} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	var value int
	if _, err := fmt.Sscanf(os.Getenv(key), "%d", &value); err == nil {
		return value
	}
	return fallback
}

func getenvInt64(key string, fallback int64) int64 {
	var value int64
	if _, err := fmt.Sscanf(os.Getenv(key), "%d", &value); err == nil {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "TRUE"
}
