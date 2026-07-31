package subscribeprofile

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/node"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Selection struct {
	ProfileKey string
	Items      selectionItemSet
	ItemNames  selectionItemNameIndex
}

func ResolveSelection(ctx context.Context, svcCtx *svc.ServiceContext, userSub *user.Subscribe, profileKey, presetKey string) (*Selection, error) {
	profileKey = strings.TrimSpace(profileKey)
	presetKey = strings.TrimSpace(presetKey)
	if presetKey == "" && profileKey != "" {
		return &Selection{ProfileKey: profileKey}, nil
	}
	if presetKey != "" && profileKey != "" {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "profile and preset cannot be used together")
	}

	var preset user.SubscribePreset
	query := svcCtx.DB.WithContext(ctx).
		Model(&user.SubscribePreset{}).
		Select(subscribePresetSelectionColumns).
		Where("user_subscribe_id = ? AND user_id = ?", userSub.Id, userSub.UserId).
		Where("enabled = ?", true)
	if presetKey == "" {
		query = query.Where("is_default = ?", true)
	} else {
		query = query.Where("preset_key = ?", presetKey)
	}
	err := query.First(&preset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if presetKey == "" {
				return &Selection{}, nil
			}
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "subscribe preset not found")
		}
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscribe preset error: %v", err)
	}

	selection := &Selection{ProfileKey: strings.TrimSpace(preset.ProfileKey)}
	if preset.Mode == user.SubscribePresetModeProfile {
		if selection.ProfileKey == "" {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "subscribe preset profile is empty")
		}
		return selection, nil
	}

	items, err := ParseItems(preset.Items)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "parse subscribe preset items error: %v", err)
	}
	if len(items) > 0 {
		selection.Items = newSelectionItemSet(len(items))
		itemNamesInitialized := false
		for _, item := range items {
			key, ok := newPresetItemKey(item.NodeId, item.InboundAlias)
			if !ok {
				continue
			}
			selection.Items.Add(key)
			if item.DisplayName != "" {
				if !itemNamesInitialized {
					selection.ItemNames = newSelectionItemNameIndex(len(items))
					itemNamesInitialized = true
				}
				selection.ItemNames.Set(key, item.DisplayName)
			}
		}
	}
	return selection, nil
}

func (s *Selection) DisplayName(nodeID int64, inboundAlias string) string {
	if s == nil {
		return ""
	}
	key, ok := newPresetItemKey(nodeID, inboundAlias)
	if !ok {
		return ""
	}
	name, _ := s.ItemNames.Find(key)
	return name
}

const selectionLinearLimit = 8

type selectionItemSet struct {
	items   []presetItemKey
	index   map[presetItemKey]struct{}
	capHint int
}

func newSelectionItemSet(capHint int) selectionItemSet {
	if capHint <= 0 {
		return selectionItemSet{}
	}
	return selectionItemSet{capHint: capHint}
}

func (s *selectionItemSet) Add(key presetItemKey) bool {
	if !key.Valid() {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[key]; ok {
			return false
		}
		s.index[key] = struct{}{}
		s.ensureItems()
		s.items = append(s.items, key)
		return true
	}
	for _, item := range s.items {
		if item == key {
			return false
		}
	}
	s.ensureItems()
	s.items = append(s.items, key)
	if len(s.items) > selectionLinearLimit {
		nextCap := subscribeProfileLookupCap(len(s.items), s.capHint)
		if nextCap > cap(s.items) {
			items := make([]presetItemKey, len(s.items), nextCap)
			copy(items, s.items)
			s.items = items
		}
		s.index = make(map[presetItemKey]struct{}, nextCap)
		for _, item := range s.items {
			s.index[item] = struct{}{}
		}
	}
	return true
}

func (s *selectionItemSet) ensureItems() {
	if s.items != nil {
		return
	}
	capHint := s.capHint
	if capHint <= 0 {
		capHint = selectionLinearLimit
	} else if capHint > selectionLinearLimit+1 {
		capHint = selectionLinearLimit + 1
	}
	s.items = make([]presetItemKey, 0, capHint)
}

func (s selectionItemSet) Contains(key presetItemKey) bool {
	if !key.Valid() {
		return false
	}
	if s.index != nil {
		_, ok := s.index[key]
		return ok
	}
	for _, item := range s.items {
		if item == key {
			return true
		}
	}
	return false
}

func (s selectionItemSet) Len() int {
	return len(s.items)
}

type selectionItemNameEntry struct {
	key  presetItemKey
	name string
}

type selectionItemNameIndex struct {
	rows    []selectionItemNameEntry
	index   map[presetItemKey]int
	capHint int
}

func newSelectionItemNameIndex(capHint int) selectionItemNameIndex {
	if capHint <= 0 {
		return selectionItemNameIndex{}
	}
	return selectionItemNameIndex{capHint: capHint}
}

func (i *selectionItemNameIndex) Set(key presetItemKey, name string) {
	name = strings.TrimSpace(name)
	if !key.Valid() || name == "" {
		return
	}
	if i.index != nil {
		if pos, ok := i.index[key]; ok {
			i.rows[pos].name = name
			return
		}
		i.index[key] = len(i.rows)
		i.ensureRows()
		i.rows = append(i.rows, selectionItemNameEntry{key: key, name: name})
		return
	}
	for pos := range i.rows {
		if i.rows[pos].key == key {
			i.rows[pos].name = name
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, selectionItemNameEntry{key: key, name: name})
	if len(i.rows) > selectionLinearLimit {
		nextCap := subscribeProfileLookupCap(len(i.rows), i.capHint)
		if nextCap > cap(i.rows) {
			rows := make([]selectionItemNameEntry, len(i.rows), nextCap)
			copy(rows, i.rows)
			i.rows = rows
		}
		i.index = make(map[presetItemKey]int, nextCap)
		for pos, row := range i.rows {
			i.index[row.key] = pos
		}
	}
}

func (i *selectionItemNameIndex) ensureRows() {
	if i.rows != nil {
		return
	}
	capHint := i.capHint
	if capHint <= 0 {
		capHint = selectionLinearLimit
	} else if capHint > selectionLinearLimit+1 {
		capHint = selectionLinearLimit + 1
	}
	i.rows = make([]selectionItemNameEntry, 0, capHint)
}

func (i selectionItemNameIndex) Find(key presetItemKey) (string, bool) {
	if !key.Valid() {
		return "", false
	}
	if i.index != nil {
		pos, ok := i.index[key]
		if !ok {
			return "", false
		}
		return i.rows[pos].name, true
	}
	for _, row := range i.rows {
		if row.key == key {
			return row.name, true
		}
	}
	return "", false
}

func QueryAvailableNodes(ctx context.Context, svcCtx *svc.ServiceContext, userSub *user.Subscribe) ([]*node.Node, error) {
	subDetails, err := getCurrentSubscribeProduct(ctx, svcCtx, userSub)
	if err != nil {
		return nil, err
	}
	return QueryAvailableNodesForProduct(ctx, svcCtx, subDetails)
}

func QueryAvailableNodesForSubscribeDetails(ctx context.Context, svcCtx *svc.ServiceContext, userSub *user.SubscribeDetails) ([]*node.Node, error) {
	return QueryAvailableNodesForSubscribeDetailsWithCache(ctx, svcCtx, userSub, nil)
}

func QueryAvailableNodesForSubscribeDetailsWithCache(ctx context.Context, svcCtx *svc.ServiceContext, userSub *user.SubscribeDetails, cache map[string][]*node.Node) ([]*node.Node, error) {
	if userSub == nil {
		return []*node.Node{}, nil
	}
	product, err := subscribeProductForDetails(ctx, svcCtx, userSub)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return []*node.Node{}, nil
	}
	if cache == nil {
		return QueryAvailableNodesForProduct(ctx, svcCtx, product)
	}
	nodeIDs, tags := modelSubscribe.NormalizeNodeSelectors(product.Nodes, product.NodeTags)
	key := normalizedNodeSelectorCacheKey(nodeIDs, tags)
	if nodes, ok := cache[key]; ok {
		return nodes, nil
	}
	nodes, err := queryNodesByNormalizedSelectors(ctx, svcCtx, nodeIDs, tags)
	if err != nil {
		return nil, err
	}
	cache[key] = nodes
	return nodes, nil
}

// HydrateSubscribeProductsForDetails attaches lightweight product selector data
// to subscription details so callers that render many subscriptions avoid one
// product lookup per row.
func HydrateSubscribeProductsForDetails(ctx context.Context, svcCtx *svc.ServiceContext, items []*user.SubscribeDetails) error {
	if len(items) == 0 {
		return nil
	}
	missingIDs := newSubscribeProfileIDSet(len(items))
	snapshotCache := newSubscribeSnapshotProductCache(len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if hasProductSelection(item.Subscribe) {
			continue
		}
		if snapshotSub, ok := snapshotCache.Product(item.ProductSnapshot); ok {
			item.Subscribe = snapshotSub
			continue
		}
		if item.SubscribeId <= 0 {
			continue
		}
		missingIDs.Add(item.SubscribeId)
	}
	ids := missingIDs.Values()
	if len(ids) == 0 || svcCtx == nil || svcCtx.DB == nil {
		return nil
	}

	var products []modelSubscribe.Subscribe
	if err := svcCtx.DB.WithContext(ctx).
		Model(&modelSubscribe.Subscribe{}).
		Select(subscribeProductSelectorColumns).
		Where("id IN ?", ids).
		Find(&products).Error; err != nil {
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscribe product selectors error: %v", err)
	}
	productIndex := newSubscribeProductSelectorIndex(products)
	for _, item := range items {
		if item == nil || hasProductSelection(item.Subscribe) {
			continue
		}
		if product, ok := productIndex.Find(item.SubscribeId); ok {
			item.Subscribe = product
		}
	}
	return nil
}

type subscribeSnapshotProductCacheEntry struct {
	raw     string
	product *modelSubscribe.Subscribe
	ok      bool
}

type subscribeSnapshotProductCache struct {
	rows    []subscribeSnapshotProductCacheEntry
	index   map[string]int
	capHint int
}

func newSubscribeSnapshotProductCache(capHint int) *subscribeSnapshotProductCache {
	if capHint <= 1 {
		return nil
	}
	return &subscribeSnapshotProductCache{capHint: capHint}
}

func (c *subscribeSnapshotProductCache) Product(raw string) (*modelSubscribe.Subscribe, bool) {
	key := strings.TrimSpace(raw)
	if c == nil {
		return ordersnapshot.SubscribeModel(key)
	}
	if c.index != nil {
		if pos, ok := c.index[key]; ok {
			entry := c.rows[pos]
			return entry.product, entry.ok
		}
		return c.add(key)
	}
	for _, row := range c.rows {
		if row.raw == key {
			return row.product, row.ok
		}
	}
	return c.add(key)
}

func (c *subscribeSnapshotProductCache) add(key string) (*modelSubscribe.Subscribe, bool) {
	product, ok := ordersnapshot.SubscribeModel(key)
	c.ensureRows()
	c.rows = append(c.rows, subscribeSnapshotProductCacheEntry{raw: key, product: product, ok: ok})
	if c.index != nil {
		c.index[key] = len(c.rows) - 1
	} else if len(c.rows) > subscribeProfileLinearLimit {
		nextCap := subscribeProfileLookupCap(len(c.rows), c.capHint)
		if nextCap > cap(c.rows) {
			rows := make([]subscribeSnapshotProductCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		c.index = make(map[string]int, nextCap)
		for pos, row := range c.rows {
			c.index[row.raw] = pos
		}
	}
	return product, ok
}

func (c *subscribeSnapshotProductCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = subscribeProfileLinearLimit
	} else if capHint > subscribeProfileLinearLimit+1 {
		capHint = subscribeProfileLinearLimit + 1
	}
	c.rows = make([]subscribeSnapshotProductCacheEntry, 0, capHint)
}

type subscribeProductSelectorIndex struct {
	rows []modelSubscribe.Subscribe
	byID map[int64]int
}

func newSubscribeProductSelectorIndex(products []modelSubscribe.Subscribe) subscribeProductSelectorIndex {
	index := subscribeProductSelectorIndex{rows: products}
	if len(products) <= subscribeProfileLinearLimit {
		return index
	}
	index.byID = make(map[int64]int, len(products))
	for pos := range products {
		if products[pos].Id <= 0 {
			continue
		}
		if _, ok := index.byID[products[pos].Id]; !ok {
			index.byID[products[pos].Id] = pos
		}
	}
	return index
}

func (i subscribeProductSelectorIndex) Find(id int64) (*modelSubscribe.Subscribe, bool) {
	if i.byID != nil {
		pos, ok := i.byID[id]
		if !ok {
			return nil, false
		}
		return &i.rows[pos], true
	}
	for pos := range i.rows {
		if i.rows[pos].Id == id {
			return &i.rows[pos], true
		}
	}
	return nil, false
}

var subscribeProductSelectorColumns = []string{
	"id",
	"nodes",
	"node_tags",
}

func subscribeProductForDetails(ctx context.Context, svcCtx *svc.ServiceContext, userSub *user.SubscribeDetails) (*modelSubscribe.Subscribe, error) {
	if userSub == nil {
		return nil, nil
	}
	if hasProductSelection(userSub.Subscribe) {
		return userSub.Subscribe, nil
	}
	if snapshotSub, ok := ordersnapshot.SubscribeModel(userSub.ProductSnapshot); ok {
		return snapshotSub, nil
	}
	return getCurrentSubscribeProduct(ctx, svcCtx, userSub.ToSubscribe())
}

func hasProductSelection(product *modelSubscribe.Subscribe) bool {
	return product != nil && (product.Id != 0 || strings.TrimSpace(product.Nodes) != "" || strings.TrimSpace(product.NodeTags) != "")
}

func productNodeSelectorCacheKey(product *modelSubscribe.Subscribe) string {
	if product == nil {
		return ""
	}
	nodeIDs, tags := modelSubscribe.NormalizeNodeSelectors(product.Nodes, product.NodeTags)
	return normalizedNodeSelectorCacheKey(nodeIDs, tags)
}

func normalizedNodeSelectorCacheKey(nodeIDs []int64, tags []string) string {
	sort.Slice(nodeIDs, func(i, j int) bool {
		return nodeIDs[i] < nodeIDs[j]
	})
	sort.Strings(tags)

	var builder strings.Builder
	builder.Grow(normalizedNodeSelectorCacheKeyCapacity(nodeIDs, tags))
	for i, id := range nodeIDs {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.FormatInt(id, 10))
	}
	builder.WriteByte(0)
	for i, tag := range tags {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(tag)
	}
	return builder.String()
}

func normalizedNodeSelectorCacheKeyCapacity(nodeIDs []int64, tags []string) int {
	capHint := 1
	if len(nodeIDs) > 1 {
		capHint += len(nodeIDs) - 1
	}
	for _, id := range nodeIDs {
		capHint += decimalInt64Len(id)
	}
	if len(tags) > 1 {
		capHint += len(tags) - 1
	}
	for _, tag := range tags {
		capHint += len(tag)
	}
	return capHint
}

func decimalInt64Len(value int64) int {
	length := 0
	if value < 0 {
		length = 1
	}
	for {
		length++
		value /= 10
		if value == 0 {
			return length
		}
	}
}

func QueryAvailableNodesForProduct(ctx context.Context, svcCtx *svc.ServiceContext, product *modelSubscribe.Subscribe) ([]*node.Node, error) {
	if product == nil {
		return []*node.Node{}, nil
	}
	nodeIds, tags := modelSubscribe.NormalizeNodeSelectors(product.Nodes, product.NodeTags)
	return queryNodesByNormalizedSelectors(ctx, svcCtx, nodeIds, tags)
}

func queryNodesByNormalizedSelectors(ctx context.Context, svcCtx *svc.ServiceContext, nodeIds []int64, tags []string) ([]*node.Node, error) {
	if len(nodeIds) == 0 && len(tags) == 0 {
		return []*node.Node{}, nil
	}

	var nodes []*node.Node
	query := svcCtx.DB.WithContext(ctx).
		Model(&node.Node{}).
		Select(subscribeProfileNodeColumns).
		Preload("Server", func(db *gorm.DB) *gorm.DB {
			return db.Select(subscribeProfileServerColumns)
		}).
		Where("enabled = ?", true)
	switch {
	case len(nodeIds) > 0 && len(tags) > 0:
		query = query.Where("id IN ? OR string_to_array(COALESCE(tags, ''), ',') && ?", nodeIds, pq.Array(tags))
	case len(nodeIds) > 0:
		query = query.Where("id IN ?", nodeIds)
	case len(tags) > 0:
		query = query.Where("string_to_array(COALESCE(tags, ''), ',') && ?", pq.Array(tags))
	}
	if err := query.Order("sort ASC, id ASC").Find(&nodes).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find server details error: %v", err.Error())
	}
	return nodes, nil
}

func QueryNodesBySelectors(ctx context.Context, svcCtx *svc.ServiceContext, nodeIds []int64, tags []string) ([]*node.Node, error) {
	nodeIds, tags = normalizeQueryNodeSelectors(nodeIds, tags)
	return queryNodesByNormalizedSelectors(ctx, svcCtx, nodeIds, tags)
}

func normalizeQueryNodeSelectors(nodeIds []int64, tags []string) ([]int64, []string) {
	return normalizeSubscribeProfileIDs(nodeIds), tool.NormalizeStringSlice(tags)
}

var subscribeProfileNodeColumns = []string{
	"id",
	"name",
	"tags",
	"port",
	"address",
	"server_id",
	"protocol",
	"inbound_alias",
	"sort",
	"created_at",
}

var subscribeProfileServerColumns = []string{
	"id",
	"name",
	"country",
	"city",
	"address",
}

func containsSelectionItem(selection *Selection, item *node.Node) bool {
	if selection == nil || selection.Items.Len() == 0 {
		return true
	}
	key, ok := newPresetItemKey(item.Id, item.Inbound())
	if !ok {
		return false
	}
	return selection.Items.Contains(key)
}

func FilterNodes(nodes []*node.Node, selection *Selection) []*node.Node {
	if selection == nil || (selection.ProfileKey == "" && selection.Items.Len() == 0) {
		return nodes
	}
	result := viewutil.NewRowsCollector[*node.Node](len(nodes))
	for _, item := range nodes {
		if item == nil {
			continue
		}
		if selection.ProfileKey != "" && item.Inbound() != selection.ProfileKey {
			continue
		}
		if !containsSelectionItem(selection, item) {
			continue
		}
		result.Add(item)
	}
	return result.Rows()
}

func BuildNodeInfoAndProfiles(ctx context.Context, svcCtx *svc.ServiceContext, nodes []*node.Node, uuid string) ([]types.UserSubscribeNodeInfo, []types.UserSubscribeProfile, error) {
	return BuildNodeInfoAndProfilesWithRuntimeCache(ctx, svcCtx, nodes, uuid, nil)
}

func BuildNodeInfoAndProfilesWithCache(ctx context.Context, svcCtx *svc.ServiceContext, nodes []*node.Node, uuid string, bindingCache map[int64][]*node.ServerXrayTemplate) ([]types.UserSubscribeNodeInfo, []types.UserSubscribeProfile, error) {
	var runtimeCache *NodeProfileRuntimeCache
	if bindingCache != nil {
		runtimeCache = &NodeProfileRuntimeCache{bindings: bindingCache}
	}
	return BuildNodeInfoAndProfilesWithRuntimeCache(ctx, svcCtx, nodes, uuid, runtimeCache)
}

type NodeProfileRuntimeCache struct {
	bindings       map[int64][]*node.ServerXrayTemplate
	bindingIndexes nodeSubscriptionBindingIndexCache
}

func NewNodeProfileRuntimeCache(capHint int) *NodeProfileRuntimeCache {
	cache := &NodeProfileRuntimeCache{
		bindingIndexes: newNodeSubscriptionBindingIndexCache(capHint),
	}
	if capHint > 0 {
		cache.bindings = make(map[int64][]*node.ServerXrayTemplate, capHint)
	}
	return cache
}

func (c *NodeProfileRuntimeCache) Len() int {
	if c == nil {
		return 0
	}
	return len(c.bindings)
}

func (c *NodeProfileRuntimeCache) prepare(capHint int) {
	if c == nil || capHint <= c.bindingIndexes.capHint || c.bindingIndexes.rows != nil {
		return
	}
	c.bindingIndexes.capHint = capHint
}

func BuildNodeInfoAndProfilesWithRuntimeCache(ctx context.Context, svcCtx *svc.ServiceContext, nodes []*node.Node, uuid string, runtimeCache *NodeProfileRuntimeCache) ([]types.UserSubscribeNodeInfo, []types.UserSubscribeProfile, error) {
	if len(nodes) == 0 {
		return []types.UserSubscribeNodeInfo{}, []types.UserSubscribeProfile{}, nil
	}
	if runtimeCache == nil {
		runtimeCache = &NodeProfileRuntimeCache{}
	}
	runtimeCache.prepare(len(nodes))
	nodesInfo := viewutil.NewRowsCollector[types.UserSubscribeNodeInfo](len(nodes))
	if runtimeCache.bindings == nil {
		var err error
		runtimeCache.bindings, err = QueryEnabledServerXrayTemplateCache(ctx, svcCtx, nodes)
		if err != nil {
			return nil, nil, err
		}
	} else if err := PopulateEnabledServerXrayTemplateCache(ctx, svcCtx, nodes, runtimeCache.bindings); err != nil {
		return nil, nil, err
	}
	profiles := newUserSubscribeProfileAccumulator(len(nodes))
	tagCache := newNodeTagListCache(len(nodes))
	metaCache := newProfileMetaCache(len(nodes))

	for _, item := range nodes {
		if item == nil || item.Server == nil {
			continue
		}
		binding, cfg, ok := findNodeSubscriptionBinding(ctx, svcCtx, item, runtimeCache.bindings, &runtimeCache.bindingIndexes)
		if !ok {
			continue
		}
		profileKey := item.Inbound()
		profileMeta := metaCache.Meta(binding, cfg, profileKey)

		profiles.Add(profileKey, profileMeta.name, profileMeta.protocol, profileMeta.transport, profileMeta.security, binding.Sort)

		nodesInfo.Add(types.UserSubscribeNodeInfo{
			Id:           item.Id,
			Name:         item.Name,
			Uuid:         uuid,
			Protocol:     profileMeta.protocol,
			InboundAlias: item.Inbound(),
			ProfileKey:   profileKey,
			ProfileName:  profileMeta.name,
			ItemKey:      ItemKey(item.Id, item.Inbound()),
			ServerId:     item.ServerId,
			ServerName:   item.Server.Name,
			Port:         item.Port,
			Address:      item.Address,
			Tags:         tagCache.Tags(item.Tags),
			Country:      item.Server.Country,
			City:         item.Server.City,
			CreatedAt:    timex.UnixOrZero(item.CreatedAt),
		})
	}

	profileRows := profiles.Rows()
	sort.Slice(profileRows, func(i, j int) bool {
		if profileRows[i].Sort == profileRows[j].Sort {
			return profileRows[i].Name < profileRows[j].Name
		}
		return profileRows[i].Sort < profileRows[j].Sort
	})
	return nodesInfo.Rows(), profileRows, nil
}

const subscribeProfileLinearLimit = 8

type subscribeProfileIDSet struct {
	ids     []int64
	seen    map[int64]struct{}
	capHint int
}

func newSubscribeProfileIDSet(capHint int) subscribeProfileIDSet {
	return subscribeProfileIDSet{capHint: capHint}
}

func normalizeSubscribeProfileIDs(ids []int64) []int64 {
	set := newSubscribeProfileIDSet(len(ids))
	for _, id := range ids {
		set.Add(id)
	}
	return set.Values()
}

func (s *subscribeProfileIDSet) Add(id int64) bool {
	if id <= 0 {
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
	if s.ids == nil {
		s.ids = make([]int64, 0, subscribeProfileInitialCap(s.capHint))
	}
	s.ids = append(s.ids, id)
	if len(s.ids) > subscribeProfileLinearLimit {
		s.upgradeLookup()
	}
	return true
}

func (s subscribeProfileIDSet) Values() []int64 {
	if len(s.ids) == 0 {
		return []int64{}
	}
	return s.ids
}

func (s *subscribeProfileIDSet) upgradeLookup() {
	if s.seen != nil {
		return
	}
	nextCap := subscribeProfileLookupCap(len(s.ids), s.capHint)
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

func subscribeProfileInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > subscribeProfileLinearLimit+1 {
		return subscribeProfileLinearLimit + 1
	}
	return capHint
}

func subscribeProfileLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

type userSubscribeProfileAccumulator struct {
	rows    []types.UserSubscribeProfile
	index   map[string]int
	capHint int
}

func newUserSubscribeProfileAccumulator(capHint int) userSubscribeProfileAccumulator {
	if capHint <= 0 {
		return userSubscribeProfileAccumulator{}
	}
	return userSubscribeProfileAccumulator{capHint: capHint}
}

func (a *userSubscribeProfileAccumulator) Add(key, name, protocol, transport, security string, sortValue int) {
	if a.index != nil {
		if pos, ok := a.index[key]; ok {
			a.update(pos, sortValue)
			return
		}
		a.index[key] = len(a.rows)
		a.append(key, name, protocol, transport, security, sortValue)
		return
	}
	for i := range a.rows {
		if a.rows[i].Key == key {
			a.update(i, sortValue)
			return
		}
	}
	a.append(key, name, protocol, transport, security, sortValue)
	if len(a.rows) > subscribeProfileLinearLimit {
		nextCap := subscribeProfileLookupCap(len(a.rows), a.capHint)
		if nextCap > cap(a.rows) {
			rows := make([]types.UserSubscribeProfile, len(a.rows), nextCap)
			copy(rows, a.rows)
			a.rows = rows
		}
		a.index = make(map[string]int, nextCap)
		for i, row := range a.rows {
			a.index[row.Key] = i
		}
	}
}

func (a *userSubscribeProfileAccumulator) append(key, name, protocol, transport, security string, sortValue int) {
	if a.rows == nil {
		capHint := a.capHint
		if capHint <= 0 {
			capHint = subscribeProfileLinearLimit
		} else if capHint > subscribeProfileLinearLimit+1 {
			capHint = subscribeProfileLinearLimit + 1
		}
		a.rows = make([]types.UserSubscribeProfile, 0, capHint)
	}
	a.rows = append(a.rows, types.UserSubscribeProfile{
		Key:       key,
		Name:      name,
		Protocol:  protocol,
		Transport: transport,
		Security:  security,
		NodeCount: 1,
		Sort:      sortValue,
	})
}

func (a *userSubscribeProfileAccumulator) update(pos int, sortValue int) {
	a.rows[pos].NodeCount++
	if sortValue < a.rows[pos].Sort {
		a.rows[pos].Sort = sortValue
	}
}

func (a *userSubscribeProfileAccumulator) Rows() []types.UserSubscribeProfile {
	if len(a.rows) == 0 {
		return []types.UserSubscribeProfile{}
	}
	return a.rows
}

func QueryEnabledServerXrayTemplateCache(ctx context.Context, svcCtx *svc.ServiceContext, nodes []*node.Node) (map[int64][]*node.ServerXrayTemplate, error) {
	missingIDs := missingServerXrayTemplateCacheIDs(nodes, nil)
	if len(missingIDs) == 0 {
		return nil, nil
	}
	cache := make(map[int64][]*node.ServerXrayTemplate, len(missingIDs))
	grouped, err := svcCtx.NodeModel.QueryServerXrayTemplatesByServerIDs(ctx, missingIDs, true)
	if err != nil {
		return nil, err
	}
	for _, id := range missingIDs {
		cache[id] = grouped[id]
	}
	return cache, nil
}

func PopulateEnabledServerXrayTemplateCache(ctx context.Context, svcCtx *svc.ServiceContext, nodes []*node.Node, cache map[int64][]*node.ServerXrayTemplate) error {
	missingIDs := missingServerXrayTemplateCacheIDs(nodes, cache)
	if len(missingIDs) == 0 {
		return nil
	}
	grouped, err := svcCtx.NodeModel.QueryServerXrayTemplatesByServerIDs(ctx, missingIDs, true)
	if err != nil {
		return err
	}
	for _, id := range missingIDs {
		cache[id] = grouped[id]
	}
	return nil
}

func missingServerXrayTemplateCacheIDs(nodes []*node.Node, cache map[int64][]*node.ServerXrayTemplate) []int64 {
	collector := newServerXrayTemplateCacheIDCollector(len(nodes))
	for _, item := range nodes {
		collector.AddNode(item, cache)
	}
	return collector.IDs()
}

type serverXrayTemplateCacheIDCollector struct {
	ids     []int64
	seen    map[int64]struct{}
	capHint int
}

func newServerXrayTemplateCacheIDCollector(capHint int) serverXrayTemplateCacheIDCollector {
	if capHint <= 0 {
		return serverXrayTemplateCacheIDCollector{}
	}
	return serverXrayTemplateCacheIDCollector{capHint: capHint}
}

func (c *serverXrayTemplateCacheIDCollector) AddNode(item *node.Node, cache map[int64][]*node.ServerXrayTemplate) {
	if item == nil || item.ServerId <= 0 {
		return
	}
	if _, cached := cache[item.ServerId]; cached {
		return
	}
	c.Add(item.ServerId)
}

func (c *serverXrayTemplateCacheIDCollector) Add(id int64) {
	if id <= 0 {
		return
	}
	if c.seen != nil {
		if _, ok := c.seen[id]; ok {
			return
		}
		c.seen[id] = struct{}{}
		c.ensureIDs()
		c.ids = append(c.ids, id)
		return
	}
	for _, existing := range c.ids {
		if existing == id {
			return
		}
	}
	c.ensureIDs()
	c.ids = append(c.ids, id)
	if len(c.ids) > subscribeProfileLinearLimit {
		nextCap := subscribeProfileLookupCap(len(c.ids), c.capHint)
		if nextCap > cap(c.ids) {
			ids := make([]int64, len(c.ids), nextCap)
			copy(ids, c.ids)
			c.ids = ids
		}
		c.seen = make(map[int64]struct{}, nextCap)
		for _, existing := range c.ids {
			c.seen[existing] = struct{}{}
		}
	}
}

func (c *serverXrayTemplateCacheIDCollector) ensureIDs() {
	if c.ids != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = subscribeProfileLinearLimit
	} else if capHint > subscribeProfileLinearLimit+1 {
		capHint = subscribeProfileLinearLimit + 1
	}
	c.ids = make([]int64, 0, capHint)
}

func (c serverXrayTemplateCacheIDCollector) IDs() []int64 {
	if len(c.ids) == 0 {
		return []int64{}
	}
	return c.ids
}

var subscribePresetSelectionColumns = []string{
	"mode",
	"profile_key",
	"items",
}

var subscribePresetListColumns = []string{
	"id",
	"user_subscribe_id",
	"name",
	"preset_key",
	"mode",
	"profile_key",
	"items",
	"enabled",
	"is_default",
	"created_at",
	"updated_at",
}

func QueryPresets(ctx context.Context, svcCtx *svc.ServiceContext, userID, userSubscribeID int64) ([]types.UserSubscribePreset, error) {
	grouped, err := QueryPresetsBySubscribeIDs(ctx, svcCtx, userID, []int64{userSubscribeID})
	if err != nil {
		return nil, err
	}
	presets := grouped[userSubscribeID]
	if presets == nil {
		return []types.UserSubscribePreset{}, nil
	}
	return presets, nil
}

func QueryPresetsBySubscribeIDs(ctx context.Context, svcCtx *svc.ServiceContext, userID int64, userSubscribeIDs []int64) (map[int64][]types.UserSubscribePreset, error) {
	userSubscribeIDs = normalizeUserSubscribeIDs(userSubscribeIDs)
	if userID <= 0 || len(userSubscribeIDs) == 0 {
		return map[int64][]types.UserSubscribePreset{}, nil
	}
	presets := make([]user.SubscribePreset, 0)
	if err := svcCtx.DB.WithContext(ctx).
		Model(&user.SubscribePreset{}).
		Select(subscribePresetListColumns).
		Where("user_id = ? AND user_subscribe_id IN ?", userID, userSubscribeIDs).
		Order("user_subscribe_id ASC, sort ASC, id ASC").
		Find(&presets).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscribe presets error: %v", err)
	}
	return groupPresetRowsBySubscribeID(presets, userSubscribeIDs), nil
}

const subscribePresetGroupPrecountThreshold = 8

func groupPresetRowsBySubscribeID(presets []user.SubscribePreset, userSubscribeIDs []int64) map[int64][]types.UserSubscribePreset {
	result := make(map[int64][]types.UserSubscribePreset, len(userSubscribeIDs))
	for _, id := range userSubscribeIDs {
		result[id] = []types.UserSubscribePreset{}
	}
	if len(presets) == 0 {
		return result
	}
	itemsCache := newPresetItemsCache(len(presets))
	if len(presets) <= subscribePresetGroupPrecountThreshold {
		for _, preset := range presets {
			result[preset.UserSubscribeId] = append(result[preset.UserSubscribeId], toTypePresetWithItemsCache(preset, itemsCache))
		}
		return result
	}

	counts := make(map[int64]int, len(userSubscribeIDs))
	for _, preset := range presets {
		counts[preset.UserSubscribeId]++
	}
	for subscribeID, count := range counts {
		result[subscribeID] = make([]types.UserSubscribePreset, 0, count)
	}
	for _, preset := range presets {
		result[preset.UserSubscribeId] = append(result[preset.UserSubscribeId], toTypePresetWithItemsCache(preset, itemsCache))
	}
	return result
}

func normalizeUserSubscribeIDs(ids []int64) []int64 {
	return normalizeSubscribeProfileIDs(ids)
}

func ToTypePreset(preset user.SubscribePreset) types.UserSubscribePreset {
	return toTypePresetWithItemsCache(preset, nil)
}

func toTypePresetWithItemsCache(preset user.SubscribePreset, cache *presetItemsCache) types.UserSubscribePreset {
	items, _ := cache.Parse(preset.Items)
	enabled := true
	if preset.Enabled != nil {
		enabled = *preset.Enabled
	}
	return types.UserSubscribePreset{
		Id:              preset.Id,
		UserSubscribeId: preset.UserSubscribeId,
		Name:            preset.Name,
		PresetKey:       preset.PresetKey,
		Mode:            preset.Mode,
		ProfileKey:      preset.ProfileKey,
		Items:           items,
		Enabled:         enabled,
		IsDefault:       preset.IsDefault,
		CreatedAt:       timex.UnixOrZero(preset.CreatedAt),
		UpdatedAt:       timex.UnixOrZero(preset.UpdatedAt),
	}
}

type presetItemsCacheEntry struct {
	raw   string
	items []types.UserSubscribePresetItem
	err   error
}

type presetItemsCache struct {
	rows    []presetItemsCacheEntry
	index   map[string]int
	capHint int
}

func newPresetItemsCache(capHint int) *presetItemsCache {
	if capHint <= 1 {
		return nil
	}
	return &presetItemsCache{capHint: capHint}
}

func (c *presetItemsCache) Parse(raw string) ([]types.UserSubscribePresetItem, error) {
	key := strings.TrimSpace(raw)
	if c == nil {
		return ParseItems(key)
	}
	if c.index != nil {
		if pos, ok := c.index[key]; ok {
			entry := c.rows[pos]
			return entry.items, entry.err
		}
		return c.add(key)
	}
	for _, row := range c.rows {
		if row.raw == key {
			return row.items, row.err
		}
	}
	return c.add(key)
}

func (c *presetItemsCache) add(key string) ([]types.UserSubscribePresetItem, error) {
	items, err := ParseItems(key)
	c.ensureRows()
	c.rows = append(c.rows, presetItemsCacheEntry{raw: key, items: items, err: err})
	if c.index != nil {
		c.index[key] = len(c.rows) - 1
	} else if len(c.rows) > selectionLinearLimit {
		nextCap := subscribeProfileLookupCap(len(c.rows), c.capHint)
		if nextCap > cap(c.rows) {
			rows := make([]presetItemsCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		c.index = make(map[string]int, nextCap)
		for pos, row := range c.rows {
			c.index[row.raw] = pos
		}
	}
	return items, err
}

func (c *presetItemsCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = selectionLinearLimit
	} else if capHint > selectionLinearLimit+1 {
		capHint = selectionLinearLimit + 1
	}
	c.rows = make([]presetItemsCacheEntry, 0, capHint)
}

func ParseItems(raw string) ([]types.UserSubscribePresetItem, error) {
	if strings.TrimSpace(raw) == "" {
		return []types.UserSubscribePresetItem{}, nil
	}
	var items []types.UserSubscribePresetItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	return normalizeItems(items), nil
}

func MarshalItems(items []types.UserSubscribePresetItem) string {
	cleaned := normalizeItems(items)
	data, _ := json.Marshal(cleaned)
	return string(data)
}

func ValidateSelectedItems(available []*node.Node, selected []types.UserSubscribePresetItem) ([]types.UserSubscribePresetItem, error) {
	return validateSelectedPresetItems(availableNodePresetKeys(available), selected)
}

func ValidateSelectedNodeInfoItems(available []types.UserSubscribeNodeInfo, selected []types.UserSubscribePresetItem) ([]types.UserSubscribePresetItem, error) {
	return validateSelectedPresetItems(availableNodeInfoPresetKeys(available), selected)
}

func availableNodePresetKeys(available []*node.Node) *presetItemKeySet {
	allowed := newPresetItemKeySet(len(available))
	for _, item := range available {
		if item == nil {
			continue
		}
		if key, ok := newPresetItemKey(item.Id, item.Inbound()); ok {
			allowed.Add(key)
		}
	}
	return allowed
}

func availableNodeInfoPresetKeys(available []types.UserSubscribeNodeInfo) *presetItemKeySet {
	allowed := newPresetItemKeySet(len(available))
	for _, item := range available {
		if key, ok := newPresetItemKey(item.Id, item.InboundAlias); ok {
			allowed.Add(key)
		}
	}
	return allowed
}

func validateSelectedPresetItems(allowed *presetItemKeySet, selected []types.UserSubscribePresetItem) ([]types.UserSubscribePresetItem, error) {
	result := viewutil.NewRowsCollector[types.UserSubscribePresetItem](len(selected))
	seen := newPresetItemKeySet(len(selected))
	for _, item := range selected {
		key, ok := newPresetItemKey(item.NodeId, item.InboundAlias)
		if !ok {
			continue
		}
		item.InboundAlias = key.inboundAlias
		item.DisplayName = normalizePresetItemDisplayName(item.DisplayName)
		if item.DisplayName != "" && len([]rune(item.DisplayName)) > MaxPresetItemDisplayNameLength {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "subscription item display name is too long")
		}
		if !allowed.Contains(key) {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "selected subscription item is not available")
		}
		if !seen.Add(key) {
			continue
		}
		result.Add(item)
	}
	rows := result.Rows()
	if len(rows) == 0 {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "selected subscription items cannot be empty")
	}
	return rows, nil
}

func ItemKey(nodeID int64, inboundAlias string) string {
	return strconv.FormatInt(nodeID, 10) + ":" + strings.TrimSpace(inboundAlias)
}

type presetItemKey struct {
	nodeID       int64
	inboundAlias string
}

func (k presetItemKey) Valid() bool {
	return k.nodeID > 0 && k.inboundAlias != ""
}

const presetItemKeySetLinearLimit = 8

type presetItemKeySet struct {
	items   []presetItemKey
	index   map[presetItemKey]struct{}
	capHint int
}

func newPresetItemKeySet(capHint int) *presetItemKeySet {
	if capHint <= 0 {
		return &presetItemKeySet{}
	}
	return &presetItemKeySet{capHint: capHint}
}

func (s *presetItemKeySet) Add(key presetItemKey) bool {
	if s == nil {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[key]; ok {
			return false
		}
		s.index[key] = struct{}{}
		s.ensureItems()
		s.items = append(s.items, key)
		return true
	}
	for _, existing := range s.items {
		if existing == key {
			return false
		}
	}
	s.ensureItems()
	s.items = append(s.items, key)
	if len(s.items) > presetItemKeySetLinearLimit {
		nextCap := subscribeProfileLookupCap(len(s.items), s.capHint)
		if nextCap > cap(s.items) {
			items := make([]presetItemKey, len(s.items), nextCap)
			copy(items, s.items)
			s.items = items
		}
		s.index = make(map[presetItemKey]struct{}, nextCap)
		for _, existing := range s.items {
			s.index[existing] = struct{}{}
		}
	}
	return true
}

func (s *presetItemKeySet) ensureItems() {
	if s.items != nil {
		return
	}
	capHint := s.capHint
	if capHint <= 0 {
		capHint = presetItemKeySetLinearLimit
	} else if capHint > presetItemKeySetLinearLimit+1 {
		capHint = presetItemKeySetLinearLimit + 1
	}
	s.items = make([]presetItemKey, 0, capHint)
}

func (s *presetItemKeySet) Contains(key presetItemKey) bool {
	if s == nil {
		return false
	}
	if s.index != nil {
		_, ok := s.index[key]
		return ok
	}
	for _, existing := range s.items {
		if existing == key {
			return true
		}
	}
	return false
}

func (s *presetItemKeySet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.items)
}

func newPresetItemKey(nodeID int64, inboundAlias string) (presetItemKey, bool) {
	inboundAlias = strings.TrimSpace(inboundAlias)
	if nodeID <= 0 || inboundAlias == "" {
		return presetItemKey{}, false
	}
	return presetItemKey{nodeID: nodeID, inboundAlias: inboundAlias}, true
}

func normalizeItems(items []types.UserSubscribePresetItem) []types.UserSubscribePresetItem {
	cleaned := viewutil.NewRowsCollector[types.UserSubscribePresetItem](len(items))
	seen := newPresetItemKeySet(len(items))
	for _, item := range items {
		key, ok := newPresetItemKey(item.NodeId, item.InboundAlias)
		if !ok {
			continue
		}
		item.InboundAlias = key.inboundAlias
		item.DisplayName = normalizePresetItemDisplayName(item.DisplayName)
		if !seen.Add(key) {
			continue
		}
		cleaned.Add(item)
	}
	rows := cleaned.Rows()
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].NodeId == rows[j].NodeId {
			return rows[i].InboundAlias < rows[j].InboundAlias
		}
		return rows[i].NodeId < rows[j].NodeId
	})
	return rows
}

const MaxPresetItemDisplayNameLength = 100

func normalizePresetItemDisplayName(value string) string {
	return strings.TrimSpace(value)
}

func nodeTagList(value string) []string {
	return tool.StringMergeAndRemoveDuplicates(value)
}

type nodeTagListCacheEntry struct {
	raw  string
	tags []string
}

type nodeTagListCache struct {
	rows    []nodeTagListCacheEntry
	index   map[string]int
	capHint int
}

func newNodeTagListCache(capHint int) *nodeTagListCache {
	if capHint <= 1 {
		return nil
	}
	return &nodeTagListCache{capHint: capHint}
}

func (c *nodeTagListCache) Tags(raw string) []string {
	if c == nil {
		return nodeTagList(raw)
	}
	if c.index != nil {
		if pos, ok := c.index[raw]; ok {
			return c.rows[pos].tags
		}
		return c.add(raw)
	}
	for _, row := range c.rows {
		if row.raw == raw {
			return row.tags
		}
	}
	return c.add(raw)
}

func (c *nodeTagListCache) add(raw string) []string {
	tags := nodeTagList(raw)
	c.ensureRows()
	c.rows = append(c.rows, nodeTagListCacheEntry{raw: raw, tags: tags})
	if c.index != nil {
		c.index[raw] = len(c.rows) - 1
	} else if len(c.rows) > subscribeProfileLinearLimit {
		nextCap := subscribeProfileLookupCap(len(c.rows), c.capHint)
		if nextCap > cap(c.rows) {
			rows := make([]nodeTagListCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		c.index = make(map[string]int, nextCap)
		for pos, row := range c.rows {
			c.index[row.raw] = pos
		}
	}
	return tags
}

func (c *nodeTagListCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = subscribeProfileLinearLimit
	} else if capHint > subscribeProfileLinearLimit+1 {
		capHint = subscribeProfileLinearLimit + 1
	}
	c.rows = make([]nodeTagListCacheEntry, 0, capHint)
}

func getCurrentSubscribeProduct(ctx context.Context, svcCtx *svc.ServiceContext, userSub *user.Subscribe) (*modelSubscribe.Subscribe, error) {
	subDetails, err := svcCtx.SubscribeModel.FindOneForNodeSelection(ctx, userSub.SubscribeId)
	if err == nil {
		return subDetails, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if snapshotSub, ok := ordersnapshot.SubscribeModel(userSub.ProductSnapshot); ok {
			return snapshotSub, nil
		}
	}
	return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find subscribe details error: %v", err.Error())
}

type nodeSubscriptionBindingEntry struct {
	binding *node.ServerXrayTemplate
	config  xraytemplate.ConfigObject
}

type nodeSubscriptionBindingAliasEntry struct {
	alias string
	entry nodeSubscriptionBindingEntry
}

type nodeSubscriptionBindingIndex struct {
	rows      []nodeSubscriptionBindingAliasEntry
	byInbound map[string]int
	capHint   int
}

type nodeSubscriptionBindingIndexCacheEntry struct {
	serverID int64
	index    *nodeSubscriptionBindingIndex
}

type nodeSubscriptionBindingIndexCache struct {
	rows    []nodeSubscriptionBindingIndexCacheEntry
	index   map[int64]int
	capHint int
}

func newNodeSubscriptionBindingIndexCache(capHint int) nodeSubscriptionBindingIndexCache {
	return nodeSubscriptionBindingIndexCache{capHint: capHint}
}

func (c *nodeSubscriptionBindingIndexCache) Get(serverID int64) (*nodeSubscriptionBindingIndex, bool) {
	if c == nil {
		return nil, false
	}
	if c.index != nil {
		pos, ok := c.index[serverID]
		if !ok {
			return nil, false
		}
		return c.rows[pos].index, true
	}
	for _, row := range c.rows {
		if row.serverID == serverID {
			return row.index, true
		}
	}
	return nil, false
}

func (c *nodeSubscriptionBindingIndexCache) Set(serverID int64, bindingIndex *nodeSubscriptionBindingIndex) {
	if c == nil || serverID <= 0 {
		return
	}
	if c.index != nil {
		if pos, ok := c.index[serverID]; ok {
			c.rows[pos].index = bindingIndex
			return
		}
		c.index[serverID] = len(c.rows)
		c.rows = append(c.rows, nodeSubscriptionBindingIndexCacheEntry{serverID: serverID, index: bindingIndex})
		return
	}
	for i, row := range c.rows {
		if row.serverID == serverID {
			c.rows[i].index = bindingIndex
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, nodeSubscriptionBindingIndexCacheEntry{serverID: serverID, index: bindingIndex})
	if len(c.rows) > subscribeProfileLinearLimit {
		c.upgradeLookup()
	}
}

func (c *nodeSubscriptionBindingIndexCache) ensureRows() {
	if c.rows != nil {
		return
	}
	c.rows = make([]nodeSubscriptionBindingIndexCacheEntry, 0, subscribeProfileIndexInitialCap(c.capHint))
}

func (c *nodeSubscriptionBindingIndexCache) upgradeLookup() {
	if c.index != nil {
		return
	}
	nextCap := subscribeProfileLookupCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]nodeSubscriptionBindingIndexCacheEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.index = make(map[int64]int, nextCap)
	for i, row := range c.rows {
		c.index[row.serverID] = i
	}
}

func findNodeSubscriptionBinding(ctx context.Context, svcCtx *svc.ServiceContext, item *node.Node, cache map[int64][]*node.ServerXrayTemplate, indexCache *nodeSubscriptionBindingIndexCache) (*node.ServerXrayTemplate, xraytemplate.ConfigObject, bool) {
	if item == nil || item.Server == nil || item.ServerId <= 0 {
		return nil, nil, false
	}

	bindings, ok := cache[item.ServerId]
	if !ok {
		var err error
		bindings, err = svcCtx.NodeModel.QueryServerXrayTemplates(ctx, item.ServerId, true)
		if err != nil {
			return nil, nil, false
		}
		cache[item.ServerId] = bindings
	}

	var index *nodeSubscriptionBindingIndex
	if indexCache != nil {
		index, _ = indexCache.Get(item.ServerId)
	}
	if index == nil {
		index = buildNodeSubscriptionBindingIndex(item.Server, bindings)
		if indexCache != nil {
			indexCache.Set(item.ServerId, index)
		}
	}
	if index == nil || index.Len() == 0 {
		return nil, nil, false
	}
	entry, ok := index.Find(item.Inbound())
	if !ok {
		return nil, nil, false
	}
	return entry.binding, entry.config, true
}

func buildNodeSubscriptionBindingIndex(server *node.Server, bindings []*node.ServerXrayTemplate) *nodeSubscriptionBindingIndex {
	index := &nodeSubscriptionBindingIndex{}
	if server == nil || len(bindings) == 0 {
		return index
	}

	rendered, err := xraytemplate.RenderBindingConfigList(server, bindings, xraytemplate.RenderBindingConfigOptions{
		TemplateType:            node.XrayTemplateInbound,
		SubscriptionEnabledOnly: true,
	})
	if err != nil {
		return index
	}
	index.capHint = len(rendered)
	for _, item := range rendered {
		if item.Binding == nil || item.Config == nil {
			continue
		}
		entry := nodeSubscriptionBindingEntry{binding: item.Binding, config: item.Config}
		for _, alias := range xraytemplate.BindingInboundAliasesFromConfig(item.Binding, item.Config) {
			if _, exists := index.Find(alias); exists {
				continue
			}
			index.Set(alias, entry)
		}
	}
	return index
}

func (i *nodeSubscriptionBindingIndex) Set(alias string, entry nodeSubscriptionBindingEntry) {
	if i == nil || entry.binding == nil {
		return
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	if i.byInbound != nil {
		if pos, ok := i.byInbound[alias]; ok {
			i.rows[pos].entry = entry
			return
		}
		i.byInbound[alias] = len(i.rows)
		i.rows = append(i.rows, nodeSubscriptionBindingAliasEntry{alias: alias, entry: entry})
		return
	}
	for pos := range i.rows {
		if i.rows[pos].alias == alias {
			i.rows[pos].entry = entry
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, nodeSubscriptionBindingAliasEntry{alias: alias, entry: entry})
	if len(i.rows) > subscribeProfileLinearLimit {
		i.upgradeLookup()
	}
}

func (i *nodeSubscriptionBindingIndex) ensureRows() {
	if i.rows != nil {
		return
	}
	i.rows = make([]nodeSubscriptionBindingAliasEntry, 0, subscribeProfileIndexInitialCap(i.capHint))
}

func (i *nodeSubscriptionBindingIndex) upgradeLookup() {
	if i.byInbound != nil {
		return
	}
	nextCap := subscribeProfileLookupCap(len(i.rows), i.capHint)
	if nextCap > cap(i.rows) {
		rows := make([]nodeSubscriptionBindingAliasEntry, len(i.rows), nextCap)
		copy(rows, i.rows)
		i.rows = rows
	}
	i.byInbound = make(map[string]int, nextCap)
	for pos, row := range i.rows {
		i.byInbound[row.alias] = pos
	}
}

func subscribeProfileIndexInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > subscribeProfileLinearLimit+1 {
		return subscribeProfileLinearLimit + 1
	}
	return capHint
}

func (i *nodeSubscriptionBindingIndex) Find(alias string) (nodeSubscriptionBindingEntry, bool) {
	if i == nil {
		return nodeSubscriptionBindingEntry{}, false
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return nodeSubscriptionBindingEntry{}, false
	}
	if i.byInbound != nil {
		pos, ok := i.byInbound[alias]
		if !ok {
			return nodeSubscriptionBindingEntry{}, false
		}
		return i.rows[pos].entry, true
	}
	for _, row := range i.rows {
		if row.alias == alias {
			return row.entry, true
		}
	}
	return nodeSubscriptionBindingEntry{}, false
}

func (i *nodeSubscriptionBindingIndex) Len() int {
	if i == nil {
		return 0
	}
	return len(i.rows)
}

func profileDisplayName(binding *node.ServerXrayTemplate, fallback string) string {
	if binding != nil && binding.Template != nil && strings.TrimSpace(binding.Template.Name) != "" {
		return strings.TrimSpace(binding.Template.Name)
	}
	if binding != nil && strings.TrimSpace(binding.Alias) != "" {
		return strings.TrimSpace(binding.Alias)
	}
	return fallback
}

type profileMeta struct {
	name      string
	protocol  string
	transport string
	security  string
}

type profileMetaCacheKey struct {
	binding  *node.ServerXrayTemplate
	fallback string
}

type profileMetaCacheEntry struct {
	key  profileMetaCacheKey
	meta profileMeta
}

type profileMetaCache struct {
	rows    []profileMetaCacheEntry
	index   map[profileMetaCacheKey]int
	capHint int
}

func newProfileMetaCache(capHint int) *profileMetaCache {
	if capHint <= 1 {
		return nil
	}
	return &profileMetaCache{capHint: capHint}
}

func (c *profileMetaCache) Meta(binding *node.ServerXrayTemplate, cfg xraytemplate.ConfigObject, fallback string) profileMeta {
	key := profileMetaCacheKey{binding: binding, fallback: fallback}
	if c == nil {
		return profileMetaFromBinding(binding, cfg, fallback)
	}
	if c.index != nil {
		if pos, ok := c.index[key]; ok {
			return c.rows[pos].meta
		}
		return c.add(key, cfg)
	}
	for _, row := range c.rows {
		if row.key == key {
			return row.meta
		}
	}
	return c.add(key, cfg)
}

func (c *profileMetaCache) add(key profileMetaCacheKey, cfg xraytemplate.ConfigObject) profileMeta {
	meta := profileMetaFromBinding(key.binding, cfg, key.fallback)
	c.ensureRows()
	c.rows = append(c.rows, profileMetaCacheEntry{key: key, meta: meta})
	if c.index != nil {
		c.index[key] = len(c.rows) - 1
	} else if len(c.rows) > subscribeProfileLinearLimit {
		nextCap := subscribeProfileLookupCap(len(c.rows), c.capHint)
		if nextCap > cap(c.rows) {
			rows := make([]profileMetaCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		c.index = make(map[profileMetaCacheKey]int, nextCap)
		for pos, row := range c.rows {
			c.index[row.key] = pos
		}
	}
	return meta
}

func (c *profileMetaCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = subscribeProfileLinearLimit
	} else if capHint > subscribeProfileLinearLimit+1 {
		capHint = subscribeProfileLinearLimit + 1
	}
	c.rows = make([]profileMetaCacheEntry, 0, capHint)
}

func profileMetaFromBinding(binding *node.ServerXrayTemplate, cfg xraytemplate.ConfigObject, fallback string) profileMeta {
	protocol, transport, security := profileProtocolMeta(cfg)
	return profileMeta{
		name:      profileDisplayName(binding, fallback),
		protocol:  protocol,
		transport: transport,
		security:  security,
	}
}

func profileProtocolMeta(cfg xraytemplate.ConfigObject) (protocol, transport, security string) {
	if value, ok := cfg["protocol"].(string); ok {
		protocol = value
	}
	if stream, ok := cfg["streamSettings"].(xraytemplate.ConfigObject); ok {
		if value, ok := stream["network"].(string); ok {
			transport = value
		}
		if value, ok := stream["security"].(string); ok {
			security = value
		}
	}
	return
}
