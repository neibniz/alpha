package coupon

import (
	"context"
	"fmt"
	"strconv"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type customCouponLogicModel interface {
	UpdateCount(ctx context.Context, code string, tx ...*gorm.DB) error
	QueryCouponListByPage(ctx context.Context, page, size int, subscribe int64, search string) (total int64, list []*Coupon, err error)
	BatchDelete(ctx context.Context, ids []int64) error
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customCouponModel{
		defaultCouponModel: newCouponModel(conn, c),
	}
}

// QueryCouponListByPage query coupon list by page
func (m *customCouponModel) QueryCouponListByPage(ctx context.Context, page, size int, subscribe int64, search string) (total int64, list []*Coupon, err error) {
	list = make([]*Coupon, 0)
	err = m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		countQuery := applyCouponListFilter(conn.Model(&Coupon{}), subscribe, search)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		list = make([]*Coupon, 0, capacity)
		return applyCouponListFilter(conn.Model(&Coupon{}), subscribe, search).
			Select(couponListColumns).
			Order("id DESC").
			Limit(size).
			Offset(offset).
			Find(&list).Error
	})
	return total, list, err
}

var couponListColumns = []string{
	"id",
	"name",
	"code",
	"count",
	"type",
	"discount",
	"start_time",
	"expire_time",
	"user_limit",
	"used_count",
	"enable",
}

var couponCacheColumns = []string{
	"id",
	"code",
}

func (m *customCouponModel) queryCouponCacheRows(ctx context.Context, ids []int64) ([]*Coupon, error) {
	coupons := make([]*Coupon, 0)
	err := m.QueryNoCacheCtx(ctx, &coupons, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Coupon{}).
			Select(couponCacheColumns).
			Where("id IN ?", ids).
			Find(v).Error
	})
	return coupons, err
}

func (m *customCouponModel) findOneByCodeForCache(ctx context.Context, code string) (*Coupon, error) {
	var coupon Coupon
	err := m.QueryNoCacheCtx(ctx, &coupon, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Coupon{}).
			Select(couponCacheColumns).
			Where("code = ?", code).
			First(v).Error
	})
	return &coupon, err
}

func applyCouponListFilter(conn *gorm.DB, subscribe int64, search string) *gorm.DB {
	if subscribe != 0 {
		conn = conn.Where("string_to_array(COALESCE(subscribe, ''), ',') && ?", pq.Array([]string{strconv.FormatInt(subscribe, 10)}))
	}
	if search != "" {
		conn = conn.Where("name like ? or code like ?", "%"+search+"%", "%"+search+"%")
	}
	return conn
}

func (m *customCouponModel) BatchDelete(ctx context.Context, ids []int64) error {
	ids = normalizeCouponIDs(ids)
	if len(ids) == 0 {
		return nil
	}

	coupons, err := m.queryCouponCacheRows(ctx, ids)
	if err != nil {
		return err
	}

	cacheKeys := newCouponCacheKeyCollector(len(coupons)*2 + len(ids))
	for _, coupon := range coupons {
		for _, key := range m.getCacheKeys(coupon) {
			cacheKeys.Add(key)
		}
	}
	for _, id := range ids {
		cacheKeys.Add(cachekeys.Int(cacheCouponIdPrefix, id))
	}

	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Where("id IN ?", ids).Delete(&Coupon{}).Error
	}, cacheKeys.Values()...)
}

const smallCouponSetThreshold = 8

func normalizeCouponIDs(ids []int64) []int64 {
	set := newCouponIDSet(len(ids))
	for _, id := range ids {
		set.Add(id)
	}
	return set.Values()
}

type couponIDSet struct {
	ids     []int64
	seen    map[int64]struct{}
	capHint int
}

func newCouponIDSet(capHint int) couponIDSet {
	return couponIDSet{capHint: capHint}
}

func (s *couponIDSet) Add(id int64) bool {
	if s == nil || id <= 0 {
		return false
	}
	if s.seen != nil {
		if _, ok := s.seen[id]; ok {
			return false
		}
		s.seen[id] = struct{}{}
		s.ids = append(s.ids, id)
		return true
	}
	for _, existing := range s.ids {
		if existing == id {
			return false
		}
	}
	s.ensureIDs()
	s.ids = append(s.ids, id)
	if len(s.ids) > smallCouponSetThreshold {
		s.upgradeLookup()
	}
	return true
}

func (s couponIDSet) Values() []int64 {
	if len(s.ids) == 0 {
		return []int64{}
	}
	return s.ids
}

func (s *couponIDSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	s.ids = make([]int64, 0, couponSetInitialCap(s.capHint))
}

func (s *couponIDSet) upgradeLookup() {
	if s.seen != nil {
		return
	}
	nextCap := couponSetLookupCap(len(s.ids), s.capHint)
	if nextCap > cap(s.ids) {
		ids := make([]int64, len(s.ids), nextCap)
		copy(ids, s.ids)
		s.ids = ids
	}
	s.seen = make(map[int64]struct{}, nextCap)
	for _, id := range s.ids {
		s.seen[id] = struct{}{}
	}
}

type couponCacheKeyCollector struct {
	keys    []string
	seen    map[string]struct{}
	capHint int
}

func newCouponCacheKeyCollector(capHint int) couponCacheKeyCollector {
	return couponCacheKeyCollector{capHint: capHint}
}

func (c *couponCacheKeyCollector) Add(key string) bool {
	if c == nil || key == "" {
		return false
	}
	if c.seen != nil {
		if _, ok := c.seen[key]; ok {
			return false
		}
		c.seen[key] = struct{}{}
		c.keys = append(c.keys, key)
		return true
	}
	for _, existing := range c.keys {
		if existing == key {
			return false
		}
	}
	c.ensureKeys()
	c.keys = append(c.keys, key)
	if len(c.keys) > smallCouponSetThreshold {
		c.upgradeLookup()
	}
	return true
}

func (c couponCacheKeyCollector) Values() []string {
	if len(c.keys) == 0 {
		return []string{}
	}
	return c.keys
}

func (c *couponCacheKeyCollector) ensureKeys() {
	if c.keys != nil {
		return
	}
	c.keys = make([]string, 0, couponSetInitialCap(c.capHint))
}

func (c *couponCacheKeyCollector) upgradeLookup() {
	if c.seen != nil {
		return
	}
	nextCap := couponSetLookupCap(len(c.keys), c.capHint)
	if nextCap > cap(c.keys) {
		keys := make([]string, len(c.keys), nextCap)
		copy(keys, c.keys)
		c.keys = keys
	}
	c.seen = make(map[string]struct{}, nextCap)
	for _, key := range c.keys {
		c.seen[key] = struct{}{}
	}
}

func couponSetInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallCouponSetThreshold+1 {
		return smallCouponSetThreshold + 1
	}
	return capHint
}

func couponSetLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (m *customCouponModel) UpdateCount(ctx context.Context, code string, tx ...*gorm.DB) error {
	data, err := m.findOneByCodeForCache(ctx, code)
	if err != nil {
		return err
	}
	var rowsAffected int64
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		result := conn.Model(&Coupon{}).
			Where("code = ? AND (count = 0 OR used_count < count)", code).
			Update("used_count", gorm.Expr("used_count + ?", 1))
		rowsAffected = result.RowsAffected
		return result.Error
	}, m.getCacheKeys(data)...)
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("coupon usage limit exceeded")
	}
	return nil
}
