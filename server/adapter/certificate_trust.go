package adapter

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"
)

func isHysteria2Type(value string) bool {
	switch strings.ToLower(value) {
	case "hysteria", "hysteria2", "hy2":
		return true
	default:
		return false
	}
}

func normalizeHysteriaCertificateTrust(proxy Proxy) (Proxy, error) {
	pins := make(map[string]bool)
	ordered := make([]string, 0)
	for _, value := range strings.FieldsFunc(proxy.PinnedPeerCertSha256, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	}) {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		value = strings.NewReplacer(":", "", "-", "").Replace(value)
		decoded, err := hex.DecodeString(value)
		if err != nil || len(decoded) != sha256.Size {
			return Proxy{}, fmt.Errorf("certificate SHA-256 must contain exactly 32 bytes")
		}
		if !pins[value] {
			pins[value] = true
			ordered = append(ordered, value)
		}
	}
	if strings.TrimSpace(proxy.PinnedPeerCertSha256) != "" && len(ordered) == 0 {
		return Proxy{}, fmt.Errorf("certificate SHA-256 must contain exactly 32 bytes")
	}
	proxy.PinnedPeerCertSha256 = strings.Join(ordered, ",")
	certificates := make([]string, 0)
	seen := make(map[string]bool)
	for _, source := range proxy.TLSCertificate {
		for remaining := []byte(source); len(remaining) > 0; {
			block, rest := pem.Decode(remaining)
			if block == nil {
				break
			}
			remaining = rest
			// Never expose private keys or non-certificate PEM blocks to subscriptions.
			if block.Type != "CERTIFICATE" {
				continue
			}
			certificate, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				continue
			}
			sum := sha256.Sum256(certificate.Raw)
			fingerprint := hex.EncodeToString(sum[:])
			if seen[fingerprint] || (len(pins) > 0 && !pins[fingerprint]) {
				continue
			}
			seen[fingerprint] = true
			certificates = append(certificates, string(pem.EncodeToMemory(&pem.Block{
				Type: "CERTIFICATE", Bytes: certificate.Raw,
			})))
		}
	}
	proxy.TLSCertificate = certificates
	return proxy, nil
}
