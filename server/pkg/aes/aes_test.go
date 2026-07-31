package pkgaes

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAes(t *testing.T) {
	params := map[string]interface{}{
		"method":   "email",
		"account":  "admin@alpha.dev",
		"password": "password",
	}
	marshal, _ := json.Marshal(params)
	jsonStr := string(marshal)
	encrypt, iv, err := Encrypt([]byte(jsonStr), "123456")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	decrypt, err := Decrypt(encrypt, "123456", iv)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	assert.Equal(t, jsonStr, decrypt, "decrypt failed")
	assert.True(t, strings.HasPrefix(iv, cipherVersionPrefix))

}

func TestAesRejectsTamperedCiphertext(t *testing.T) {
	cipherText, nonce, err := Encrypt([]byte("sensitive"), "device-secret")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	tampered := []byte(cipherText)
	if tampered[len(tampered)-2] == 'A' {
		tampered[len(tampered)-2] = 'B'
	} else {
		tampered[len(tampered)-2] = 'A'
	}
	if _, err := Decrypt(string(tampered), "device-secret", nonce); err == nil {
		t.Fatal("tampered ciphertext must be rejected")
	}
}

func TestAesRejectsLegacyNonce(t *testing.T) {
	if _, err := Decrypt("ZmFrZQ==", "device-secret", "legacy-time"); err == nil {
		t.Fatal("legacy unauthenticated ciphertext must be rejected")
	}
}
