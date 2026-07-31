package xray

import (
	"encoding/json"
	"testing"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

func TestInjectUsersMergesUserDefaults(t *testing.T) {
	raw := []byte(`{
		"inbounds": [
			{
				"tag": "main",
				"protocol": "vless",
				"settings": {
					"clients": [
						{"flow": "xtls-rprx-vision", "level": 3, "custom": "keep"}
					],
					"decryption": "none"
				}
			}
		]
	}`)

	rendered, err := InjectUsers(raw, []panel.XrayUser{
		{
			UserSubscribeID: 1001,
			Email:           "u_1001",
			UUID:            "e6c7a44e-9fc6-4c2c-a148-293b41f1982f",
			Level:           0,
			Inbounds:        []string{"main"},
			Enabled:         true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	client := firstRenderedClient(t, rendered)
	if client["flow"] != "xtls-rprx-vision" {
		t.Fatalf("flow default lost: %#v", client)
	}
	if client["custom"] != "keep" {
		t.Fatalf("custom default lost: %#v", client)
	}
	if client["email"] != "u_1001" {
		t.Fatalf("email not injected: %#v", client)
	}
	if client["id"] != "e6c7a44e-9fc6-4c2c-a148-293b41f1982f" {
		t.Fatalf("id not injected: %#v", client)
	}
	if client["level"].(float64) != 0 {
		t.Fatalf("server level should override default level: %#v", client)
	}
}

func TestInjectUsersUsesXrayClientField(t *testing.T) {
	raw := []byte(`{
		"inbounds": [
			{
				"tag": "main",
				"protocol": "vless",
				"settings": {
					"clients": [
						{"flow": "xtls-rprx-vision"}
					],
					"users": [
						{"flow": "legacy-field-should-be-removed"}
					],
					"decryption": "none"
				}
			}
		]
	}`)

	rendered, err := InjectUsers(raw, []panel.XrayUser{
		{
			Email:    "u_1001",
			UUID:     "e6c7a44e-9fc6-4c2c-a148-293b41f1982f",
			Enabled:  true,
			Inbounds: []string{"main"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(rendered, &cfg); err != nil {
		t.Fatal(err)
	}
	inbound := cfg["inbounds"].([]interface{})[0].(map[string]interface{})
	settings := inbound["settings"].(map[string]interface{})
	if _, ok := settings["users"]; ok {
		t.Fatalf("vless runtime config should not contain settings.users: %#v", settings)
	}
	clients := settings["clients"].([]interface{})
	client := clients[0].(map[string]interface{})
	if client["flow"] != "xtls-rprx-vision" {
		t.Fatalf("client flow = %v, want xtls-rprx-vision", client["flow"])
	}
	if client["id"] != "e6c7a44e-9fc6-4c2c-a148-293b41f1982f" {
		t.Fatalf("client id = %v", client["id"])
	}
}

func TestBuildInboundUsersUsesXrayClientDefaults(t *testing.T) {
	raw := []byte(`{
		"inbounds": [
			{
				"tag": "main",
				"protocol": "vless",
				"settings": {
					"clients": [
						{"flow": "xtls-rprx-vision", "level": 3}
					],
					"decryption": "none"
				}
			}
		]
	}`)

	inbounds, err := BuildInboundUsers(raw, []panel.XrayUser{
		{
			Email:    "u_1001",
			UUID:     "e6c7a44e-9fc6-4c2c-a148-293b41f1982f",
			Enabled:  true,
			Inbounds: []string{"main"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inbounds) != 1 || len(inbounds[0].Users) != 1 {
		t.Fatalf("BuildInboundUsers() = %#v", inbounds)
	}
	fields := inbounds[0].Users[0].Fields
	if fields["flow"] != "xtls-rprx-vision" {
		t.Fatalf("hot reload flow default lost: %#v", fields)
	}
	if fields["id"] != "e6c7a44e-9fc6-4c2c-a148-293b41f1982f" {
		t.Fatalf("hot reload id = %v", fields["id"])
	}
	if uint32Field(fields, "level") != 0 {
		t.Fatalf("server level should override default level: %#v", fields)
	}
}

func TestInjectUsersPreservesProtocolSpecificDefaults(t *testing.T) {
	cases := []struct {
		name     string
		protocol string
		defaults string
		wantKey  string
		want     interface{}
	}{
		{
			name:     "vmess",
			protocol: "vmess",
			defaults: `{"alterId": 0, "security": "auto"}`,
			wantKey:  "security",
			want:     "auto",
		},
		{
			name:     "shadowsocks",
			protocol: "shadowsocks",
			defaults: `{"method": "2022-blake3-aes-128-gcm", "network": "tcp,udp"}`,
			wantKey:  "network",
			want:     "tcp,udp",
		},
		{
			name:     "trojan",
			protocol: "trojan",
			defaults: `{"flow": "xtls-rprx-vision"}`,
			wantKey:  "flow",
			want:     "xtls-rprx-vision",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userField := userListField(tc.protocol)
			raw := []byte(`{"inbounds":[{"tag":"main","protocol":"` + tc.protocol + `","settings":{"` + userField + `":[` + tc.defaults + `]}}]}`)
			rendered, err := InjectUsers(raw, []panel.XrayUser{
				{
					Email:    "u_1001",
					UUID:     "uuid",
					Password: "password",
					Enabled:  true,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			client := firstRenderedClient(t, rendered)
			if client[tc.wantKey] != tc.want {
				t.Fatalf("%s default lost: %#v", tc.wantKey, client)
			}
		})
	}
}

func TestInjectUsersSupportsHysteriaUsers(t *testing.T) {
	raw := []byte(`{
		"inbounds": [
			{
				"tag": "hy2",
				"protocol": "hysteria",
				"settings": {
					"version": 2,
					"users": [
						{"level": 3, "custom": "keep"}
					]
				}
			}
		]
	}`)

	rendered, err := InjectUsers(raw, []panel.XrayUser{
		{
			Email:    "u_1001",
			UUID:     "uuid",
			Password: "hy2-password",
			Level:    0,
			Inbounds: []string{"hy2"},
			Enabled:  true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	client := firstRenderedClient(t, rendered)
	if client["auth"] != "hy2-password" {
		t.Fatalf("auth not injected: %#v", client)
	}
	if client["email"] != "u_1001" {
		t.Fatalf("email not injected: %#v", client)
	}
	if client["custom"] != "keep" {
		t.Fatalf("custom default lost: %#v", client)
	}
	if client["level"].(float64) != 0 {
		t.Fatalf("server level should override default level: %#v", client)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(rendered, &cfg); err != nil {
		t.Fatal(err)
	}
	inbound := cfg["inbounds"].([]interface{})[0].(map[string]interface{})
	settings := inbound["settings"].(map[string]interface{})
	if _, ok := settings["clients"]; ok {
		t.Fatalf("hysteria runtime config should not contain settings.clients: %#v", settings)
	}
	if _, ok := settings["users"]; !ok {
		t.Fatalf("hysteria runtime config is missing settings.users: %#v", settings)
	}
}

func firstRenderedClient(t *testing.T, data []byte) map[string]interface{} {
	t.Helper()
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	inbounds := cfg["inbounds"].([]interface{})
	inbound := inbounds[0].(map[string]interface{})
	protocol := stringValue(inbound["protocol"])
	settings := inbound["settings"].(map[string]interface{})
	clients := settings[userListField(protocol)].([]interface{})
	return clients[0].(map[string]interface{})
}
