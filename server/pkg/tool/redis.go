package tool

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func ParseRedisURI(uri string) (addr, password string, database int, err error) {
	parsedURI, err := url.Parse(uri)

	if err != nil {
		return "", "", 0, err
	}
	host := parsedURI.Hostname()
	port := parsedURI.Port()
	if port == "" {
		port = "6379"
	}
	addr = fmt.Sprintf("%s:%s", host, port)

	// password
	if parsedURI.User != nil {
		password, _ = parsedURI.User.Password()
	}
	if len(parsedURI.Path) > 1 { // Path: "/0"
		var dbIndex int
		_, err = fmt.Sscanf(parsedURI.Path, "/%d", &dbIndex)
		if err == nil {
			database = dbIndex
		}
	}
	return
}

func RedisPing(addr, password string, database int) error {
	rds := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       database,
	})
	defer rds.Close()
	return rds.Ping(context.Background()).Err()
}

func RedisWritableCheck(addr, password string, database int) error {
	rds := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       database,
	})
	defer rds.Close()
	return RedisClientWritableCheck(context.Background(), rds)
}

func RedisClientWritableCheck(ctx context.Context, rds *redis.Client) error {
	if err := rds.Ping(ctx).Err(); err != nil {
		return err
	}
	key := fmt.Sprintf("alpha:redis:writable-check:%d", time.Now().UnixNano())
	if err := rds.Set(ctx, key, "1", 30*time.Second).Err(); err != nil {
		if strings.Contains(err.Error(), "READONLY") {
			return fmt.Errorf("redis is a read-only replica; configure Redis.Host to a writable primary: %w", err)
		}
		return fmt.Errorf("redis write check failed: %w", err)
	}
	if err := rds.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis cleanup check key failed: %w", err)
	}
	return nil
}
