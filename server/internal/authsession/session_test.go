package authsession

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/jwt"
	"github.com/redis/go-redis/v9"
)

func TestIssueStoresSessionAndCompleteClaims(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cfg := config.JwtAuth{AccessSecret: "test-secret", AccessExpire: 600}

	token, err := Issue(context.Background(), client, cfg, 42, "web")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := jwt.ParseJwtToken(token, cfg.AccessSecret)
	if err != nil {
		t.Fatal(err)
	}
	if claims["UserId"] != float64(42) || claims["LoginType"] != "web" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	sessionID, ok := claims["SessionId"].(string)
	if !ok || sessionID == "" {
		t.Fatalf("missing session id claim: %#v", claims)
	}
	key := config.SessionCacheKey(sessionID)
	got, err := mini.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Fatalf("stored user id = %q, want 42", got)
	}
	if ttl := mini.TTL(key); ttl != 10*time.Minute {
		t.Fatalf("session ttl = %v, want 10m", ttl)
	}
	indexKey := config.UserSessionCacheKey(42)
	if member, err := client.SIsMember(context.Background(), indexKey, sessionID).Result(); err != nil || !member {
		t.Fatalf("session %q is not indexed for user", sessionID)
	}
	if ttl := mini.TTL(indexKey); ttl != 10*time.Minute {
		t.Fatalf("session index ttl = %v, want 10m", ttl)
	}
}

func TestRevokeSessionAndUser(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cfg := config.JwtAuth{AccessSecret: "test-secret", AccessExpire: 600}

	first, err := Issue(context.Background(), client, cfg, 42, "web")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Issue(context.Background(), client, cfg, 42, "web")
	if err != nil {
		t.Fatal(err)
	}
	firstClaims, _ := jwt.ParseJwtToken(first, cfg.AccessSecret)
	secondClaims, _ := jwt.ParseJwtToken(second, cfg.AccessSecret)
	firstID := firstClaims["SessionId"].(string)
	secondID := secondClaims["SessionId"].(string)
	if err := client.Set(context.Background(), config.SessionCacheKey("legacy-session"), 42, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	if err := RevokeSession(context.Background(), client, 42, firstID); err != nil {
		t.Fatal(err)
	}
	if mini.Exists(config.SessionCacheKey(firstID)) {
		t.Fatal("revoked session still exists")
	}
	if !mini.Exists(config.SessionCacheKey(secondID)) {
		t.Fatal("unrelated session was revoked")
	}
	if err := RevokeUser(context.Background(), client, 42); err != nil {
		t.Fatal(err)
	}
	if mini.Exists(config.SessionCacheKey(secondID)) || mini.Exists(config.SessionCacheKey("legacy-session")) || mini.Exists(config.UserSessionCacheKey(42)) {
		t.Fatal("user sessions were not fully revoked")
	}
}

func TestIssueRejectsInvalidConfiguration(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	for name, testCase := range map[string]struct {
		client *redis.Client
		cfg    config.JwtAuth
		userID int64
	}{
		"nil redis":    {cfg: config.JwtAuth{AccessSecret: "secret", AccessExpire: 60}, userID: 1},
		"empty secret": {client: client, cfg: config.JwtAuth{AccessExpire: 60}, userID: 1},
		"invalid ttl":  {client: client, cfg: config.JwtAuth{AccessSecret: "secret"}, userID: 1},
		"invalid user": {client: client, cfg: config.JwtAuth{AccessSecret: "secret", AccessExpire: 60}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Issue(context.Background(), testCase.client, testCase.cfg, testCase.userID, "web"); !errors.Is(err, ErrInvalidConfiguration) {
				t.Fatalf("Issue() error = %v, want ErrInvalidConfiguration", err)
			}
		})
	}
}

func TestResolveLoginType(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.LoginType, "desktop")
	if got := ResolveLoginType(ctx, "web"); got != "desktop" {
		t.Fatalf("ResolveLoginType() = %q, want desktop", got)
	}
	badType := context.WithValue(context.Background(), constant.LoginType, 7)
	if got := ResolveLoginType(badType, "web"); got != "web" {
		t.Fatalf("ResolveLoginType() with invalid context value = %q, want web", got)
	}
}
