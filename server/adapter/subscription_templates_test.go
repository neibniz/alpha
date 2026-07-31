package adapter

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type migrationTemplate struct {
	id           int
	name         string
	templateText string
	outputFormat string
}

func TestBuiltInSubscriptionTemplatesRender(t *testing.T) {
	templates := loadEffectiveMigrationTemplates(t)
	if len(templates) != 15 {
		t.Fatalf("expected 15 built-in subscription templates, got %d", len(templates))
	}

	proxies := []Proxy{
		{
			Sort:                    1,
			Name:                    "VLESS Reality gRPC",
			Server:                  "node.example.com",
			Port:                    443,
			Type:                    "vless",
			Security:                "reality",
			SNI:                     "www.ebay.com",
			RealityPublicKey:        "preview-public-key",
			RealityShortId:          "0123456789abcdef",
			Transport:               "grpc",
			ServiceName:             "grpc",
			GrpcAuthority:           "www.ebay.com",
			GrpcMultiMode:           true,
			GrpcUserAgent:           "alpha-test",
			GrpcIdleTimeout:         60,
			GrpcHealthCheckTimeout:  20,
			GrpcPermitWithoutStream: true,
			GrpcInitialWindowsSize:  65536,
			Flow:                    "none",
			Fingerprint:             "chrome",
			AllowInsecure:           true,
			RealityServerAddr:       "172.16.0.2/32",
			RealityServerPort:       51820,
		},
		{
			Sort:      2,
			Name:      "VMess WS",
			Server:    "vmess.example.com",
			Port:      8443,
			Type:      "vmess",
			Security:  "tls",
			SNI:       "vmess.example.com",
			Transport: "ws",
			Host:      "vmess.example.com",
			Path:      "/ws",
		},
		{
			Sort:      3,
			Name:      "Trojan WS",
			Server:    "trojan.example.com",
			Port:      443,
			Type:      "trojan",
			Security:  "tls",
			SNI:       "trojan.example.com",
			Transport: "ws",
			Host:      "trojan.example.com",
			Path:      "/trojan",
		},
		{
			Sort:      4,
			Name:      "SS 2022",
			Server:    "ss.example.com",
			Port:      8388,
			Type:      "shadowsocks",
			Method:    "2022-blake3-aes-128-gcm",
			ServerKey: "server-secret-key",
		},
		{
			Sort:                   5,
			Name:                   "Hysteria2",
			Server:                 "hy2.example.com",
			Port:                   443,
			Type:                   "hysteria2",
			SNI:                    "hy2.example.com",
			ServerKey:              "hy2-password",
			ObfsPassword:           "obfs-password",
			HopPorts:               "443,8443",
			HopInterval:            30,
			AllowInsecure:          true,
			UpMbps:                 100,
			DownMbps:               100,
			HysteriaUDPIdleTimeout: 60,
			Fingerprint:            "chrome",
			DisableSNI:             false,
			ReduceRtt:              true,
			UDPRelayMode:           "native",
			EncryptionMode:         "native",
		},
		{
			Sort:                 6,
			Name:                 "TUIC",
			Server:               "tuic.example.com",
			Port:                 443,
			Type:                 "tuic",
			ServerKey:            "00000000-0000-0000-0000-000000000000",
			SNI:                  "tuic.example.com",
			AllowInsecure:        true,
			DisableSNI:           true,
			ReduceRtt:            true,
			UDPRelayMode:         "native",
			CongestionController: "bbr",
		},
		{
			Sort:          7,
			Name:          "AnyTLS",
			Server:        "anytls.example.com",
			Port:          443,
			Type:          "anytls",
			SNI:           "anytls.example.com",
			AllowInsecure: true,
			Fingerprint:   "chrome",
		},
	}

	for _, item := range templates {
		t.Run(item.name, func(t *testing.T) {
			client := &Client{
				SiteName:       "Alpha",
				SubscribeName:  "Test Plan",
				ClientTemplate: item.templateText,
				OutputFormat:   item.outputFormat,
				Proxies:        proxies,
				UserInfo: User{
					Password:     "00000000-0000-0000-0000-000000000000",
					ExpiredAt:    time.Unix(1_893_456_000, 0),
					Download:     1024,
					Upload:       2048,
					Traffic:      1024 * 1024 * 1024,
					SubscribeURL: "https://example.com/sub/token",
				},
				Params: map[string]string{
					"xray_inbound_host":       "127.0.0.1",
					"xray_inbound_start_port": "10800",
				},
			}
			output, err := client.Build()
			if err != nil {
				t.Fatalf("build template: %v", err)
			}
			if len(output) == 0 {
				t.Fatal("template rendered empty output")
			}
			rendered := string(output)
			if item.outputFormat == "base64" {
				decoded, err := base64.StdEncoding.DecodeString(rendered)
				if err != nil {
					t.Fatalf("decode base64 output: %v", err)
				}
				rendered = string(decoded)
			}
			if strings.Contains(rendered, "<no value>") {
				t.Fatalf("rendered output contains unresolved template value")
			}
			switch item.outputFormat {
			case "json":
				var data interface{}
				if err := json.Unmarshal([]byte(rendered), &data); err != nil {
					t.Fatalf("invalid json output: %v", err)
				}
			case "yaml":
				var data interface{}
				if err := yaml.Unmarshal([]byte(rendered), &data); err != nil {
					t.Fatalf("invalid yaml output: %v", err)
				}
			}
		})
	}
}

func TestXrayCoreTemplateKeepsGrpcSettings(t *testing.T) {
	var xrayCore migrationTemplate
	for _, item := range loadEffectiveMigrationTemplates(t) {
		if item.id == 14 {
			xrayCore = item
			break
		}
	}
	if xrayCore.id == 0 {
		t.Fatal("missing Xray Core subscription template")
	}

	client := &Client{
		SiteName:       "Alpha",
		SubscribeName:  "Test Plan",
		ClientTemplate: xrayCore.templateText,
		OutputFormat:   xrayCore.outputFormat,
		Proxies: []Proxy{
			{
				Name:                    "VLESS Reality gRPC",
				Server:                  "node.example.com",
				Port:                    443,
				Type:                    "vless",
				Security:                "reality",
				SNI:                     "www.ebay.com",
				RealityPublicKey:        "preview-public-key",
				RealityShortId:          "0123456789abcdef",
				Transport:               "grpc",
				ServiceName:             "grpc",
				GrpcAuthority:           "www.ebay.com",
				GrpcMultiMode:           true,
				GrpcUserAgent:           "alpha-test",
				GrpcIdleTimeout:         60,
				GrpcHealthCheckTimeout:  20,
				GrpcPermitWithoutStream: true,
				GrpcInitialWindowsSize:  65536,
			},
		},
		UserInfo: User{
			Password:  "00000000-0000-0000-0000-000000000000",
			ExpiredAt: time.Unix(1_893_456_000, 0),
			Traffic:   1024 * 1024 * 1024,
		},
		Params: map[string]string{
			"xray_inbound_host":       "127.0.0.1",
			"xray_inbound_start_port": "10800",
		},
	}
	output, err := client.Build()
	if err != nil {
		t.Fatalf("build template: %v", err)
	}

	var rendered struct {
		Outbounds []struct {
			StreamSettings struct {
				Network      string                 `json:"network"`
				GrpcSettings map[string]interface{} `json:"grpcSettings"`
			} `json:"streamSettings"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("invalid xray core json: %v", err)
	}
	if len(rendered.Outbounds) != 1 {
		t.Fatalf("outbounds len = %d, want 1", len(rendered.Outbounds))
	}
	stream := rendered.Outbounds[0].StreamSettings
	if stream.Network != "grpc" {
		t.Fatalf("network = %q, want grpc", stream.Network)
	}
	assertJSONField(t, stream.GrpcSettings, "serviceName", "grpc")
	assertJSONField(t, stream.GrpcSettings, "authority", "www.ebay.com")
	assertJSONField(t, stream.GrpcSettings, "multiMode", true)
	assertJSONField(t, stream.GrpcSettings, "user_agent", "alpha-test")
	assertJSONField(t, stream.GrpcSettings, "idle_timeout", float64(60))
	assertJSONField(t, stream.GrpcSettings, "health_check_timeout", float64(20))
	assertJSONField(t, stream.GrpcSettings, "permit_without_stream", true)
	assertJSONField(t, stream.GrpcSettings, "initial_windows_size", float64(65536))
}

func TestXrayCoreTemplateUsesTLSVerificationPins(t *testing.T) {
	xrayCore := findMigrationTemplate(t, 14)
	output := renderTemplateString(t, xrayCore, []Proxy{
		{
			Name:                 "VMess TLS",
			Server:               "vmess.example.com",
			Port:                 443,
			Type:                 "vmess",
			Security:             "tls",
			SNI:                  "vmess.example.com",
			Transport:            "ws",
			Path:                 "/ws",
			AllowInsecure:        true,
			Fingerprint:          "chrome",
			PinnedPeerCertSha256: "AA:BB:CC",
			VerifyPeerCertByName: "vmess.example.com",
		},
	}, User{
		Password:  "00000000-0000-0000-0000-000000000000",
		ExpiredAt: time.Unix(1_893_456_000, 0),
		Traffic:   1024 * 1024 * 1024,
	}, map[string]string{
		"xray_inbound_host":       "127.0.0.1",
		"xray_inbound_start_port": "10800",
	})
	if strings.Contains(output, "allowInsecure") {
		t.Fatalf("xray core template should not emit deprecated allowInsecure:\n%s", output)
	}
	var rendered struct {
		Outbounds []struct {
			StreamSettings struct {
				TLSSettings map[string]interface{} `json:"tlsSettings"`
			} `json:"streamSettings"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(output), &rendered); err != nil {
		t.Fatalf("invalid xray core json: %v", err)
	}
	tls := rendered.Outbounds[0].StreamSettings.TLSSettings
	assertJSONField(t, tls, "serverName", "vmess.example.com")
	assertJSONField(t, tls, "pinnedPeerCertSha256", "AA:BB:CC")
	assertJSONField(t, tls, "verifyPeerCertByName", "vmess.example.com")
	assertJSONField(t, tls, "fingerprint", "chrome")
}

func TestVLESSFlowIsOmittedWhenEmptyOrNone(t *testing.T) {
	user := User{
		Password:  "00000000-0000-0000-0000-000000000000",
		ExpiredAt: time.Unix(1_893_456_000, 0),
		Traffic:   1024 * 1024 * 1024,
	}
	proxies := []Proxy{
		{
			Name:             "VLESS Empty Flow",
			Server:           "empty-flow.example.com",
			Port:             443,
			Type:             "vless",
			Security:         "reality",
			SNI:              "www.ebay.com",
			RealityPublicKey: "preview-public-key",
			RealityShortId:   "0123456789abcdef",
			Transport:        "grpc",
			ServiceName:      "grpc",
			Fingerprint:      "chrome",
		},
		{
			Name:             "VLESS None Flow",
			Server:           "none-flow.example.com",
			Port:             8443,
			Type:             "vless",
			Security:         "reality",
			SNI:              "www.ebay.com",
			RealityPublicKey: "preview-public-key",
			RealityShortId:   "0123456789abcdef",
			Transport:        "grpc",
			ServiceName:      "grpc",
			Fingerprint:      "chrome",
			Flow:             "none",
		},
	}
	params := map[string]string{
		"xray_inbound_host":       "127.0.0.1",
		"xray_inbound_start_port": "10800",
	}
	badPatterns := []string{
		"flow=none",
		"flow=&",
		"flow=,",
		"flow: none",
		"flow: ,",
		"flow: }",
		"flow:  }",
		`"flow": "none"`,
		`"flow": ""`,
		`flow={{`,
	}
	for _, item := range loadEffectiveMigrationTemplates(t) {
		t.Run(item.name, func(t *testing.T) {
			rendered := renderTemplateString(t, item, proxies, user, params)
			for _, bad := range badPatterns {
				if strings.Contains(rendered, bad) {
					t.Fatalf("%s rendered disabled flow pattern %q:\n%s", item.name, bad, rendered)
				}
			}
		})
	}
}

func TestSingbox111TemplateRendersVLESSFlow(t *testing.T) {
	singbox111 := findMigrationTemplate(t, 13)
	rendered := renderTemplateString(t, singbox111, []Proxy{
		{
			Name:             "VLESS Vision",
			Server:           "vision.example.com",
			Port:             443,
			Type:             "vless",
			Security:         "reality",
			SNI:              "www.ebay.com",
			RealityPublicKey: "preview-public-key",
			RealityShortId:   "0123456789abcdef",
			Transport:        "tcp",
			Flow:             "xtls-rprx-vision",
			Fingerprint:      "chrome",
		},
	}, User{
		Password:  "00000000-0000-0000-0000-000000000000",
		ExpiredAt: time.Unix(1_893_456_000, 0),
		Traffic:   1024 * 1024 * 1024,
	}, map[string]string{})

	var config struct {
		Outbounds []map[string]interface{} `json:"outbounds"`
	}
	if err := json.Unmarshal([]byte(rendered), &config); err != nil {
		t.Fatalf("invalid sing-box 1.11 json: %v", err)
	}
	for _, outbound := range config.Outbounds {
		if outbound["type"] == "vless" {
			assertJSONField(t, outbound, "flow", "xtls-rprx-vision")
			return
		}
	}
	t.Fatalf("missing vless outbound:\n%s", rendered)
}

func TestXrayCoreTemplateRendersHysteriaOutbound(t *testing.T) {
	xrayCore := findMigrationTemplate(t, 14)
	client := &Client{
		SiteName:       "Alpha",
		SubscribeName:  "Test Plan",
		ClientTemplate: xrayCore.templateText,
		OutputFormat:   xrayCore.outputFormat,
		Proxies: []Proxy{
			{
				Name:                   "Hysteria2",
				Server:                 "hy2.example.com",
				Port:                   443,
				Type:                   "hysteria2",
				ServerKey:              "hy2-password",
				SNI:                    "hy2.example.com",
				AllowInsecure:          true,
				Fingerprint:            "chrome",
				HysteriaUDPIdleTimeout: 60,
			},
		},
		UserInfo: User{
			Password:  "00000000-0000-0000-0000-000000000000",
			ExpiredAt: time.Unix(1_893_456_000, 0),
			Traffic:   1024 * 1024 * 1024,
		},
		Params: map[string]string{
			"xray_inbound_host":       "127.0.0.1",
			"xray_inbound_start_port": "10800",
		},
	}
	output, err := client.Build()
	if err != nil {
		t.Fatalf("build template: %v", err)
	}

	var rendered struct {
		Outbounds []struct {
			Protocol       string                 `json:"protocol"`
			Settings       map[string]interface{} `json:"settings"`
			StreamSettings struct {
				Network          string                 `json:"network"`
				HysteriaSettings map[string]interface{} `json:"hysteriaSettings"`
			} `json:"streamSettings"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(output, &rendered); err != nil {
		t.Fatalf("invalid xray core json: %v", err)
	}
	if len(rendered.Outbounds) != 1 {
		t.Fatalf("outbounds len = %d, want 1", len(rendered.Outbounds))
	}
	outbound := rendered.Outbounds[0]
	if outbound.Protocol != "hysteria" {
		t.Fatalf("protocol = %q, want hysteria", outbound.Protocol)
	}
	assertJSONField(t, outbound.Settings, "version", float64(2))
	assertJSONField(t, outbound.Settings, "address", "hy2.example.com")
	assertJSONField(t, outbound.Settings, "port", float64(443))
	stream := outbound.StreamSettings
	if stream.Network != "hysteria" {
		t.Fatalf("network = %q, want hysteria", stream.Network)
	}
	assertJSONField(t, stream.HysteriaSettings, "version", float64(2))
	assertJSONField(t, stream.HysteriaSettings, "auth", "hy2-password")
	assertJSONField(t, stream.HysteriaSettings, "udpIdleTimeout", float64(60))
}

func TestSurgeXrayBridgeOrderMatchesXrayCore(t *testing.T) {
	xrayCore := findMigrationTemplate(t, 14)
	surgeXray := findMigrationTemplate(t, 15)
	proxies := []Proxy{
		{
			Sort:      20,
			Name:      "WS Node",
			Server:    "ws.example.com",
			Port:      8443,
			Type:      "vmess",
			Security:  "tls",
			SNI:       "ws.example.com",
			Transport: "ws",
			Host:      "ws.example.com",
			Path:      "/ws",
		},
		{
			Sort:              10,
			Name:              "Grpc Node",
			Server:            "grpc.example.com",
			Port:              443,
			Type:              "vless",
			Security:          "reality",
			SNI:               "www.ebay.com",
			RealityPublicKey:  "preview-public-key",
			RealityShortId:    "0123456789abcdef",
			Transport:         "grpc",
			ServiceName:       "grpc",
			GrpcAuthority:     "www.ebay.com",
			GrpcUserAgent:     "alpha-test",
			GrpcIdleTimeout:   60,
			GrpcMultiMode:     true,
			Fingerprint:       "chrome",
			RealityServerPort: 443,
		},
		{
			Sort:                   15,
			Name:                   "Hysteria2 Node",
			Server:                 "hy2.example.com",
			Port:                   443,
			Type:                   "hysteria2",
			ServerKey:              "hy2-password",
			SNI:                    "hy2.example.com",
			HysteriaUDPIdleTimeout: 60,
		},
	}
	user := User{
		Password:     "00000000-0000-0000-0000-000000000000",
		ExpiredAt:    time.Unix(1_893_456_000, 0),
		Traffic:      1024 * 1024 * 1024,
		SubscribeURL: "https://example.com/sub/token",
	}
	params := map[string]string{
		"xray_inbound_host":       "127.0.0.1",
		"xray_inbound_start_port": "10800",
	}

	xrayOutput, err := (&Client{
		SiteName:       "Alpha",
		SubscribeName:  "Test Plan",
		ClientTemplate: xrayCore.templateText,
		OutputFormat:   xrayCore.outputFormat,
		Proxies:        proxies,
		UserInfo:       user,
		Params:         params,
	}).Build()
	if err != nil {
		t.Fatalf("build xray core template: %v", err)
	}
	var rendered struct {
		Inbounds []struct {
			Port int `json:"port"`
		} `json:"inbounds"`
		Outbounds []struct {
			Settings map[string]interface{} `json:"settings"`
		} `json:"outbounds"`
	}
	if err := json.Unmarshal(xrayOutput, &rendered); err != nil {
		t.Fatalf("invalid xray core json: %v", err)
	}
	if len(rendered.Inbounds) != 3 || len(rendered.Outbounds) != 3 {
		t.Fatalf("bridge entries = %d/%d, want 3/3", len(rendered.Inbounds), len(rendered.Outbounds))
	}
	if rendered.Inbounds[0].Port != 10800 || rendered.Inbounds[1].Port != 10801 || rendered.Inbounds[2].Port != 10802 {
		t.Fatalf("inbound ports = %d/%d/%d, want 10800/10801/10802", rendered.Inbounds[0].Port, rendered.Inbounds[1].Port, rendered.Inbounds[2].Port)
	}
	if got := xrayOutboundAddress(t, rendered.Outbounds[0].Settings); got != "grpc.example.com" {
		t.Fatalf("first xray outbound = %q, want grpc.example.com", got)
	}
	if got := xrayOutboundAddress(t, rendered.Outbounds[1].Settings); got != "hy2.example.com" {
		t.Fatalf("second xray outbound = %q, want hy2.example.com", got)
	}
	if got := xrayOutboundAddress(t, rendered.Outbounds[2].Settings); got != "ws.example.com" {
		t.Fatalf("second xray outbound = %q, want ws.example.com", got)
	}

	surgeOutput, err := (&Client{
		SiteName:       "Alpha",
		SubscribeName:  "Test Plan",
		ClientTemplate: surgeXray.templateText,
		OutputFormat:   surgeXray.outputFormat,
		Proxies:        proxies,
		UserInfo:       user,
		Params:         params,
	}).Build()
	if err != nil {
		t.Fatalf("build surge xray template: %v", err)
	}
	surge := string(surgeOutput)
	grpcLine := "Grpc Node = socks5, 127.0.0.1, 10800"
	hy2Line := "Hysteria2 Node = socks5, 127.0.0.1, 10801"
	wsLine := "WS Node = socks5, 127.0.0.1, 10802"
	grpcIndex := strings.Index(surge, grpcLine)
	hy2Index := strings.Index(surge, hy2Line)
	wsIndex := strings.Index(surge, wsLine)
	if grpcIndex < 0 || hy2Index < 0 || wsIndex < 0 {
		t.Fatalf("missing bridge lines in surge output:\n%s", surge)
	}
	if grpcIndex > hy2Index || hy2Index > wsIndex {
		t.Fatalf("surge xray bridge order does not match xray core order")
	}
}

func TestXHTTPClientTemplateSupportMatrix(t *testing.T) {
	xhttpProxy := Proxy{
		Sort:             1,
		Name:             "XHTTP Node",
		Server:           "xhttp.example.com",
		Port:             443,
		Type:             "vless",
		Security:         "reality",
		SNI:              "www.ebay.com",
		RealityPublicKey: "preview-public-key",
		RealityShortId:   "0123456789abcdef",
		Transport:        "xhttp",
		Host:             "xhttp.example.com",
		Path:             "/xhttp",
		XhttpMode:        "stream-up",
		XhttpExtra:       `{"noGRPCHeader":true,"xPaddingBytes":"100-1000","xmux":{"maxConcurrency":0,"maxConnections":1}}`,
		Fingerprint:      "chrome",
	}
	fallbackProxy := Proxy{
		Sort:      2,
		Name:      "WS Fallback",
		Server:    "fallback.example.com",
		Port:      8443,
		Type:      "vmess",
		Security:  "tls",
		SNI:       "fallback.example.com",
		Transport: "ws",
		Host:      "fallback.example.com",
		Path:      "/ws",
	}
	user := User{
		Password:     "00000000-0000-0000-0000-000000000000",
		ExpiredAt:    time.Unix(1_893_456_000, 0),
		Download:     1024,
		Upload:       2048,
		Traffic:      1024 * 1024 * 1024,
		SubscribeURL: "https://example.com/sub/token",
	}
	params := map[string]string{
		"xray_inbound_host":       "127.0.0.1",
		"xray_inbound_start_port": "10800",
	}
	directXHTTPTemplates := map[int]bool{1: true, 2: true, 3: true, 6: true, 14: true, 15: true}

	for _, item := range loadEffectiveMigrationTemplates(t) {
		t.Run(item.name, func(t *testing.T) {
			rendered := renderTemplateString(t, item, []Proxy{xhttpProxy, fallbackProxy}, user, params)
			hasXHTTPNode := strings.Contains(rendered, "XHTTP Node") || strings.Contains(rendered, "XHTTP%20Node")
			if !directXHTTPTemplates[item.id] {
				if hasXHTTPNode {
					t.Fatalf("%s should not render unsupported xhttp node:\n%s", item.name, rendered)
				}
				return
			}

			switch item.id {
			case 1, 2:
				for _, want := range []string{"type=xhttp", "host=xhttp.example.com", "mode=stream-up", "extra="} {
					if !strings.Contains(rendered, want) {
						t.Fatalf("%s missing %q in xhttp URI:\n%s", item.name, want, rendered)
					}
				}
				if !strings.Contains(rendered, "path=%2Fxhttp") && !strings.Contains(rendered, "path=/xhttp") {
					t.Fatalf("%s missing xhttp path in URI:\n%s", item.name, rendered)
				}
			case 3, 6:
				proxy := findYAMLProxy(t, rendered, "XHTTP Node")
				if proxy["network"] != "xhttp" {
					t.Fatalf("%s network = %#v, want xhttp", item.name, proxy["network"])
				}
				if item.id == 3 && strings.Contains(rendered, "max-connections: \"1\"\n\n  tls: true") {
					t.Fatalf("clash xhttp proxy has an extra blank line before tls:\n%s", rendered)
				}
				opts, ok := proxy["xhttp-opts"].(map[string]interface{})
				if !ok {
					t.Fatalf("%s missing xhttp-opts: %#v", item.name, proxy)
				}
				assertYAMLField(t, opts, "path", "/xhttp")
				assertYAMLField(t, opts, "host", "xhttp.example.com")
				assertYAMLField(t, opts, "mode", "stream-up")
				assertYAMLField(t, opts, "no-grpc-header", true)
				assertYAMLField(t, opts, "x-padding-bytes", "100-1000")
				reuse, ok := opts["reuse-settings"].(map[string]interface{})
				if !ok {
					t.Fatalf("%s missing reuse-settings: %#v", item.name, opts)
				}
				assertYAMLField(t, reuse, "max-connections", "1")
				if _, ok := reuse["max-concurrency"]; ok {
					t.Fatalf("%s should omit disabled max-concurrency: %#v", item.name, reuse)
				}
			case 14:
				var config struct {
					Outbounds []struct {
						StreamSettings struct {
							Network       string                 `json:"network"`
							XHTTPSettings map[string]interface{} `json:"xhttpSettings"`
						} `json:"streamSettings"`
					} `json:"outbounds"`
				}
				if err := json.Unmarshal([]byte(rendered), &config); err != nil {
					t.Fatalf("invalid xray core json: %v", err)
				}
				if len(config.Outbounds) == 0 {
					t.Fatal("missing xray core outbounds")
				}
				stream := config.Outbounds[0].StreamSettings
				if stream.Network != "xhttp" {
					t.Fatalf("xray core network = %q, want xhttp", stream.Network)
				}
				assertJSONField(t, stream.XHTTPSettings, "path", "/xhttp")
				assertJSONField(t, stream.XHTTPSettings, "host", "xhttp.example.com")
				assertJSONField(t, stream.XHTTPSettings, "mode", "stream-up")
				extra, ok := stream.XHTTPSettings["extra"].(map[string]interface{})
				if !ok {
					t.Fatalf("xray core missing xhttp extra: %#v", stream.XHTTPSettings)
				}
				xmux, ok := extra["xmux"].(map[string]interface{})
				if !ok {
					t.Fatalf("xray core missing xmux: %#v", extra)
				}
				assertJSONField(t, xmux, "maxConnections", float64(1))
			case 15:
				if !strings.Contains(rendered, "XHTTP Node = socks5, 127.0.0.1, 10800") {
					t.Fatalf("surge xray missing xhttp bridge:\n%s", rendered)
				}
			}
		})
	}
}

func findMigrationTemplate(t *testing.T, id int) migrationTemplate {
	t.Helper()
	for _, item := range loadEffectiveMigrationTemplates(t) {
		if item.id == id {
			return item
		}
	}
	t.Fatalf("missing subscription template id %d", id)
	return migrationTemplate{}
}

func xrayOutboundAddress(t *testing.T, settings map[string]interface{}) string {
	t.Helper()
	if address, ok := settings["address"].(string); ok && address != "" {
		return address
	}
	vnext, ok := settings["vnext"].([]interface{})
	if !ok || len(vnext) == 0 {
		t.Fatalf("missing vnext settings: %#v", settings)
	}
	server, ok := vnext[0].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid vnext item: %#v", vnext[0])
	}
	address, ok := server["address"].(string)
	if !ok {
		t.Fatalf("missing outbound address: %#v", server)
	}
	return address
}

func renderTemplateString(t *testing.T, item migrationTemplate, proxies []Proxy, user User, params map[string]string) string {
	t.Helper()
	output, err := (&Client{
		SiteName:       "Alpha",
		SubscribeName:  "Test Plan",
		ClientTemplate: item.templateText,
		OutputFormat:   item.outputFormat,
		Proxies:        proxies,
		UserInfo:       user,
		Params:         params,
	}).Build()
	if err != nil {
		t.Fatalf("build template: %v", err)
	}
	rendered := string(output)
	if item.outputFormat == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(rendered)
		if err != nil {
			t.Fatalf("decode base64 output: %v", err)
		}
		rendered = string(decoded)
	}
	return rendered
}

func findYAMLProxy(t *testing.T, rendered string, name string) map[string]interface{} {
	t.Helper()
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(rendered), &data); err != nil {
		t.Fatalf("invalid yaml: %v\n%s", err, rendered)
	}
	proxies, ok := data["proxies"].([]interface{})
	if !ok {
		t.Fatalf("missing proxies list: %#v", data["proxies"])
	}
	for _, item := range proxies {
		proxy, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if proxy["name"] == name {
			return proxy
		}
	}
	t.Fatalf("missing proxy %q in:\n%s", name, rendered)
	return nil
}

func assertYAMLField(t *testing.T, data map[string]interface{}, key string, want interface{}) {
	t.Helper()
	if data[key] != want {
		t.Fatalf("%s = %#v, want %#v", key, data[key], want)
	}
}

func assertJSONField(t *testing.T, data map[string]interface{}, key string, want interface{}) {
	t.Helper()
	if data[key] != want {
		t.Fatalf("%s = %#v, want %#v", key, data[key], want)
	}
}

func loadEffectiveMigrationTemplates(t *testing.T) []migrationTemplate {
	t.Helper()
	paths := []string{
		"../initialize/migrate/database/02132_update_subscription_templates.up.sql",
		"../initialize/migrate/database/02135_refresh_subscription_protocol_fields.up.sql",
		"../initialize/migrate/database/02162_refresh_xray_core_hysteria_template.up.sql",
	}
	byID := map[int]migrationTemplate{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, item := range extractMigrationTemplates(string(data)) {
			byID[item.id] = item
		}
	}
	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	out := make([]migrationTemplate, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out
}

func extractMigrationTemplates(sql string) []migrationTemplate {
	pattern := regexp.MustCompile(`(?s)VALUES \((\d+), '([^']+)'.*?\$alpha_subscription_template\$(.*?)\$alpha_subscription_template\$, '([^']+)'`)
	matches := pattern.FindAllStringSubmatch(sql, -1)
	out := make([]migrationTemplate, 0, len(matches))
	for _, match := range matches {
		id, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		out = append(out, migrationTemplate{
			id:           id,
			name:         match[2],
			templateText: match[3],
			outputFormat: match[4],
		})
	}
	return out
}
