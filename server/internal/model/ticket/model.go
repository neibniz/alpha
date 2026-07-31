package ticket

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/internal/model/pagination"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var cacheTicketDetailPrefix = "cache:ticket:detail:"

var ErrInvalidTicketTransition = errors.New("invalid ticket status transition")

type Details struct {
	Id          int64        `gorm:"primaryKey"`
	Title       string       `gorm:"type:varchar(255);not null;default:'';comment:Title"`
	Description string       `gorm:"type:text;comment:Description"`
	Images      ImageGallery `gorm:"type:text[];not null;default:'{}';comment:Image Gallery"`
	UserId      int64        `gorm:"type:bigint;not null;default:0;comment:UserId"`
	Status      uint8        `gorm:"type:smallint;not null;default:1;comment:Status"`
	Follows     []Follow     `gorm:"foreignKey:TicketId;references:Id"`
	CreatedAt   time.Time    `gorm:"<-:create;comment:Create Time"`
	UpdatedAt   time.Time    `gorm:"comment:Update Time"`
}
type customTicketLogicModel interface {
	FindOneForFollow(ctx context.Context, id int64) (*Ticket, error)
	QueryTicketDetail(ctx context.Context, id int64) (*Details, error)
	InsertTicketFollow(ctx context.Context, data *Follow) error
	InsertTicketFollowAndUpdateStatus(ctx context.Context, data *Follow, userID int64, status uint8) error
	QueryTicketList(ctx context.Context, page, size int, userId int64, status *uint8, search string, options TicketListOptions) (int64, []*Ticket, error)
	UpdateTicketStatus(ctx context.Context, id, userId int64, status uint8) error
	QueryWaitReplyTotal(ctx context.Context) (int64, error)
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customTicketModel{
		defaultTicketModel: newTicketModel(conn, c),
	}
}

func (m *customTicketModel) FindOneForFollow(ctx context.Context, id int64) (*Ticket, error) {
	var data Ticket
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Ticket{}).
			Select(ticketFollowLookupColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &data, err
}

// QueryTicketDetail returns the ticket details.
func (m *customTicketModel) QueryTicketDetail(ctx context.Context, id int64) (*Details, error) {
	key := cachekeys.Int(cacheTicketDetailPrefix, id)
	var data Details
	err := m.QueryCtx(ctx, &data, key, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Ticket{}).
			Select(ticketDetailColumns).
			Where("id = ?", id).
			Preload("Follows", func(db *gorm.DB) *gorm.DB {
				return db.Select(ticketFollowDetailColumns).Order("id ASC")
			}).
			First(v).Error
	})
	return &data, err
}

// InsertTicketFollow inserts a follow record.
func (m *customTicketModel) InsertTicketFollow(ctx context.Context, data *Follow) error {
	key := cachekeys.Int(cacheTicketDetailPrefix, data.TicketId)
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Model(&Follow{}).Create(data).Error
	}, key)
}

// InsertTicketFollowAndUpdateStatus persists the reply and state transition atomically.
// A positive userID also acts as the ownership condition for user-side replies.
func (m *customTicketModel) InsertTicketFollowAndUpdateStatus(ctx context.Context, data *Follow, userID int64, status uint8) error {
	if data == nil || data.TicketId <= 0 {
		return gorm.ErrRecordNotFound
	}
	if (userID > 0 && status != Pending) || (userID == 0 && status != Waiting) {
		return ErrInvalidTicketTransition
	}
	key := cachekeys.Int(cacheTicketDetailPrefix, data.TicketId)
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Transaction(func(tx *gorm.DB) error {
			update := tx.Model(&Ticket{}).
				Where("id = ?", data.TicketId).
				Where("status != ?", Closed)
			if userID > 0 {
				update = update.Where("user_id = ?", userID)
			}
			result := update.Update("status", status)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
			return tx.Create(data).Error
		})
	}, key)
}

type TicketListOptions struct {
	IncludeUserID             bool
	IncludeDescriptionPreview bool
}

// QueryTicketList returns the ticket list.
func (m *customTicketModel) QueryTicketList(ctx context.Context, page, size int, userId int64, status *uint8, search string, options TicketListOptions) (int64, []*Ticket, error) {
	data := make([]*Ticket, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		countQuery := applyTicketListFilter(conn.Model(&Ticket{}), userId, status, search)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		data = make([]*Ticket, 0, capacity)
		return applyTicketListFilter(conn.Model(&Ticket{}), userId, status, search).
			Select(ticketListSelectColumns(options)).
			Order("id desc").
			Limit(size).
			Offset(offset).
			Find(&data).Error
	})
	return total, data, err
}

const ticketDescriptionPreviewLimit = 240

var ticketListBaseColumns = []string{
	"id",
	"title",
	"status",
	"updated_at",
}

const ticketListUserIDColumn = "user_id"

var ticketFollowLookupColumns = []string{
	"id",
	"user_id",
}

func ticketListSelectColumns(options TicketListOptions) []string {
	columns := make([]string, 0, len(ticketListBaseColumns)+2)
	columns = append(columns, ticketListBaseColumns...)
	if options.IncludeUserID {
		columns = append(columns, ticketListUserIDColumn)
	}
	if options.IncludeDescriptionPreview {
		columns = append(columns, fmt.Sprintf("substr(description, 1, %d) AS description", ticketDescriptionPreviewLimit))
	}
	return columns
}

var ticketDetailColumns = []string{
	"id",
	"title",
	"description",
	"images",
	"user_id",
	"status",
	"created_at",
	"updated_at",
}

var ticketFollowDetailColumns = []string{
	"id",
	"ticket_id",
	"from",
	"type",
	"content",
	"images",
	"created_at",
}

func applyTicketListFilter(conn *gorm.DB, userId int64, status *uint8, search string) *gorm.DB {
	if userId > 0 {
		conn = conn.Where("user_id = ?", userId)
	}
	if status != nil {
		conn = conn.Where("status = ?", *status)
	} else {
		conn = conn.Where("status != ?", 4)
	}
	if search != "" {
		conn = conn.Where("title like ? or description like ?", "%"+search+"%", "%"+search+"%")
	}
	return conn
}

// UpdateTicketStatus updates the ticket status.
func (m *customTicketModel) UpdateTicketStatus(ctx context.Context, id, userId int64, status uint8) error {
	sources, ok := ticketStatusSources(status)
	if id <= 0 || !ok {
		return ErrInvalidTicketTransition
	}
	key := cachekeys.Int(cacheTicketDetailPrefix, id)
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		conn = conn.Model(&Ticket{})
		if userId > 0 {
			conn = conn.Where("user_id = ?", userId)
		}
		result := conn.
			Where("id = ?", id).
			Where("status IN ?", sources).
			Update("status", status)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}, key)
}

func ticketStatusSources(status uint8) ([]int, bool) {
	switch status {
	case Pending:
		return []int{int(Pending), int(Closed)}, true
	case Processed:
		return []int{int(Pending), int(Waiting), int(Processed)}, true
	case Closed:
		return []int{int(Pending), int(Waiting), int(Processed), int(Closed)}, true
	default:
		return nil, false
	}
}

// QueryWaitReplyTotal returns the total number of tickets that are waiting for a reply.
func (m *customTicketModel) QueryWaitReplyTotal(ctx context.Context) (int64, error) {
	var total int64
	err := m.QueryNoCacheCtx(ctx, &total, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Ticket{}).Where("status = ?", Pending).Count(&total).Error
	})
	return total, err
}
