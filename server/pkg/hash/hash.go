package hash

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/spaolacci/murmur3"
)

// Hash returns the hash value of data.
func Hash(data []byte) uint64 {
	return murmur3.Sum64(data)
}

// SHA256 returns the SHA-256 bytes of data.
func SHA256(data []byte) []byte {
	digest := sha256.Sum256(data)
	return digest[:]
}

// SHA256Hex returns the SHA-256 hex string of data.
func SHA256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
