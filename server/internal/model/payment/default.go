package payment

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customPaymentModel)(nil)
var (
	cachePaymentIdPrefix           = "cache:payment:id:"
	cachePaymentTokenPrefix        = "cache:payment:token:"
	cachePaymentPublicOrderPrefix  = "cache:payment:public-order:"
	cachePaymentOrderRecordPrefix  = "cache:payment:order-record:"
	cachePaymentCheckoutPrefix     = "cache:payment:checkout:"
	cachePaymentGatewayCheckPrefix = "cache:payment:gateway-check:"
)

type (
	Model interface {
		paymentModel
		customPaymentLogicModel
	}
	paymentModel interface {
		Insert(ctx context.Context, data *Payment, tx ...*gorm.DB) error
		FindOne(ctx context.Context, id int64) (*Payment, error)
		Update(ctx context.Context, data *Payment, tx ...*gorm.DB) error
		Delete(ctx context.Context, id int64, tx ...*gorm.DB) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customPaymentModel struct {
		*defaultPaymentModel
	}
	defaultPaymentModel struct {
		cache.CachedConn
		table string
	}
)

func newPaymentModel(db *gorm.DB, c *redis.Client) *defaultPaymentModel {
	return &defaultPaymentModel{
		CachedConn: cache.NewConn(db, c),
		table:      "Payment",
	}
}

//nolint:unused
func (m *defaultPaymentModel) batchGetCacheKeys(Payments ...*Payment) []string {
	keys := make([]string, 0, len(Payments)*6)
	for _, payment := range Payments {
		keys = append(keys, m.getCacheKeys(payment)...)
	}
	return keys
}
func (m *defaultPaymentModel) getCacheKeys(data *Payment) []string {
	if data == nil {
		return []string{}
	}
	paymentIdKey := cachekeys.Int(cachePaymentIdPrefix, data.Id)
	paymentNameKey := cachekeys.String(cachePaymentTokenPrefix, data.Token)
	cacheKeys := []string{
		paymentIdKey,
		cachekeys.Int(cachePaymentPublicOrderPrefix, data.Id),
		cachekeys.Int(cachePaymentOrderRecordPrefix, data.Id),
		cachekeys.Int(cachePaymentCheckoutPrefix, data.Id),
		cachekeys.Int(cachePaymentGatewayCheckPrefix, data.Id),
		paymentNameKey,
	}
	return cacheKeys
}

func (m *defaultPaymentModel) Insert(ctx context.Context, data *Payment, tx ...*gorm.DB) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultPaymentModel) FindOne(ctx context.Context, id int64) (*Payment, error) {
	PaymentIdKey := cachekeys.Int(cachePaymentIdPrefix, id)
	var resp Payment
	err := m.QueryCtx(ctx, &resp, PaymentIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultPaymentModel) findOneForCache(ctx context.Context, id int64) (*Payment, error) {
	var resp Payment
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentCacheColumns).
			Where("id = ?", id).
			First(v).Error
	})
	if err != nil {
		return nil, err
	}
	return &resp, err
}

func (m *defaultPaymentModel) Update(ctx context.Context, data *Payment, tx ...*gorm.DB) error {
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

func (m *defaultPaymentModel) Delete(ctx context.Context, id int64, tx ...*gorm.DB) error {
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
		return conn.Delete(&Payment{}, id).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultPaymentModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
