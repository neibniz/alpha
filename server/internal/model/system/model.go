package system

import (
	"context"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type customSystemLogicModel interface {
	GetSiteConfig(ctx context.Context) ([]*System, error)
	GetSubscribeConfig(ctx context.Context) ([]*System, error)
	GetRegisterConfig(ctx context.Context) ([]*System, error)
	GetVerifyConfig(ctx context.Context) ([]*System, error)
	GetNodeConfig(ctx context.Context) ([]*System, error)
	GetInviteConfig(ctx context.Context) ([]*System, error)
	GetTosConfig(ctx context.Context) ([]*System, error)
	GetCurrencyConfig(ctx context.Context) ([]*System, error)
	GetVerifyCodeConfig(ctx context.Context) ([]*System, error)
	GetLogConfig(ctx context.Context) ([]*System, error)
	UpdateNodeMultiplierConfig(ctx context.Context, config string) error
	FindNodeMultiplierConfig(ctx context.Context) (*System, error)
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customSystemModel{
		defaultSystemModel: newSystemModel(conn, c),
	}
}

var systemConfigColumns = []string{
	"key",
	"value",
	"type",
}

var systemKeyLookupColumns = []string{
	"id",
	"key",
	"value",
}

var systemCacheColumns = []string{
	"id",
	"key",
}

var systemValueColumns = []string{"value"}

func (m *customSystemModel) getConfigByCategory(ctx context.Context, cacheKey, category string) ([]*System, error) {
	var configs []*System
	err := m.QueryCtx(ctx, &configs, cacheKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&System{}).
			Select(systemConfigColumns).
			Where("category = ?", category).
			Find(v).Error
	})
	return configs, err
}

func (m *customSystemModel) getConfigByCategoryNoCache(ctx context.Context, category string) ([]*System, error) {
	var configs []*System
	err := m.QueryNoCacheCtx(ctx, &configs, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&System{}).
			Select(systemConfigColumns).
			Where("category = ?", category).
			Find(v).Error
	})
	return configs, err
}

// GetSiteConfig returns the site config.
func (m *customSystemModel) GetSiteConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.SiteConfigKey, "site")
}

// GetEmailConfig returns the email config.
func (m *customSystemModel) GetEmailConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.EmailSmtpConfigKey, "email")
}

// GetSubscribeConfig returns the subscribe config.
func (m *customSystemModel) GetSubscribeConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.SubscribeConfigKey, "subscribe")
}

// GetRegisterConfig returns the register config.
func (m *customSystemModel) GetRegisterConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.RegisterConfigKey, "register")
}

// GetVerifyConfig returns the verify config.
func (m *customSystemModel) GetVerifyConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.VerifyConfigKey, "verify")
}

// GetNodeConfig returns the server config.
func (m *customSystemModel) GetNodeConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.NodeConfigKey, "server")
}

// GetInviteConfig returns the invite config.
func (m *customSystemModel) GetInviteConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.InviteConfigKey, "invite")
}

// GetTelegramConfig returns the telegram config.
func (m *customSystemModel) GetTelegramConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.TelegramConfigKey, "telegram")
}

// GetTosConfig returns the tos config.
func (m *customSystemModel) GetTosConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.TosConfigKey, "tos")
}

// GetCurrencyConfig returns the currency config.
func (m *customSystemModel) GetCurrencyConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.CurrencyConfigKey, "currency")
}

func (m *customSystemModel) UpdateNodeMultiplierConfig(ctx context.Context, config string) error {
	return m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		return conn.Model(&System{}).Where("category = ? AND key = ?", "server", "NodeMultiplierConfig").Update("value", config).Error
	})
}

func (m *customSystemModel) FindNodeMultiplierConfig(ctx context.Context) (*System, error) {
	var data System
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&System{}).
			Select(systemValueColumns).
			Where("category = ? AND key = ?", "server", "NodeMultiplierConfig").
			Find(v).Error
	})
	return &data, err
}

// GetVerifyCodeConfig returns the verify code config.

func (m *customSystemModel) GetVerifyCodeConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategory(ctx, config.VerifyCodeConfigKey, "verify_code")
}

// GetLogConfig returns the log config.
func (m *customSystemModel) GetLogConfig(ctx context.Context) ([]*System, error) {
	return m.getConfigByCategoryNoCache(ctx, "log")
}
