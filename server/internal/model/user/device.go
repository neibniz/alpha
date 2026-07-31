package user

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/internal/privacy"

	"gorm.io/gorm"
)

func (m *customUserModel) FindOneDevice(ctx context.Context, id int64) (*Device, error) {
	var resp Device
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Device{}).Where("id = ?", id).First(&resp).Error
	})
	if err == nil {
		err = m.openDevice(&resp)
	}
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *customUserModel) FindOneDeviceForStatusUpdate(ctx context.Context, id int64) (*Device, error) {
	var resp Device
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Device{}).
			Select(m.deviceStatusColumns(userDeviceStatusUpdateColumns)).
			Where("id = ?", id).
			First(v).Error
	})
	if err == nil {
		err = m.openDevice(&resp)
	}
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *customUserModel) FindOneDeviceByIdentifierForStatusUpdate(ctx context.Context, id string) (*Device, error) {
	var resp Device
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return m.deviceIdentifierLookup(conn.Model(&Device{}).
			Select(m.deviceStatusColumns(userDeviceStatusUpdateColumns)).
			Where("1 = 1"), id).
			First(v).Error
	})
	if err == nil {
		err = m.openDevice(&resp)
	}
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *customUserModel) findOneDeviceForCache(ctx context.Context, id int64) (*Device, error) {
	var resp Device
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Device{}).
			Select(m.deviceCacheColumns(userDeviceCacheColumns)).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *customUserModel) FindOneDeviceByIdentifier(ctx context.Context, id string) (*Device, error) {
	deviceIdKey := cachekeys.String(cacheUserDeviceNumberPrefix, m.deviceCacheToken(id))
	var resp Device
	err := m.QueryCtx(ctx, &resp, deviceIdKey, func(conn *gorm.DB, v interface{}) error {
		return m.deviceIdentifierLookup(conn.Model(&Device{}), id).First(&resp).Error
	})
	if err == nil {
		err = m.openDevice(&resp)
	}
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

var userDeviceListColumns = []string{
	"id",
	"ip",
	"identifier",
	"user_agent",
	"online",
	"enabled",
	"created_at",
	"updated_at",
}

var userDeviceCacheColumns = []string{
	"id",
	"identifier",
}

var userDeviceStatusUpdateColumns = []string{
	"id",
	"user_id",
	"identifier",
	"online",
	"enabled",
}

// QueryDevicePageList  returns a list of records that meet the conditions.
func (m *customUserModel) QueryDevicePageList(ctx context.Context, userId, _ int64, page, size int) ([]*Device, int64, error) {
	list := make([]*Device, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		// Device sessions are user-scoped; subscribe_id is not a reliable device boundary.
		countQuery := conn.Model(&Device{}).Where("user_id = ?", userId)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		list = make([]*Device, 0, capacity)
		return conn.Model(&Device{}).
			Select(m.deviceListColumns(userDeviceListColumns)).
			Where("user_id = ?", userId).
			Order("id DESC").
			Limit(size).
			Offset(offset).
			Find(&list).Error
	})
	if err == nil {
		for _, item := range list {
			if err = m.openDevice(item); err != nil {
				return nil, 0, err
			}
		}
	}
	return list, total, err
}

// QueryDeviceList  returns a list of records that meet the conditions.
func (m *customUserModel) QueryDeviceList(ctx context.Context, userId int64, page, size int) ([]*Device, int64, error) {
	list := make([]*Device, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.NormalizeWithLimit(page, size, pagination.MaxPageSize, pagination.MaxPageSize)
		if err := conn.Model(&Device{}).Where("user_id = ?", userId).Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		list = make([]*Device, 0, capacity)
		return conn.Model(&Device{}).
			Select(m.deviceListColumns(userDeviceListColumns)).
			Where("user_id = ?", userId).
			Order("id DESC").
			Limit(size).
			Offset(offset).
			Find(&list).Error
	})
	if err == nil {
		for _, item := range list {
			if err = m.openDevice(item); err != nil {
				return nil, 0, err
			}
		}
	}
	return list, total, err
}

func (m *customUserModel) UpdateDevice(ctx context.Context, data *Device, tx ...*gorm.DB) error {
	old, err := m.findOneDeviceForCache(ctx, data.Id)
	if err != nil {
		return err
	}
	protected, err := m.protectDevice(data)
	if err != nil {
		return err
	}
	cacheKeys := cacheKeysForModels(old, data)
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Save(protected).Error
	}, cacheKeys...)
	return err
}

func (m *customUserModel) UpdateDeviceFields(ctx context.Context, current *Device, update DeviceFieldsUpdate, tx ...*gorm.DB) error {
	if current == nil || current.Id == 0 {
		return nil
	}
	updates := update.updates()
	if m.privacyCodec.Enabled() {
		targetUserID := current.UserId
		if update.UserID != nil {
			targetUserID = *update.UserID
			sealed, err := m.privacyCodec.Seal(privacy.DeviceScope(targetUserID, "identifier"), current.Identifier)
			if err != nil {
				return err
			}
			updates["identifier"] = sealed
			updates["identifier_bidx"] = m.privacyCodec.BlindIndex("device", current.Identifier)
			updates["identifier_display"] = privacy.MaskIdentifier(current.Identifier)
			if update.IP == nil {
				sealedIP, err := m.privacyCodec.Seal(privacy.DeviceScope(targetUserID, "ip"), current.Ip)
				if err != nil {
					return err
				}
				updates["ip"] = sealedIP
			}
			if update.UserAgent == nil {
				sealedUserAgent, err := m.privacyCodec.Seal(privacy.DeviceScope(targetUserID, "user_agent"), current.UserAgent)
				if err != nil {
					return err
				}
				updates["user_agent"] = sealedUserAgent
			}
		}
		if update.IP != nil {
			sealed, err := m.privacyCodec.Seal(privacy.DeviceScope(targetUserID, "ip"), *update.IP)
			if err != nil {
				return err
			}
			updates["ip"] = sealed
			updates["ip_display"] = privacy.MaskIP(*update.IP)
		}
		if update.UserAgent != nil {
			sealed, err := m.privacyCodec.Seal(privacy.DeviceScope(targetUserID, "user_agent"), *update.UserAgent)
			if err != nil {
				return err
			}
			updates["user_agent"] = sealed
			updates["user_agent_display"] = privacy.MaskUserAgent(*update.UserAgent)
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Device{}).
			Where("id = ?", current.Id).
			Updates(updates).Error
	}, current.GetCacheKeys()...)
}

func (m *customUserModel) DeleteDevice(ctx context.Context, id int64, tx ...*gorm.DB) error {
	data, err := m.findOneDeviceForCache(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Delete(&Device{}, id).Error
	}, data.GetCacheKeys()...)
	return err
}

func (m *customUserModel) DeleteDeviceByModel(ctx context.Context, data *Device, tx ...*gorm.DB) error {
	if data == nil || data.Id == 0 {
		return nil
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Delete(&Device{}, data.Id).Error
	}, data.GetCacheKeys()...)
}

func (m *customUserModel) InsertDevice(ctx context.Context, data *Device, tx ...*gorm.DB) error {
	protected, err := m.protectDevice(data)
	if err != nil {
		return err
	}
	defer func() {
		if clearErr := m.ClearDeviceCache(ctx, data); clearErr != nil {
			// log cache clear error
		}
	}()

	err = m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Create(protected).Error
	})
	if err == nil && protected != data {
		data.Id = protected.Id
		data.CreatedAt = protected.CreatedAt
		data.UpdatedAt = protected.UpdatedAt
	}
	return err
}
