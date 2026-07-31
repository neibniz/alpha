package log

import (
	"time"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/types"
)

const smallTrafficDetailsAvailabilityCacheThreshold = 8

var serverTrafficLogDailyColumns = []string{
	"date",
	"server_id",
	"download",
	"upload",
	"total",
}

var userSubscribeTrafficLogDailyColumns = []string{
	"date",
	"user_id",
	"subscribe_id",
	"download",
	"upload",
	"total",
}

func serverTrafficLogDailySelectColumns() []string {
	return serverTrafficLogDailyColumns
}

func userSubscribeTrafficLogDailySelectColumns() []string {
	return userSubscribeTrafficLogDailyColumns
}

type trafficPageWindow struct {
	todayOffset   int
	todayLimit    int
	historyOffset int
	historyLimit  int
}

func splitTrafficPage(page, size int, todayTotal int64) trafficPageWindow {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		return trafficPageWindow{}
	}
	offset := int64(pagination.Offset(page, size))
	if offset >= todayTotal {
		return trafficPageWindow{
			historyOffset: int(offset - todayTotal),
			historyLimit:  size,
		}
	}
	todayLimit := size
	if remaining := todayTotal - offset; remaining < int64(size) {
		todayLimit = int(remaining)
	}
	return trafficPageWindow{
		todayOffset:  int(offset),
		todayLimit:   todayLimit,
		historyLimit: size - todayLimit,
	}
}

func newTrafficPageRows[T any](total int64, page, limit int) ([]T, int, bool) {
	offset, capacity, ok := pagination.PageWindow(total, page, limit)
	return make([]T, 0, capacity), offset, ok
}

func localDayRange(now time.Time) (time.Time, time.Time) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return start, start.Add(24 * time.Hour)
}

func serverTrafficLogFromDaily(item traffic.DailyServerTraffic, details bool) types.ServerTrafficLog {
	return types.ServerTrafficLog{
		ServerId: item.ServerId,
		Upload:   item.Upload,
		Download: item.Download,
		Total:    item.Total,
		Date:     item.Date,
		Details:  details,
	}
}

func userSubscribeTrafficLogFromDaily(item traffic.DailyUserTraffic, details bool) types.UserSubscribeTrafficLog {
	return types.UserSubscribeTrafficLog{
		UserId:      item.UserId,
		SubscribeId: item.SubscribeId,
		Upload:      item.Upload,
		Download:    item.Download,
		Total:       item.Total,
		Date:        item.Date,
		Details:     details,
	}
}

func applyServerTrafficMarker(list []types.ServerTrafficLog, date string, details bool) {
	for i := range list {
		list[i].Date = date
		list[i].Details = details
	}
}

func applyUserSubscribeTrafficMarker(list []types.UserSubscribeTrafficLog, date string, details bool) {
	for i := range list {
		list[i].Date = date
		list[i].Details = details
	}
}

type trafficDetailsAvailabilityResult struct {
	available bool
	err       error
}

type trafficDetailsAvailabilityCacheEntry struct {
	date   string
	result trafficDetailsAvailabilityResult
}

type trafficDetailsAvailabilityCache struct {
	now       time.Time
	autoClear bool
	clearDays int64
	rows      []trafficDetailsAvailabilityCacheEntry
	index     map[string]int
	capHint   int
}

func newTrafficDetailsAvailabilityCache(now time.Time, autoClear bool, clearDays int64) *trafficDetailsAvailabilityCache {
	return newTrafficDetailsAvailabilityCacheWithHint(now, autoClear, clearDays, 0)
}

func newTrafficDetailsAvailabilityCacheWithHint(now time.Time, autoClear bool, clearDays int64, capHint int) *trafficDetailsAvailabilityCache {
	return &trafficDetailsAvailabilityCache{
		now:       now,
		autoClear: autoClear,
		clearDays: clearDays,
		capHint:   capHint,
	}
}

func (c *trafficDetailsAvailabilityCache) available(date string) (bool, error) {
	if c == nil {
		return true, nil
	}
	if result, ok := c.lookup(date); ok {
		return result.available, result.err
	}
	available, err := trafficDetailsAvailable(date, c.now, c.autoClear, c.clearDays)
	result := trafficDetailsAvailabilityResult{
		available: available,
		err:       err,
	}
	c.add(date, result)
	return available, err
}

func (c *trafficDetailsAvailabilityCache) lookup(date string) (trafficDetailsAvailabilityResult, bool) {
	if c.index != nil {
		row, ok := c.index[date]
		if !ok {
			return trafficDetailsAvailabilityResult{}, false
		}
		return c.rows[row].result, true
	}
	for _, row := range c.rows {
		if row.date == date {
			return row.result, true
		}
	}
	return trafficDetailsAvailabilityResult{}, false
}

func (c *trafficDetailsAvailabilityCache) add(date string, result trafficDetailsAvailabilityResult) {
	if c.index != nil {
		if row, ok := c.index[date]; ok {
			c.rows[row].result = result
			return
		}
		c.ensureRows()
		c.rows = append(c.rows, trafficDetailsAvailabilityCacheEntry{date: date, result: result})
		c.index[date] = len(c.rows) - 1
		return
	}
	for i, row := range c.rows {
		if row.date == date {
			c.rows[i].result = result
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, trafficDetailsAvailabilityCacheEntry{date: date, result: result})
	if len(c.rows) > smallTrafficDetailsAvailabilityCacheThreshold {
		nextCap := trafficDetailsAvailabilityCacheCap(len(c.rows), c.capHint)
		if nextCap > cap(c.rows) {
			rows := make([]trafficDetailsAvailabilityCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		c.index = make(map[string]int, nextCap)
		for i, row := range c.rows {
			c.index[row.date] = i
		}
	}
}

func (c *trafficDetailsAvailabilityCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallTrafficDetailsAvailabilityCacheThreshold
	} else if capHint > smallTrafficDetailsAvailabilityCacheThreshold+1 {
		capHint = smallTrafficDetailsAvailabilityCacheThreshold + 1
	}
	c.rows = make([]trafficDetailsAvailabilityCacheEntry, 0, capHint)
}

func trafficDetailsAvailabilityCacheCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func applyServerTrafficDetailsAvailability(list []types.ServerTrafficLog, cache *trafficDetailsAvailabilityCache) error {
	var firstErr error
	for i := range list {
		available, err := cache.available(list[i].Date)
		list[i].Details = available
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func applyUserSubscribeTrafficDetailsAvailability(list []types.UserSubscribeTrafficLog, cache *trafficDetailsAvailabilityCache) error {
	var firstErr error
	for i := range list {
		available, err := cache.available(list[i].Date)
		list[i].Details = available
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func trafficDetailsAvailable(date string, now time.Time, autoClear bool, clearDays int64) (bool, error) {
	if !autoClear {
		return true, nil
	}
	dataTime, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return true, err
	}
	return !dataTime.Before(now.AddDate(0, 0, int(-clearDays))), nil
}
