package user

import (
	"context"
	"encoding/json"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type VerifyEmailLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Verify Email
func NewVerifyEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyEmailLogic {
	return &VerifyEmailLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type CacheKeyPayload struct {
	Code   string `json:"code"`
	LastAt int64  `json:"lastAt"`
}

func (l *VerifyEmailLogic) VerifyEmail(req *types.VerifyEmailRequest) error {
	cacheKey := config.EmailAuthCodeCacheKey(constant.Security.String(), l.svcCtx.Privacy.CacheToken("email", req.Email))
	value, err := l.svcCtx.Redis.Get(l.ctx, cacheKey).Result()
	if err != nil {
		l.Errorw("Redis Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.VerifyCodeError), "code error")
	}

	var payload CacheKeyPayload
	err = json.Unmarshal([]byte(value), &payload)
	if err != nil {
		l.Errorw("Redis Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.VerifyCodeError), "code error")
	}
	if payload.Code != req.Code {
		return errors.Wrapf(xerr.NewErrCode(xerr.VerifyCodeError), "code error")
	}
	l.svcCtx.Redis.Del(l.ctx, cacheKey)

	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	method, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "email", req.Email)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindUserAuthMethodByOpenID error")
	}
	if method.UserId != u.Id {
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "invalid access")
	}
	err = l.svcCtx.UserModel.UpdateUserAuthMethods(l.ctx, method.UserId, method.AuthType, user.UserAuthMethodFieldsUpdate{
		Verified: user.UpdateValue(true),
	})
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "UpdateUserAuthMethods error")
	}
	return nil
}
