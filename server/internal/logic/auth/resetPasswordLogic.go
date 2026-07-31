package auth

import (
	"context"

	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/internal/model/user"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/verificationcode"
)

type ResetPasswordLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewResetPasswordLogic Reset password
func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetPasswordLogic) ResetPassword(req *types.ResetPasswordRequest) (resp *types.LoginResponse, err error) {
	var userInfo *user.User
	loginStatus := false

	defer func() {
		if userInfo != nil && userInfo.Id != 0 && loginStatus {
			if err := recordLoginAudit(l.ctx, l.svcCtx, userInfo.Id, "email", req.IP, req.UserAgent, loginStatus); err != nil {
				l.Errorw("failed to insert login log",
					logger.Field("user_id", userInfo.Id),
					logger.Field("error", err.Error()),
				)
			}
		}
	}()

	cacheKey := config.EmailAuthCodeCacheKey(constant.Security.String(), l.svcCtx.Privacy.CacheToken("email", req.Email))
	verified, verifyErr := verificationcode.Consume(l.ctx, l.svcCtx.Redis, cacheKey, req.Code)
	if verifyErr != nil {
		l.Errorw("verification code storage error", logger.Field("error", verifyErr.Error()))
	}
	if verifyErr != nil || !verified {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.VerifyCodeError), "Verification code error")
	}

	// Check user
	authMethod, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "email", req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authFailedError()
		}
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "find user by email failed")
	}

	userInfo, err = l.svcCtx.UserModel.FindOneForLogin(l.ctx, authMethod.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authFailedError()
		}
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query user info failed")
	}
	if userInfo == nil || userInfo.DeletedAt.Valid {
		return nil, authFailedError()
	}

	// Update password
	passwordHash, err := tool.EncodePassWord(req.Password)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "password does not meet policy")
	}
	if err = l.svcCtx.UserModel.UpdateUserPassword(l.ctx, userInfo.Id, passwordHash); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseUpdateError), "update user info failed")
	}
	if err = authsession.RevokeUser(l.ctx, l.svcCtx.Redis, userInfo.Id); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "revoke existing authentication sessions")
	}

	// Bind device to user if identifier is provided
	if req.Identifier != "" {
		bindLogic := NewBindDeviceLogic(l.ctx, l.svcCtx)
		if err := bindLogic.BindDeviceToUser(req.Identifier, req.IP, req.UserAgent, userInfo.Id); err != nil {
			l.Errorw("failed to bind device to user",
				logger.Field("user_id", userInfo.Id),
				logger.Field("identifier", logIdentifier(req.Identifier)),
				logger.Field("error", err.Error()),
			)
			// Don't fail register if device binding fails, just log the error
		}
	}
	loginType := authsession.ResolveLoginType(l.ctx, req.LoginType)
	token, err := authsession.Issue(l.ctx, l.svcCtx.Redis, l.svcCtx.Config.JwtAuth, userInfo.Id, loginType)
	if err != nil {
		l.Errorw("failed to issue authentication session",
			logger.Field("user_id", userInfo.Id),
			logger.Field("error", err.Error()),
		)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "create authentication session")
	}
	loginStatus = true
	return &types.LoginResponse{
		Token: token,
	}, nil
}
