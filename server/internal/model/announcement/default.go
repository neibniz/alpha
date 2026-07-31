package announcement

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customAnnouncementModel)(nil)
var (
	cacheAnnouncementIdPrefix = "cache:announcement:id:"
)

type (
	Model interface {
		announcementModel
		customAnnouncementLogicModel
	}
	announcementModel interface {
		Insert(ctx context.Context, data *Announcement) error
		FindOne(ctx context.Context, id int64) (*Announcement, error)
		Update(ctx context.Context, data *Announcement) error
		Delete(ctx context.Context, id int64) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customAnnouncementModel struct {
		*defaultAnnouncementModel
	}
	defaultAnnouncementModel struct {
		cache.CachedConn
		table string
	}
)

func newAnnouncementModel(db *gorm.DB, c *redis.Client) *defaultAnnouncementModel {
	return &defaultAnnouncementModel{
		CachedConn: cache.NewConn(db, c),
		table:      "announcement",
	}
}

//nolint:unused
func (m *defaultAnnouncementModel) batchGetCacheKeys(Announcements ...*Announcement) []string {
	keys := make([]string, 0, len(Announcements))
	for _, announcement := range Announcements {
		keys = append(keys, m.getCacheKeys(announcement)...)
	}
	return keys
}
func (m *defaultAnnouncementModel) getCacheKeys(data *Announcement) []string {
	if data == nil {
		return []string{}
	}
	cacheKeys := []string{
		cachekeys.Int(cacheAnnouncementIdPrefix, data.Id),
	}
	return cacheKeys
}

func (m *defaultAnnouncementModel) Insert(ctx context.Context, data *Announcement) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultAnnouncementModel) FindOne(ctx context.Context, id int64) (*Announcement, error) {
	announcementIdKey := cachekeys.Int(cacheAnnouncementIdPrefix, id)
	var resp Announcement
	err := m.QueryCtx(ctx, &resp, announcementIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Announcement{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultAnnouncementModel) Update(ctx context.Context, data *Announcement) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Save(data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultAnnouncementModel) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return nil
	}
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Delete(&Announcement{}, id).Error
	}, m.getCacheKeys(&Announcement{Id: id})...)
	return err
}

func (m *defaultAnnouncementModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
