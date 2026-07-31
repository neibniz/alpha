package ads

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type customAdsLogicModel interface {
	FindOneForAdmin(ctx context.Context, id int64, includeContent bool) (*Ads, error)
	GetAdsListByPage(ctx context.Context, page, size int, filter Filter) (int64, []*Ads, error)
	UpdateFields(ctx context.Context, current *Ads, update AdsFieldsUpdate, tx ...*gorm.DB) error
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customAdsModel{
		defaultAdsModel: newAdsModel(conn, c),
	}
}

type Filter struct {
	Status         *int
	Search         string
	IncludeContent bool
}

type AdsFieldsUpdate struct {
	Title       *string
	Type        *string
	Content     *string
	Description *string
	TargetURL   *string
	StartTime   *time.Time
	EndTime     *time.Time
	Status      *int
}

func (u AdsFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 8)
	if u.Title != nil {
		updates["title"] = *u.Title
	}
	if u.Type != nil {
		updates["type"] = *u.Type
	}
	if u.Content != nil {
		updates["content"] = *u.Content
	}
	if u.Description != nil {
		updates["description"] = *u.Description
	}
	if u.TargetURL != nil {
		updates["target_url"] = *u.TargetURL
	}
	if u.StartTime != nil {
		updates["start_time"] = *u.StartTime
	}
	if u.EndTime != nil {
		updates["end_time"] = *u.EndTime
	}
	if u.Status != nil {
		updates["status"] = *u.Status
	}
	return updates
}

func (m *customAdsModel) FindOneForAdmin(ctx context.Context, id int64, includeContent bool) (*Ads, error) {
	var resp Ads
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Ads{}).
			Select(adsListSelectColumns(includeContent)).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

// GetAdsListByPage  get ads list by page
func (m *customAdsModel) GetAdsListByPage(ctx context.Context, page, size int, filter Filter) (int64, []*Ads, error) {
	list := make([]*Ads, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		countQuery := applyAdsListFilter(conn.Model(&Ads{}), filter)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		list = make([]*Ads, 0, capacity)
		listQuery := applyAdsListFilter(conn.Model(&Ads{}), filter).
			Order("id DESC").
			Offset(offset).
			Limit(size)
		if !filter.IncludeContent {
			listQuery = listQuery.Select(adsListColumns)
		}
		return listQuery.Find(&list).Error
	})
	return total, list, err
}

func (m *customAdsModel) UpdateFields(ctx context.Context, current *Ads, update AdsFieldsUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Ads{}).Where("id = ?", current.Id).Updates(updates).Error
	}, m.getCacheKeys(current)...)
}

func applyAdsListFilter(conn *gorm.DB, filter Filter) *gorm.DB {
	if filter.Status != nil {
		conn = conn.Where("status = ?", *filter.Status)
	}
	if filter.Search != "" {
		keyword := "%" + filter.Search + "%"
		if filter.IncludeContent {
			conn = conn.Where("title LIKE ? OR description LIKE ? OR content LIKE ?", keyword, keyword, keyword)
		} else {
			conn = conn.Where("title LIKE ? OR description LIKE ?", keyword, keyword)
		}
	}
	return conn
}

var adsListColumns = []string{
	"id",
	"title",
	"type",
	"description",
	"target_url",
	"start_time",
	"end_time",
	"status",
	"created_at",
	"updated_at",
}

func adsListSelectColumns(includeContent bool) []string {
	if !includeContent {
		return adsListColumns
	}
	columns := make([]string, 0, len(adsListColumns)+1)
	columns = append(columns, adsListColumns...)
	return append(columns, "content")
}
