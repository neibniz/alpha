package adapter

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type hysteriaTrustPatch struct{ old, updated, format string }

func hysteriaTrustMigrationPatches(t *testing.T) []hysteriaTrustPatch {
	t.Helper()
	sql, err := os.ReadFile("../initialize/migrate/database/02219_hy2_certificate_trust.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`(?s)REPLACE\("subscribe_template",\s*\$hy2_old\$(.*?)\$hy2_old\$,\s*\$hy2_new\$(.*?)\$hy2_new\$\),\s*"updated_at" = NOW\(\)\s*WHERE "output_format" = '([^']+)'`)
	patches := make([]hysteriaTrustPatch, 0)
	for _, match := range pattern.FindAllStringSubmatch(string(sql), -1) {
		patches = append(patches, hysteriaTrustPatch{match[1], match[2], match[3]})
	}
	if len(patches) != 10 {
		t.Fatalf("migration patches = %d, want 10", len(patches))
	}
	return patches
}

func trustTestCertificate(t *testing.T) (string, string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "hy2.example.test"}, DNSNames: []string{"hy2.example.test"}, NotBefore: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), NotAfter: time.Date(2040, 1, 1, 0, 0, 0, 0, time.UTC), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pin := sha256.Sum256(der)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})), string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private})), hex.EncodeToString(pin[:])
}

func TestHysteriaTrustNormalizesPinsAndNeverExposesKeys(t *testing.T) {
	certificate, private, pin := trustTestCertificate(t)
	colon := strings.Join(regexp.MustCompile(`..`).FindAllString(strings.ToUpper(pin), -1), ":")
	proxy, err := normalizeCertificateTrust(Proxy{PinnedPeerCertSha256: colon + "," + pin, TLSCertificate: []string{certificate + private, certificate}})
	if err != nil {
		t.Fatal(err)
	}
	if proxy.PinnedPeerCertSha256 != pin || len(proxy.TLSCertificate) != 1 || proxy.TLSCertificate[0] != certificate {
		t.Fatal("certificate trust did not normalize or deduplicate")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(proxy.TLSCertificate[0])) {
		t.Fatal("invalid trust anchor")
	}
	block, _ := pem.Decode([]byte(certificate))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = cert.Verify(x509.VerifyOptions{DNSName: "hy2.example.test", Roots: roots, CurrentTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	other, _, _ := trustTestCertificate(t)
	block, _ = pem.Decode([]byte(other))
	cert, _ = x509.ParseCertificate(block.Bytes)
	if _, err = cert.Verify(x509.VerifyOptions{DNSName: "hy2.example.test", Roots: roots, CurrentTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}); err == nil {
		t.Fatal("untrusted replacement certificate was accepted")
	}
	for _, invalid := range []string{"chrome", "AA:BB:CC", ",;", strings.Repeat("a", 63), strings.Repeat("g", 64)} {
		if _, err := normalizeCertificateTrust(Proxy{PinnedPeerCertSha256: invalid}); err == nil {
			t.Fatalf("accepted invalid pin %q", invalid)
		}
	}
}

func TestHysteriaTemplatesCarryClientSpecificCertificateTrust(t *testing.T) {
	certificate, private, pin := trustTestCertificate(t)
	for _, id := range []int{1, 2, 3, 4, 5, 6, 8, 11, 12, 13, 14} {
		item := findMigrationTemplate(t, id)
		t.Run(item.name, func(t *testing.T) {
			output := renderTemplateString(t, item, []Proxy{{Name: "HY2", Type: "hysteria2", Transport: "hysteria", Security: "tls", Server: "hy2.example.test", Port: 443, SNI: "hy2.example.test", Fingerprint: "chrome", ServerKey: "user-password", PinnedPeerCertSha256: pin, TLSCertificate: []string{certificate + private}, AllowInsecure: true}}, User{Password: "user-password", ExpiredAt: time.Unix(1893456000, 0)}, nil)
			if strings.Contains(output, "PRIVATE KEY") {
				t.Fatal("private key leaked into subscription")
			}
			switch id {
			case 1, 2:
				var line string
				for _, value := range strings.Split(output, "\n") {
					if strings.HasPrefix(value, "hysteria2://") {
						line = value
						break
					}
				}
				parsed, err := url.Parse(line)
				if err != nil || parsed == nil {
					t.Fatalf("invalid URI: %v", err)
				}
				if parsed.Query().Get("pinSHA256") != pin || parsed.Query().Get("insecure") != "1" {
					t.Fatal("URI is missing pinned certificate mode")
				}
			case 3, 6:
				proxy := findYAMLProxy(t, output, "HY2")
				field := "fingerprint"
				if id == 6 {
					field = "server-cert-fingerprint"
					assertYAMLField(t, proxy, "auth", "user-password")
				}
				assertYAMLField(t, proxy, field, pin)
				if proxy["skip-cert-verify"] == true {
					t.Fatal("unnecessary skip-cert-verify with certificate pin")
				}
			case 4, 12, 13:
				var config struct {
					Outbounds []struct {
						Type string `json:"type"`
						TLS  struct {
							Certificate []string `json:"certificate"`
							Insecure    bool     `json:"insecure"`
						} `json:"tls"`
					} `json:"outbounds"`
				}
				if err := json.Unmarshal([]byte(output), &config); err != nil {
					t.Fatal(err)
				}
				found := false
				for _, outbound := range config.Outbounds {
					if outbound.Type == "hysteria2" {
						found = true
						if len(outbound.TLS.Certificate) != 1 || outbound.TLS.Certificate[0] != certificate || outbound.TLS.Insecure {
							t.Fatal("missing strict public certificate trust")
						}
					}
				}
				if !found {
					t.Fatal("HY2 outbound missing")
				}
			case 5:
				if !strings.Contains(output, "server-cert-fingerprint-sha256="+pin) {
					t.Fatal("Surge certificate pin missing")
				}
			case 8:
				if !strings.Contains(output, "tls-cert-sha256="+pin) {
					t.Fatal("Loon certificate pin missing")
				}
			case 11:
				var config struct {
					Proxies []struct {
						Hysteria2 map[string]any `yaml:"hysteria2"`
					} `yaml:"proxies"`
				}
				if err := yaml.Unmarshal([]byte(output), &config); err != nil {
					t.Fatal(err)
				}
				if len(config.Proxies) != 1 {
					t.Fatal("Egern HY2 node missing")
				}
				assertYAMLField(t, config.Proxies[0].Hysteria2, "fingerprint_sha256", pin)
				assertYAMLField(t, config.Proxies[0].Hysteria2, "auth", "user-password")
			case 14:
				if !strings.Contains(output, `"pinnedPeerCertSha256": "`+pin+`"`) {
					t.Fatal("Xray certificate pin missing")
				}
			}
		})
	}
}

func TestLegacySingBoxFailsClosedWithoutMatchingCertificate(t *testing.T) {
	certificate, _, pin := trustTestCertificate(t)
	for _, id := range []int{4, 12, 13} {
		for _, certificates := range [][]string{nil, {certificate}} {
			item := findMigrationTemplate(t, id)
			_, err := (&Client{ClientTemplate: item.templateText, OutputFormat: item.outputFormat, Proxies: []Proxy{{Name: "HY2", Type: "hysteria2", Security: "tls", PinnedPeerCertSha256: strings.Repeat("0", 64), TLSCertificate: certificates, AllowInsecure: true}}}).Build()
			if err == nil || !strings.Contains(err.Error(), "matching inline public PEM") {
				t.Fatalf("template %d did not fail closed: %v (pin %s)", id, err, pin)
			}
		}
	}
}

func TestHysteriaTrustMigrationPreservesCustomizationsAndIsIdempotent(t *testing.T) {
	for _, patch := range hysteriaTrustMigrationPatches(t) {
		original := "custom-prefix\n" + patch.old + "\ncustom-suffix"
		updated := strings.ReplaceAll(original, patch.old, patch.updated)
		if !strings.HasPrefix(updated, "custom-prefix\n") || !strings.HasSuffix(updated, "\ncustom-suffix") {
			t.Fatal("migration dropped custom content")
		}
		if strings.ReplaceAll(updated, patch.old, patch.updated) != updated {
			t.Fatal("migration is not idempotent")
		}
		customized := strings.Replace(patch.old, "$proxy.SNI", "$proxy.CustomSNI", 1)
		if customized != patch.old && strings.ReplaceAll(customized, patch.old, patch.updated) != customized {
			t.Fatal("migration overwrote a customized block")
		}
	}
}
