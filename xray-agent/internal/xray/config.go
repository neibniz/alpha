package xray

import (
	"encoding/json"
	"fmt"
)

const ipLimitOutboundTag = "alpha-ip-limit-block"

type RuntimeLogConfig struct {
	AccessPath string
	ErrorPath  string
	Level      string
}

func BuildRuntimeConfig(raw []byte, apiPort int, injectAPI bool) ([]byte, error) {
	return BuildRuntimeConfigWithLog(raw, apiPort, injectAPI, RuntimeLogConfig{})
}

func BuildRuntimeConfigWithLog(raw []byte, apiPort int, injectAPI bool, logConfig RuntimeLogConfig) ([]byte, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if injectAPI {
		ensureAPI(cfg, apiPort)
		cfg["outbounds"] = ensureIPLimitOutbound(asSlice(cfg["outbounds"]))
	}
	ensureLog(cfg, logConfig)
	return json.MarshalIndent(cfg, "", "  ")
}

func ensureIPLimitOutbound(outbounds []interface{}) []interface{} {
	block := map[string]interface{}{
		"tag":      ipLimitOutboundTag,
		"protocol": "blackhole",
		"settings": map[string]interface{}{},
	}
	result := make([]interface{}, 0, len(outbounds)+1)
	for _, item := range outbounds {
		if asMap(item)["tag"] == ipLimitOutboundTag {
			continue
		}
		result = append(result, item)
	}
	return append(result, block)
}

func ensureLog(cfg map[string]interface{}, logConfig RuntimeLogConfig) {
	if logConfig.AccessPath == "" && logConfig.ErrorPath == "" && logConfig.Level == "" {
		return
	}
	logs := asMap(cfg["log"])
	if logConfig.AccessPath != "" {
		logs["access"] = logConfig.AccessPath
	}
	if logConfig.ErrorPath != "" {
		logs["error"] = logConfig.ErrorPath
	}
	if logConfig.Level != "" {
		logs["loglevel"] = logConfig.Level
	}
	cfg["log"] = logs
}

func ensureAPI(cfg map[string]interface{}, apiPort int) {
	if _, ok := cfg["stats"]; !ok {
		cfg["stats"] = map[string]interface{}{}
	}
	ensurePolicy(cfg)
	cfg["api"] = map[string]interface{}{
		"tag": "api",
		"services": []interface{}{
			"HandlerService",
			"StatsService",
			"RoutingService",
		},
	}
	cfg["inbounds"] = ensureAPIInbound(asSlice(cfg["inbounds"]), apiPort)
	cfg["routing"] = ensureAPIRouting(asMap(cfg["routing"]))
}

func ensurePolicy(cfg map[string]interface{}) {
	policy := asMap(cfg["policy"])
	levels := asMap(policy["levels"])
	level0 := asMap(levels["0"])
	level0["statsUserUplink"] = true
	level0["statsUserDownlink"] = true
	levels["0"] = level0
	policy["levels"] = levels

	system := asMap(policy["system"])
	system["statsInboundUplink"] = true
	system["statsInboundDownlink"] = true
	system["statsOutboundUplink"] = true
	system["statsOutboundDownlink"] = true
	policy["system"] = system
	cfg["policy"] = policy
}

func ensureAPIInbound(inbounds []interface{}, apiPort int) []interface{} {
	apiInbound := map[string]interface{}{
		"tag":      "api",
		"listen":   "127.0.0.1",
		"port":     apiPort,
		"protocol": "dokodemo-door",
		"settings": map[string]interface{}{
			"address": "127.0.0.1",
		},
	}
	result := make([]interface{}, 0, len(inbounds)+1)
	inserted := false
	for _, item := range inbounds {
		if inbound := asMap(item); inbound["tag"] == "api" {
			if !inserted {
				result = append(result, apiInbound)
				inserted = true
			}
			continue
		}
		result = append(result, item)
	}
	if !inserted {
		result = append(result, apiInbound)
	}
	return result
}

func ensureAPIRouting(routing map[string]interface{}) map[string]interface{} {
	rules := asSlice(routing["rules"])
	next := make([]interface{}, 0, len(rules)+1)
	next = append(next, map[string]interface{}{
		"type":        "field",
		"inboundTag":  []interface{}{"api"},
		"outboundTag": "api",
	})
	for _, item := range rules {
		rule := asMap(item)
		if isAPIRoute(rule) {
			continue
		}
		next = append(next, item)
	}
	routing["rules"] = next
	return routing
}

func isAPIRoute(rule map[string]interface{}) bool {
	if rule["outboundTag"] == "api" {
		return true
	}
	return valueContainsString(rule["inboundTag"], "api")
}

func valueContainsString(value interface{}, expected string) bool {
	switch typed := value.(type) {
	case string:
		return typed == expected
	case []interface{}:
		for _, item := range typed {
			if valueContainsString(item, expected) {
				return true
			}
		}
	case []string:
		for _, item := range typed {
			if item == expected {
				return true
			}
		}
	}
	return false
}

func asMap(value interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	if typed, ok := value.(map[string]interface{}); ok {
		return typed
	}
	return map[string]interface{}{}
}

func asSlice(value interface{}) []interface{} {
	if value == nil {
		return []interface{}{}
	}
	if typed, ok := value.([]interface{}); ok {
		return typed
	}
	return []interface{}{}
}

func ValidateRuntimeConfig(raw []byte) error {
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return err
	}
	if len(asSlice(cfg["inbounds"])) == 0 {
		return fmt.Errorf("xray config has no inbounds")
	}
	return nil
}
