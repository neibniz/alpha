package client

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"gorm.io/gorm"
)

type (
	Model interface {
		subscribeApplicationModel
	}
	subscribeApplicationModel interface {
		Insert(ctx context.Context, data *SubscribeApplication) error
		FindOne(ctx context.Context, id int64) (*SubscribeApplication, error)
		FindOneForUpdate(ctx context.Context, id int64) (*SubscribeApplication, error)
		FindOneForPreview(ctx context.Context, id int64) (*SubscribeApplication, error)
		Update(ctx context.Context, data *SubscribeApplication) error
		UpdateFields(ctx context.Context, id int64, update SubscribeApplicationFieldsUpdate) error
		Delete(ctx context.Context, id int64) error
		List(ctx context.Context, page, size int) (int64, []*SubscribeApplication, error)
		Filter(ctx context.Context, page, size int, filter SubscribeApplicationListFilter) (int64, []*SubscribeApplication, error)
		ListEnabled(ctx context.Context) ([]*SubscribeApplication, error)
		ListEnabledUserAgents(ctx context.Context) ([]string, error)
		ListEnabledMatchers(ctx context.Context) ([]*SubscribeApplication, error)
		ListEnabledRuntime(ctx context.Context) ([]*SubscribeApplication, error)
		ListEnabledSummary(ctx context.Context) ([]*SubscribeApplication, error)
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}
	DefaultSubscribeApplicationModel struct {
		*gorm.DB
	}
)

type SubscribeApplicationListFilter struct {
	Search  string
	Enabled *bool
}

func NewSubscribeApplicationModel(db *gorm.DB) Model {
	return &DefaultSubscribeApplicationModel{
		DB: db,
	}
}

func (m *DefaultSubscribeApplicationModel) Insert(ctx context.Context, data *SubscribeApplication) error {
	if err := m.WithContext(ctx).Model(&SubscribeApplication{}).Create(data).Error; err != nil {
		return err
	}
	return nil
}

func (m *DefaultSubscribeApplicationModel) FindOne(ctx context.Context, id int64) (*SubscribeApplication, error) {
	var resp SubscribeApplication
	if err := m.WithContext(ctx).Model(&SubscribeApplication{}).Where("id = ?", id).First(&resp).Error; err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *DefaultSubscribeApplicationModel) FindOneForUpdate(ctx context.Context, id int64) (*SubscribeApplication, error) {
	var resp SubscribeApplication
	if err := m.WithContext(ctx).
		Model(&SubscribeApplication{}).
		Select(subscribeApplicationUpdateReadColumns).
		Where("id = ?", id).
		First(&resp).Error; err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *DefaultSubscribeApplicationModel) FindOneForPreview(ctx context.Context, id int64) (*SubscribeApplication, error) {
	var resp SubscribeApplication
	if err := m.WithContext(ctx).
		Model(&SubscribeApplication{}).
		Select(subscribeApplicationPreviewColumns).
		Where("id = ?", id).
		First(&resp).Error; err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *DefaultSubscribeApplicationModel) Update(ctx context.Context, data *SubscribeApplication) error {
	result := m.WithContext(ctx).
		Model(&SubscribeApplication{}).
		Where("id = ?", data.Id).
		Select("*").
		Updates(data)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type SubscribeApplicationFieldsUpdate struct {
	Name              *string
	Icon              *string
	Description       *string
	Scheme            *string
	UserAgent         *string
	Enabled           *bool
	IsDefault         *bool
	SubscribeTemplate *string
	OutputFormat      *string
	DownloadLink      *string
}

func (u SubscribeApplicationFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 10)
	if u.Name != nil {
		updates["name"] = *u.Name
	}
	if u.Icon != nil {
		updates["icon"] = *u.Icon
	}
	if u.Description != nil {
		updates["description"] = *u.Description
	}
	if u.Scheme != nil {
		updates["scheme"] = *u.Scheme
	}
	if u.UserAgent != nil {
		updates["user_agent"] = *u.UserAgent
	}
	if u.Enabled != nil {
		updates["enabled"] = *u.Enabled
	}
	if u.IsDefault != nil {
		updates["is_default"] = *u.IsDefault
	}
	if u.SubscribeTemplate != nil {
		updates["subscribe_template"] = *u.SubscribeTemplate
	}
	if u.OutputFormat != nil {
		updates["output_format"] = *u.OutputFormat
	}
	if u.DownloadLink != nil {
		updates["download_link"] = *u.DownloadLink
	}
	return updates
}

func (m *DefaultSubscribeApplicationModel) UpdateFields(ctx context.Context, id int64, update SubscribeApplicationFieldsUpdate) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	result := m.WithContext(ctx).
		Model(&SubscribeApplication{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *DefaultSubscribeApplicationModel) Delete(ctx context.Context, id int64) error {
	if err := m.WithContext(ctx).Model(&SubscribeApplication{}).Where("id = ?", id).Delete(&SubscribeApplication{}).Error; err != nil {
		return err
	}
	return nil
}

func (m *DefaultSubscribeApplicationModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	tx := m.WithContext(ctx).Begin()
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			return rbErr
		}
		return err
	}
	return tx.Commit().Error
}

func (m *DefaultSubscribeApplicationModel) List(ctx context.Context, page, size int) (int64, []*SubscribeApplication, error) {
	return m.Filter(ctx, page, size, SubscribeApplicationListFilter{})
}

func (m *DefaultSubscribeApplicationModel) Filter(ctx context.Context, page, size int, filter SubscribeApplicationListFilter) (int64, []*SubscribeApplication, error) {
	page, size = pagination.Normalize(page, size)
	var total int64
	if err := applySubscribeApplicationListFilter(
		m.WithContext(ctx).Model(&SubscribeApplication{}),
		filter,
	).Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset, capacity, ok := pagination.PageWindow(total, page, size)
	if !ok {
		return total, []*SubscribeApplication{}, nil
	}

	resp := make([]*SubscribeApplication, 0, capacity)
	if err := applySubscribeApplicationListFilter(
		m.WithContext(ctx).Model(&SubscribeApplication{}),
		filter,
	).
		Select(subscribeApplicationAdminListColumns).
		Order("id ASC").
		Offset(offset).
		Limit(size).
		Find(&resp).Error; err != nil {
		return 0, nil, err
	}
	return total, resp, nil
}

func applySubscribeApplicationListFilter(query *gorm.DB, filter SubscribeApplicationListFilter) *gorm.DB {
	if search := strings.ToLower(strings.TrimSpace(filter.Search)); search != "" {
		pattern := "%" + search + "%"
		query = query.Where(
			"LOWER(name) LIKE ? OR LOWER(user_agent) LIKE ? OR LOWER(output_format) LIKE ?",
			pattern,
			pattern,
			pattern,
		)
	}
	if filter.Enabled != nil {
		query = query.Where("enabled = ?", *filter.Enabled)
	}
	return query
}

func (m *DefaultSubscribeApplicationModel) ListEnabled(ctx context.Context) ([]*SubscribeApplication, error) {
	resp := make([]*SubscribeApplication, 0)
	if err := m.WithContext(ctx).
		Select(subscribeApplicationAdminListColumns).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&resp).Error; err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *DefaultSubscribeApplicationModel) ListEnabledUserAgents(ctx context.Context) ([]string, error) {
	resp := make([]string, 0)
	if err := m.WithContext(ctx).
		Model(&SubscribeApplication{}).
		Where("enabled = ?", true).
		Order("id ASC").
		Pluck("user_agent", &resp).Error; err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *DefaultSubscribeApplicationModel) ListEnabledMatchers(ctx context.Context) ([]*SubscribeApplication, error) {
	resp := make([]*SubscribeApplication, 0)
	if err := m.WithContext(ctx).
		Select(subscribeApplicationMatcherColumns).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&resp).Error; err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *DefaultSubscribeApplicationModel) ListEnabledRuntime(ctx context.Context) ([]*SubscribeApplication, error) {
	resp := make([]*SubscribeApplication, 0)
	if err := m.WithContext(ctx).
		Select(subscribeApplicationRuntimeColumns).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&resp).Error; err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *DefaultSubscribeApplicationModel) ListEnabledSummary(ctx context.Context) ([]*SubscribeApplication, error) {
	resp := make([]*SubscribeApplication, 0)
	if err := m.WithContext(ctx).
		Select(subscribeApplicationClientColumns).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&resp).Error; err != nil {
		return nil, err
	}
	return resp, nil
}

var subscribeApplicationClientColumns = []string{
	"id",
	"name",
	"icon",
	"description",
	"scheme",
	"enabled",
	"is_default",
	"download_link",
}

var subscribeApplicationAdminListColumns = []string{
	"id",
	"name",
	"icon",
	"description",
	"scheme",
	"user_agent",
	"enabled",
	"is_default",
	"output_format",
	"download_link",
	"created_at",
	"updated_at",
}

var subscribeApplicationUpdateReadColumns = []string{
	"id",
	"name",
	"icon",
	"description",
	"scheme",
	"user_agent",
	"enabled",
	"is_default",
	"output_format",
	"download_link",
	"created_at",
	"updated_at",
}

var subscribeApplicationRuntimeColumns = []string{
	"id",
	"name",
	"user_agent",
	"is_default",
	"subscribe_template",
	"output_format",
}

var subscribeApplicationMatcherColumns = []string{
	"id",
	"name",
	"user_agent",
	"is_default",
}

var subscribeApplicationPreviewColumns = []string{
	"id",
	"subscribe_template",
	"output_format",
}
