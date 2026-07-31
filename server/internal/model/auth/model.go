package auth

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type customAuthLogicModel interface {
	GetAuthListByPage(ctx context.Context) ([]*Auth, error)
	FindAllSummaries(ctx context.Context) ([]*Auth, error)
	FindAllForGlobalConfig(ctx context.Context) ([]*Auth, error)
	FindOneByMethod(ctx context.Context, platform string) (*Auth, error)
	FindOneByMethodForAdmin(ctx context.Context, method string, includeConfig bool) (*Auth, error)
	FindAll(ctx context.Context) ([]*Auth, error)
	UpdateFields(ctx context.Context, current *Auth, update AuthFieldsUpdate, tx ...*gorm.DB) error
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customAuthModel{
		defaultAuthModel: newAuthModel(conn, c),
	}
}

type Filter struct {
	Show   *bool
	Pinned *bool
	Popup  *bool
	Search string
}

// GetAuthListByPage  get auth list by page
func (m *customAuthModel) GetAuthListByPage(ctx context.Context) ([]*Auth, error) {
	list := make([]*Auth, 0)
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		return activeAuthMethods(conn).Select(authSummaryColumns).Order("id ASC").Find(v).Error
	})
	return list, err
}

func (m *customAuthModel) FindAllSummaries(ctx context.Context) ([]*Auth, error) {
	list := make([]*Auth, 0)
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		return activeAuthMethods(conn).Select(authSummaryColumns).Order("id ASC").Find(v).Error
	})
	return list, err
}

func (m *customAuthModel) FindAllForGlobalConfig(ctx context.Context) ([]*Auth, error) {
	list := make([]*Auth, 0)
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		return activeAuthMethods(conn).Select(authSummaryColumns).Order("id ASC").Find(v).Error
	})
	return list, err
}

// FindOneByMethod  find one by method
func (m *customAuthModel) FindOneByMethod(ctx context.Context, method string) (*Auth, error) {
	key := cachekeys.String(cacheAuthMethodPrefix, method)
	var data Auth
	err := m.QueryCtx(ctx, &data, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Auth{}).
			Select(authRuntimeColumns).
			Where("method = ?", method).
			First(v).Error
	})

	return &data, err
}

func (m *customAuthModel) FindOneByMethodForAdmin(ctx context.Context, method string, includeConfig bool) (*Auth, error) {
	var data Auth
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Auth{}).
			Select(authAdminSelectColumns(includeConfig)).
			Where("method = ?", method).
			First(v).Error
	})
	return &data, err
}

// FindAll find all
func (m *customAuthModel) FindAll(ctx context.Context) ([]*Auth, error) {
	list := make([]*Auth, 0)
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		return activeAuthMethods(conn).Select(authSummaryColumns).Order("id ASC").Find(v).Error
	})
	return list, err
}

type AuthFieldsUpdate struct {
	Enabled *bool
	Config  *string
}

func (u AuthFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 2)
	if u.Enabled != nil {
		updates["enabled"] = *u.Enabled
	}
	if u.Config != nil {
		updates["config"] = *u.Config
	}
	return updates
}

func (m *customAuthModel) UpdateFields(ctx context.Context, current *Auth, update AuthFieldsUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Auth{}).Where("id = ?", current.Id).Updates(updates).Error
	}, m.getCacheKeys(current)...)
}

var authSummaryColumns = []string{
	"id",
	"method",
	"enabled",
}

var authRuntimeColumns = []string{
	"id",
	"method",
	"config",
	"enabled",
}

func activeAuthMethods(conn *gorm.DB) *gorm.DB {
	return conn.Model(&Auth{}).Where("method NOT IN ?", []string{"mobile", "device"})
}

var authCacheColumns = []string{
	"id",
	"method",
}

func authAdminSelectColumns(includeConfig bool) []string {
	if !includeConfig {
		return authSummaryColumns
	}
	columns := make([]string, 0, len(authSummaryColumns)+1)
	columns = append(columns, authSummaryColumns...)
	return append(columns, "config")
}
