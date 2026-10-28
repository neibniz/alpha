package adapter

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type clientCryptoPatch struct{ old, updated, format, marker string }

func clientCryptoMigrationPatches(t *testing.T) []clientCryptoPatch {
	t.Helper()
	sql, err := os.ReadFile("../initialize/migrate/database/02220_client_tls_reality_compatibility.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`(?s)REPLACE\("subscribe_template",\s*\$crypto_old\$(.*?)\$crypto_old\$,\s*\$crypto_new\$(.*?)\$crypto_new\$\),\s*"updated_at" = NOW\(\)\s*WHERE "output_format" = '([^']+)'\s*AND strpos\("subscribe_template", '([^']+)'\) > 0`)
	var patches []clientCryptoPatch
	for _, match := range pattern.FindAllStringSubmatch(string(sql), -1) {
		patches = append(patches, clientCryptoPatch{match[1], match[2], match[3], match[4]})
	}
	if len(patches) != 12 {
		t.Fatalf("migration patches = %d, want 12", len(patches))
	}
	return patches
}

func TestClientCryptoMigrationMatchesReferences(t *testing.T) {
	for _, item := range loadEffectiveMigrationTemplates(t) {
		paths, err := filepath.Glob(fmt.Sprintf("../reference/subscription-templates/%02d-*", item.id))
		if err != nil || len(paths) != 1 {
			t.Fatalf("reference %d: %v", item.id, err)
		}
		data, err := os.ReadFile(paths[0])
		if err != nil {
			t.Fatal(err)
		}
		_, body, ok := strings.Cut(string(data), "*/}}")
		if !ok || strings.TrimSpace(body) != strings.TrimSpace(item.templateText) {
			t.Errorf("migration differs from reference %s", paths[0])
			actual, expected := strings.Split(strings.TrimSpace(item.templateText), "\n"), strings.Split(strings.TrimSpace(body), "\n")
			for i := 0; i < len(actual) && i < len(expected); i++ {
				if actual[i] != expected[i] {
					t.Logf("first mismatch at line %d: migration %q, reference %q", i+1, actual[i], expected[i])
					break
				}
			}
		}
	}
	for _, patch := range clientCryptoMigrationPatches(t) {
		original := "custom-prefix\n" + patch.old + "\ncustom-suffix"
		updated := strings.ReplaceAll(original, patch.old, patch.updated)
		if strings.ReplaceAll(updated, patch.old, patch.updated) != updated {
			t.Fatalf("non-idempotent patch for %s", patch.marker)
		}
		if !strings.HasPrefix(updated, "custom-prefix\n") || !strings.HasSuffix(updated, "\ncustom-suffix") {
			t.Fatal("custom content lost")
		}
	}
}

func TestRealityTemplatesEnableCompatibleHandshake(t *testing.T) {
	user := User{Password: "00000000-0000-0000-0000-000000000001", ExpiredAt: time.Unix(1893456000, 0)}
	for _, fingerprint := range []string{"", "edge", "chrome140"} {
		for _, id := range []int{1, 2, 3, 4, 6, 8, 12, 13, 14} {
			item := findMigrationTemplate(t, id)
			t.Run(fmt.Sprintf("%s/%s", item.name, fingerprint), func(t *testing.T) {
				proxy := Proxy{Name: "REALITY", Type: "vless", Security: "reality", Server: "node.example.test", Port: 443, SNI: "www.example.test", Transport: "grpc", RealityPublicKey: "public-key", RealityShortId: "01234567", Fingerprint: fingerprint}
				output := renderTemplateString(t, item, []Proxy{proxy}, user, nil)
				switch id {
				case 1, 2:
					for _, line := range strings.Split(output, "\n") {
						if !strings.HasPrefix(line, "vless://") {
							continue
						}
						u, err := url.Parse(line)
						if err != nil {
							t.Fatal(err)
						}
						want := "chrome"
						if id == 2 {
							want = "chrome140"
						}
						if u.Query().Get("fp") != want {
							t.Fatalf("wrong handshake fingerprint: %s", u.Query().Get("fp"))
						}
						return
					}
					t.Fatal("missing VLESS URI")
				case 3, 6:
					node := findYAMLProxy(t, output, "REALITY")
					opts, ok := node["reality-opts"].(map[string]any)
					if !ok || opts["support-x25519mlkem768"] != true || node["client-fingerprint"] != "chrome" {
						t.Fatalf("missing ML-KEM handshake: %#v", node)
					}
				case 8:
					for _, field := range []string{"tls-profile=chrome147", "public-key=public-key", "short-id=01234567", "over-tls=true"} {
						if !strings.Contains(output, field) {
							t.Errorf("missing %s", field)
						}
					}
				default:
					var doc any
					if err := json.Unmarshal([]byte(output), &doc); err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(output, `"chrome"`) || strings.Contains(output, `"chrome140"`) || strings.Contains(output, `"edge"`) {
						t.Fatal("incompatible REALITY ClientHello")
					}
				}
			})
		}
	}
}

func TestTLSCertificatePinsAcrossClients(t *testing.T) {
	certificate, private, pin := trustTestCertificate(t)
	user := User{Password: "00000000-0000-0000-0000-000000000001", ExpiredAt: time.Unix(1893456000, 0)}
	for _, kind := range []string{"vmess", "vless", "trojan", "tuic", "anytls"} {
		for _, id := range []int{3, 4, 5, 6, 8, 12, 13, 14} {
			if id == 13 && kind == "anytls" {
				continue
			}
			if id == 14 && (kind == "tuic" || kind == "anytls") {
				continue
			}
			if id == 8 && (kind == "tuic" || kind == "anytls") {
				continue
			}
			item := findMigrationTemplate(t, id)
			t.Run(item.name+"/"+kind, func(t *testing.T) {
				proxy := Proxy{Name: "TLS", Type: kind, Security: "tls", Server: "hy2.example.test", Port: 443, SNI: "hy2.example.test", Fingerprint: "chrome", PinnedPeerCertSha256: strings.ToUpper(pin), TLSCertificate: []string{certificate + private}, AllowInsecure: true}
				output := renderTemplateString(t, item, []Proxy{proxy}, user, nil)
				if strings.Contains(output, "PRIVATE KEY") || strings.Contains(output, "support-x25519mlkem768") {
					t.Fatal("leaked key or REALITY options into ordinary TLS")
				}
				switch id {
				case 3, 6:
					node := findYAMLProxy(t, output, "TLS")
					field := "fingerprint"
					if id == 6 {
						field = "server-cert-fingerprint"
					}
					if node[field] != pin || node["skip-cert-verify"] == true {
						t.Fatalf("incorrect certificate trust: %#v", node)
					}
				case 5, 8:
					field := "server-cert-fingerprint-sha256="
					if id == 8 {
						field = "tls-cert-sha256="
					}
					if !strings.Contains(output, field+pin) || strings.Contains(output, "skip-cert-verify=true") {
						t.Fatal("missing pin or verification disabled")
					}
				default:
					var doc any
					if err := json.Unmarshal([]byte(output), &doc); err != nil {
						t.Fatal(err)
					}
					if id == 14 {
						if !strings.Contains(output, pin) {
							t.Fatal("missing Xray certificate pin")
						}
					} else if !strings.Contains(output, `"certificate"`) || strings.Contains(output, `"insecure": true`) {
						t.Fatal("missing certificate trust or verification disabled")
					}
				}
			})
		}
	}
}

func TestInvalidTLSPinFailsBeforeTemplateRendering(t *testing.T) {
	for _, kind := range []string{"vmess", "vless", "trojan", "hysteria2", "tuic", "anytls"} {
		_, err := (&Client{ClientTemplate: `{{ len .Proxies }}`, Proxies: []Proxy{{Type: kind, PinnedPeerCertSha256: "chrome"}}}).Build()
		if err == nil {
			t.Errorf("accepted invalid certificate pin for %s", kind)
		}
	}
}

func TestShadowrocketTLSUsesChrome140(t *testing.T) {
	item := findMigrationTemplate(t, 2)
	output := renderTemplateString(t, item, []Proxy{{Name: "TLS", Type: "vless", Security: "tls", Server: "node.example.test", Port: 443, Fingerprint: "edge"}}, User{Password: "test-user", ExpiredAt: time.Unix(1893456000, 0)}, nil)
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "vless://") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil {
			t.Fatal(err)
		}
		if u.Query().Get("fp") != "chrome140" || u.Query().Get("security") != "tls" || u.Query().Has("pbk") {
			t.Fatal("incorrect ordinary TLS fingerprint or security")
		}
		return
	}
	t.Fatal("missing VLESS TLS URI")
}

func TestAdditionalClientsRetainCertificateVerification(t *testing.T) {
	pin := strings.Repeat("ab", 32)
	user := User{Password: "test-user", ExpiredAt: time.Unix(1893456000, 0)}
	for _, id := range []int{7, 10} {
		for _, kind := range []string{"vmess", "trojan", "https"} {
			if id == 10 && kind == "https" {
				kind = "http"
			}
			item := findMigrationTemplate(t, id)
			output := renderTemplateString(t, item, []Proxy{{Name: "TLS", Type: kind, Security: "tls", TLS: true, Transport: "ws", Server: "node.example.test", SNI: "node.example.test", Port: 443, PinnedPeerCertSha256: pin, AllowInsecure: true}}, user, nil)
			field := "server-cert-fingerprint-sha256="
			if id == 10 {
				field = "tls-cert-sha256="
			}
			if !strings.Contains(output, field+pin) || strings.Contains(output, "tls-verification=false") || strings.Contains(output, "skip-cert-verify=true") {
				t.Errorf("%s/%s dropped certificate verification", item.name, kind)
			}
		}
	}
	for _, id := range []int{4, 12, 13} {
		item := findMigrationTemplate(t, id)
		_, err := (&Client{ClientTemplate: item.templateText, OutputFormat: item.outputFormat, Proxies: []Proxy{{Name: "TLS", Type: "trojan", Security: "tls", Server: "node.example.test", Port: 443, PinnedPeerCertSha256: pin, AllowInsecure: true}}, UserInfo: user}).Build()
		if err == nil || !strings.Contains(err.Error(), "matching inline public PEM") {
			t.Errorf("%s silently ignored a TLS pin without PEM: %v", item.name, err)
		}
	}
}
