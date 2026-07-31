package authsession

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/jwt"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/redis/go-redis/v9"
)

var ErrInvalidConfiguration = errors.New("invalid authentication session configuration")

func Issue(ctx context.Context, client *redis.Client, cfg config.JwtAuth, userID int64, loginType string) (string, error) {
	if client == nil || userID <= 0 || cfg.AccessSecret == "" || cfg.AccessExpire <= 0 {
		return "", ErrInvalidConfiguration
	}

	sessionID := uuidx.NewUUID().String()
	token, err := jwt.NewJwtToken(
		cfg.AccessSecret,
		time.Now().Unix(),
		cfg.AccessExpire,
		jwt.WithOption("UserId", userID),
		jwt.WithOption("SessionId", sessionID),
		jwt.WithOption("LoginType", loginType),
	)
	if err != nil {
		return "", fmt.Errorf("sign authentication token: %w", err)
	}

	ttl := time.Duration(cfg.AccessExpire) * time.Second
	indexKey := config.UserSessionCacheKey(userID)
	pipe := client.TxPipeline()
	pipe.Set(ctx, config.SessionCacheKey(sessionID), userID, ttl)
	pipe.SAdd(ctx, indexKey, sessionID)
	pipe.Expire(ctx, indexKey, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("store authentication session: %w", err)
	}
	return token, nil
}

func RevokeSession(ctx context.Context, client *redis.Client, userID int64, sessionID string) error {
	if client == nil || userID <= 0 || strings.TrimSpace(sessionID) == "" {
		return ErrInvalidConfiguration
	}
	pipe := client.TxPipeline()
	pipe.Del(ctx, config.SessionCacheKey(sessionID))
	pipe.SRem(ctx, config.UserSessionCacheKey(userID), sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("revoke authentication session: %w", err)
	}
	return nil
}

func RevokeUser(ctx context.Context, client *redis.Client, userID int64) error {
	if client == nil || userID <= 0 {
		return ErrInvalidConfiguration
	}
	indexKey := config.UserSessionCacheKey(userID)
	sessionIDs, err := client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return fmt.Errorf("load user authentication sessions: %w", err)
	}
	keys := make([]string, 0, len(sessionIDs)+1)
	keys = append(keys, indexKey)
	seen := make(map[string]struct{}, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		if sessionID != "" {
			key := config.SessionCacheKey(sessionID)
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
	}
	var cursor uint64
	for {
		legacyKeys, next, scanErr := client.Scan(ctx, cursor, config.SessionIdKey+":*", 500).Result()
		if scanErr != nil {
			return fmt.Errorf("scan authentication sessions: %w", scanErr)
		}
		if len(legacyKeys) > 0 {
			values, getErr := client.MGet(ctx, legacyKeys...).Result()
			if getErr != nil {
				return fmt.Errorf("load authentication sessions: %w", getErr)
			}
			for index, value := range values {
				if fmt.Sprint(value) != config.SessionUserValue(userID) {
					continue
				}
				if _, exists := seen[legacyKeys[index]]; !exists {
					keys = append(keys, legacyKeys[index])
					seen[legacyKeys[index]] = struct{}{}
				}
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if err := client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("revoke user authentication sessions: %w", err)
	}
	return nil
}

func ResolveLoginType(ctx context.Context, fallback string) string {
	if ctx != nil {
		if loginType, ok := ctx.Value(constant.LoginType).(string); ok && strings.TrimSpace(loginType) != "" {
			return loginType
		}
	}
	return fallback
}
