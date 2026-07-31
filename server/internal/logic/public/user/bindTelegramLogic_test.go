package user

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/config"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
)

func TestBindTelegramIssuesOneTimeTokenInsteadOfSessionID(t *testing.T) {
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	service := &svc.ServiceContext{Redis: rds}
	service.SetTelegramRuntime(config.Telegram{BotName: "alpha_bot"}, nil)
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{Id: 7})

	response, err := NewBindTelegramLogic(ctx, service).BindTelegram()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(response.Url)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(parsed.Query().Get("start"), " ")
	if len(token) != 64 {
		t.Fatalf("binding token length = %d, want 64", len(token))
	}
	if got := rds.Get(context.Background(), config.TelegramBindCacheKey(token)).Val(); got != "7" {
		t.Fatalf("stored user = %q, want 7", got)
	}
}
