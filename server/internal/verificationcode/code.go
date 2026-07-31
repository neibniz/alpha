package verificationcode

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"
)

var compareAndDelete = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	redis.call("DEL", KEYS[2])
	return redis.call("DEL", KEYS[1])
end
return 0
`)

var recordMismatch = redis.NewScript(`
local attempts = redis.call("INCR", KEYS[2])
if attempts == 1 then
	local ttl = redis.call("TTL", KEYS[1])
	if ttl < 1 then ttl = 300 end
	redis.call("EXPIRE", KEYS[2], ttl)
end
if attempts >= tonumber(ARGV[1]) then
	redis.call("DEL", KEYS[1], KEYS[2])
end
return attempts
`)

const maxVerificationAttempts = 5

type payload struct {
	Code string `json:"code"`
}

func Verify(ctx context.Context, client redis.Cmdable, key, expected string) (bool, error) {
	value, ok, err := load(ctx, client, key)
	if err != nil || !ok {
		return false, err
	}
	if matches(value, expected) {
		return true, nil
	}
	if err := registerMismatch(ctx, client, key); err != nil {
		return false, err
	}
	return false, nil
}

func Consume(ctx context.Context, client redis.Cmdable, key, expected string) (bool, error) {
	value, ok, err := load(ctx, client, key)
	if err != nil || !ok {
		return false, err
	}
	if !matches(value, expected) {
		if err := registerMismatch(ctx, client, key); err != nil {
			return false, err
		}
		return false, nil
	}
	deleted, err := compareAndDelete.Run(ctx, client, []string{key, attemptKey(key)}, value).Int64()
	if err != nil {
		return false, err
	}
	return deleted == 1, nil
}

func registerMismatch(ctx context.Context, client redis.Cmdable, key string) error {
	return recordMismatch.Run(ctx, client, []string{key, attemptKey(key)}, maxVerificationAttempts).Err()
}

func attemptKey(key string) string {
	return key + ":attempts"
}

func load(ctx context.Context, client redis.Cmdable, key string) (string, bool, error) {
	if client == nil {
		return "", false, errors.New("verification code redis client is nil")
	}
	value, err := client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	return value, err == nil, err
}

func matches(value, expected string) bool {
	if expected == "" {
		return false
	}
	actual := value
	var decoded payload
	if json.Unmarshal([]byte(value), &decoded) == nil {
		actual = decoded.Code
	}
	if len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
