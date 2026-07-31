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

type KickOfflineByUserDeviceLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// kick offline user device
func NewKickOfflineByUserDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickOfflineByUserDeviceLogic {
	return &KickOfflineByUserDeviceLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *KickOfflineByUserDeviceLogic) KickOfflineByUserDevice(req *types.KickOfflineRequest) error {
	device, err := l.svcCtx.UserModel.FindOneDeviceForStatusUpdate(l.ctx, req.Id)
	if err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get Device  error: %v", err.Error())
	}
	l.svcCtx.DeviceManager.KickDevice(device.UserId, device.Identifier)
	err = l.svcCtx.UserModel.UpdateDeviceFields(l.ctx, device, user.DeviceFieldsUpdate{
		Online: user.UpdateValue(false),
	})
	if err != nil {
		l.Logger.Error("[KickOfflineByUserDeviceLogic] Update Device Error:", logger.Field("err", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update Device error: %v", err.Error())
	}

	return nil
}
