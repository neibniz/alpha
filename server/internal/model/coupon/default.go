package coupon

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customCouponModel)(nil)
var (
	cacheCouponIdPrefix   = "cache:coupon:id:"
	cacheCouponCodePrefix = "cache:coupon:code:"
)

type (
	Model interface {
		couponModel
		customCouponLogicModel
	}
	couponModel interface {
		Insert(ctx context.Context, data *Coupon) error
		FindOne(ctx context.Context, id int64) (*Coupon, error)
		FindOneByCode(ctx context.Context, code string) (*Coupon, error)
		Update(ctx context.Context, data *Coupon) error
		Delete(ctx context.Context, id int64) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customCouponModel struct {
		*defaultCouponModel
	}
	defaultCouponModel struct {
		cache.CachedConn
		table string
	}
)

func newCouponModel(db *gorm.DB, c *redis.Client) *defaultCouponModel {
	return &defaultCouponModel{
		CachedConn: cache.NewConn(db, c),
		table:      "coupon",
	}
}

//nolint:unused
func (m *defaultCouponModel) batchGetCacheKeys(Coupons ...*Coupon) []string {
	keys := newCouponCacheKeyCollector(len(Coupons) * 2)
	for _, coupon := range Coupons {
		for _, key := range m.getCacheKeys(coupon) {
			keys.Add(key)
		}
	}
	return keys.Values()
}
func (m *defaultCouponModel) getCacheKeys(data *Coupon) []string {
	if data == nil {
		return []string{}
	}
	cacheKeys := []string{
		cachekeys.Int(cacheCouponIdPrefix, data.Id),
		cachekeys.String(cacheCouponCodePrefix, data.Code),
	}
	return cacheKeys
}

func (m *defaultCouponModel) Insert(ctx context.Context, data *Coupon) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultCouponModel) FindOne(ctx context.Context, id int64) (*Coupon, error) {
	couponIdKey := cachekeys.Int(cacheCouponIdPrefix, id)
	var resp Coupon
	err := m.QueryCtx(ctx, &resp, couponIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Coupon{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultCouponModel) FindOneByCode(ctx context.Context, code string) (*Coupon, error) {
	couponCodeKey := cachekeys.String(cacheCouponCodePrefix, code)
	var resp Coupon
	err := m.QueryCtx(ctx, &resp, couponCodeKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Coupon{}).Where("code = ?", code).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultCouponModel) findOneForCache(ctx context.Context, id int64) (*Coupon, error) {
	var resp Coupon
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Coupon{}).
			Select(couponCacheColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, err
}

func (m *defaultCouponModel) Update(ctx context.Context, data *Coupon) error {
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

func (m *defaultCouponModel) Delete(ctx context.Context, id int64) error {
	data, err := m.findOneForCache(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Delete(&Coupon{}, id).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultCouponModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
