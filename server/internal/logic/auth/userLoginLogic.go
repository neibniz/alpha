package auth

import (
	"context"

	"github.com/neibniz/alpha/server/internal/authlimit"
	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/pkg/logger"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
)

type UserLoginLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUserLoginLogic User login
func NewUserLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserLoginLogic {
	return &UserLoginLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UserLoginLogic) UserLogin(req *types.UserLoginRequest) (resp *types.LoginResponse, err error) {
	loginLimits := authlimit.LoginLimits("email", req.Email, req.IP)
	blocked, err := authlimit.Blocked(l.ctx, l.svcCtx.Redis, loginLimits)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "check authentication failure limit")
	}
	if blocked {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.TooManyRequests), "too many failed login attempts")
	}
	authFailure := func() error {
		if recordErr := authlimit.RecordFailure(l.ctx, l.svcCtx.Redis, loginLimits); recordErr != nil {
			return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "record authentication failure")
		}
		return authFailedError()
	}

	loginStatus := false
	var userInfo *user.User
	defer func() {
		if userInfo != nil && userInfo.Id != 0 {
			if err := recordLoginAudit(l.ctx, l.svcCtx, userInfo.Id, "email", req.IP, req.UserAgent, loginStatus); err != nil {
				l.Errorw("failed to insert login log",
					logger.Field("user_id", userInfo.Id),
					logger.Field("error", err.Error()),
				)
			}
		}
	}()

	userInfo, err = l.svcCtx.UserModel.FindOneForLoginByEmail(l.ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, authFailure()
		}
		logger.WithContext(l.ctx).Error(err)
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query user info failed")
	}

	if userInfo == nil || userInfo.DeletedAt.Valid {
		return nil, authFailure()
	}
	if userInfo.Enable == nil || !*userInfo.Enable {
		return nil, authFailure()
	}

	// Verify password
	if !tool.MultiPasswordVerify(userInfo.Algo, userInfo.Salt, req.Password, userInfo.Password) {
		return nil, authFailure()
	}
	if tool.PasswordNeedsUpgrade(userInfo.Algo, userInfo.Password) {
		if upgraded, hashErr := tool.RehashPassWord(req.Password); hashErr == nil {
			if updateErr := l.svcCtx.UserModel.UpdateUserPassword(l.ctx, userInfo.Id, upgraded); updateErr != nil {
				l.Errorw("failed to upgrade legacy password hash", logger.Field("user_id", userInfo.Id), logger.Field("error", updateErr.Error()))
			}
		}
	}
	if err := authlimit.ResetSuccessful(l.ctx, l.svcCtx.Redis, loginLimits); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "reset authentication failure limit")
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
			// Don't fail login if device binding fails, just log the error
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
