package authlimit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	loginWindow       = 15 * time.Minute
	accountFailureMax = int64(5)
	pairFailureMax    = int64(5)
	ipFailureMax      = int64(50)
)

type Limit struct {
	Key            string
	Max            int64
	ResetOnSuccess bool
}

var incrementFailure = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

func LoginLimits(kind, account, ip string) []Limit {
	accountDigest := digest(kind, account)
	ipDigest := digest(ip)
	return []Limit{
		{Key: "auth:failure:login:account:" + accountDigest, Max: accountFailureMax, ResetOnSuccess: true},
		{Key: "auth:failure:login:pair:" + digest(accountDigest, ipDigest), Max: pairFailureMax, ResetOnSuccess: true},
		{Key: "auth:failure:login:ip:" + ipDigest, Max: ipFailureMax},
	}
}

func Blocked(ctx context.Context, client redis.Cmdable, limits []Limit) (bool, error) {
	if client == nil {
		return false, errors.New("authentication limiter redis client is nil")
	}
	for _, limit := range limits {
		value, err := client.Get(ctx, limit.Key).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return false, err
		}
		count, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return false, err
		}
		if count >= limit.Max {
			return true, nil
		}
	}
	return false, nil
}

func RecordFailure(ctx context.Context, client redis.Cmdable, limits []Limit) error {
	if client == nil {
		return errors.New("authentication limiter redis client is nil")
	}
	for _, limit := range limits {
		if _, err := incrementFailure.Run(ctx, client, []string{limit.Key}, int64(loginWindow/time.Second)).Result(); err != nil {
			return err
		}
	}
	return nil
}

func ResetSuccessful(ctx context.Context, client redis.Cmdable, limits []Limit) error {
	if client == nil {
		return errors.New("authentication limiter redis client is nil")
	}
	keys := make([]string, 0, len(limits))
	for _, limit := range limits {
		if limit.ResetOnSuccess {
			keys = append(keys, limit.Key)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return client.Del(ctx, keys...).Err()
}

func digest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		hash.Write([]byte(strings.ToLower(strings.TrimSpace(part))))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
