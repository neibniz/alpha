package auth

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type BindDeviceLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBindDeviceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindDeviceLogic {
	return &BindDeviceLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// BindDeviceToUser records the device used by an authenticated user.
func (l *BindDeviceLogic) BindDeviceToUser(identifier, ip, userAgent string, currentUserId int64) error {
	if identifier == "" {
		// No device identifier provided, skip binding
		return nil
	}

	l.Infow("binding device to user",
		logger.Field("identifier", logIdentifier(identifier)),
		logger.Field("user_id", currentUserId),
		logger.Field("ip", logIP(ip)),
	)

	// Check if device exists
	deviceInfo, err := l.svcCtx.UserModel.FindOneDeviceByIdentifierForStatusUpdate(l.ctx, identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Device not found, create new device record
			return l.createDeviceForUser(identifier, ip, userAgent, currentUserId)
		}
		l.Errorw("failed to query device",
			logger.Field("identifier", logIdentifier(identifier)),
			logger.Field("error", err.Error()),
		)
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query device failed: %v", err.Error())
	}

	// Device exists, check if it's bound to current user
	if deviceInfo.UserId == currentUserId {
		// Already bound to current user, just update IP and UserAgent
		l.Infow("device already bound to current user, updating info",
			logger.Field("identifier", logIdentifier(identifier)),
			logger.Field("user_id", currentUserId),
		)
		if err := l.svcCtx.UserModel.UpdateDeviceFields(l.ctx, deviceInfo, user.DeviceFieldsUpdate{
			IP:        user.UpdateValue(ip),
			UserAgent: user.UpdateValue(userAgent),
		}); err != nil {
			l.Errorw("failed to update device",
				logger.Field("identifier", logIdentifier(identifier)),
				logger.Field("error", err.Error()),
			)
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update device failed: %v", err.Error())
		}
		return nil
	}

	// Device records are not sign-in identities. Reassign the record without
	// changing either user's authentication methods or account status.
	l.Infow("device bound to another user, rebinding",
		logger.Field("identifier", logIdentifier(identifier)),
		logger.Field("old_user_id", deviceInfo.UserId),
		logger.Field("new_user_id", currentUserId),
	)

	return l.rebindDeviceToNewUser(deviceInfo, ip, userAgent, currentUserId)
}

func (l *BindDeviceLogic) createDeviceForUser(identifier, ip, userAgent string, userId int64) error {
	l.Infow("creating new device for user",
		logger.Field("identifier", logIdentifier(identifier)),
		logger.Field("user_id", userId),
	)

	err := l.svcCtx.UserModel.Transaction(l.ctx, func(db *gorm.DB) error {
		// Create device record
		deviceInfo := &user.Device{
			Ip:         ip,
			UserId:     userId,
			UserAgent:  userAgent,
			Identifier: identifier,
			Enabled:    true,
			Online:     false,
		}
		if err := l.svcCtx.UserModel.InsertDevice(l.ctx, deviceInfo, db); err != nil {
			l.Errorw("failed to create device",
				logger.Field("user_id", userId),
				logger.Field("identifier", logIdentifier(identifier)),
				logger.Field("error", err.Error()),
			)
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create device failed: %v", err)
		}

		return nil
	})

	if err != nil {
		l.Errorw("device creation failed",
			logger.Field("identifier", logIdentifier(identifier)),
			logger.Field("user_id", userId),
			logger.Field("error", err.Error()),
		)
		return err
	}

	l.Infow("device created successfully",
		logger.Field("identifier", logIdentifier(identifier)),
		logger.Field("user_id", userId),
	)

	return nil
}

func (l *BindDeviceLogic) rebindDeviceToNewUser(deviceInfo *user.Device, ip, userAgent string, newUserId int64) error {
	oldUserId := deviceInfo.UserId
	if err := l.svcCtx.UserModel.UpdateDeviceFields(l.ctx, deviceInfo, user.DeviceFieldsUpdate{
		UserID:    user.UpdateValue(newUserId),
		IP:        user.UpdateValue(ip),
		UserAgent: user.UpdateValue(userAgent),
		Enabled:   user.UpdateValue(true),
	}); err != nil {
		l.Errorw("device rebinding failed",
			logger.Field("identifier", logIdentifier(deviceInfo.Identifier)),
			logger.Field("old_user_id", oldUserId),
			logger.Field("new_user_id", newUserId),
			logger.Field("error", err.Error()),
		)
		return err
	}

	l.Infow("device rebound successfully",
		logger.Field("identifier", logIdentifier(deviceInfo.Identifier)),
		logger.Field("old_user_id", oldUserId),
		logger.Field("new_user_id", newUserId),
	)

	return nil
}
