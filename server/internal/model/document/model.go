package document

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type customDocumentLogicModel interface {
	QueryDocumentDetail(ctx context.Context, id int64) (*Document, error)
	QueryDocumentList(ctx context.Context, page, size int, tag string, search string) (int64, []*Document, error)
	GetDocumentListByAll(ctx context.Context, page, size int) (int64, []*Document, error)
	BatchDelete(ctx context.Context, ids []int64) error
}

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customDocumentModel{
		defaultDocumentModel: newDocumentModel(conn, c),
	}
}

// QueryDocumentDetail queries the details of a document.
func (m *customDocumentModel) QueryDocumentDetail(ctx context.Context, id int64) (*Document, error) {
	var data Document
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Document{}).
			Select(documentDetailColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &data, err
}

func (m *customDocumentModel) queryDocumentCacheRows(ctx context.Context, ids []int64) ([]*Document, error) {
	documents := make([]*Document, 0)
	err := m.QueryNoCacheCtx(ctx, &documents, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Document{}).
			Select("id").
			Where("id IN ?", ids).
			Find(v).Error
	})
	return documents, err
}

func (m *customDocumentModel) BatchDelete(ctx context.Context, ids []int64) error {
	ids = normalizeDocumentIDs(ids)
	if len(ids) == 0 {
		return nil
	}
	documents, err := m.queryDocumentCacheRows(ctx, ids)
	if err != nil {
		return err
	}
	cacheKeys := newDocumentCacheKeyCollector(len(documents) + len(ids))
	for _, document := range documents {
		for _, key := range m.getCacheKeys(document) {
			cacheKeys.Add(key)
		}
	}
	for _, id := range ids {
		cacheKeys.Add(cachekeys.Int(cacheDocumentIdPrefix, id))
	}

	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Where("id IN ?", ids).Delete(&Document{}).Error
	}, cacheKeys.Values()...)
}

const smallDocumentSetThreshold = 8

func normalizeDocumentIDs(ids []int64) []int64 {
	set := newDocumentIDSet(len(ids))
	for _, id := range ids {
		set.Add(id)
	}
	return set.Values()
}

type documentIDSet struct {
	ids     []int64
	seen    map[int64]struct{}
	capHint int
}

func newDocumentIDSet(capHint int) documentIDSet {
	return documentIDSet{capHint: capHint}
}

func (s *documentIDSet) Add(id int64) bool {
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
	if len(s.ids) > smallDocumentSetThreshold {
		s.upgradeLookup()
	}
	return true
}

func (s documentIDSet) Values() []int64 {
	if len(s.ids) == 0 {
		return []int64{}
	}
	return s.ids
}

func (s *documentIDSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	s.ids = make([]int64, 0, documentSetInitialCap(s.capHint))
}

func (s *documentIDSet) upgradeLookup() {
	if s.seen != nil {
		return
	}
	nextCap := documentSetLookupCap(len(s.ids), s.capHint)
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

type documentCacheKeyCollector struct {
	keys    []string
	seen    map[string]struct{}
	capHint int
}

func newDocumentCacheKeyCollector(capHint int) documentCacheKeyCollector {
	return documentCacheKeyCollector{capHint: capHint}
}

func (c *documentCacheKeyCollector) Add(key string) bool {
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
	if len(c.keys) > smallDocumentSetThreshold {
		c.upgradeLookup()
	}
	return true
}

func (c documentCacheKeyCollector) Values() []string {
	if len(c.keys) == 0 {
		return []string{}
	}
	return c.keys
}

func (c *documentCacheKeyCollector) ensureKeys() {
	if c.keys != nil {
		return
	}
	c.keys = make([]string, 0, documentSetInitialCap(c.capHint))
}

func (c *documentCacheKeyCollector) upgradeLookup() {
	if c.seen != nil {
		return
	}
	nextCap := documentSetLookupCap(len(c.keys), c.capHint)
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

func documentSetInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallDocumentSetThreshold+1 {
		return smallDocumentSetThreshold + 1
	}
	return capHint
}

func documentSetLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

// QueryDocumentList queries a list of documents.
func (m *customDocumentModel) QueryDocumentList(ctx context.Context, page, size int, tag string, search string) (int64, []*Document, error) {
	data := make([]*Document, 0)
	var total int64
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.Normalize(page, size)
		countQuery := applyDocumentListFilter(conn.Model(&Document{}), tag, search)
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		data = make([]*Document, 0, capacity)
		return applyDocumentListFilter(conn.Model(&Document{}), tag, search).
			Select(documentListColumns).
			Order("id DESC").
			Offset(offset).
			Limit(size).
			Find(&data).Error
	})
	return total, data, err
}

func applyDocumentListFilter(db *gorm.DB, tag string, search string) *gorm.DB {
	if tag != "" {
		db = db.Where("string_to_array(COALESCE(tags, ''), ',') && ?", pq.Array([]string{tag}))
	}
	if search != "" {
		db = db.Where("title LIKE ? OR content LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	return db
}

// GetDocumentListByAll queries a list of documents.
func (m *customDocumentModel) GetDocumentListByAll(ctx context.Context, page, size int) (int64, []*Document, error) {
	data := make([]*Document, 0)
	var total int64
	show := true
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		page, size = pagination.NormalizeWithLimit(page, size, pagination.MaxPageSize, pagination.MaxPageSize)
		if err := conn.Model(&Document{}).Where("show = ?", &show).Count(&total).Error; err != nil {
			return err
		}
		offset, capacity, ok := pagination.PageWindow(total, page, size)
		if !ok {
			return nil
		}
		data = make([]*Document, 0, capacity)
		return conn.Model(&Document{}).
			Select(documentListColumns).
			Where("show = ?", &show).
			Order("id DESC").
			Offset(offset).
			Limit(size).
			Find(&data).Error
	})
	return total, data, err
}

var documentListColumns = []string{
	"id",
	"title",
	"tags",
	"show",
	"created_at",
	"updated_at",
}

var documentDetailColumns = []string{
	"id",
	"title",
	"content",
	"tags",
	"show",
	"created_at",
	"updated_at",
}
