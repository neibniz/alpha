package accesspolicy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/redis/go-redis/v9"
)

const (
	defaultActiveTTL  = 3 * time.Minute
	defaultBlockedTTL = 30 * time.Second
)

var evaluateScript = redis.NewScript(`
local key = KEYS[1]
local member = ARGV[1]
local now = tonumber(ARGV[2])
local active_until = tonumber(ARGV[3])
local key_ttl = tonumber(ARGV[4])
local limit = tonumber(ARGV[5])
local blocked_until = tonumber(ARGV[6])

if limit <= 0 then
  redis.call('DEL', key)
  return {0, now, 0}
end

redis.call('ZREMRANGEBYSCORE', key, '-inf', now)
local existing = redis.call('ZSCORE', key, member)
if existing then
  redis.call('ZADD', key, active_until, member)
  redis.call('PEXPIRE', key, key_ttl)
  return {0, active_until, redis.call('ZCARD', key)}
end

local active = redis.call('ZCARD', key)
if active < limit then
  redis.call('ZADD', key, active_until, member)
  redis.call('PEXPIRE', key, key_ttl)
  return {0, active_until, active + 1}
end

redis.call('PEXPIRE', key, key_ttl)
return {1, blocked_until, active}
`)

type Decision struct {
	Blocked   bool
	ExpiresAt time.Time
	Active    int64
	Limit     int64
}

type Store struct {
	redis      *redis.Client
	privacy    *privacy.Codec
	activeTTL  time.Duration
	blockedTTL time.Duration
	now        func() time.Time
}

func NewStore(redisClient *redis.Client, privacyCodec *privacy.Codec) *Store {
	return &Store{
		redis:      redisClient,
		privacy:    privacyCodec,
		activeTTL:  defaultActiveTTL,
		blockedTTL: defaultBlockedTTL,
		now:        time.Now,
	}
}

func (s *Store) ActiveTTL() time.Duration {
	if s == nil || s.activeTTL <= 0 {
		return defaultActiveTTL
	}
	return s.activeTTL
}

func (s *Store) Evaluate(ctx context.Context, userSubscribeID int64, sourceNetwork string, limit int64) (Decision, error) {
	if s == nil || s.redis == nil {
		return Decision{}, errors.New("access IP limiter is unavailable")
	}
	if userSubscribeID <= 0 {
		return Decision{}, errors.New("user subscription ID must be positive")
	}
	network, ok := privacy.NormalizeIPNetwork(sourceNetwork)
	if !ok {
		return Decision{}, errors.New("source network is invalid")
	}

	now := s.now().UTC()
	activeUntil := now.Add(s.ActiveTTL())
	blockedUntil := now.Add(s.blockedTTL)
	keyTTL := s.ActiveTTL() + time.Minute
	result, err := evaluateScript.Run(ctx, s.redis,
		[]string{fmt.Sprintf("access:ip-limit:%d", userSubscribeID)},
		s.member(network),
		strconv.FormatInt(now.UnixMilli(), 10),
		strconv.FormatInt(activeUntil.UnixMilli(), 10),
		strconv.FormatInt(keyTTL.Milliseconds(), 10),
		strconv.FormatInt(limit, 10),
		strconv.FormatInt(blockedUntil.UnixMilli(), 10),
	).Slice()
	if err != nil {
		return Decision{}, fmt.Errorf("evaluate active IP limit: %w", err)
	}
	if len(result) != 3 {
		return Decision{}, fmt.Errorf("evaluate active IP limit returned %d fields", len(result))
	}
	blocked, err := redisInt64(result[0])
	if err != nil {
		return Decision{}, err
	}
	expiresAt, err := redisInt64(result[1])
	if err != nil {
		return Decision{}, err
	}
	active, err := redisInt64(result[2])
	if err != nil {
		return Decision{}, err
	}
	return Decision{
		Blocked:   blocked == 1,
		ExpiresAt: time.UnixMilli(expiresAt).UTC(),
		Active:    active,
		Limit:     limit,
	}, nil
}

func (s *Store) member(network string) string {
	if s.privacy != nil && s.privacy.Enabled() {
		return s.privacy.BlindIndex("xray_source_network", network)
	}
	sum := sha256.Sum256([]byte(network))
	return hex.EncodeToString(sum[:])
}

func redisInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case string:
		result, err := strconv.ParseInt(typed, 10, 64)
		if err == nil {
			return result, nil
		}
	}
	return 0, fmt.Errorf("unexpected Redis integer %T", value)
}
