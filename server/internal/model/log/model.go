package log

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/pagination"

	"gorm.io/gorm"
)

func NewModel(db *gorm.DB, hooks ...Hooks) Model {
	var h Hooks
	if len(hooks) > 0 {
		h = hooks[0]
	}
	return &customSystemLogModel{
		defaultLogModel: newSystemLogModel(db, h),
	}
}

type FilterParams struct {
	Page     int
	Size     int
	MaxSize  int
	Type     uint8
	Types    []int
	Data     string
	Search   string
	ObjectID int64
}

type SubscribeAccessFilterParams struct {
	Page            int
	Size            int
	Date            string
	UserId          int64
	UserSubscribeId int64
}

type customSystemLogLogicModel interface {
	FilterSystemLog(ctx context.Context, filter *FilterParams) ([]*SystemLog, int64, error)
	FilterLoginLogs(ctx context.Context, filter *FilterParams) ([]LoginLogRow, int64, error)
	FilterRegisterLogs(ctx context.Context, filter *FilterParams) ([]RegisterLogRow, int64, error)
	FilterMessageLogs(ctx context.Context, filter *FilterParams) ([]MessageLogRow, int64, error)
	FilterResetSubscribeLogs(ctx context.Context, filter *FilterParams) ([]ResetSubscribeLogRow, int64, error)
	FilterGiftLogs(ctx context.Context, filter *FilterParams) ([]GiftLogRow, int64, error)
	FilterBalanceLogs(ctx context.Context, filter *FilterParams) ([]BalanceLogRow, int64, error)
	FilterCommissionLogs(ctx context.Context, filter *FilterParams) ([]CommissionLogRow, int64, error)
	FilterSubscribeAccessLog(ctx context.Context, filter *SubscribeAccessFilterParams) ([]*SubscribeAccessLog, int64, error)
}

func (m *customSystemLogModel) FilterSystemLog(ctx context.Context, filter *FilterParams) ([]*SystemLog, int64, error) {
	filter = normalizeSystemLogFilter(filter)

	var total int64
	countQuery := applySystemLogFilter(m.WithContext(ctx).Model(&SystemLog{}), filter)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, capacity, ok := pagination.PageWindow(total, filter.Page, filter.Size)
	logs := make([]*SystemLog, 0, capacity)
	if !ok {
		return logs, total, nil
	}
	err := applySystemLogFilter(m.WithContext(ctx).Model(&SystemLog{}), filter).
		Select(systemLogListColumns).
		Order("id DESC").
		Limit(filter.Size).
		Offset(offset).
		Find(&logs).Error
	return logs, total, err
}

func (m *customSystemLogModel) FilterBalanceLogs(ctx context.Context, filter *FilterParams) ([]BalanceLogRow, int64, error) {
	return filterProjectedSystemLogs[BalanceLogRow](m, ctx, filter, balanceLogSelectColumns)
}

func (m *customSystemLogModel) FilterLoginLogs(ctx context.Context, filter *FilterParams) ([]LoginLogRow, int64, error) {
	return filterProjectedSystemLogs[LoginLogRow](m, ctx, filterWithType(filter, TypeLogin), loginLogSelectColumns)
}

func (m *customSystemLogModel) FilterRegisterLogs(ctx context.Context, filter *FilterParams) ([]RegisterLogRow, int64, error) {
	return filterProjectedSystemLogs[RegisterLogRow](m, ctx, filterWithType(filter, TypeRegister), registerLogSelectColumns)
}

func (m *customSystemLogModel) FilterMessageLogs(ctx context.Context, filter *FilterParams) ([]MessageLogRow, int64, error) {
	return filterProjectedSystemLogs[MessageLogRow](m, ctx, filter, messageLogSelectColumns)
}

func (m *customSystemLogModel) FilterResetSubscribeLogs(ctx context.Context, filter *FilterParams) ([]ResetSubscribeLogRow, int64, error) {
	return filterProjectedSystemLogs[ResetSubscribeLogRow](m, ctx, filterWithType(filter, TypeResetSubscribe), resetSubscribeLogSelectColumns)
}

func (m *customSystemLogModel) FilterGiftLogs(ctx context.Context, filter *FilterParams) ([]GiftLogRow, int64, error) {
	return filterProjectedSystemLogs[GiftLogRow](m, ctx, filterWithType(filter, TypeGift), giftLogSelectColumns)
}

func (m *customSystemLogModel) FilterCommissionLogs(ctx context.Context, filter *FilterParams) ([]CommissionLogRow, int64, error) {
	return filterProjectedSystemLogs[CommissionLogRow](m, ctx, filter, commissionLogSelectColumns)
}

func filterProjectedSystemLogs[T any](m *customSystemLogModel, ctx context.Context, filter *FilterParams, columns []string) ([]T, int64, error) {
	filter = normalizeSystemLogFilter(filter)

	var total int64
	countQuery := applySystemLogFilter(m.WithContext(ctx).Model(&SystemLog{}), filter)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, capacity, ok := pagination.PageWindow(total, filter.Page, filter.Size)
	rows := make([]T, 0, capacity)
	if !ok {
		return rows, total, nil
	}
	err := applySystemLogFilter(m.WithContext(ctx).Model(&SystemLog{}), filter).
		Select(columns).
		Order("id DESC").
		Limit(filter.Size).
		Offset(offset).
		Scan(&rows).Error
	return rows, total, err
}

func normalizeSystemLogFilter(filter *FilterParams) *FilterParams {
	if filter == nil {
		filter = &FilterParams{}
	}
	maxSize := filter.MaxSize
	if maxSize <= 0 {
		maxSize = pagination.MaxPageSize
	}
	filter.Page, filter.Size = pagination.NormalizeWithLimit(filter.Page, filter.Size, 10, maxSize)
	return filter
}

func normalizeSubscribeAccessLogFilter(filter *SubscribeAccessFilterParams) *SubscribeAccessFilterParams {
	if filter == nil {
		filter = &SubscribeAccessFilterParams{}
	}
	filter.Page, filter.Size = pagination.NormalizeWithLimit(filter.Page, filter.Size, 10, pagination.MaxPageSize)
	return filter
}

type subscribeAccessLogQueryFilter struct {
	params    *SubscribeAccessFilterParams
	dateFrom  time.Time
	dateUntil time.Time
	hasDate   bool
}

func prepareSubscribeAccessLogQueryFilter(filter *SubscribeAccessFilterParams) (subscribeAccessLogQueryFilter, error) {
	filter = normalizeSubscribeAccessLogFilter(filter)
	queryFilter := subscribeAccessLogQueryFilter{params: filter}
	if filter.Date == "" {
		return queryFilter, nil
	}
	day, err := time.ParseInLocation(time.DateOnly, filter.Date, time.Local)
	if err != nil {
		return queryFilter, err
	}
	queryFilter.dateFrom = day
	queryFilter.dateUntil = day.Add(24 * time.Hour)
	queryFilter.hasDate = true
	return queryFilter, nil
}

func (m *customSystemLogModel) FilterSubscribeAccessLog(ctx context.Context, filter *SubscribeAccessFilterParams) ([]*SubscribeAccessLog, int64, error) {
	queryFilter, err := prepareSubscribeAccessLogQueryFilter(filter)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	countQuery := applySubscribeAccessLogFilter(m.WithContext(ctx).Model(&SubscribeAccessLog{}), queryFilter)
	if err = countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, capacity, ok := pagination.PageWindow(total, queryFilter.params.Page, queryFilter.params.Size)
	logs := make([]*SubscribeAccessLog, 0, capacity)
	if !ok {
		return logs, total, nil
	}
	listQuery := applySubscribeAccessLogFilter(m.WithContext(ctx).Model(&SubscribeAccessLog{}), queryFilter)
	err = listQuery.
		Select(subscribeAccessLogListColumns).
		Order("created_at DESC, id DESC").
		Limit(queryFilter.params.Size).
		Offset(offset).
		Find(&logs).Error
	return logs, total, err
}

func applySystemLogFilter(tx *gorm.DB, filter *FilterParams) *gorm.DB {
	if filter.Type != 0 {
		tx = tx.Where("type = ?", filter.Type)
	} else if len(filter.Types) > 0 {
		tx = tx.Where("type IN ?", filter.Types)
	}
	if filter.Data != "" {
		tx = tx.Where("date = ?", filter.Data)
	}
	if filter.ObjectID != 0 {
		tx = tx.Where("object_id = ?", filter.ObjectID)
	}
	if filter.Search != "" {
		tx = tx.Where("content LIKE ?", "%"+filter.Search+"%")
	}
	return tx
}

func filterWithType(filter *FilterParams, typ Type) *FilterParams {
	next := FilterParams{}
	if filter != nil {
		next = *filter
	}
	next.Type = typ.Uint8()
	return &next
}

func applySubscribeAccessLogFilter(tx *gorm.DB, filter subscribeAccessLogQueryFilter) *gorm.DB {
	if filter.hasDate {
		tx = tx.Where("created_at >= ? AND created_at < ?", filter.dateFrom, filter.dateUntil)
	}
	if filter.params.UserId != 0 {
		tx = tx.Where("user_id = ?", filter.params.UserId)
	}
	if filter.params.UserSubscribeId != 0 {
		tx = tx.Where("user_subscribe_id = ?", filter.params.UserSubscribeId)
	}
	return tx
}

type BalanceLogRow struct {
	UserId    int64
	Type      uint16
	Amount    int64
	OrderNo   string
	Balance   int64
	Timestamp int64
}

type CommissionLogRow struct {
	UserId    int64
	Type      uint16
	Amount    int64
	OrderNo   string
	Timestamp int64
}

type MessageLogRow struct {
	Id        int64
	Type      uint8
	Platform  string
	To        string
	Subject   string
	Content   string
	Status    uint8
	CreatedAt time.Time
}

type ResetSubscribeLogRow struct {
	Id              int64
	Type            uint16
	UserId          int64
	UserSubscribeId int64
	OrderNo         string
	Timestamp       int64
}

type GiftLogRow struct {
	UserId      int64
	Type        uint16
	OrderNo     string
	SubscribeId int64
	Amount      int64
	Balance     int64
	Remark      string
	Timestamp   int64
}

type LoginLogRow struct {
	Id           int64
	UserId       int64
	Method       string
	LoginIP      string
	CountryShort string
	Region       string
	Location     string
	UserAgent    string
	Success      bool
	CreatedAt    time.Time
}

type RegisterLogRow struct {
	UserId       int64
	AuthMethod   string
	Identifier   string
	RegisterIP   string
	CountryShort string
	Region       string
	Location     string
	UserAgent    string
	Timestamp    int64
}

var systemLogListColumns = []string{
	"id",
	"type",
	"date",
	"object_id",
	"ip",
	"country_short",
	"region",
	"location",
	"method",
	"identifier",
	"user_agent",
	"success",
	"event_at",
	"content",
	"created_at",
}

var loginLogSelectColumns = []string{
	"id",
	"object_id AS user_id",
	"country_short",
	"region",
	"location",
	"method",
	"ip AS login_ip",
	"user_agent",
	"success",
	"created_at",
}

var registerLogSelectColumns = []string{
	"object_id AS user_id",
	"country_short",
	"region",
	"location",
	"method AS auth_method",
	"identifier",
	"ip AS register_ip",
	"user_agent",
	"event_at AS timestamp",
}

var messageLogSelectColumns = []string{
	"id",
	"type",
	"COALESCE(content::jsonb->>'platform', '') AS platform",
	"COALESCE(content::jsonb->>'to', '') AS to",
	"COALESCE(content::jsonb->>'subject', '') AS subject",
	"(content::jsonb->'content')::text AS content",
	"COALESCE(NULLIF(content::jsonb->>'status', ''), '0')::int AS status",
	"created_at",
}

var resetSubscribeLogSelectColumns = []string{
	"id",
	"object_id AS user_subscribe_id",
	"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
	"COALESCE(NULLIF(content::jsonb->>'user_id', ''), '0')::bigint AS user_id",
	"COALESCE(content::jsonb->>'order_no', '') AS order_no",
	"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
}

var giftLogSelectColumns = []string{
	"object_id AS user_id",
	"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
	"COALESCE(content::jsonb->>'order_no', '') AS order_no",
	"COALESCE(NULLIF(content::jsonb->>'subscribe_id', ''), '0')::bigint AS subscribe_id",
	"COALESCE(NULLIF(content::jsonb->>'amount', ''), '0')::bigint AS amount",
	"COALESCE(NULLIF(content::jsonb->>'balance', ''), '0')::bigint AS balance",
	"COALESCE(content::jsonb->>'remark', '') AS remark",
	"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
}

var balanceLogSelectColumns = []string{
	"object_id AS user_id",
	"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
	"COALESCE(NULLIF(content::jsonb->>'amount', ''), '0')::bigint AS amount",
	"COALESCE(content::jsonb->>'order_no', '') AS order_no",
	"COALESCE(NULLIF(content::jsonb->>'balance', ''), '0')::bigint AS balance",
	"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
}

var commissionLogSelectColumns = []string{
	"object_id AS user_id",
	"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
	"COALESCE(NULLIF(content::jsonb->>'amount', ''), '0')::bigint AS amount",
	"COALESCE(content::jsonb->>'order_no', '') AS order_no",
	"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
}

var subscribeAccessLogListColumns = []string{
	"id",
	"user_id",
	"user_subscribe_id",
	"token_prefix",
	"user_agent",
	"client_ip",
	"country_short",
	"region",
	"location",
	"created_at",
}
