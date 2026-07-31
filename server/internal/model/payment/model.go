package payment

import (
	"context"
	"fmt"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	paymentPlatform "github.com/neibniz/alpha/server/pkg/payment"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type customPaymentLogicModel interface {
	FindOneByPaymentToken(ctx context.Context, token string) (*Payment, error)
	FindOneForPublicOrder(ctx context.Context, id int64) (*Payment, error)
	FindOneForOrderRecord(ctx context.Context, id int64) (*Payment, error)
	FindOneForCheckout(ctx context.Context, id int64) (*Payment, error)
	FindOneForGatewayCheck(ctx context.Context, id int64) (*Payment, error)
	FindOneForAdmin(ctx context.Context, id int64, includeConfig bool) (*Payment, error)
	FindAll(ctx context.Context) ([]*Payment, error)
	FindListByPage(ctx context.Context, page, size int, req *Filter) (int64, []*Payment, error)
	FindAvailableMethods(ctx context.Context) ([]*Payment, error)
	UpdateFields(ctx context.Context, current *Payment, update PaymentFieldsUpdate, tx ...*gorm.DB) error
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customPaymentModel{
		defaultPaymentModel: newPaymentModel(conn, c),
	}
}

func (m *customPaymentModel) FindOneByPaymentToken(ctx context.Context, token string) (*Payment, error) {
	var resp Payment
	key := cachePaymentTokenPrefix + token
	err := m.QueryCtx(ctx, &resp, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentNotifyColumns).
			Where("token = ?", token).
			First(v).Error
	})
	return &resp, err
}

func (m *customPaymentModel) FindOneForPublicOrder(ctx context.Context, id int64) (*Payment, error) {
	var resp Payment
	key := cachePaymentPublicOrderPrefix + fmt.Sprint(id)
	err := m.QueryCtx(ctx, &resp, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentPublicOrderColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *customPaymentModel) FindOneForOrderRecord(ctx context.Context, id int64) (*Payment, error) {
	var resp Payment
	key := cachePaymentOrderRecordPrefix + fmt.Sprint(id)
	err := m.QueryCtx(ctx, &resp, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentOrderRecordColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *customPaymentModel) FindOneForCheckout(ctx context.Context, id int64) (*Payment, error) {
	var resp Payment
	key := cachePaymentCheckoutPrefix + fmt.Sprint(id)
	err := m.QueryCtx(ctx, &resp, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentCheckoutColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *customPaymentModel) FindOneForGatewayCheck(ctx context.Context, id int64) (*Payment, error) {
	var resp Payment
	key := cachePaymentGatewayCheckPrefix + fmt.Sprint(id)
	err := m.QueryCtx(ctx, &resp, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentGatewayCheckColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *customPaymentModel) FindOneForAdmin(ctx context.Context, id int64, includeConfig bool) (*Payment, error) {
	var resp Payment
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentAdminDetailSelectColumns(includeConfig)).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *customPaymentModel) FindAll(ctx context.Context) ([]*Payment, error) {
	resp := make([]*Payment, 0)
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).Select(paymentListColumns).Order("id ASC").Find(v).Error
	})
	return resp, err
}

func (m *customPaymentModel) FindAvailableMethods(ctx context.Context) ([]*Payment, error) {
	resp := make([]*Payment, 0)
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Payment{}).
			Select(paymentAvailableColumns).
			Where("enable = ?", true).
			Order("id ASC").
			Find(v).Error
	})
	return resp, err
}

func (m *customPaymentModel) FindListByPage(ctx context.Context, page, size int, req *Filter) (int64, []*Payment, error) {
	resp := make([]*Payment, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		countQuery := applyPaymentListFilter(conn.Model(&Payment{}), req)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		resp = make([]*Payment, 0, capacity)
		return applyPaymentListFilter(conn.Model(&Payment{}), req).
			Select(paymentListColumns).
			Order("id DESC").
			Offset(offset).
			Limit(size).
			Find(&resp).Error
	})
	return total, resp, err
}

var paymentCacheColumns = []string{
	"id",
	"token",
}

type PaymentFieldsUpdate struct {
	Name        *string
	Platform    *string
	Description *string
	Icon        *string
	Domain      *string
	Config      *string
	FeeMode     *uint
	FeePercent  *int64
	FeeAmount   *int64
	Enable      *bool
}

func (u PaymentFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 10)
	if u.Name != nil {
		updates["name"] = *u.Name
	}
	if u.Platform != nil {
		updates["platform"] = *u.Platform
	}
	if u.Description != nil {
		updates["description"] = *u.Description
	}
	if u.Icon != nil {
		updates["icon"] = *u.Icon
	}
	if u.Domain != nil {
		updates["domain"] = *u.Domain
	}
	if u.Config != nil {
		updates["config"] = *u.Config
	}
	if u.FeeMode != nil {
		updates["fee_mode"] = *u.FeeMode
	}
	if u.FeePercent != nil {
		updates["fee_percent"] = *u.FeePercent
	}
	if u.FeeAmount != nil {
		updates["fee_amount"] = *u.FeeAmount
	}
	if u.Enable != nil {
		updates["enable"] = *u.Enable
	}
	return updates
}

func (m *customPaymentModel) UpdateFields(ctx context.Context, current *Payment, update PaymentFieldsUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Payment{}).Where("id = ?", current.Id).Updates(updates).Error
	}, m.getCacheKeys(current)...)
}

func applyPaymentListFilter(conn *gorm.DB, req *Filter) *gorm.DB {
	if req == nil {
		return conn
	}
	if req.Enable != nil {
		conn = conn.Where("enable = ?", *req.Enable)
	}
	if req.Mark != "" {
		conn = conn.Where("platform IN ?", paymentPlatform.CompatiblePlatformNames(req.Mark))
	}
	if req.Search != "" {
		conn = conn.Where("name LIKE ?", "%"+req.Search+"%")
	}
	return conn
}

var paymentListColumns = []string{
	"id",
	"name",
	"platform",
	"icon",
	"domain",
	"description",
	"fee_mode",
	"fee_percent",
	"fee_amount",
	"enable",
	"token",
}

var paymentAvailableColumns = []string{
	"id",
	"name",
	"platform",
	"description",
	"icon",
	"fee_mode",
	"fee_percent",
	"fee_amount",
}

var paymentPublicOrderColumns = []string{
	"id",
	"name",
	"platform",
	"description",
	"icon",
	"fee_mode",
	"fee_percent",
	"fee_amount",
	"enable",
}

var paymentOrderRecordColumns = []string{
	"id",
	"platform",
	"token",
}

var paymentCheckoutColumns = []string{
	"id",
	"platform",
	"domain",
	"config",
	"enable",
	"token",
}

var paymentNotifyColumns = []string{
	"id",
	"platform",
	"domain",
	"config",
	"token",
}

var paymentGatewayCheckColumns = []string{
	"id",
	"platform",
	"config",
}

func paymentAdminDetailSelectColumns(includeConfig bool) []string {
	if !includeConfig {
		return paymentListColumns
	}
	columns := make([]string, 0, len(paymentListColumns)+1)
	columns = append(columns, paymentListColumns...)
	return append(columns, "config")
}
