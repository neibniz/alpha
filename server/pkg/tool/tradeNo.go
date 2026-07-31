package tool

import (
	"fmt"
	"time"

	"github.com/neibniz/alpha/server/pkg/random"
)

func GenerateTradeNo() (string, error) {
	now := time.Now()
	digits, err := random.SecureKey(10, 0)
	if err != nil {
		return "", err
	}
	return now.Format("20060102150405") + fmt.Sprintf("%03d", now.Nanosecond()/1_000_000) + digits, nil
}
