package user

import (
	"context"

	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserNotifyLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update User Notify
func NewUpdateUserNotifyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserNotifyLogic {
	return &UpdateUserNotifyLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserNotifyLogic) UpdateUserNotify(req *types.UpdateUserNotifyRequest) error {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	if u.Id == 0 {
		return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "user not login")
	}
	update := user.UserNotifySettingsUpdate{
		EnableBalanceNotify:   req.EnableBalanceNotify,
		EnableLoginNotify:     req.EnableLoginNotify,
		EnableSubscribeNotify: req.EnableSubscribeNotify,
		EnableTradeNotify:     req.EnableTradeNotify,
	}
	if err := l.svcCtx.UserModel.UpdateUserNotifySettings(l.ctx, u.Id, update); err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "update user notify error: %v", err.Error())
	}
	return nil
}
