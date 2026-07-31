package user

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type UpdateBindEmailLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateBindEmailLogic Update Bind Email
func NewUpdateBindEmailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBindEmailLogic {
	return &UpdateBindEmailLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateBindEmailLogic) UpdateBindEmail(req *types.UpdateBindEmailRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	method, err := l.svcCtx.UserModel.FindUserAuthMethodByUserId(l.ctx, "email", u.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindUserAuthMethodByOpenID error")
	}
	m, err := l.svcCtx.UserModel.FindUserAuthMethodByOpenID(l.ctx, "email", req.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindUserAuthMethodByOpenID error")
	}
	// email already bind
	if m.Id > 0 {
		return errors.Wrapf(xerr.NewErrCode(xerr.UserExist), "email already bind")
	}
	if method.Id == 0 {
		method = &user.AuthMethods{
			UserId:         u.Id,
			AuthType:       "email",
			AuthIdentifier: req.Email,
			Verified:       false,
		}
		if err := l.svcCtx.UserModel.InsertUserAuthMethods(l.ctx, method); err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "InsertUserAuthMethods error")
		}
	} else {
		if err := l.svcCtx.UserModel.UpdateUserAuthMethods(l.ctx, method.UserId, method.AuthType, user.UserAuthMethodFieldsUpdate{
			AuthIdentifier: user.UpdateValue(req.Email),
			Verified:       user.UpdateValue(false),
		}); err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "UpdateUserAuthMethods error")
		}
	}
	return nil
}
