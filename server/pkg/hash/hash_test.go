package hash

import (
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	text         = "hello, world!\n"
	sha256Digest = "4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc"
)

func TestSHA256(t *testing.T) {
	actual := fmt.Sprintf("%x", SHA256([]byte(text)))
	assert.Equal(t, sha256Digest, actual)
}

func TestSHA256Hex(t *testing.T) {
	actual := SHA256Hex([]byte(text))
	assert.Equal(t, sha256Digest, actual)
}

func BenchmarkHashFnv(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := fnv.New32()
		new(big.Int).SetBytes(h.Sum([]byte(text))).Int64()
	}
}

func BenchmarkHashSHA256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := sha256.New()
		bytes := h.Sum([]byte(text))
		new(big.Int).SetBytes(bytes).Int64()
	}
}

func BenchmarkMurmur3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Hash([]byte(text))
	}
}
