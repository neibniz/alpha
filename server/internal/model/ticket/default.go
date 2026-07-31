package ticket

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var _ Model = (*customTicketModel)(nil)
var (
	cacheTicketIdPrefix = "cache:ticket:id:"
)

type (
	Model interface {
		ticketModel
		customTicketLogicModel
	}
	ticketModel interface {
		Insert(ctx context.Context, data *Ticket) error
		FindOne(ctx context.Context, id int64) (*Ticket, error)
		Update(ctx context.Context, data *Ticket) error
		Delete(ctx context.Context, id int64) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customTicketModel struct {
		*defaultTicketModel
	}
	defaultTicketModel struct {
		cache.CachedConn
		table string
	}
)

func newTicketModel(db *gorm.DB, c *redis.Client) *defaultTicketModel {
	return &defaultTicketModel{
		CachedConn: cache.NewConn(db, c),
		table:      "ticket",
	}
}

//nolint:unused
func (m *defaultTicketModel) batchGetCacheKeys(Tickets ...*Ticket) []string {
	keys := make([]string, 0, len(Tickets))
	for _, ticket := range Tickets {
		keys = append(keys, m.getCacheKeys(ticket)...)
	}
	return keys
}
func (m *defaultTicketModel) getCacheKeys(data *Ticket) []string {
	if data == nil {
		return []string{}
	}
	cacheKeys := []string{
		cachekeys.Int(cacheTicketIdPrefix, data.Id),
	}
	return cacheKeys
}

func (m *defaultTicketModel) Insert(ctx context.Context, data *Ticket) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Create(&data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultTicketModel) FindOne(ctx context.Context, id int64) (*Ticket, error) {
	ticketIdKey := cachekeys.Int(cacheTicketIdPrefix, id)
	var resp Ticket
	err := m.QueryCtx(ctx, &resp, ticketIdKey, func(conn *gorm.DB, v interface{}) error {

		return conn.Model(&Ticket{}).Where("id = ?", id).First(&resp).Error
	})
	switch {
	case err == nil:
		return &resp, nil
	default:
		return nil, err
	}
}

func (m *defaultTicketModel) Update(ctx context.Context, data *Ticket) error {
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Save(data).Error
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultTicketModel) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return nil
	}
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		db := conn
		return db.Delete(&Ticket{}, id).Error
	}, m.getCacheKeys(&Ticket{Id: id})...)
	return err
}

func (m *defaultTicketModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
