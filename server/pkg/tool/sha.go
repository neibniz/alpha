package tool

import (
	"crypto/sha256"
	"encoding/hex"
)

func GenerateShortID(privateKey string) string {
	hashValue := sha256.Sum256([]byte(privateKey))
	return hex.EncodeToString(hashValue[:4])
}
