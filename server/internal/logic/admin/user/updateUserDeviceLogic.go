package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateUserDeviceLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// User device
func NewUpdateUserDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserDeviceLogic {
	return &UpdateUserDeviceLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserDeviceLogic) UpdateUserDevice(req *types.UserDevice) error {
	device, err := l.svcCtx.UserModel.FindOneDeviceForStatusUpdate(l.ctx, req.Id)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get Device  error: %v", err.Error())
	}
	err = l.svcCtx.UserModel.UpdateDeviceFields(l.ctx, device, user.DeviceFieldsUpdate{
		Enabled: user.UpdateValue(req.Enabled),
	})
	if err != nil {
		l.Logger.Error("[UpdateUserDeviceLogic] Update Device Error:", logger.Field("err", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update Device error: %v", err.Error())
	}
	return nil
}
