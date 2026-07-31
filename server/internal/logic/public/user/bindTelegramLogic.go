package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type BindTelegramLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Bind Telegram
func NewBindTelegramLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindTelegramLogic {
	return &BindTelegramLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindTelegramLogic) BindTelegram() (resp *types.BindTelegramResponse, err error) {
	currentUser, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User)
	if !ok || currentUser == nil || currentUser.Id <= 0 {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is unavailable")
	}
	telegramConfig, _ := l.svcCtx.TelegramRuntime()
	if strings.TrimSpace(telegramConfig.BotName) == "" {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "telegram bot is unavailable")
	}
	token, err := uuidx.NewSubscribeToken()
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate telegram binding token")
	}
	const bindTTL = 5 * time.Minute
	if err := l.svcCtx.Redis.Set(l.ctx, config.TelegramBindCacheKey(token), currentUser.Id, bindTTL).Err(); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "store telegram binding token")
	}
	return &types.BindTelegramResponse{
		Url:       fmt.Sprintf("https://t.me/%s?start=%s", telegramConfig.BotName, token),
		ExpiredAt: time.Now().Add(bindTTL).UnixMilli(),
	}, nil
}
