package document

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customDocumentModel)(nil)
var (
	cacheDocumentIdPrefix = "cache:document:id:"
)

type (
	Model interface {
		documentModel
		customDocumentLogicModel
	}
	documentModel interface {
		Insert(ctx context.Context, data *Document) error
		FindOne(ctx context.Context, id int64) (*Document, error)
		Update(ctx context.Context, data *Document) error
		Delete(ctx context.Context, id int64) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customDocumentModel struct {
		*defaultDocumentModel
	}
	defaultDocumentModel struct {
		cache.CachedConn
		table string
	}
)

func newDocumentModel(db *gorm.DB, c *redis.Client) *defaultDocumentModel {
	return &defaultDocumentModel{
		CachedConn: cache.NewConn(db, c),
		table:      "document",
	}
}

//nolint:unused
func (m *defaultDocumentModel) batchGetCacheKeys(Documents ...*Document) []string {
	keys := newDocumentCacheKeyCollector(len(Documents))
	for _, document := range Documents {
		for _, key := range m.getCacheKeys(document) {
			keys.Add(key)
		}
	}
	return keys.Values()
}
func (m *defaultDocumentModel) getCacheKeys(data *Document) []string {
	if data == nil {
		return []string{}
	}
	cacheKeys := []string{
		cachekeys.Int(cacheDocumentIdPrefix, data.Id),
	}
	return cacheKeys
}

func (m *defaultDocumentModel) Insert(ctx context.Context, data *Document) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultDocumentModel) FindOne(ctx context.Context, id int64) (*Document, error) {
	documentIdKey := cachekeys.Int(cacheDocumentIdPrefix, id)
	var resp Document
	err := m.QueryCtx(ctx, &resp, documentIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Document{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultDocumentModel) Update(ctx context.Context, data *Document) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Save(data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultDocumentModel) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return nil
	}
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Delete(&Document{}, id).Error
	}, m.getCacheKeys(&Document{Id: id})...)
	return err
}

func (m *defaultDocumentModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
