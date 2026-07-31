package auth

import (
	"context"

	"github.com/neibniz/alpha/server/internal/authaudit"
	"github.com/neibniz/alpha/server/internal/svc"
)

func recordLoginAudit(ctx context.Context, svcCtx *svc.ServiceContext, userID int64, method, ip, userAgent string, success bool) error {
	return authaudit.RecordLogin(ctx, svcCtx.LogModel, authaudit.LoginEvent{
		UserID:    userID,
		Method:    method,
		IP:        ip,
		UserAgent: userAgent,
		Success:   success,
	})
}

func recordRegisterAudit(ctx context.Context, svcCtx *svc.ServiceContext, userID int64, method, identifier, ip, userAgent string) error {
	return authaudit.RecordRegister(ctx, svcCtx.LogModel, authaudit.RegisterEvent{
		UserID:     userID,
		Method:     method,
		Identifier: identifier,
		IP:         ip,
		UserAgent:  userAgent,
	})
}
