package nodeauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strconv"
)

const tokenPrefix = "alpha-node-v1_" // #nosec G101 -- public token format marker, not a credential

var ErrInvalidInput = errors.New("node authentication secret and server id are required")

func DeriveToken(masterSecret string, serverID int64) (string, error) {
	if masterSecret == "" || serverID <= 0 {
		return "", ErrInvalidInput
	}
	mac := hmac.New(sha256.New, []byte(masterSecret))
	mac.Write([]byte("alpha-node-auth:v1:"))
	mac.Write([]byte(strconv.FormatInt(serverID, 10)))
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func VerifyToken(masterSecret string, serverID int64, provided string) bool {
	expected, err := DeriveToken(masterSecret, serverID)
	if err != nil || len(expected) != len(provided) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}
