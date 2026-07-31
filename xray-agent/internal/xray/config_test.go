package xray

import (
	"encoding/json"
	"testing"
)

func TestBuildRuntimeConfigPreservesGeodata(t *testing.T) {
	raw := []byte(`{
		"inbounds": [{"tag": "main"}],
		"outbounds": [{"tag": "direct", "protocol": "freedom"}],
		"geodata": {
			"cron": "0 4 * * *",
			"assets": [
				{"url": "https://example.com/geoip.dat", "file": "geoip.dat"}
			]
		}
	}`)

	rendered, err := BuildRuntimeConfig(raw, 10085, false)
	if err != nil {
		t.Fatalf("build runtime config: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(rendered, &cfg); err != nil {
		t.Fatalf("unmarshal runtime config: %v", err)
	}
	if cfg["geodata"] == nil {
		t.Fatal("expected geodata to be preserved")
	}
}

func TestBuildRuntimeConfigPlacesAPIRouteFirst(t *testing.T) {
	raw := []byte(`{
		"inbounds": [{"tag": "main"}],
		"outbounds": [{"tag": "direct", "protocol": "freedom"}],
		"routing": {
			"rules": [
				{"type": "field", "outboundTag": "direct"},
				{"type": "field", "inboundTag": ["api"], "outboundTag": "api"}
			]
		}
	}`)

	rendered, err := BuildRuntimeConfig(raw, 10085, true)
	if err != nil {
		t.Fatalf("build runtime config: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(rendered, &cfg); err != nil {
		t.Fatalf("unmarshal runtime config: %v", err)
	}
	routing := cfg["routing"].(map[string]interface{})
	rules := routing["rules"].([]interface{})
	if len(rules) != 2 {
		t.Fatalf("expected api route plus original non-api rule, got %d", len(rules))
	}
	first := rules[0].(map[string]interface{})
	if first["outboundTag"] != "api" {
		t.Fatalf("expected api route first, got %#v", first)
	}
	inboundTags := first["inboundTag"].([]interface{})
	if len(inboundTags) != 1 || inboundTags[0] != "api" {
		t.Fatalf("expected api inbound tag, got %#v", inboundTags)
	}
}

func TestBuildRuntimeConfigOwnsAPIInbound(t *testing.T) {
	raw := []byte(`{
		"inbounds": [
			{"tag": "api", "listen": "0.0.0.0", "port": 9999, "protocol": "freedom"},
			{"tag": "main"}
		],
		"outbounds": [{"tag": "direct", "protocol": "freedom"}]
	}`)

	rendered, err := BuildRuntimeConfig(raw, 10085, true)
	if err != nil {
		t.Fatalf("build runtime config: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(rendered, &cfg); err != nil {
		t.Fatalf("unmarshal runtime config: %v", err)
	}
	inbounds := cfg["inbounds"].([]interface{})
	apiCount := 0
	for _, item := range inbounds {
		inbound := item.(map[string]interface{})
		if inbound["tag"] != "api" {
			continue
		}
		apiCount++
		if inbound["listen"] != "127.0.0.1" ||
			inbound["protocol"] != "dokodemo-door" ||
			int(inbound["port"].(float64)) != 10085 {
			t.Fatalf("unexpected api inbound: %#v", inbound)
		}
	}
	if apiCount != 1 {
		t.Fatalf("expected exactly one api inbound, got %d", apiCount)
	}
}

func TestBuildRuntimeConfigInjectsLogFiles(t *testing.T) {
	raw := []byte(`{
		"log": {"loglevel": "info"},
		"inbounds": [{"tag": "main"}],
		"outbounds": [{"tag": "direct", "protocol": "freedom"}]
	}`)

	rendered, err := BuildRuntimeConfigWithLog(raw, 10085, false, RuntimeLogConfig{
		AccessPath: "/var/log/alpha/xray-agent/xray-access.log",
		ErrorPath:  "/var/log/alpha/xray-agent/xray-error.log",
		Level:      "warning",
	})
	if err != nil {
		t.Fatalf("build runtime config: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(rendered, &cfg); err != nil {
		t.Fatalf("unmarshal runtime config: %v", err)
	}
	logConfig := cfg["log"].(map[string]interface{})
	if logConfig["access"] != "/var/log/alpha/xray-agent/xray-access.log" {
		t.Fatalf("unexpected access log: %#v", logConfig)
	}
	if logConfig["error"] != "/var/log/alpha/xray-agent/xray-error.log" {
		t.Fatalf("unexpected error log: %#v", logConfig)
	}
	if logConfig["loglevel"] != "warning" {
		t.Fatalf("unexpected log level: %#v", logConfig)
	}
}

func TestBuildRuntimeConfigOwnsIPLimitBlackhole(t *testing.T) {
	raw := []byte(`{
		"inbounds": [{"tag": "main"}],
		"outbounds": [
			{"tag": "direct", "protocol": "freedom"},
			{"tag": "alpha-ip-limit-block", "protocol": "freedom"}
		]
	}`)

	rendered, err := BuildRuntimeConfig(raw, 10085, true)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(rendered, &cfg); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range cfg["outbounds"].([]interface{}) {
		outbound := item.(map[string]interface{})
		if outbound["tag"] != ipLimitOutboundTag {
			continue
		}
		count++
		if outbound["protocol"] != "blackhole" {
			t.Fatalf("managed block outbound = %#v", outbound)
		}
	}
	if count != 1 {
		t.Fatalf("managed block outbound count = %d", count)
	}
}
