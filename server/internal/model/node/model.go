package node

import (
	"context"
	"strconv"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type customServerLogicModel interface {
	FilterServerList(ctx context.Context, params *FilterParams) (int64, []*Server, error)
	FilterNodeList(ctx context.Context, params *FilterNodeParams) (int64, []*Node, error)
	FilterXrayTemplateList(ctx context.Context, params *FilterXrayTemplateParams) (int64, []*XrayTemplate, error)
	QueryXrayTemplateCatalog(ctx context.Context, params *FilterXrayTemplateParams) (int64, []*XrayTemplate, error)
	ClearNodeCache(ctx context.Context, params *FilterNodeParams) error
	ClearServerCache(ctx context.Context, serverId int64) error
}

const (
	// ServerUserListCacheKey Server User List Cache Key
	ServerUserListCacheKey = "server:user:"

	// ServerConfigCacheKey Server Config Cache Key
	ServerConfigCacheKey = "server:config:"
)

// FilterParams Filter Server Params
type FilterParams struct {
	Page    int
	Size    int
	MaxSize int
	Ids     []int64 // Server IDs
	Search  string
}

type FilterNodeParams struct {
	Page         int      // Page Number
	Size         int      // Page Size
	MaxSize      int      // Internal maximum page size
	NodeId       []int64  // Node IDs
	ServerId     []int64  // Server IDs
	Tag          []string // Tags
	Search       string   // Search Address or Name
	Protocol     string   // Protocol
	InboundAlias string   // Xray inbound alias
	Preload      bool     // Preload Server
	Enabled      *bool    // Enabled
}

type FilterXrayTemplateParams struct {
	Page    int
	Size    int
	MaxSize int
	Type    string
	Search  string
	Enabled *bool
}

type XrayTemplatePreloadMode uint8

const (
	XrayTemplatePreloadNone XrayTemplatePreloadMode = iota
	XrayTemplatePreloadSummary
	XrayTemplatePreloadRuntime
	XrayTemplatePreloadFull
)

type QueryServerXrayTemplateOptions struct {
	EnabledOnly     bool
	TemplatePreload XrayTemplatePreloadMode
}

type nodeCacheInvalidationPlan struct {
	userListServerIDs []int64
	configServerIDs   []int64
}

// FilterServerList Filter Server List
func (m *customServerModel) FilterServerList(ctx context.Context, params *FilterParams) (int64, []*Server, error) {
	servers := make([]*Server, 0)
	var total int64
	if params == nil {
		params = &FilterParams{}
	}
	normalizeServerFilterParams(params)
	countQuery := applyServerFilter(m.WithContext(ctx).Model(&Server{}), params)
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset, capacity, ok := pagination.PageWindow(total, params.Page, params.Size)
	if !ok {
		return total, servers, nil
	}
	servers = make([]*Server, 0, capacity)
	err := applyServerFilter(m.WithContext(ctx).Model(&Server{}), params).
		Select(serverAdminListColumns).
		Order("sort ASC").
		Limit(params.Size).
		Offset(offset).
		Find(&servers).Error
	return total, servers, err
}

func applyServerFilter(query *gorm.DB, params *FilterParams) *gorm.DB {
	if params.Search != "" {
		s := "%" + params.Search + "%"
		query = query.Where("name LIKE ? OR address LIKE ?", s, s)
	}
	if len(params.Ids) > 0 {
		query = query.Where("id IN ?", params.Ids)
	}
	return query
}

func (m *customServerModel) QueryServerList(ctx context.Context, ids []int64) (servers []*Server, err error) {
	ids = normalizeServerIDs(ids)
	if len(ids) == 0 {
		return []*Server{}, nil
	}
	servers = make([]*Server, 0, nodeSmallSetInitialCap(len(ids)))
	if err = m.WithContext(ctx).
		Model(&Server{}).
		Select(serverListColumns).
		Where("id IN ?", ids).
		Find(&servers).Error; err != nil {
		return nil, err
	}
	return orderServersByIDs(servers, ids), nil
}

func (m *customServerModel) QueryServerIPList(ctx context.Context) ([]string, error) {
	ips := make([]string, 0)
	err := m.WithContext(ctx).
		Model(&Server{}).
		Where("address <> ''").
		Order("sort ASC").
		Pluck("address", &ips).Error
	return ips, err
}

func (m *customServerModel) QueryDeployServerList(ctx context.Context) ([]*Server, error) {
	servers := make([]*Server, 0)
	err := m.WithContext(ctx).
		Model(&Server{}).
		Select(serverDeployListColumns).
		Where("address <> ''").
		Order("sort ASC").
		Find(&servers).Error
	return servers, err
}

// FilterNodeList Filter Node List
func (m *customServerModel) FilterNodeList(ctx context.Context, params *FilterNodeParams) (int64, []*Node, error) {
	nodes := make([]*Node, 0)
	var total int64
	if params == nil {
		params = &FilterNodeParams{}
	}
	normalizeNodeFilterParams(params)
	countQuery := applyNodeFilter(m.WithContext(ctx).Model(&Node{}), params)
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset, capacity, ok := pagination.PageWindow(total, params.Page, params.Size)
	if !ok {
		return total, nodes, nil
	}
	nodes = make([]*Node, 0, capacity)
	query := applyNodeFilter(m.WithContext(ctx).Model(&Node{}), params)
	query = query.Select(nodeListColumns)
	if params.Preload {
		query = query.Preload("Server", func(db *gorm.DB) *gorm.DB {
			return db.Select(serverListColumns)
		})
	}
	err := query.Order("sort ASC").
		Limit(params.Size).
		Offset(offset).
		Find(&nodes).Error
	return total, nodes, err
}

func applyNodeFilter(query *gorm.DB, params *FilterNodeParams) *gorm.DB {
	if params.Search != "" {
		s := "%" + params.Search + "%"
		query = query.Where("name LIKE ? OR address LIKE ? OR tags LIKE ? OR port::text LIKE ? ", s, s, s, s)
	}
	if len(params.NodeId) > 0 {
		query = query.Where("id IN ?", params.NodeId)
	}
	if len(params.ServerId) > 0 {
		query = query.Where("server_id IN ?", params.ServerId)
	}
	if len(params.Tag) > 0 {
		query = query.Scopes(InSet("tags", params.Tag))
	}
	if params.Protocol != "" {
		query = query.Where("inbound_alias = ? OR (inbound_alias = '' AND protocol = ?)", params.Protocol, params.Protocol)
	}
	if params.InboundAlias != "" {
		query = query.Where("inbound_alias = ? OR (inbound_alias = '' AND protocol = ?)", params.InboundAlias, params.InboundAlias)
	}

	if params.Enabled != nil {
		query = query.Where("enabled = ?", *params.Enabled)
	}
	return query
}

var serverListColumns = []string{
	"id",
	"name",
	"country",
	"city",
	"address",
	"sort",
	"last_reported_at",
	"created_at",
	"updated_at",
}

var serverAdminListColumns = append(append([]string{}, serverListColumns...),
	"(SELECT COUNT(*) FROM nodes WHERE nodes.server_id = servers.id) AS node_count",
	"(SELECT COUNT(*) FROM server_xray_templates WHERE server_xray_templates.server_id = servers.id) AS template_count",
)

func orderServersByIDs(servers []*Server, ids []int64) []*Server {
	if len(servers) == 0 || len(ids) == 0 {
		return []*Server{}
	}
	ordered := make([]*Server, 0, len(servers))
	if len(servers) <= smallServerOrderLookupLimit {
		for _, id := range ids {
			for _, server := range servers {
				if server != nil && server.Id == id {
					ordered = append(ordered, server)
					break
				}
			}
		}
		return ordered
	}
	byID := make(map[int64]*Server, len(servers))
	for _, server := range servers {
		if server != nil {
			byID[server.Id] = server
		}
	}
	for _, id := range ids {
		if server, ok := byID[id]; ok {
			ordered = append(ordered, server)
		}
	}
	return ordered
}

const smallServerOrderLookupLimit = 8

var serverDeployListColumns = []string{
	"id",
	"name",
	"address",
}

var serverRuntimeColumns = []string{
	"id",
	"name",
	"address",
}

var nodeListColumns = []string{
	"id",
	"name",
	"tags",
	"port",
	"address",
	"server_id",
	"protocol",
	"inbound_alias",
	"enabled",
	"sort",
	"created_at",
	"updated_at",
}

var nodeCacheInvalidationColumns = []string{
	"server_id",
	"protocol",
	"inbound_alias",
}

func (m *customServerModel) FilterXrayTemplateList(ctx context.Context, params *FilterXrayTemplateParams) (int64, []*XrayTemplate, error) {
	templates := make([]*XrayTemplate, 0)
	var total int64
	if params == nil {
		params = &FilterXrayTemplateParams{}
	}
	normalizeXrayTemplateFilterParams(params)
	countQuery := applyXrayTemplateFilter(m.WithContext(ctx).Model(&XrayTemplate{}), params)
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset, capacity, ok := pagination.PageWindow(total, params.Page, params.Size)
	if !ok {
		return total, templates, nil
	}
	templates = make([]*XrayTemplate, 0, capacity)
	err := applyXrayTemplateFilter(m.WithContext(ctx).Model(&XrayTemplate{}), params).
		Select(xrayTemplateListColumns).
		Order("id DESC").
		Limit(params.Size).
		Offset(offset).
		Find(&templates).Error
	return total, templates, err
}

func (m *customServerModel) QueryXrayTemplateCatalog(ctx context.Context, params *FilterXrayTemplateParams) (int64, []*XrayTemplate, error) {
	templates := make([]*XrayTemplate, 0)
	var total int64
	if params == nil {
		params = &FilterXrayTemplateParams{}
	}
	normalizeXrayTemplateFilterParams(params)
	countQuery := applyXrayTemplateFilter(m.WithContext(ctx).Model(&XrayTemplate{}), params)
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	offset, capacity, ok := pagination.PageWindow(total, params.Page, params.Size)
	if !ok {
		return total, templates, nil
	}
	templates = make([]*XrayTemplate, 0, capacity)
	err := applyXrayTemplateFilter(m.WithContext(ctx).Model(&XrayTemplate{}), params).
		Select(xrayTemplateCatalogColumns).
		Order("id DESC").
		Limit(params.Size).
		Offset(offset).
		Find(&templates).Error
	return total, templates, err
}

func applyXrayTemplateFilter(query *gorm.DB, params *FilterXrayTemplateParams) *gorm.DB {
	if params.Search != "" {
		s := "%" + params.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", s, s)
	}
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}
	if params.Enabled != nil {
		query = query.Where("enabled = ?", *params.Enabled)
	}
	return query
}

var xrayTemplateListColumns = []string{
	`"xray_templates"."id"`,
	`"xray_templates"."name"`,
	`"xray_templates"."type"`,
	`"xray_templates"."description"`,
	`"xray_templates"."enabled"`,
	`"xray_templates"."updated_at"`,
	`(
		SELECT COUNT(DISTINCT binding.server_id)
		FROM server_xray_templates AS binding
		WHERE binding.template_id = xray_templates.id
			AND binding.enabled = TRUE
	) AS binding_count`,
	`(
		SELECT COUNT(DISTINCT binding.server_id)
		FROM server_xray_templates AS binding
		WHERE binding.template_id = xray_templates.id
			AND binding.enabled = TRUE
			AND binding.subscription_enabled = TRUE
	) AS subscription_binding_count`,
}

var xrayTemplateCatalogColumns = []string{
	"id",
	"name",
	"type",
	"config",
	"config_template",
	"variables_schema",
	"default_variables",
	"subscription_meta",
	"description",
	"enabled",
}

var serverXrayTemplateRuntimeColumns = []string{
	`"server_xray_templates"."id"`,
	`"server_xray_templates"."server_id"`,
	`"server_xray_templates"."template_id"`,
	`"server_xray_templates"."alias"`,
	`"server_xray_templates"."variables"`,
	`"server_xray_templates"."subscription_enabled"`,
	`"server_xray_templates"."subscription_name"`,
	`"server_xray_templates"."subscription_variables"`,
	`"server_xray_templates"."sort"`,
	`"server_xray_templates"."enabled"`,
}

var xrayTemplateRuntimeColumns = []string{
	"id",
	"name",
	"type",
	"config",
	"config_template",
	"default_variables",
	"subscription_meta",
	"enabled",
}

var xrayTemplateBindingSummaryColumns = []string{
	"id",
	"name",
	"type",
	"config",
	"default_variables",
	"enabled",
}

// ClearNodeCache Clear Node Cache
func (m *customServerModel) ClearNodeCache(ctx context.Context, params *FilterNodeParams) error {
	if params == nil {
		params = &FilterNodeParams{}
	}
	normalizeNodeFilterParams(params)
	nodes := make([]*Node, 0)
	if err := applyNodeFilter(m.WithContext(ctx).Model(&Node{}), params).
		Select(nodeCacheInvalidationColumns).
		Find(&nodes).Error; err != nil {
		return err
	}

	plan := buildNodeCacheInvalidationPlan(nodes)
	cacheKeys := newNodeCacheKeyCollector(len(plan.userListServerIDs) + len(plan.configServerIDs))
	for _, serverID := range plan.userListServerIDs {
		cacheKeys.Add(serverUserListCacheKey(serverID))
	}
	for _, serverID := range plan.configServerIDs {
		keys, err := m.scanServerConfigCacheKeys(ctx, serverID)
		if err != nil {
			return err
		}
		for _, key := range keys {
			cacheKeys.Add(key)
		}
	}
	return m.deleteCacheKeys(ctx, cacheKeys.Values())
}

func buildNodeCacheInvalidationPlan(nodes []*Node) nodeCacheInvalidationPlan {
	userListServerIDs := newNodeServerIDSet(len(nodes))
	configServerIDs := newNodeServerIDSet(len(nodes))
	for _, item := range nodes {
		if item == nil || item.ServerId <= 0 {
			continue
		}
		userListServerIDs.Add(item.ServerId)
		if item.Inbound() == "" {
			continue
		}
		configServerIDs.Add(item.ServerId)
	}
	return nodeCacheInvalidationPlan{
		userListServerIDs: userListServerIDs.Values(),
		configServerIDs:   configServerIDs.Values(),
	}
}

func appendUniqueNodeServerID(ids []int64, id int64, seen map[int64]struct{}) ([]int64, map[int64]struct{}) {
	if id <= 0 {
		return ids, seen
	}
	if seen != nil {
		if _, ok := seen[id]; ok {
			return ids, seen
		}
		seen[id] = struct{}{}
		return append(ids, id), seen
	}
	for _, existing := range ids {
		if existing == id {
			return ids, seen
		}
	}
	ids = append(ids, id)
	if len(ids) > smallNodeServerIDSetThreshold {
		seen = make(map[int64]struct{}, max(cap(ids), len(ids)))
		for _, existing := range ids {
			seen[existing] = struct{}{}
		}
	}
	return ids, seen
}

const smallNodeServerIDSetThreshold = 8

type nodeServerIDSet struct {
	ids     []int64
	seen    map[int64]struct{}
	capHint int
}

func newNodeServerIDSet(capHint int) nodeServerIDSet {
	return nodeServerIDSet{capHint: capHint}
}

func (s *nodeServerIDSet) Add(id int64) bool {
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
	if len(s.ids) > smallNodeServerIDSetThreshold {
		s.upgradeLookup()
	}
	return true
}

func (s nodeServerIDSet) Values() []int64 {
	if len(s.ids) == 0 {
		return []int64{}
	}
	return s.ids
}

func (s *nodeServerIDSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	s.ids = make([]int64, 0, nodeSmallSetInitialCap(s.capHint))
}

func (s *nodeServerIDSet) upgradeLookup() {
	if s.seen != nil {
		return
	}
	nextCap := nodeLookupCap(len(s.ids), s.capHint)
	if nextCap > cap(s.ids) {
		ids := make([]int64, len(s.ids), nextCap)
		copy(ids, s.ids)
		s.ids = ids
	}
	s.seen = make(map[int64]struct{}, nextCap)
	for _, existing := range s.ids {
		s.seen[existing] = struct{}{}
	}
}

func nodeSmallSetInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallNodeServerIDSetThreshold+1 {
		return smallNodeServerIDSetThreshold + 1
	}
	return capHint
}

func nodeLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (m *customServerModel) scanServerConfigCacheKeys(ctx context.Context, serverID int64) ([]string, error) {
	cacheKeys := make([]string, 0, 16)
	var cursor uint64
	pattern := serverConfigCacheScanPattern(serverID)
	for {
		keys, newCursor, err := m.Cache.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		if len(keys) > 0 {
			cacheKeys = append(cacheKeys, keys...)
		}
		cursor = newCursor
		if cursor == 0 {
			break
		}
	}
	return cacheKeys, nil
}

func (m *customServerModel) deleteCacheKeys(ctx context.Context, cacheKeys []string) error {
	cacheKeys = uniqueNodeCacheKeys(cacheKeys)
	if len(cacheKeys) == 0 {
		return nil
	}
	return m.Cache.Del(ctx, cacheKeys...).Err()
}

func uniqueNodeCacheKeys(cacheKeys []string) []string {
	collector := newNodeCacheKeyCollector(len(cacheKeys))
	for _, key := range cacheKeys {
		collector.Add(key)
	}
	return collector.Values()
}

type nodeCacheKeyCollector struct {
	keys    []string
	seen    map[string]struct{}
	capHint int
}

func newNodeCacheKeyCollector(capHint int) nodeCacheKeyCollector {
	return nodeCacheKeyCollector{capHint: capHint}
}

func (c *nodeCacheKeyCollector) Add(key string) bool {
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
	if len(c.keys) > smallNodeServerIDSetThreshold {
		c.upgradeLookup()
	}
	return true
}

func (c nodeCacheKeyCollector) Values() []string {
	if len(c.keys) == 0 {
		return []string{}
	}
	return c.keys
}

func (c *nodeCacheKeyCollector) ensureKeys() {
	if c.keys != nil {
		return
	}
	c.keys = make([]string, 0, nodeSmallSetInitialCap(c.capHint))
}

func (c *nodeCacheKeyCollector) upgradeLookup() {
	if c.seen != nil {
		return
	}
	nextCap := nodeLookupCap(len(c.keys), c.capHint)
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

func normalizeServerFilterParams(params *FilterParams) {
	if params == nil {
		return
	}
	params.Page, params.Size = normalizeModelPage(params.Page, params.Size, params.MaxSize)
	params.Ids = normalizeServerIDs(params.Ids)
}

func normalizeNodeFilterParams(params *FilterNodeParams) {
	if params == nil {
		return
	}
	params.Page, params.Size = normalizeModelPage(params.Page, params.Size, params.MaxSize)
	params.NodeId = normalizeNodePositiveIDs(params.NodeId)
	params.ServerId = normalizeServerIDs(params.ServerId)
	params.Tag = tool.NormalizeStringSlice(params.Tag)
}

func normalizeNodePositiveIDs(ids []int64) []int64 {
	set := newNodeServerIDSet(len(ids))
	for _, id := range ids {
		set.Add(id)
	}
	return set.Values()
}

func normalizeXrayTemplateFilterParams(params *FilterXrayTemplateParams) {
	if params == nil {
		return
	}
	params.Page, params.Size = normalizeModelPage(params.Page, params.Size, params.MaxSize)
}

func normalizeModelPage(page, size, maxSize int) (int, int) {
	if maxSize <= 0 {
		maxSize = pagination.MaxPageSize
	}
	return pagination.NormalizeWithLimit(page, size, 10, maxSize)
}

// ClearServerCache Clear Server Cache
func (m *customServerModel) ClearServerCache(ctx context.Context, serverId int64) error {
	if serverId <= 0 {
		return nil
	}
	cacheKeys := newNodeCacheKeyCollector(1)
	cacheKeys.Add(serverUserListCacheKey(serverId))
	keys, err := m.scanServerConfigCacheKeys(ctx, serverId)
	if err != nil {
		return err
	}
	for _, key := range keys {
		cacheKeys.Add(key)
	}
	return m.deleteCacheKeys(ctx, cacheKeys.Values())
}

func serverUserListCacheKey(serverID int64) string {
	return ServerUserListCacheKey + strconv.FormatInt(serverID, 10)
}

func serverConfigCacheScanPattern(serverID int64) string {
	return ServerConfigCacheKey + strconv.FormatInt(serverID, 10) + "*"
}

// InSet 支持多值 OR 查询
func InSet(field string, values []string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(values) == 0 {
			return db
		}

		return db.Where("string_to_array(COALESCE("+field+", ''), ',') && ?", pq.Array(values))
	}
}
