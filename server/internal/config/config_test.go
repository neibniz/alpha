package config

import "testing"

func TestNodeConfigMarshalShapeAndNilReceiver(t *testing.T) {
	cfg := &NodeConfig{
		NodeSecret:             "secret",
		NodePullInterval:       60,
		NodePushInterval:       30,
		TrafficReportThreshold: 1024,
		IPStrategy:             "UseIPv4",
	}
	data, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("NodeConfig.Marshal() error = %v", err)
	}
	want := `{"NodePullInterval":60,"NodePushInterval":30,"TrafficReportThreshold":1024,"IPStrategy":"UseIPv4"}`
	if string(data) != want {
		t.Fatalf("NodeConfig.Marshal() = %s, want %s", data, want)
	}

	var nilCfg *NodeConfig
	data, err = nilCfg.Marshal()
	if err != nil {
		t.Fatalf("nil NodeConfig.Marshal() error = %v", err)
	}
	if string(data) != "{}" {
		t.Fatalf("nil NodeConfig.Marshal() = %s, want {}", data)
	}
}

func TestNodeConfigUnmarshalDirectShape(t *testing.T) {
	var cfg NodeConfig
	if err := cfg.Unmarshal([]byte(`{"NodeSecret":"secret","NodePullInterval":60,"NodePushInterval":30,"TrafficReportThreshold":1024,"IPStrategy":"UseIPv4"}`)); err != nil {
		t.Fatalf("NodeConfig.Unmarshal() error = %v", err)
	}
	if cfg.NodeSecret != "" || cfg.NodePullInterval != 60 || cfg.NodePushInterval != 30 || cfg.TrafficReportThreshold != 1024 || cfg.IPStrategy != "UseIPv4" {
		t.Fatalf("NodeConfig.Unmarshal() = %#v", cfg)
	}
}
