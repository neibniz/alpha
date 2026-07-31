package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const (
	envelopePrefix = "enc:v1:"
	maxKeyringSize = 64 * 1024
)

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

type Keyring struct {
	ActiveKeyID   string            `json:"active_key_id"`
	EncryptionKey map[string]string `json:"encryption_keys"`
	IndexKey      string            `json:"index_key"`
}

type Codec struct {
	enabled     bool
	activeKeyID string
	keys        map[string][]byte
	indexKey    []byte
	rand        io.Reader
}

func DisabledCodec() *Codec {
	return &Codec{rand: rand.Reader}
}

func LoadCodec(enabled bool, keyFile string) (*Codec, error) {
	if !enabled {
		return DisabledCodec(), nil
	}
	if strings.TrimSpace(keyFile) == "" {
		return nil, errors.New("privacy key file is required when privacy encryption is enabled")
	}
	info, err := os.Stat(keyFile)
	if err != nil {
		return nil, fmt.Errorf("stat privacy key file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("privacy key file must be a regular file")
	}
	if info.Size() > maxKeyringSize {
		return nil, fmt.Errorf("privacy key file exceeds %d bytes", maxKeyringSize)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("privacy key file permissions must not allow group or other access: %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(keyFile) // #nosec G304 -- operator path was validated as a private regular file above
	if err != nil {
		return nil, fmt.Errorf("read privacy key file: %w", err)
	}
	var keyring Keyring
	if err := json.Unmarshal(data, &keyring); err != nil {
		return nil, fmt.Errorf("decode privacy key file: %w", err)
	}
	return NewCodec(keyring)
}

func NewCodec(keyring Keyring) (*Codec, error) {
	if !keyIDPattern.MatchString(keyring.ActiveKeyID) {
		return nil, fmt.Errorf("invalid active privacy key id %q", keyring.ActiveKeyID)
	}
	keys := make(map[string][]byte, len(keyring.EncryptionKey))
	for keyID, encoded := range keyring.EncryptionKey {
		if !keyIDPattern.MatchString(keyID) {
			return nil, fmt.Errorf("invalid privacy key id %q", keyID)
		}
		key, err := decodeKey(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode privacy encryption key %q: %w", keyID, err)
		}
		keys[keyID] = key
	}
	if _, ok := keys[keyring.ActiveKeyID]; !ok {
		return nil, fmt.Errorf("active privacy key %q is missing", keyring.ActiveKeyID)
	}
	indexKey, err := decodeKey(keyring.IndexKey)
	if err != nil {
		return nil, fmt.Errorf("decode privacy index key: %w", err)
	}
	return &Codec{
		enabled:     true,
		activeKeyID: keyring.ActiveKeyID,
		keys:        keys,
		indexKey:    indexKey,
		rand:        rand.Reader,
	}, nil
}

func decodeKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key length is %d, want 32", len(key))
	}
	return key, nil
}

func (c *Codec) Enabled() bool {
	return c != nil && c.enabled
}

func IsSealed(value string) bool {
	return strings.HasPrefix(value, envelopePrefix)
}

func (c *Codec) Seal(scope, plaintext string) (string, error) {
	if !c.Enabled() || plaintext == "" {
		return plaintext, nil
	}
	key := c.keys[c.activeKeyID]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(c.rand, nonce); err != nil {
		return "", fmt.Errorf("generate privacy nonce: %w", err)
	}
	sealed := aead.Seal(nonce, nonce, []byte(plaintext), []byte(scope))
	return envelopePrefix + c.activeKeyID + ":" + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *Codec) Open(scope, value string) (string, error) {
	if !c.Enabled() || value == "" || !strings.HasPrefix(value, envelopePrefix) {
		return value, nil
	}
	rest := strings.TrimPrefix(value, envelopePrefix)
	keyID, encoded, ok := strings.Cut(rest, ":")
	if !ok || !keyIDPattern.MatchString(keyID) {
		return "", errors.New("invalid privacy envelope")
	}
	key, ok := c.keys[keyID]
	if !ok {
		return "", fmt.Errorf("privacy key %q is unavailable", keyID)
	}
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode privacy envelope: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < aead.NonceSize() {
		return "", errors.New("privacy envelope is truncated")
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, []byte(scope))
	if err != nil {
		return "", errors.New("privacy envelope authentication failed")
	}
	return string(plaintext), nil
}

func (c *Codec) BlindIndex(kind, value string) string {
	normalized := Normalize(kind, value)
	if normalized == "" {
		return ""
	}
	if !c.Enabled() {
		return normalized
	}
	mac := hmac.New(sha256.New, c.indexKey)
	_, _ = mac.Write([]byte(kind))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(normalized))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Codec) CacheToken(kind, value string) string {
	return c.BlindIndex(kind, value)
}

func Normalize(kind, value string) string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(kind) {
	case "email":
		return strings.ToLower(value)
	default:
		return value
	}
}

func IdentityScope(userID int64, authType string) string {
	return fmt.Sprintf("user_auth_methods.auth_identifier:%d:%s", userID, strings.ToLower(strings.TrimSpace(authType)))
}

func EmailTargetScope(taskID int64) string {
	return fmt.Sprintf("task_email_targets.email:%d", taskID)
}

func DeviceScope(userID int64, field string) string {
	return fmt.Sprintf("user_device.%s:%d", field, userID)
}

func WithdrawalScope(userID int64, field string) string {
	return fmt.Sprintf("user_withdrawal.%s:%d", field, userID)
}

func (c *Codec) ProtectIdentity(userID int64, authType, value string) (ciphertext, blindIndex, display string, err error) {
	normalized := Normalize(authType, value)
	ciphertext, err = c.Seal(IdentityScope(userID, authType), normalized)
	if err != nil {
		return "", "", "", err
	}
	return ciphertext, c.BlindIndex(authType, normalized), MaskIdentifier(normalized), nil
}

func (c *Codec) OpenIdentity(userID int64, authType, value string) (string, error) {
	return c.Open(IdentityScope(userID, authType), value)
}
