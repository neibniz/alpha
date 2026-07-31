package auth

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type LogoutLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout(ip, userAgent string) error {
	userInfo, _ := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	sessionID, _ := l.ctx.Value(constant.CtxKeySessionID).(string)
	if sessionID != "" && userInfo != nil && userInfo.Id > 0 {
		if err := authsession.RevokeSession(l.ctx, l.svcCtx.Redis, userInfo.Id, sessionID); err != nil {
			l.Debugw("[Logout] delete session error", logger.Field("error", err.Error()), logger.Field("session_id", sessionID))
		}
	}
	if userInfo == nil || userInfo.Id <= 0 {
		return nil
	}
	content, _ := (&log.Logout{
		LogoutIP:  ip,
		UserAgent: userAgent,
		Timestamp: time.Now().UnixMilli(),
	}).Marshal()
	if err := l.svcCtx.LogModel.Insert(l.ctx, &log.SystemLog{
		Type:     log.TypeLogout.Uint8(),
		Date:     time.Now().Format(time.DateOnly),
		ObjectID: userInfo.Id,
		Content:  string(content),
	}); err != nil {
		l.Debugw("[Logout] insert logout log error", logger.Field("error", err.Error()), logger.Field("user_id", userInfo.Id))
	}
	return nil
}
