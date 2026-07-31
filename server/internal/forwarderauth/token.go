package forwarderauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

const tokenPrefix = "alpha-forwarder-v1_" // #nosec G101 -- public token format marker

var ErrInvalidInput = errors.New("forwarder authentication secret and instance id are required")

func DeriveToken(masterSecret, instanceID string) (string, error) {
	masterSecret = strings.TrimSpace(masterSecret)
	instanceID = strings.TrimSpace(instanceID)
	if masterSecret == "" || instanceID == "" {
		return "", ErrInvalidInput
	}
	mac := hmac.New(sha256.New, []byte(masterSecret))
	mac.Write([]byte("alpha-forwarder-auth:v1:"))
	mac.Write([]byte(instanceID))
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func VerifyToken(masterSecret, instanceID, provided string) bool {
	expected, err := DeriveToken(masterSecret, instanceID)
	if err != nil || len(expected) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
