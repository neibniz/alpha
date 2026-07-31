package announcement

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type customAnnouncementLogicModel interface {
	GetAnnouncementListByPage(ctx context.Context, page, size int, filter Filter) (int64, []*Announcement, error)
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customAnnouncementModel{
		defaultAnnouncementModel: newAnnouncementModel(conn, c),
	}
}

type Filter struct {
	Show           *bool
	Pinned         *bool
	Popup          *bool
	Search         string
	IncludeContent bool
}

// GetAnnouncementListByPage  get announcement list by page
func (m *customAnnouncementModel) GetAnnouncementListByPage(ctx context.Context, page, size int, filter Filter) (int64, []*Announcement, error) {
	list := make([]*Announcement, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		countQuery := applyAnnouncementListFilter(conn.Model(&Announcement{}), filter)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		list = make([]*Announcement, 0, capacity)
		listQuery := applyAnnouncementListFilter(conn.Model(&Announcement{}), filter)
		if !filter.IncludeContent {
			listQuery = listQuery.Select(announcementListColumns)
		}
		return listQuery.
			Order("id DESC").
			Offset(offset).
			Limit(size).
			Find(&list).Error
	})
	return total, list, err
}

func applyAnnouncementListFilter(conn *gorm.DB, filter Filter) *gorm.DB {
	if filter.Show != nil {
		conn = conn.Where("show = ?", *filter.Show)
	}
	if filter.Pinned != nil {
		conn = conn.Where("pinned = ?", *filter.Pinned)
	}
	if filter.Popup != nil {
		conn = conn.Where("popup = ?", *filter.Popup)
	}
	if filter.Search != "" {
		conn = conn.Where("title LIKE ? OR content LIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	return conn
}

var announcementListColumns = []string{
	"id",
	"title",
	"show",
	"pinned",
	"popup",
	"created_at",
	"updated_at",
}
