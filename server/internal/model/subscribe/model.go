package subscribe

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type FilterParams struct {
	Page            int      // Page Number
	Size            int      // Page Size
	MaxSize         int      // Internal maximum page size
	Ids             []int64  // Subscribe IDs
	Node            []int64  // Node IDs
	Tags            []string // Node Tags
	Show            bool     // Show Portal Page
	UserVisible     bool     // Show authenticated user subscribe page
	Sell            bool     // Sell
	Language        string   // Language
	DefaultLanguage bool     // Default Subscribe Language Data
	Search          string   // Search Keywords
	Projection      FilterListProjection
	SkipCount       bool // Skip total count for full-list callers; total becomes len(list).
}

func (p *FilterParams) Normalize() {
	maxSize := p.MaxSize
	if maxSize <= 0 {
		maxSize = pagination.MaxPageSize
	}
	p.Page, p.Size = pagination.NormalizeWithLimit(p.Page, p.Size, 10, maxSize)
	p.Ids = normalizeSubscribeIDs(p.Ids)
	p.Node = normalizeSubscribeIDs(p.Node)
	p.Tags = tool.NormalizeStringSlice(p.Tags)
}

type customSubscribeLogicModel interface {
	FilterList(ctx context.Context, params *FilterParams) (int64, []*Subscribe, error)
	QueryGroupList(ctx context.Context) ([]*Group, error)
	ClearCache(ctx context.Context, id ...int64) error
	DeleteByIds(ctx context.Context, ids []int64, tx ...*gorm.DB) error
}

type FilterListProjection uint8

const (
	FilterListProjectionFull FilterListProjection = iota
	FilterListProjectionAdmin
	FilterListProjectionPublic
)

// NewModel returns a model for the database table.
func NewModel(conn *gorm.DB, c *redis.Client) Model {
	return &customSubscribeModel{
		defaultSubscribeModel: newSubscribeModel(conn, c),
	}
}

func (m *customSubscribeModel) ClearCache(ctx context.Context, ids ...int64) error {
	ids = normalizeSubscribeIDs(ids)
	if len(ids) <= 0 {
		return nil
	}

	cacheKeys, err := m.cacheKeysForSubscribeIds(ctx, ids)
	if err != nil {
		return err
	}
	if len(cacheKeys) == 0 {
		return nil
	}
	return m.CachedConn.DelCacheCtx(ctx, cacheKeys...)
}

func (m *customSubscribeModel) DeleteByIds(ctx context.Context, ids []int64, tx ...*gorm.DB) error {
	ids = normalizeSubscribeIDs(ids)
	if len(ids) == 0 {
		return nil
	}

	var db *gorm.DB
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0].WithContext(ctx)
	}
	subscribes, err := m.querySubscribeCacheRows(ctx, ids, db)
	if err != nil {
		return err
	}
	cacheKeys, err := m.cacheKeysForSubscribes(ctx, ids, subscribes, db)
	if err != nil {
		return err
	}
	if err := m.CachedConn.DelCacheCtx(ctx, cacheKeys...); err != nil {
		return err
	}

	sort.Slice(subscribes, func(i, j int) bool {
		if subscribes[i].Sort == subscribes[j].Sort {
			return subscribes[i].Id > subscribes[j].Id
		}
		return subscribes[i].Sort > subscribes[j].Sort
	})

	if err := m.ExecNoCacheCtx(ctx, func(conn *gorm.DB) error {
		execDB := conn
		if db != nil {
			execDB = db
		}
		for _, item := range subscribes {
			if item == nil {
				continue
			}
			if err := execDB.Delete(item).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return m.CachedConn.DelCacheCtx(ctx, cacheKeys...)
}

var subscribeCacheColumns = []string{"id", "nodes", "node_tags", "sort"}
var subscribeRuntimeColumns = []string{"id", "name", "nodes", "node_tags"}
var subscribeNodeSelectorColumns = []string{"id", "nodes", "node_tags"}
var subscribeOrderActivationColumns = []string{"id", "name", "unit_time", "traffic", "nodes", "node_tags", "renewal_reset"}
var subscribePublicPurchaseColumns = []string{
	"id",
	"name",
	"description",
	"unit_price",
	"unit_time",
	"discount",
	"replacement",
	"inventory",
	"traffic",
	"speed_limit",
	"device_limit",
	"quota",
	"nodes",
	"node_tags",
	"user_visible",
	"sell",
	"allow_renewal",
	"allow_reset_traffic",
	"reset_cycle",
	"renewal_reset",
	"show_original_price",
	"display_template",
	"badge",
	"min_quantity",
	"max_quantity",
	"sale_starts_at",
	"sale_ends_at",
}
var subscribePortalPrePurchaseColumns = []string{
	"id",
	"unit_price",
	"discount",
	"inventory",
	"show",
	"sell",
	"min_quantity",
	"max_quantity",
	"sale_starts_at",
	"sale_ends_at",
}
var subscribeOrderSnapshotColumns = []string{
	"id",
	"name",
	"description",
	"unit_price",
	"unit_time",
	"discount",
	"replacement",
	"traffic",
	"speed_limit",
	"device_limit",
	"quota",
	"nodes",
	"node_tags",
	"show_original_price",
	"reset_cycle",
	"renewal_reset",
	"allow_renewal",
	"allow_reset_traffic",
}
var subscribeSubscriptionExpiryColumns = []string{
	"id",
	"unit_time",
}
var subscribeUpdateDefaultsColumns = []string{
	"id",
	"show",
	"user_visible",
	"sell",
	"allow_deduction",
	"allow_renewal",
	"allow_reset_traffic",
	"renewal_reset",
}
var subscribeAdminListColumns = []string{
	"id",
	"name",
	"language",
	"unit_price",
	"unit_time",
	"replacement",
	"inventory",
	"traffic",
	"speed_limit",
	"device_limit",
	"quota",
	"show",
	"user_visible",
	"sell",
	"allow_renewal",
	"reset_cycle",
	"display_template",
	"badge",
	"min_quantity",
	"max_quantity",
	"sale_starts_at",
	"sale_ends_at",
	"sort",
	"created_at",
	"updated_at",
}
var subscribePublicListColumns = []string{
	"id",
	"name",
	"description",
	"unit_price",
	"unit_time",
	"discount",
	"replacement",
	"inventory",
	"traffic",
	"speed_limit",
	"device_limit",
	"user_visible",
	"sell",
	"show_original_price",
	"display_template",
	"badge",
	"min_quantity",
	"max_quantity",
	"sale_starts_at",
	"sale_ends_at",
}
var subscribeCacheNodeServerColumns = []string{"DISTINCT server_id"}
var subscribeGroupListColumns = []string{"id", "name", "description", "created_at", "updated_at"}

func (m *customSubscribeModel) QueryGroupList(ctx context.Context) ([]*Group, error) {
	list := make([]*Group, 0)
	err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Group{}).
			Select(subscribeGroupListColumns).
			Order("id ASC").
			Find(v).Error
	})
	return list, err
}

func (m *customSubscribeModel) querySubscribeCacheRows(ctx context.Context, ids []int64, dbs ...*gorm.DB) ([]*Subscribe, error) {
	subscribes := make([]*Subscribe, 0, subscribeSeenIDInitialCap(len(ids)))
	query := func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&Subscribe{}).
			Select(subscribeCacheColumns).
			Where("id IN ?", ids).
			Find(v).Error
	}
	if len(dbs) > 0 && dbs[0] != nil {
		if err := query(dbs[0].WithContext(ctx), &subscribes); err != nil {
			return nil, err
		}
		return subscribes, nil
	}
	if err := m.QueryNoCacheCtx(ctx, &subscribes, query); err != nil {
		return nil, err
	}
	return subscribes, nil
}

func (m *customSubscribeModel) cacheKeysForSubscribeIds(ctx context.Context, ids []int64) ([]string, error) {
	ids = normalizeSubscribeIDs(ids)
	if len(ids) == 0 {
		return []string{}, nil
	}
	subscribes, err := m.querySubscribeCacheRows(ctx, ids)
	if err != nil {
		return nil, err
	}
	return m.cacheKeysForSubscribes(ctx, ids, subscribes)
}

func (m *customSubscribeModel) cacheKeysForSubscribes(ctx context.Context, ids []int64, subscribes []*Subscribe, dbs ...*gorm.DB) ([]string, error) {
	cacheKeys := newSubscribeCacheKeyCollector(subscribeBaseCacheKeyCount * (len(ids) + len(subscribes)))

	seenIDs := newSubscribeSeenIDSet(len(subscribes))
	nodeCapHint, tagCapHint := subscribeListSelectorCapHints(subscribes)
	selectors := newSubscribeNodeSelectorSetWithHints(nodeCapHint, tagCapHint)
	for _, data := range subscribes {
		if data == nil || data.Id <= 0 {
			continue
		}
		seenIDs.Add(data.Id)
		addSubscribeBaseCacheKeys(cacheKeys.AddKey, data.Id)
		selectors.AddRaw(data.Nodes, data.NodeTags)
	}
	for _, id := range ids {
		if !seenIDs.Contains(id) {
			addSubscribeBaseCacheKeys(cacheKeys.AddKey, id)
		}
	}

	if selectors.Len() > 0 {
		rows := make([]subscribeCacheNodeServerRow, 0)
		query := func(conn *gorm.DB, v interface{}) error {
			return querySubscribeCacheNodeServerRows(conn, v, selectors.NodeIDs(), selectors.Tags())
		}
		if err := m.querySubscribeCacheRelatedRows(ctx, &rows, query, dbs...); err != nil {
			return nil, err
		}
		addSubscribeServerUserCacheKeys(cacheKeys.AddKey, rows)
	}

	return cacheKeys.Values(), nil
}

type subscribeCacheNodeServerRow struct {
	ServerId int64
}

const (
	subscribeBaseCacheKeyCount        = 4
	subscribeSmallSetLinearDedupLimit = 8
)

type subscribeNodeSelectorSet struct {
	nodeIDs     []int64
	tags        []string
	seenNodeIDs map[int64]struct{}
	seenTags    map[string]struct{}
	nodeCapHint int
	tagCapHint  int
}

func newSubscribeNodeSelectorSet(capHint int) *subscribeNodeSelectorSet {
	return newSubscribeNodeSelectorSetWithHints(capHint, capHint)
}

func newSubscribeNodeSelectorSetWithHints(nodeCapHint, tagCapHint int) *subscribeNodeSelectorSet {
	if nodeCapHint <= 0 && tagCapHint <= 0 {
		return &subscribeNodeSelectorSet{}
	}
	return &subscribeNodeSelectorSet{
		nodeCapHint: nodeCapHint,
		tagCapHint:  tagCapHint,
	}
}

func (s *subscribeNodeSelectorSet) AddRaw(nodes, nodeTags string) {
	if s == nil {
		return
	}
	forEachSubscribeCSVItem(nodes, func(value string) {
		id, err := strconv.ParseInt(value, 10, 64)
		if err == nil && id > 0 {
			s.AddNodeID(id)
		}
	})
	forEachSubscribeCSVItem(nodeTags, s.AddTag)
}

func (s *subscribeNodeSelectorSet) AddNodeID(id int64) {
	if s == nil || id <= 0 {
		return
	}
	if s.seenNodeIDs != nil {
		if _, ok := s.seenNodeIDs[id]; ok {
			return
		}
		s.seenNodeIDs[id] = struct{}{}
		s.nodeIDs = append(s.nodeIDs, id)
		return
	}
	for _, existing := range s.nodeIDs {
		if existing == id {
			return
		}
	}
	s.ensureNodeIDs()
	s.nodeIDs = append(s.nodeIDs, id)
	if len(s.nodeIDs) > subscribeSmallSetLinearDedupLimit {
		s.upgradeNodeIDLookup()
	}
}

func (s *subscribeNodeSelectorSet) AddTag(tag string) {
	if s == nil || tag == "" {
		return
	}
	if s.seenTags != nil {
		if _, ok := s.seenTags[tag]; ok {
			return
		}
		s.seenTags[tag] = struct{}{}
		s.tags = append(s.tags, tag)
		return
	}
	for _, existing := range s.tags {
		if existing == tag {
			return
		}
	}
	s.ensureTags()
	s.tags = append(s.tags, tag)
	if len(s.tags) > subscribeSmallSetLinearDedupLimit {
		s.upgradeTagLookup()
	}
}

func (s *subscribeNodeSelectorSet) ensureNodeIDs() {
	if s.nodeIDs != nil || s.nodeCapHint <= 0 {
		return
	}
	s.nodeIDs = make([]int64, 0, subscribeSelectorInitialCap(s.nodeCapHint))
}

func (s *subscribeNodeSelectorSet) ensureTags() {
	if s.tags != nil || s.tagCapHint <= 0 {
		return
	}
	s.tags = make([]string, 0, subscribeSelectorInitialCap(s.tagCapHint))
}

func (s *subscribeNodeSelectorSet) upgradeNodeIDLookup() {
	if s.seenNodeIDs != nil {
		return
	}
	nextCap := subscribeSetLookupCap(len(s.nodeIDs), s.nodeCapHint)
	if nextCap > cap(s.nodeIDs) {
		nodeIDs := make([]int64, len(s.nodeIDs), nextCap)
		copy(nodeIDs, s.nodeIDs)
		s.nodeIDs = nodeIDs
	}
	s.seenNodeIDs = make(map[int64]struct{}, nextCap)
	for _, existing := range s.nodeIDs {
		s.seenNodeIDs[existing] = struct{}{}
	}
}

func (s *subscribeNodeSelectorSet) upgradeTagLookup() {
	if s.seenTags != nil {
		return
	}
	nextCap := subscribeSetLookupCap(len(s.tags), s.tagCapHint)
	if nextCap > cap(s.tags) {
		tags := make([]string, len(s.tags), nextCap)
		copy(tags, s.tags)
		s.tags = tags
	}
	s.seenTags = make(map[string]struct{}, nextCap)
	for _, existing := range s.tags {
		s.seenTags[existing] = struct{}{}
	}
}

func subscribeSelectorInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > subscribeSmallSetLinearDedupLimit+1 {
		return subscribeSmallSetLinearDedupLimit + 1
	}
	return capHint
}

func subscribeSetLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (s *subscribeNodeSelectorSet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.nodeIDs) + len(s.tags)
}

func (s *subscribeNodeSelectorSet) NodeIDs() []int64 {
	if s == nil {
		return nil
	}
	return s.nodeIDs
}

func (s *subscribeNodeSelectorSet) Tags() []string {
	if s == nil {
		return nil
	}
	return s.tags
}

func NormalizeNodeSelectors(nodes, nodeTags string) ([]int64, []string) {
	selectors := newSubscribeNodeSelectorSetWithHints(
		subscribeCSVItemCapHint(nodes),
		subscribeCSVItemCapHint(nodeTags),
	)
	selectors.AddRaw(nodes, nodeTags)
	return selectors.NodeIDs(), selectors.Tags()
}

func subscribeListSelectorCapHints(subscribes []*Subscribe) (int, int) {
	nodeCapHint := 0
	tagCapHint := 0
	for _, data := range subscribes {
		if data == nil {
			continue
		}
		nodeCapHint += subscribeCSVItemCapHint(data.Nodes)
		tagCapHint += subscribeCSVItemCapHint(data.NodeTags)
	}
	return nodeCapHint, tagCapHint
}

func subscribeCSVItemCapHint(value string) int {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	return strings.Count(value, ",") + 1
}

func ServerUserCacheKeysForSubscribe(ctx context.Context, db *gorm.DB, data *Subscribe) ([]string, error) {
	if db == nil || data == nil {
		return []string{}, nil
	}
	nodeIDs, tags := NormalizeNodeSelectors(data.Nodes, data.NodeTags)
	return ServerUserCacheKeysForNodeSelectors(ctx, db, nodeIDs, tags)
}

func ServerUserCacheKeysForNodeSelectors(ctx context.Context, db *gorm.DB, nodeIDs []int64, tags []string) ([]string, error) {
	if db == nil {
		return []string{}, nil
	}
	nodeIDs = normalizeSubscribeIDs(nodeIDs)
	tags = tool.NormalizeStringSlice(tags)
	if len(nodeIDs) == 0 && len(tags) == 0 {
		return []string{}, nil
	}
	rows := make([]subscribeCacheNodeServerRow, 0)
	query := db
	if ctx != nil {
		query = db.WithContext(ctx)
	}
	if err := querySubscribeCacheNodeServerRows(query, &rows, nodeIDs, tags); err != nil {
		return nil, err
	}
	return subscribeServerUserCacheKeys(rows), nil
}

func querySubscribeCacheNodeServerRows(conn *gorm.DB, v interface{}, nodeIDs []int64, tags []string) error {
	if len(nodeIDs) == 0 && len(tags) == 0 {
		return nil
	}
	return buildSubscribeCacheNodeServerQuery(conn, nodeIDs, tags).Find(v).Error
}

func buildSubscribeCacheNodeServerQuery(conn *gorm.DB, nodeIDs []int64, tags []string) *gorm.DB {
	query := conn.Model(&node.Node{}).Select(subscribeCacheNodeServerColumns)
	switch {
	case len(nodeIDs) > 0 && len(tags) > 0:
		query = query.Where("id IN ? OR string_to_array(COALESCE(tags, ''), ',') && ?", nodeIDs, pq.Array(tags))
	case len(nodeIDs) > 0:
		query = query.Where("id IN ?", nodeIDs)
	case len(tags) > 0:
		query = query.Where("string_to_array(COALESCE(tags, ''), ',') && ?", pq.Array(tags))
	}
	return query
}

func addSubscribeServerUserCacheKeys(addCacheKey func(string), rows []subscribeCacheNodeServerRow) {
	serverIDs := newSubscribeSeenIDSet(len(rows))
	for _, row := range rows {
		if row.ServerId > 0 {
			serverIDs.Add(row.ServerId)
		}
	}
	for _, serverID := range serverIDs.ids {
		addCacheKey(subscribeServerUserListCacheKey(serverID))
	}
}

func subscribeServerUserListCacheKey(serverID int64) string {
	return node.ServerUserListCacheKey + strconv.FormatInt(serverID, 10)
}

func subscribeServerUserCacheKeys(rows []subscribeCacheNodeServerRow) []string {
	if len(rows) == 0 {
		return []string{}
	}
	keys := newSubscribeCacheKeyCollector(len(rows))
	addSubscribeServerUserCacheKeys(keys.AddKey, rows)
	return keys.Values()
}

func forEachSubscribeCSVItem(value string, visit func(string)) {
	for len(value) > 0 {
		part := value
		if index := strings.IndexByte(value, ','); index >= 0 {
			part = value[:index]
			value = value[index+1:]
		} else {
			value = ""
		}
		item := strings.TrimSpace(part)
		if item != "" {
			visit(item)
		}
	}
}

func (m *customSubscribeModel) querySubscribeCacheRelatedRows(ctx context.Context, v interface{}, query func(conn *gorm.DB, v interface{}) error, dbs ...*gorm.DB) error {
	if len(dbs) > 0 && dbs[0] != nil {
		return query(dbs[0].WithContext(ctx), v)
	}
	return m.QueryNoCacheCtx(ctx, v, query)
}

func normalizeSubscribeIDs(ids []int64) []int64 {
	seen := newSubscribeSeenIDSet(len(ids))
	for _, id := range ids {
		seen.Add(id)
	}
	return seen.Values()
}

type subscribeSeenIDSet struct {
	ids     []int64
	index   map[int64]struct{}
	capHint int
}

func newSubscribeSeenIDSet(capHint int) subscribeSeenIDSet {
	return subscribeSeenIDSet{capHint: capHint}
}

func (s *subscribeSeenIDSet) Add(id int64) bool {
	if s == nil || id <= 0 {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[id]; ok {
			return false
		}
		s.index[id] = struct{}{}
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
	if len(s.ids) > subscribeSmallSetLinearDedupLimit {
		s.upgradeLookup()
	}
	return true
}

func (s subscribeSeenIDSet) Contains(id int64) bool {
	if id <= 0 {
		return false
	}
	if s.index != nil {
		_, ok := s.index[id]
		return ok
	}
	for _, existing := range s.ids {
		if existing == id {
			return true
		}
	}
	return false
}

func (s *subscribeSeenIDSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	s.ids = make([]int64, 0, subscribeSeenIDInitialCap(s.capHint))
}

func (s *subscribeSeenIDSet) upgradeLookup() {
	if s.index != nil {
		return
	}
	nextCap := subscribeSetLookupCap(len(s.ids), s.capHint)
	if nextCap > cap(s.ids) {
		ids := make([]int64, len(s.ids), nextCap)
		copy(ids, s.ids)
		s.ids = ids
	}
	s.index = make(map[int64]struct{}, nextCap)
	for _, existing := range s.ids {
		s.index[existing] = struct{}{}
	}
}

func (s subscribeSeenIDSet) Values() []int64 {
	if len(s.ids) == 0 {
		return []int64{}
	}
	return s.ids
}

func subscribeSeenIDInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > subscribeSmallSetLinearDedupLimit+1 {
		return subscribeSmallSetLinearDedupLimit + 1
	}
	return capHint
}

func appendSubscribeSeenID(ids []int64, index map[int64]struct{}, id int64) ([]int64, map[int64]struct{}) {
	set := subscribeSeenIDSet{ids: ids, index: index, capHint: cap(ids)}
	set.Add(id)
	return set.ids, set.index
}

func containsSubscribeSeenID(ids []int64, index map[int64]struct{}, id int64) bool {
	return subscribeSeenIDSet{ids: ids, index: index}.Contains(id)
}

func appendSubscribeBaseCacheKeys(keys []string, id int64) []string {
	addSubscribeBaseCacheKeys(func(key string) {
		keys = append(keys, key)
	}, id)
	return keys
}

func addSubscribeBaseCacheKeys(addCacheKey func(string), id int64) {
	if id <= 0 {
		return
	}
	idText := strconv.FormatInt(id, 10)
	addCacheKey(cacheSubscribeIdPrefix + idText)
	addCacheKey(cacheSubscribeRuntimePrefix + idText)
	addCacheKey(cacheSubscribeNodeSelectorPrefix + idText)
	addCacheKey(cacheSubscribeOrderSnapshotPrefix + idText)
}

func uniqueSubscribeCacheKeys(keys []string) []string {
	collector := newSubscribeCacheKeyCollector(len(keys))
	for _, key := range keys {
		collector.Add(key)
	}
	return collector.Values()
}

type subscribeCacheKeyCollector struct {
	keys    []string
	seen    map[string]struct{}
	capHint int
}

func newSubscribeCacheKeyCollector(capHint int) subscribeCacheKeyCollector {
	return subscribeCacheKeyCollector{capHint: capHint}
}

func (c *subscribeCacheKeyCollector) Add(key string) bool {
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
	if len(c.keys) > subscribeSmallSetLinearDedupLimit {
		c.upgradeLookup()
	}
	return true
}

func (c *subscribeCacheKeyCollector) AddKey(key string) {
	c.Add(key)
}

func (c subscribeCacheKeyCollector) Values() []string {
	if len(c.keys) == 0 {
		return []string{}
	}
	return c.keys
}

func (c *subscribeCacheKeyCollector) ensureKeys() {
	if c.keys != nil {
		return
	}
	c.keys = make([]string, 0, subscribeSeenIDInitialCap(c.capHint))
}

func (c *subscribeCacheKeyCollector) upgradeLookup() {
	if c.seen != nil {
		return
	}
	nextCap := subscribeSetLookupCap(len(c.keys), c.capHint)
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

// FilterList Filter Subscribe List
func (m *customSubscribeModel) FilterList(ctx context.Context, params *FilterParams) (int64, []*Subscribe, error) {
	if params == nil {
		params = &FilterParams{}
	}
	params.Normalize()

	list := make([]*Subscribe, 0)
	var total int64

	// 构建查询函数
	buildQuery := func(conn *gorm.DB, lang string) *gorm.DB {
		query := conn.Model(&Subscribe{})

		if params.Search != "" {
			s := "%" + params.Search + "%"
			query = query.Where("name LIKE ? OR description LIKE ?", s, s)
		}
		if params.Show {
			query = query.Where("show = true")
		}
		if params.UserVisible {
			query = query.Where("user_visible = true")
		}
		if params.Sell {
			query = query.Where("sell = true")
		}

		if len(params.Ids) > 0 {
			query = query.Where("id IN ?", params.Ids)
		}
		if len(params.Node) > 0 {
			query = query.Scopes(InSet("nodes", tool.Int64SliceToStringSlice(params.Node)))
		}

		if len(params.Tags) > 0 {
			query = query.Scopes(InSet("node_tags", params.Tags))
		}
		if lang != "" {
			query = query.Where("language = ?", lang)
		} else if params.DefaultLanguage {
			query = query.Where("language = ''")
		}

		return query
	}

	// 查询数据
	queryFunc := func(lang string) error {
		list = make([]*Subscribe, 0)
		total = 0
		err := m.QueryNoCacheCtx(ctx, &list, func(conn *gorm.DB, v interface{}) error {
			offset := pagination.Offset(params.Page, params.Size)
			if !params.SkipCount {
				countQuery := buildQuery(conn, lang)
				if err := countQuery.Count(&total).Error; err != nil {
					return err
				}
				pageOffset, capacity, ok := pagination.PageWindow(total, params.Page, params.Size)
				if !ok {
					return nil
				}
				offset = pageOffset
				list = make([]*Subscribe, 0, capacity)
			} else {
				list = make([]*Subscribe, 0, params.Size)
			}
			listQuery := buildQuery(conn, lang).Order("sort ASC")
			if columns := subscribeFilterListColumns(params.Projection); len(columns) > 0 {
				listQuery = listQuery.Select(columns)
			}
			return listQuery.
				Limit(params.Size).
				Offset(offset).
				Find(&list).Error
		})
		if err != nil {
			return err
		}
		if params.SkipCount {
			total = int64(len(list))
		}
		return nil
	}

	err := queryFunc(params.Language)
	if err != nil {
		return 0, nil, err
	}

	// fallback 默认语言
	if params.DefaultLanguage && total == 0 {
		err = queryFunc("")
		if err != nil {
			return 0, nil, err
		}
	}

	return total, list, nil
}

func subscribeFilterListColumns(projection FilterListProjection) []string {
	switch projection {
	case FilterListProjectionAdmin:
		return subscribeAdminListColumns
	case FilterListProjectionPublic:
		return subscribePublicListColumns
	default:
		return nil
	}
}

func InSet(field string, values []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(values) == 0 {
			return db
		}

		return db.Where("string_to_array(COALESCE("+field+", ''), ',') && ?", pq.Array(values))
	}
}
