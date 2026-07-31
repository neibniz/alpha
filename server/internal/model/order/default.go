package order

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customOrderModel)(nil)
var (
	cacheOrderIdPrefix = "cache:order:id:"
	cacheOrderNoPrefix = "cache:order:no:"
)

type (
	Model interface {
		orderModel
		customOrderLogicModel
	}
	orderModel interface {
		Insert(ctx context.Context, data *Order, tx ...*gorm.DB) error
		FindOne(ctx context.Context, id int64) (*Order, error)
		FindOneByOrderNo(ctx context.Context, orderNo string) (*Order, error)
		Update(ctx context.Context, data *Order, tx ...*gorm.DB) error
		Delete(ctx context.Context, id int64, tx ...*gorm.DB) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customOrderModel struct {
		*defaultOrderModel
	}
	defaultOrderModel struct {
		cache.CachedConn
		table string
	}
)

func newOrderModel(db *gorm.DB, c *redis.Client) *defaultOrderModel {
	return &defaultOrderModel{
		CachedConn: cache.NewConn(db, c),
		table:      "order",
	}
}

//nolint:unused
func (m *defaultOrderModel) batchGetCacheKeys(Orders ...*Order) []string {
	keys := make([]string, 0, len(Orders)*2)
	for _, order := range Orders {
		keys = append(keys, m.getCacheKeys(order)...)
	}
	return keys
}
func (m *defaultOrderModel) getCacheKeys(data *Order) []string {
	if data == nil {
		return []string{}
	}
	orderIdKey := cachekeys.Int(cacheOrderIdPrefix, data.Id)
	orderNoKey := cachekeys.String(cacheOrderNoPrefix, data.OrderNo)
	cacheKeys := []string{
		orderIdKey,
		orderNoKey,
	}
	return cacheKeys
}

func (m *defaultOrderModel) Insert(ctx context.Context, data *Order, tx ...*gorm.DB) error {
	if data != nil {
		data.ClientIP = privacy.MaskIP(data.ClientIP)
	}
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultOrderModel) FindOne(ctx context.Context, id int64) (*Order, error) {
	OrderIdKey := cachekeys.Int(cacheOrderIdPrefix, id)
	var resp Order
	err := m.QueryCtx(ctx, &resp, OrderIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultOrderModel) FindOneByOrderNo(ctx context.Context, orderNo string) (*Order, error) {
	OrderNoKey := cachekeys.String(cacheOrderNoPrefix, orderNo)
	var resp Order
	err := m.QueryCtx(ctx, &resp, OrderNoKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).Where("order_no = ?", orderNo).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultOrderModel) findOneForCache(ctx context.Context, id int64) (*Order, error) {
	var resp Order
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Order{}).
			Select(orderCacheColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, err
}

func (m *defaultOrderModel) Update(ctx context.Context, data *Order, tx ...*gorm.DB) error {
	old, err := m.findOneForCache(ctx, data.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	cacheKeys := cachekeys.Merge(m.getCacheKeys(old), m.getCacheKeys(data))
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Save(data).Error
	}, cacheKeys...)
	return err
}

func (m *defaultOrderModel) Delete(ctx context.Context, id int64, tx ...*gorm.DB) error {
	data, err := m.findOneForCache(ctx, id)
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
		return conn.Delete(&Order{}, id).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultOrderModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
