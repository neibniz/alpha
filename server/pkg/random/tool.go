package random

import (
	cryptorand "crypto/rand"
	"math/big"
)

func RandomInRange(min, max int) int {
	if min > max {
		min, max = max, min
	}
	minimum := big.NewInt(int64(min))
	span := new(big.Int).Sub(big.NewInt(int64(max)), minimum)
	span.Add(span, big.NewInt(1))
	offset, err := cryptorand.Int(cryptorand.Reader, span)
	if err != nil {
		panic("secure random source unavailable")
	}
	return int(new(big.Int).Add(minimum, offset).Int64()) // #nosec G115 -- result is bounded by the int inputs
}
