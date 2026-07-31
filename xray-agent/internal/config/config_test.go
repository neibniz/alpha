package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNormalizeYAMLConfig(t *testing.T) {
	var cfg Config
	raw := []byte(`
server:
  base_url: "https://panel.example.com"
  server_id: 10
  secret_key: "secret"
xray:
  executable: "/usr/local/bin/xray"
  work_dir: "/tmp/xray-agent-test"
  config_path: "/tmp/xray-agent-test/current.json"
  api_address: "127.0.0.1:11085"
agent:
  pull_interval: "15s"
report:
  status_interval: "2s"
  traffic_interval: "20s"
log:
  agent_path: "log/agent.log"
  xray_error_path: "/tmp/xray-agent-test/xray-error.log"
  xray_access_path: "log/xray-access.log"
  xray_level: "warning"
`)
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	cfg, err := normalize(cfg)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg.Panel.BaseURL != "https://panel.example.com" {
		t.Fatalf("panel alias not populated: %q", cfg.Panel.BaseURL)
	}
	if cfg.Xray.BinPath != "/usr/local/bin/xray" {
		t.Fatalf("legacy bin path not populated: %q", cfg.Xray.BinPath)
	}
	if cfg.Xray.APIPort != 11085 {
		t.Fatalf("api port = %d, want 11085", cfg.Xray.APIPort)
	}
	if cfg.Log.AgentPath != "/tmp/xray-agent-test/log/agent.log" {
		t.Fatalf("agent log path = %q", cfg.Log.AgentPath)
	}
	if cfg.Log.XrayAccessPath != "/tmp/xray-agent-test/log/xray-access.log" {
		t.Fatalf("xray access log path = %q", cfg.Log.XrayAccessPath)
	}
	if cfg.Log.XrayErrorPath != "/tmp/xray-agent-test/xray-error.log" {
		t.Fatalf("xray error log path = %q", cfg.Log.XrayErrorPath)
	}
	if cfg.Log.XrayLevel != "warning" {
		t.Fatalf("xray log level = %q", cfg.Log.XrayLevel)
	}
}

func TestNormalizeDefaultBackedConfigDerivesAPIPortFromAPIAddress(t *testing.T) {
	t.Setenv("XRAY_AGENT_WORK_DIR", t.TempDir())
	cfg := Default()
	raw := []byte(`
server:
  base_url: "https://panel.example.com"
  server_id: 12
  secret_key: "secret"
xray:
  api_address: "127.0.0.1:11085"
`)
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	cfg, err := normalize(cfg)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg.Xray.APIPort != 11085 {
		t.Fatalf("api port = %d, want 11085", cfg.Xray.APIPort)
	}
}

func TestApplyEnvDerivesAPIPortFromAPIAddress(t *testing.T) {
	t.Setenv("XRAY_AGENT_WORK_DIR", t.TempDir())
	t.Setenv("XRAY_API_ADDRESS", "127.0.0.1:12085")
	cfg := Default()
	cfg.Server.BaseURL = "https://panel.example.com"
	cfg.Server.ServerID = 13
	cfg.Server.SecretKey = "secret"
	applyEnv(&cfg)
	cfg, err := normalize(cfg)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg.Xray.APIPort != 12085 {
		t.Fatalf("api port = %d, want 12085", cfg.Xray.APIPort)
	}
}

func TestNormalizeDefaultBackedConfigDerivesAPIAddressFromAPIPort(t *testing.T) {
	t.Setenv("XRAY_AGENT_WORK_DIR", t.TempDir())
	cfg := Default()
	raw := []byte(`
server:
  base_url: "https://panel.example.com"
  server_id: 14
  secret_key: "secret"
xray:
  api_port: 13085
`)
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	cfg, err := normalize(cfg)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg.Xray.APIAddress != "127.0.0.1:13085" {
		t.Fatalf("api address = %q, want 127.0.0.1:13085", cfg.Xray.APIAddress)
	}
}

func TestNormalizeMinimalYAMLConfig(t *testing.T) {
	t.Setenv("XRAY_AGENT_WORK_DIR", t.TempDir())
	cfg := Default()
	raw := []byte(`
server:
  base_url: "https://panel.example.com"
  server_id: 12
  secret_key: "secret"
`)
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	cfg, err := normalize(cfg)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg.Xray.Executable != "/usr/local/bin/xray" {
		t.Fatalf("executable = %q, want /usr/local/bin/xray", cfg.Xray.Executable)
	}
	if cfg.Xray.WorkDir == "" {
		t.Fatalf("work dir = %q", cfg.Xray.WorkDir)
	}
	if cfg.Xray.ConfigPath != cfg.Xray.WorkDir+"/current.json" {
		t.Fatalf("config path = %q", cfg.Xray.ConfigPath)
	}
	if cfg.Xray.APIAddress != "127.0.0.1:10085" {
		t.Fatalf("api address = %q", cfg.Xray.APIAddress)
	}
	if cfg.Agent.PullIntervalText != "30s" || cfg.Report.StatusIntervalText != "5s" || cfg.Report.TrafficIntervalText != "5s" {
		t.Fatalf("interval defaults not populated: %#v %#v", cfg.Agent, cfg.Report)
	}
	if cfg.Report.Traffic.BatchSize != 12 || cfg.Report.Traffic.FlushIntervalText != "5s" {
		t.Fatalf("traffic spool defaults not populated: %#v", cfg.Report.Traffic)
	}
}

func TestNormalizeLegacyJSONFields(t *testing.T) {
	var cfg Config
	raw := []byte(`{
  "panel": {
    "base_url": "https://panel.example.com",
    "server_id": 11,
    "secret_key": "secret"
  },
  "xray": {
    "bin_path": "/usr/bin/xray",
    "work_dir": "/tmp/xray-agent-test",
    "config_file": "/tmp/xray-agent-test/current.json",
    "api_port": 10085
  },
  "agent": {
    "pull_interval": "30s"
  },
  "report": {
    "status_interval": "5s",
    "traffic_interval": "30s"
  }
}`)
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	cfg, err := normalize(cfg)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if cfg.Server.ServerID != 11 {
		t.Fatalf("server id = %d, want 11", cfg.Server.ServerID)
	}
	if cfg.Xray.Executable != "/usr/bin/xray" {
		t.Fatalf("executable = %q, want /usr/bin/xray", cfg.Xray.Executable)
	}
	if cfg.Xray.APIAddress != "127.0.0.1:10085" {
		t.Fatalf("api address = %q, want 127.0.0.1:10085", cfg.Xray.APIAddress)
	}
}
