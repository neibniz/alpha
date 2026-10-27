package xray

import (
	"testing"

	"github.com/xtls/xray-core/common/protocol"
	ss "github.com/xtls/xray-core/proxy/shadowsocks"
)

func TestVMessSecurityTypes(t *testing.T) {
	for value, want := range map[string]protocol.SecurityType{
		"":                  protocol.SecurityType_AUTO,
		"auto":              protocol.SecurityType_AUTO,
		" AES-128-GCM ":     protocol.SecurityType_AES128_GCM,
		"chacha20_poly1305": protocol.SecurityType_CHACHA20_POLY1305,
	} {
		t.Run(value, func(t *testing.T) {
			got, err := vmessSecurityType(value)
			if err != nil || got != want {
				t.Fatalf("security = %v, err = %v; want %v", got, err, want)
			}
		})
	}
}

func TestShadowsocksCipherTypes(t *testing.T) {
	for value, want := range map[string]ss.CipherType{
		"":                        ss.CipherType_AES_128_GCM,
		"aes-128-gcm":             ss.CipherType_AES_128_GCM,
		"aead_aes_128_gcm":        ss.CipherType_AES_128_GCM,
		"AES_256_GCM":             ss.CipherType_AES_256_GCM,
		"aead_aes_256_gcm":        ss.CipherType_AES_256_GCM,
		"chacha20-poly1305":       ss.CipherType_CHACHA20_POLY1305,
		"aead_chacha20_poly1305":  ss.CipherType_CHACHA20_POLY1305,
		"chacha20-ietf-poly1305":  ss.CipherType_CHACHA20_POLY1305,
		"xchacha20-poly1305":      ss.CipherType_XCHACHA20_POLY1305,
		"aead_xchacha20_poly1305": ss.CipherType_XCHACHA20_POLY1305,
		"xchacha20-ietf-poly1305": ss.CipherType_XCHACHA20_POLY1305,
	} {
		t.Run(value, func(t *testing.T) {
			got, err := shadowsocksCipherType(value)
			if err != nil || got != want {
				t.Fatalf("cipher = %v, err = %v; want %v", got, err, want)
			}
		})
	}
}

func TestHotUsersRejectUnsupportedEncryption(t *testing.T) {
	for _, tc := range []struct {
		protocol string
		field    string
		values   []string
	}{
		{"vmess", "security", []string{"none", "zero", "invalid"}},
		{"shadowsocks", "method", []string{"none", "invalid"}},
	} {
		for _, value := range tc.values {
			t.Run(tc.protocol+"/"+value, func(t *testing.T) {
				user, err := buildProtocolUser(tc.protocol, RuntimeUser{
					Email: "test",
					Fields: map[string]interface{}{
						"id":       "b1710a33-a698-4b38-81dc-1d6b493442c0",
						"password": "test-password",
						tc.field:   value,
					},
				})
				if err == nil || user != nil {
					t.Fatalf("unsupported encryption was accepted: user = %v, err = %v", user, err)
				}
			})
		}
	}
}
