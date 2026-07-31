package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/authsession"
	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type UpdateUserPasswordLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update User Password
func NewUpdateUserPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserPasswordLogic {
	return &UpdateUserPasswordLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserPasswordLogic) UpdateUserPassword(req *types.UpdateUserPasswordRequest) error {
	currentUser, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok || currentUser == nil || currentUser.Id <= 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is not available")
	}
	userInfo, err := l.svcCtx.UserModel.FindOneForLogin(l.ctx, currentUser.Id)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query user password")
	}
	if !tool.MultiPasswordVerify(userInfo.Algo, userInfo.Salt, req.CurrentPassword, userInfo.Password) {
		return errors.Wrap(xerr.NewErrCode(xerr.UserAuthFailed), "invalid current password")
	}
	if tool.MultiPasswordVerify(userInfo.Algo, userInfo.Salt, req.Password, userInfo.Password) {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "new password must differ from current password")
	}
	passwordHash, err := tool.EncodePassWord(req.Password)
	if err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "password does not meet policy")
	}
	if err := l.svcCtx.UserModel.UpdateUserPassword(l.ctx, userInfo.Id, passwordHash); err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "Update user password error")
	}
	if err := authsession.RevokeUser(l.ctx, l.svcCtx.Redis, userInfo.Id); err != nil {
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "revoke existing authentication sessions")
	}
	return nil
}
