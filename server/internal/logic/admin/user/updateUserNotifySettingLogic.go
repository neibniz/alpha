package user

import (
	"context"

	userModel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserNotifySettingLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateUserNotifySettingLogic Update user notify setting
func NewUpdateUserNotifySettingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserNotifySettingLogic {
	return &UpdateUserNotifySettingLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserNotifySettingLogic) UpdateUserNotifySetting(req *types.UpdateUserNotifySettingRequest) error {
	update := userModel.UserNotifySettingsUpdate{
		EnableBalanceNotify:   userModel.UpdateValue(req.EnableBalanceNotify),
		EnableLoginNotify:     userModel.UpdateValue(req.EnableLoginNotify),
		EnableSubscribeNotify: userModel.UpdateValue(req.EnableSubscribeNotify),
		EnableTradeNotify:     userModel.UpdateValue(req.EnableTradeNotify),
	}
	if err := l.svcCtx.UserModel.UpdateUserNotifySettings(l.ctx, req.UserId, update); err != nil {
		l.Errorw("[UpdateUserNotifySettingLogic] Update User Error:", logger.Field("err", err.Error()), logger.Field("userId", req.UserId))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "Update User Error")
	}
	return nil
}
