package svc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"

	"github.com/neibniz/alpha/server/pkg/device"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func NewDeviceManager(srv *ServiceContext) *device.DeviceManager {
	ctx := context.Background()
	manager := device.NewDeviceManager(30, 30)

	//设备离线处理
	manager.OnDeviceOffline = func(userID int64, deviceID, session string, createAt time.Time) {
		oneDevice, err := srv.UserModel.FindOneDeviceByIdentifierForStatusUpdate(ctx, deviceID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Errorw("failed to find device", logger.Field("error", err.Error()), logger.Field("device_id", deviceID))
			}
			return
		}

		if err = srv.UserModel.UpdateDeviceFields(ctx, oneDevice, user.DeviceFieldsUpdate{
			Online: user.UpdateValue(false),
		}); err != nil {
			logger.Errorw("[DeviceManager] failed to update device", logger.Field("error", err.Error()), logger.Field("device_id", deviceID))
		}

		//当前时间为设备离线时间
		currentTime := time.Now()
		endTime := currentTime.Format("2006-01-02 00:00:00")
		parseStart, _ := time.Parse(time.DateTime, endTime)
		startTime := parseStart.Add(time.Hour * 24).Format(time.DateTime)
		deviceOnlineRecord := user.DeviceOnlineRecord{
			UserId:        userID,
			Identifier:    privacy.MaskIdentifier(deviceID),
			OnlineTime:    createAt,
			OfflineTime:   currentTime,
			OnlineSeconds: int64(currentTime.Sub(createAt).Seconds()),
		}

		//获取设备昨日在线记录
		var onlineRecord user.DeviceOnlineRecord
		if err := srv.DB.Model(&onlineRecord).Where("user_id = ? and create_at >= ? and  create_at < ?", userID, startTime, endTime).First(&onlineRecord).Error; err != nil {
			//昨日未在线，连续在线天数为1
			deviceOnlineRecord.DurationDays = 1
		} else {
			//昨日在线，连续在线天数为，昨天连续在线天数+1，等于当前连续在线天数
			deviceOnlineRecord.DurationDays = onlineRecord.DurationDays + 1
		}

		if err := srv.DB.Create(&deviceOnlineRecord).Error; err != nil {
			logger.Errorw("[DeviceOnlineRecord] failed to DeviceOnlineRecord", logger.Field("error", err.Error()), logger.Field("device_id", deviceID))
		}
	}

	//设备上线处理
	manager.OnDeviceOnline = func(userID int64, deviceID, session string) {
		oneDevice, err := srv.UserModel.FindOneDeviceByIdentifierForStatusUpdate(ctx, deviceID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Errorw("failed to find device", logger.Field("error", err.Error()), logger.Field("device_id", deviceID))
			}
			return
		}
		if err = srv.UserModel.UpdateDeviceFields(ctx, oneDevice, user.DeviceFieldsUpdate{
			Online: user.UpdateValue(true),
		}); err != nil {
			logger.Errorw("[DeviceManager] failed to update device", logger.Field("error", err.Error()), logger.Field("device_id", deviceID))
			return
		}
	}

	manager.OnDeviceKicked = func(userID int64, deviceID, session string, operator device.Operator) {
		//管理员踢下线
		if operator == device.Admin {
			message := DeviceMessage{Method: DeviceKickedAdmin}
			_ = manager.SendToDevice(userID, deviceID, message.Json())
			//将登陆凭证从缓存中删除
			srv.Redis.Del(ctx, config.SessionCacheKey(session))
			return
		}

		//登陆设备超过限制踢下线
		if operator == device.MaxDevices {
			message := DeviceMessage{Method: DeviceKickedMax}
			_ = manager.SendToDevice(userID, deviceID, message.Json())
			//将登陆凭证从缓存中删除
			srv.Redis.Del(ctx, config.SessionCacheKey(session))
			return
		}

	}

	manager.OnMessage = func(userID int64, deviceID, session string, message string) {
		logger.Infow("[DeviceManager] message received",
			logger.Field("user_id", userID),
			logger.Field("device_id", deviceID),
			logger.Field("session", session),
			logger.Field("message_size", len(message)),
		)
	}
	return manager
}

type DeviceMessage struct {
	Method DeviceMessageMethod `json:"method"`
}

func (dm *DeviceMessage) Json() string {
	jsonData, _ := json.Marshal(dm)
	return string(jsonData)
}

type DeviceMessageMethod string

const (
	// DeviceKickedMax 设备数量超出限制
	DeviceKickedMax DeviceMessageMethod = "kicked_device"
	// DeviceKickedAdmin 管理员踢下线
	DeviceKickedAdmin DeviceMessageMethod = "kicked_admin"
	// SubscribeUpdate 订阅有更新
	SubscribeUpdate DeviceMessageMethod = "subscribe_update"
)
