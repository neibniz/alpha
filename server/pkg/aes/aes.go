package pkgaes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

const cipherVersionPrefix = "v2."

// Encrypt protects device payloads with AES-256-GCM. The second return value is
// an opaque, versioned nonce token and must be passed back unchanged to Decrypt.
func Encrypt(plainText []byte, keyStr string) (string, string, error) {
	gcm, err := newGCM(keyStr)
	if err != nil {
		return "", "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", fmt.Errorf("generate AES-GCM nonce: %w", err)
	}

	cipherText := gcm.Seal(nil, nonce, plainText, nil)
	nonceToken := cipherVersionPrefix + base64.RawURLEncoding.EncodeToString(nonce)
	return base64.StdEncoding.EncodeToString(cipherText), nonceToken, nil
}

// Decrypt authenticates and decrypts a device payload. Legacy unauthenticated
// CBC payloads are intentionally rejected; callers must use the v2 GCM format.
func Decrypt(cipherText string, keyStr string, nonceToken string) (string, error) {
	encodedNonce, ok := strings.CutPrefix(nonceToken, cipherVersionPrefix)
	if !ok {
		return "", fmt.Errorf("unsupported device cipher version")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(encodedNonce)
	if err != nil {
		return "", fmt.Errorf("decode AES-GCM nonce: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("decode AES-GCM ciphertext: %w", err)
	}

	gcm, err := newGCM(keyStr)
	if err != nil {
		return "", err
	}
	if len(nonce) != gcm.NonceSize() {
		return "", fmt.Errorf("invalid AES-GCM nonce length")
	}
	plainText, err := gcm.Open(nil, nonce, decoded, nil)
	if err != nil {
		return "", fmt.Errorf("authenticate AES-GCM ciphertext: %w", err)
	}
	return string(plainText), nil
}

func newGCM(key string) (cipher.AEAD, error) {
	block, err := aes.NewCipher(generateKey(key))
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM cipher: %w", err)
	}
	return gcm, nil
}

func generateKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:]
}
