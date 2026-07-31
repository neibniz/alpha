package traffic

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/pagination"

	"gorm.io/gorm"
)

type customTrafficLogicModel interface {
	QueryServerTrafficByDay(ctx context.Context, serverId int64, date time.Time) (*TotalTraffic, error)
	QueryTrafficByDay(ctx context.Context, date time.Time) (*TotalTraffic, error)
	QueryTrafficByMonthly(ctx context.Context, date time.Time) (*TotalTraffic, error)
	TopServersTrafficByDay(ctx context.Context, date time.Time, limit int) ([]ServerTrafficRanking, error)
	TopServersTrafficByMonthly(ctx context.Context, date time.Time, limit int) ([]ServerTrafficRanking, error)
	TopUsersTrafficByDay(ctx context.Context, date time.Time, limit int) ([]UserTrafficRanking, error)
	TopUsersTrafficByMonthly(ctx context.Context, date time.Time, limit int) ([]UserTrafficRanking, error)
	QueryTrafficLogPageList(ctx context.Context, userId, subscribeId int64, page, size int) ([]*TrafficLog, int64, error)
	QueryTrafficLogDetailsPage(ctx context.Context, filter *TrafficLogDetailsFilterParams) ([]*TrafficLog, int64, error)
	QueryDailyUserTrafficPage(ctx context.Context, date string, page, size int) ([]*DailyUserTraffic, int64, error)
	QueryDailyServerTrafficPage(ctx context.Context, date string, page, size int) ([]*DailyServerTraffic, int64, error)
	TopDailyUserTraffic(ctx context.Context, date string, limit int) ([]DailyUserTraffic, error)
	TopDailyServerTraffic(ctx context.Context, date string, limit int) ([]DailyServerTraffic, error)
	GetDailyTrafficStat(ctx context.Context, date string) (*DailyTrafficStat, error)
}

type TrafficLogDetailsFilterParams struct {
	Page        int
	Size        int
	ServerId    int64
	UserId      int64
	SubscribeId int64
	Start       time.Time
	End         time.Time
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB) Model {
	return &customTrafficModel{
		defaultTrafficModel: newTrafficModel(conn),
	}
}

func (m *customTrafficModel) QueryServerTrafficByDay(ctx context.Context, serverId int64, date time.Time) (*TotalTraffic, error) {
	var data TotalTraffic
	start, end := dayTrafficRange(date)
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select("sum(download) as download, sum(upload) as upload").
		Where("server_id = ? AND timestamp >= ? AND timestamp < ?", serverId, start, end).
		Scan(&data).Error
	return &data, err
}

func (m *customTrafficModel) QueryTrafficByDay(ctx context.Context, date time.Time) (*TotalTraffic, error) {
	var data TotalTraffic
	start, end := dayTrafficRange(date)
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select("sum(download) as download, sum(upload) as upload").
		Where("timestamp >= ? AND timestamp < ?", start, end).
		Scan(&data).Error
	return &data, err
}

func (m *customTrafficModel) QueryTrafficByMonthly(ctx context.Context, date time.Time) (*TotalTraffic, error) {
	var data TotalTraffic
	start, end := monthTrafficRange(date)
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select("sum(download) as download, sum(upload) as upload").
		Where("timestamp >= ? AND timestamp < ?", start, end).
		Scan(&data).Error
	return &data, err
}

func (m *customTrafficModel) TopServersTrafficByDay(ctx context.Context, date time.Time, limit int) ([]ServerTrafficRanking, error) {
	limit = normalizeTrafficLimit(limit)
	summaries := make([]ServerTrafficRanking, 0, trafficLimitRowsCap(limit))
	start, end := dayTrafficRange(date)
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select("server_id, SUM(download + upload) AS total, SUM(download) AS download, SUM(upload) AS upload").
		Where("timestamp >= ? AND timestamp < ?", start, end).
		Group("server_id").
		Order("total DESC").
		Limit(limit).
		Scan(&summaries).Error
	return summaries, err
}

func (m *customTrafficModel) TopServersTrafficByMonthly(ctx context.Context, date time.Time, limit int) ([]ServerTrafficRanking, error) {
	limit = normalizeTrafficLimit(limit)
	summaries := make([]ServerTrafficRanking, 0, trafficLimitRowsCap(limit))
	start, end := monthTrafficRange(date)
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select("server_id, SUM(download + upload) AS total, SUM(download) AS download, SUM(upload) AS upload").
		Where("timestamp >= ? AND timestamp < ?", start, end).
		Group("server_id").
		Order("total DESC").
		Limit(limit).
		Scan(&summaries).Error
	return summaries, err
}

func (m *customTrafficModel) TopUsersTrafficByDay(ctx context.Context, date time.Time, limit int) ([]UserTrafficRanking, error) {
	limit = normalizeTrafficLimit(limit)
	summaries := make([]UserTrafficRanking, 0, trafficLimitRowsCap(limit))
	start, end := dayTrafficRange(date)
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select("user_id, subscribe_id, SUM(download + upload) AS total, SUM(download) AS download, SUM(upload) AS upload").
		Where("timestamp >= ? AND timestamp < ?", start, end).
		Group("user_id, subscribe_id"). // 修改这里，添加 subscribe_id 到 GROUP BY 子句
		Order("total DESC").
		Limit(limit).
		Scan(&summaries).Error
	return summaries, err
}

func (m *customTrafficModel) TopUsersTrafficByMonthly(ctx context.Context, date time.Time, limit int) ([]UserTrafficRanking, error) {
	limit = normalizeTrafficLimit(limit)
	summaries := make([]UserTrafficRanking, 0, trafficLimitRowsCap(limit))
	start, end := monthTrafficRange(date)
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select("user_id, subscribe_id, SUM(download + upload) AS total, SUM(download) AS download, SUM(upload) AS upload"). // 添加 subscribe_id 到 SELECT 列表
		Where("timestamp >= ? AND timestamp < ?", start, end).
		Group("user_id, subscribe_id"). // 修改这里，添加 subscribe_id 到 GROUP BY 子句
		Order("total DESC").
		Limit(limit).
		Scan(&summaries).Error
	return summaries, err
}

// QueryTrafficLogPageList returns a list of records that meet the conditions.
func (m *customTrafficModel) QueryTrafficLogPageList(ctx context.Context, userId, subscribeId int64, page, size int) ([]*TrafficLog, int64, error) {
	page, size = pagination.NormalizeWithLimit(page, size, 10, pagination.MaxPageSize)
	var total int64
	countQuery := m.Conn.WithContext(ctx).Model(&TrafficLog{}).Where("user_id = ? AND subscribe_id = ?", userId, subscribeId)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, capacity, ok := pagination.PageWindow(total, page, size)
	list := make([]*TrafficLog, 0, capacity)
	if !ok {
		return list, total, nil
	}
	err := m.Conn.WithContext(ctx).Model(&TrafficLog{}).
		Select(trafficLogPageColumns).
		Where("user_id = ? AND subscribe_id = ?", userId, subscribeId).
		Order("timestamp DESC, id DESC").
		Limit(size).
		Offset(offset).
		Find(&list).Error
	return list, total, err
}

func (m *customTrafficModel) QueryTrafficLogDetailsPage(ctx context.Context, filter *TrafficLogDetailsFilterParams) ([]*TrafficLog, int64, error) {
	if filter == nil {
		filter = &TrafficLogDetailsFilterParams{}
	}
	filter.Page, filter.Size = pagination.NormalizeWithLimit(filter.Page, filter.Size, 10, pagination.MaxPageSize)

	var total int64
	countQuery := applyTrafficLogDetailsFilter(m.Conn.WithContext(ctx).Model(&TrafficLog{}), filter)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, capacity, ok := pagination.PageWindow(total, filter.Page, filter.Size)
	list := make([]*TrafficLog, 0, capacity)
	if !ok {
		return list, total, nil
	}
	err := applyTrafficLogDetailsFilter(m.Conn.WithContext(ctx).Model(&TrafficLog{}), filter).
		Select(trafficLogPageColumns).
		Order("timestamp DESC, id DESC").
		Limit(filter.Size).
		Offset(offset).
		Find(&list).Error
	return list, total, err
}

var trafficLogPageColumns = []string{
	"id",
	"server_id",
	"user_id",
	"subscribe_id",
	"download",
	"upload",
	"timestamp",
}

func applyTrafficLogDetailsFilter(tx *gorm.DB, filter *TrafficLogDetailsFilterParams) *gorm.DB {
	if filter.ServerId != 0 {
		tx = tx.Where("server_id = ?", filter.ServerId)
	}
	if !filter.Start.IsZero() && !filter.End.IsZero() {
		tx = tx.Where("timestamp >= ? AND timestamp < ?", filter.Start, filter.End)
	}
	if filter.UserId != 0 {
		tx = tx.Where("user_id = ?", filter.UserId)
	}
	if filter.SubscribeId != 0 {
		tx = tx.Where("subscribe_id = ?", filter.SubscribeId)
	}
	return tx
}

func dayTrafficRange(date time.Time) (time.Time, time.Time) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	return start, start.Add(24 * time.Hour)
}

func monthTrafficRange(date time.Time) (time.Time, time.Time) {
	start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.Local)
	return start, start.AddDate(0, 1, 0)
}

func (m *customTrafficModel) QueryDailyUserTrafficPage(ctx context.Context, date string, page, size int) ([]*DailyUserTraffic, int64, error) {
	page, size = pagination.NormalizeWithLimit(page, size, 10, pagination.MaxPageSize)
	var total int64
	countQuery := m.Conn.WithContext(ctx).Model(&DailyUserTraffic{}).Where("date = ?", date)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, capacity, ok := pagination.PageWindow(total, page, size)
	list := make([]*DailyUserTraffic, 0, capacity)
	if !ok {
		return list, total, nil
	}
	err := m.Conn.WithContext(ctx).Model(&DailyUserTraffic{}).
		Select(dailyUserTrafficListColumns).
		Where("date = ?", date).
		Order("total DESC, id DESC").
		Limit(size).
		Offset(offset).
		Find(&list).Error
	return list, total, err
}

func (m *customTrafficModel) QueryDailyServerTrafficPage(ctx context.Context, date string, page, size int) ([]*DailyServerTraffic, int64, error) {
	page, size = pagination.NormalizeWithLimit(page, size, 10, pagination.MaxPageSize)
	var total int64
	countQuery := m.Conn.WithContext(ctx).Model(&DailyServerTraffic{}).Where("date = ?", date)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, capacity, ok := pagination.PageWindow(total, page, size)
	list := make([]*DailyServerTraffic, 0, capacity)
	if !ok {
		return list, total, nil
	}
	err := m.Conn.WithContext(ctx).Model(&DailyServerTraffic{}).
		Select(dailyServerTrafficListColumns).
		Where("date = ?", date).
		Order("total DESC, id DESC").
		Limit(size).
		Offset(offset).
		Find(&list).Error
	return list, total, err
}

func (m *customTrafficModel) TopDailyUserTraffic(ctx context.Context, date string, limit int) ([]DailyUserTraffic, error) {
	limit = normalizeTrafficLimit(limit)
	list := make([]DailyUserTraffic, 0, trafficLimitRowsCap(limit))
	err := m.Conn.WithContext(ctx).Model(&DailyUserTraffic{}).
		Select(dailyUserTrafficListColumns).
		Where("date = ?", date).
		Order("total DESC, id DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (m *customTrafficModel) TopDailyServerTraffic(ctx context.Context, date string, limit int) ([]DailyServerTraffic, error) {
	limit = normalizeTrafficLimit(limit)
	list := make([]DailyServerTraffic, 0, trafficLimitRowsCap(limit))
	err := m.Conn.WithContext(ctx).Model(&DailyServerTraffic{}).
		Select(dailyServerTrafficListColumns).
		Where("date = ?", date).
		Order("total DESC, id DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (m *customTrafficModel) GetDailyTrafficStat(ctx context.Context, date string) (*DailyTrafficStat, error) {
	var stat DailyTrafficStat
	err := m.Conn.WithContext(ctx).Model(&DailyTrafficStat{}).Where("date = ?", date).First(&stat).Error
	return &stat, err
}

var dailyUserTrafficListColumns = []string{
	"id",
	"date",
	"user_id",
	"subscribe_id",
	"download",
	"upload",
	"total",
}

var dailyServerTrafficListColumns = []string{
	"id",
	"date",
	"server_id",
	"download",
	"upload",
	"total",
}

func DailyUserTrafficListColumns() []string {
	return append([]string(nil), dailyUserTrafficListColumns...)
}

func DailyServerTrafficListColumns() []string {
	return append([]string(nil), dailyServerTrafficListColumns...)
}

func trafficLimitRowsCap(limit int) int {
	if limit <= 0 {
		return 0
	}
	return limit
}

func normalizeTrafficLimit(limit int) int {
	if limit < 0 {
		return 0
	}
	return limit
}
