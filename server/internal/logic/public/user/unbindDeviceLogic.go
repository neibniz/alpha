package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type UnbindDeviceLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

var unbindDeviceLookupColumns = []string{
	"id",
	"user_id",
	"identifier",
}

// Unbind Device
func NewUnbindDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnbindDeviceLogic {
	return &UnbindDeviceLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnbindDeviceLogic) UnbindDevice(req *types.UnbindDeviceRequest) error {
	userInfo, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok || userInfo == nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is unavailable")
	}

	return l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var deleteDevice user.Device
		err := tx.Model(&deleteDevice).
			Select(unbindDeviceLookupColumns).
			Where("id = ?", req.Id).
			First(&deleteDevice).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.Wrapf(xerr.NewErrCode(xerr.DeviceNotExist), "find device")
			}
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find device err: %v", err)
		}

		if deleteDevice.UserId != userInfo.Id {
			return errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "device not belong to user")
		}

		if err = l.svcCtx.UserModel.DeleteDeviceByModel(l.ctx, &deleteDevice, tx); err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "delete device err: %v", err)
		}

		return nil
	})
}
