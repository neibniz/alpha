package common

import (
	"context"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/verificationcode"
	"github.com/neibniz/alpha/server/pkg/authmethod"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type CheckVerificationCodeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Check verification code
func NewCheckVerificationCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckVerificationCodeLogic {
	return &CheckVerificationCodeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckVerificationCodeLogic) CheckVerificationCode(req *types.CheckVerificationCodeRequest) (resp *types.CheckVerificationCodeRespone, err error) {
	resp = &types.CheckVerificationCodeRespone{}
	verifyType := constant.ParseVerifyType(req.Type).String()
	if req.Method == authmethod.Email {
		cacheKey := config.EmailAuthCodeCacheKey(verifyType, l.svcCtx.Privacy.CacheToken("email", req.Account))
		resp.Status, _ = verificationcode.Verify(l.ctx, l.svcCtx.Redis, cacheKey, req.Code)
	}
	return resp, nil
}
