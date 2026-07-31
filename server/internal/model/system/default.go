package system

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	cacheSystemIdPrefix  = "cache:System:id:"
	cacheSystemKeyPrefix = "cache:System:key:"
)
var _ Model = (*customSystemModel)(nil)

type (
	Model interface {
		systemModel
		customSystemLogicModel
	}
	systemModel interface {
		Insert(ctx context.Context, data *System) error
		FindOne(ctx context.Context, id int64) (*System, error)
		FindOneByKey(ctx context.Context, email string) (*System, error)
		Update(ctx context.Context, data *System) error
		Delete(ctx context.Context, id int64) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customSystemModel struct {
		*defaultSystemModel
	}
	defaultSystemModel struct {
		cache.CachedConn
		table string
	}
)

func newSystemModel(db *gorm.DB, c *redis.Client) *defaultSystemModel {
	return &defaultSystemModel{
		CachedConn: cache.NewConn(db, c),
		table:      "System",
	}
}

func (m *defaultSystemModel) getCacheKeys(data *System) []string {
	if data == nil {
		return []string{}
	}
	cacheKeys := make([]string, 0, 2)
	cacheKeys = append(cacheKeys, cachekeys.Int(cacheSystemIdPrefix, data.Id))
	if data.Key != "" {
		cacheKeys = append(cacheKeys, cachekeys.String(cacheSystemKeyPrefix, data.Key))
	}
	return cacheKeys
}

func (m *defaultSystemModel) FindOneByKey(ctx context.Context, key string) (*System, error) {
	system := new(System)
	cacheKey := cachekeys.String(cacheSystemKeyPrefix, key)
	err := m.QueryCtx(ctx, system, cacheKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&System{}).
			Select(systemKeyLookupColumns).
			Where("key = ?", key).
			First(v).Error
	})
	return system, err
}

func (m *defaultSystemModel) Insert(ctx context.Context, data *System) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultSystemModel) FindOne(ctx context.Context, id int64) (*System, error) {
	systemIdKey := cachekeys.Int(cacheSystemIdPrefix, id)
	var resp System
	err := m.QueryCtx(ctx, &resp, systemIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&System{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultSystemModel) findOneForCache(ctx context.Context, id int64) (*System, error) {
	var resp System
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&System{}).
			Select(systemCacheColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, err
}

func (m *defaultSystemModel) Update(ctx context.Context, data *System) error {
	old, err := m.findOneForCache(ctx, data.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	cacheKeys := cachekeys.Merge(m.getCacheKeys(old), m.getCacheKeys(data))
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Save(data).Error
	}, cacheKeys...)
	return err
}

func (m *defaultSystemModel) Delete(ctx context.Context, id int64) error {
	data, err := m.findOneForCache(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Delete(&System{}, id).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultSystemModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
