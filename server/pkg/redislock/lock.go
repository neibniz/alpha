package redislock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

type Lock struct {
	client redis.Cmdable
	key    string
	token  string
}

func Acquire(ctx context.Context, client redis.Cmdable, key string, ttl time.Duration) (*Lock, bool, error) {
	token := uuid.NewString()
	acquired, err := client.SetNX(ctx, key, token, ttl).Result()
	if err != nil || !acquired {
		return nil, acquired, err
	}
	return &Lock{
		client: client,
		key:    key,
		token:  token,
	}, true, nil
}

func (l *Lock) Release(ctx context.Context) error {
	if l == nil {
		return nil
	}
	return releaseScript.Run(ctx, l.client, []string{l.key}, l.token).Err()
}
