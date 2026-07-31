package subscribe

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customSubscribeModel)(nil)
var (
	cacheSubscribeIdPrefix            = "cache:subscribe:id:"
	cacheSubscribeRuntimePrefix       = "cache:subscribe:runtime:"
	cacheSubscribeNodeSelectorPrefix  = "cache:subscribe:node-selector:"
	cacheSubscribeOrderSnapshotPrefix = "cache:subscribe:order-snapshot:"
)

type (
	Model interface {
		subscribeModel
		customSubscribeLogicModel
	}
	subscribeModel interface {
		Insert(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error
		FindOne(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForRuntime(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForNodeSelection(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForOrderActivation(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForPublicPurchase(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForPortalPrePurchase(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForOrderSnapshot(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForSubscriptionExpiry(ctx context.Context, id int64) (*Subscribe, error)
		FindOneForUpdateDefaults(ctx context.Context, id int64) (*Subscribe, error)
		Update(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error
		Delete(ctx context.Context, id int64, tx ...*gorm.DB) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customSubscribeModel struct {
		*defaultSubscribeModel
	}
	defaultSubscribeModel struct {
		cache.CachedConn
		table string
	}
)

func newSubscribeModel(db *gorm.DB, c *redis.Client) *defaultSubscribeModel {
	return &defaultSubscribeModel{
		CachedConn: cache.NewConn(db, c),
		table:      "subscribe",
	}
}

//nolint:unused
func (m *defaultSubscribeModel) batchGetCacheKeys(Subscribes ...*Subscribe) []string {
	keys := newSubscribeCacheKeyCollector(len(Subscribes) * subscribeBaseCacheKeyCount)
	for _, subscribe := range Subscribes {
		for _, key := range m.getCacheKeys(subscribe) {
			keys.Add(key)
		}
	}
	return keys.Values()
}

func (m *defaultSubscribeModel) getCacheKeys(data *Subscribe) []string {
	if data == nil {
		return []string{}
	}
	keys := newSubscribeCacheKeyCollector(subscribeBaseCacheKeyCount)
	selectors := newSubscribeNodeSelectorSetWithHints(
		subscribeCSVItemCapHint(data.Nodes),
		subscribeCSVItemCapHint(data.NodeTags),
	)
	selectors.AddRaw(data.Nodes, data.NodeTags)
	if selectors.Len() > 0 {
		rows := make([]subscribeCacheNodeServerRow, 0)
		err := m.QueryNoCacheCtx(context.Background(), &rows, func(conn *gorm.DB, v interface{}) error {
			return querySubscribeCacheNodeServerRows(conn, v, selectors.NodeIDs(), selectors.Tags())
		})
		if err == nil {
			addSubscribeServerUserCacheKeys(keys.AddKey, rows)
		}
	}

	addSubscribeBaseCacheKeys(keys.AddKey, data.Id)
	return keys.Values()
}

func (m *defaultSubscribeModel) Insert(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultSubscribeModel) FindOne(ctx context.Context, id int64) (*Subscribe, error) {
	SubscribeIdKey := cachekeys.Int(cacheSubscribeIdPrefix, id)
	var resp Subscribe
	err := m.QueryCtx(ctx, &resp, SubscribeIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultSubscribeModel) findOneForCache(ctx context.Context, id int64) (*Subscribe, error) {
	var resp Subscribe
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeCacheColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForRuntime(ctx context.Context, id int64) (*Subscribe, error) {
	subscribeRuntimeKey := cachekeys.Int(cacheSubscribeRuntimePrefix, id)
	var resp Subscribe
	err := m.QueryCtx(ctx, &resp, subscribeRuntimeKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeRuntimeColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForNodeSelection(ctx context.Context, id int64) (*Subscribe, error) {
	subscribeNodeSelectorKey := cachekeys.Int(cacheSubscribeNodeSelectorPrefix, id)
	var resp Subscribe
	err := m.QueryCtx(ctx, &resp, subscribeNodeSelectorKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeNodeSelectorColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForOrderActivation(ctx context.Context, id int64) (*Subscribe, error) {
	var resp Subscribe
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeOrderActivationColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForPublicPurchase(ctx context.Context, id int64) (*Subscribe, error) {
	var resp Subscribe
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribePublicPurchaseColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForPortalPrePurchase(ctx context.Context, id int64) (*Subscribe, error) {
	var resp Subscribe
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribePortalPrePurchaseColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForOrderSnapshot(ctx context.Context, id int64) (*Subscribe, error) {
	subscribeOrderSnapshotKey := cachekeys.Int(cacheSubscribeOrderSnapshotPrefix, id)
	var resp Subscribe
	err := m.QueryCtx(ctx, &resp, subscribeOrderSnapshotKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeOrderSnapshotColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForSubscriptionExpiry(ctx context.Context, id int64) (*Subscribe, error) {
	var resp Subscribe
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeSubscriptionExpiryColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) FindOneForUpdateDefaults(ctx context.Context, id int64) (*Subscribe, error) {
	var resp Subscribe
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeUpdateDefaultsColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *defaultSubscribeModel) Update(ctx context.Context, data *Subscribe, tx ...*gorm.DB) error {
	old, err := m.findOneForCache(ctx, data.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	cacheKeys := newSubscribeCacheKeyCollector(subscribeBaseCacheKeyCount * 2)
	for _, key := range m.getCacheKeys(old) {
		cacheKeys.Add(key)
	}
	for _, key := range m.getCacheKeys(data) {
		cacheKeys.Add(key)
	}
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		if len(tx) > 0 {
			db = tx[0]
		}
		return db.Save(data).Error
	}, cacheKeys.Values()...)
	return err
}

func (m *defaultSubscribeModel) Delete(ctx context.Context, id int64, tx ...*gorm.DB) error {
	data, err := m.findOneForCache(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		if len(tx) > 0 {
			db = tx[0]
		}
		return db.Delete(data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultSubscribeModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
