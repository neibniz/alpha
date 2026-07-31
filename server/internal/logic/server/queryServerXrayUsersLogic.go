package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	usermodel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type QueryServerXrayUsersLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

const (
	smallXrayUniqueThreshold = 8
	xrayUsersCacheTTL        = 15 * time.Second
)

func NewQueryServerXrayUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryServerXrayUsersLogic {
	return &QueryServerXrayUsersLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryServerXrayUsersLogic) QueryServerXrayUsers(req *types.QueryServerXrayUsersRequest) (*types.QueryServerXrayUsersResponse, error) {
	entry, err := l.cachedXrayUsers(req.ServerID, func() ([]types.ServerXrayUser, error) {
		return l.queryServerXrayUsers(req.ServerID)
	})
	if err != nil {
		return nil, err
	}
	return entry.Response(req.Version), nil
}

func (l *QueryServerXrayUsersLogic) queryServerXrayUsers(serverID int64) ([]types.ServerXrayUser, error) {
	serverInfo, err := l.svcCtx.NodeModel.FindOneServerForRuntime(l.ctx, serverID)
	if err != nil {
		return nil, err
	}

	bindings, err := l.svcCtx.NodeModel.QueryServerXrayTemplates(l.ctx, serverInfo.Id, true)
	if err != nil {
		return nil, err
	}
	inboundTagsByAlias, err := resolveInboundTagRefs(serverInfo, bindings)
	if err != nil {
		return nil, err
	}

	var nodes []node.Node
	if err = l.svcCtx.DB.WithContext(l.ctx).
		Model(&node.Node{}).
		Select(xrayUserNodeColumns).
		Where("server_id = ? AND enabled = ?", serverInfo.Id, true).
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	if len(nodes) == 0 || inboundTagsByAlias.Len() == 0 {
		return []types.ServerXrayUser{}, nil
	}
	nodeIDs, nodeTags := collectXrayNodeScopes(nodes)

	products, err := l.queryProductsByNodes(nodeIDs, nodeTags)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return []types.ServerXrayUser{}, nil
	}
	productIndex := indexXrayProducts(products)
	productIDs := productIndex.IDs()
	if len(productIDs) == 0 {
		return []types.ServerXrayUser{}, nil
	}

	subs, err := l.queryActiveSubscriptions(productIDs)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return []types.ServerXrayUser{}, nil
	}
	inboundNodeIndex := newXrayInboundNodeIndexFromRefs(nodes, inboundTagsByAlias)
	userIndex, err := l.queryUsers(collectXrayUserIDs(subs))
	if err != nil {
		return nil, err
	}

	return buildXrayUserRows(subs, xrayUserBuildContext{
		productIndex:     productIndex,
		inboundNodeIndex: inboundNodeIndex,
		userIndex:        userIndex,
		now:              time.Now(),
	}), nil
}

type xrayUsersCacheEntry struct {
	Version string                 `json:"version"`
	Users   []types.ServerXrayUser `json:"users"`
}

func newXrayUsersCacheEntry(users []types.ServerXrayUser) *xrayUsersCacheEntry {
	response := buildXrayUsersResponse("", users)
	return &xrayUsersCacheEntry{Version: response.Version, Users: response.Users}
}

func (e *xrayUsersCacheEntry) Response(clientVersion string) *types.QueryServerXrayUsersResponse {
	if e == nil {
		return buildXrayUsersResponse(clientVersion, nil)
	}
	full := clientVersion == "" || clientVersion != e.Version
	var users []types.ServerXrayUser
	if full {
		users = e.Users
		if users == nil {
			users = []types.ServerXrayUser{}
		}
	}
	return &types.QueryServerXrayUsersResponse{
		Version: e.Version,
		Full:    full,
		Users:   users,
	}
}

func (l *QueryServerXrayUsersLogic) cachedXrayUsers(
	serverID int64,
	load func() ([]types.ServerXrayUser, error),
) (*xrayUsersCacheEntry, error) {
	if entry, ok := l.readXrayUsersCache(serverID); ok {
		return entry, nil
	}
	loadEntry := func() (any, error) {
		if entry, ok := l.readXrayUsersCache(serverID); ok {
			return entry, nil
		}
		users, err := load()
		if err != nil {
			return nil, err
		}
		entry := newXrayUsersCacheEntry(users)
		l.writeXrayUsersCache(serverID, entry)
		return entry, nil
	}

	flight := l.svcCtx.XrayUsersSingleFlight
	if flight == nil {
		value, err := loadEntry()
		if err != nil {
			return nil, err
		}
		return value.(*xrayUsersCacheEntry), nil
	}
	value, err := flight.DoCtx(l.ctx, xrayUsersCacheKey(serverID), loadEntry)
	if err != nil {
		return nil, err
	}
	return value.(*xrayUsersCacheEntry), nil
}

func (l *QueryServerXrayUsersLogic) readXrayUsersCache(serverID int64) (*xrayUsersCacheEntry, bool) {
	if !l.xrayUsersCacheEnabled() {
		return nil, false
	}
	key := xrayUsersCacheKey(serverID)
	sealed, err := l.svcCtx.Redis.Get(l.ctx, key).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			l.Debugf("[QueryServerXrayUsers] read cache failed: %v", err)
		}
		return nil, false
	}
	if !privacycodec.IsSealed(sealed) {
		_ = l.svcCtx.Redis.Del(l.ctx, key).Err()
		return nil, false
	}
	value, err := l.svcCtx.Privacy.Open(xrayUsersCacheScope(serverID), sealed)
	if err != nil {
		_ = l.svcCtx.Redis.Del(l.ctx, key).Err()
		l.Debugf("[QueryServerXrayUsers] decrypt cache failed: %v", err)
		return nil, false
	}
	var entry xrayUsersCacheEntry
	if err := json.Unmarshal([]byte(value), &entry); err != nil || entry.Version == "" {
		_ = l.svcCtx.Redis.Del(l.ctx, key).Err()
		return nil, false
	}
	if entry.Users == nil {
		entry.Users = []types.ServerXrayUser{}
	}
	return &entry, true
}

func (l *QueryServerXrayUsersLogic) writeXrayUsersCache(serverID int64, entry *xrayUsersCacheEntry) {
	if !l.xrayUsersCacheEnabled() || entry == nil {
		return
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	sealed, err := l.svcCtx.Privacy.Seal(xrayUsersCacheScope(serverID), string(data))
	if err != nil || !privacycodec.IsSealed(sealed) {
		if err != nil {
			l.Debugf("[QueryServerXrayUsers] encrypt cache failed: %v", err)
		}
		return
	}
	if err := l.svcCtx.Redis.Set(l.ctx, xrayUsersCacheKey(serverID), sealed, xrayUsersCacheTTL).Err(); err != nil {
		l.Debugf("[QueryServerXrayUsers] write cache failed: %v", err)
	}
}

func (l *QueryServerXrayUsersLogic) xrayUsersCacheEnabled() bool {
	return l != nil && l.svcCtx != nil && l.svcCtx.Redis != nil && l.svcCtx.Privacy != nil && l.svcCtx.Privacy.Enabled()
}

func xrayUsersCacheKey(serverID int64) string {
	return node.ServerUserListCacheKey + strconv.FormatInt(serverID, 10)
}

func xrayUsersCacheScope(serverID int64) string {
	return "server_xray_users:" + strconv.FormatInt(serverID, 10)
}

type xrayUserBuildContext struct {
	productIndex       xrayProductIndex
	inboundNodeIndex   *xrayInboundNodeIndex
	userIndex          xrayUserIndex
	allowedInboundTags xrayAllowedInboundTagCache
	snapshotLimits     xraySnapshotLimitsCache
	now                time.Time
}

func buildXrayUserRows(subs []*usermodel.Subscribe, ctx xrayUserBuildContext) []types.ServerXrayUser {
	if len(subs) == 0 {
		return []types.ServerXrayUser{}
	}
	if ctx.allowedInboundTags.capHint == 0 && len(ctx.allowedInboundTags.rows) == 0 {
		ctx.allowedInboundTags = newXrayAllowedInboundTagCache(ctx.productIndex.Len())
	}
	if ctx.snapshotLimits.capHint == 0 && len(ctx.snapshotLimits.rows) == 0 {
		ctx.snapshotLimits = newXraySnapshotLimitsCache(len(subs))
	}
	collector := viewutil.NewRowsCollector[types.ServerXrayUser](len(subs))
	for _, sub := range subs {
		user, ok := xrayUserFromSubscription(sub, &ctx)
		if !ok {
			continue
		}
		collector.Add(user)
	}
	return collector.Rows()
}

func xrayUserFromSubscription(sub *usermodel.Subscribe, ctx *xrayUserBuildContext) (types.ServerXrayUser, bool) {
	if sub == nil || sub.UUID == "" || ctx == nil {
		return types.ServerXrayUser{}, false
	}
	currentProduct := ctx.productIndex.Get(sub.SubscribeId)
	if currentProduct == nil {
		return types.ServerXrayUser{}, false
	}
	inboundTags, ok := ctx.allowedInboundTags.Get(currentProduct.Id)
	if !ok {
		inboundTags = resolveAllowedInboundTags(currentProduct, ctx.inboundNodeIndex)
		ctx.allowedInboundTags.Set(currentProduct.Id, inboundTags)
	}
	if len(inboundTags) == 0 {
		return types.ServerXrayUser{}, false
	}
	userInfo := ctx.userIndex.Get(sub.UserId)
	if userInfo == nil {
		return types.ServerXrayUser{}, false
	}
	speedLimit, deviceLimit := xrayRuntimeLimitsWithCache(sub, currentProduct, &ctx.snapshotLimits)
	enabled, expiredAt := xrayRuntimeState(sub, userInfo, ctx.now)
	return types.ServerXrayUser{
		UserSubscribeID: sub.Id,
		UserID:          sub.UserId,
		Email:           xrayUserEmail(sub.Id),
		UUID:            sub.UUID,
		Password:        sub.UUID,
		Level:           0,
		Inbounds:        inboundTags,
		SpeedLimit:      speedLimit,
		DeviceLimit:     deviceLimit,
		Enabled:         enabled,
		ExpiredAt:       expiredAt,
	}, true
}

func xrayUserEmail(userSubscribeID int64) string {
	return "u_" + strconv.FormatInt(userSubscribeID, 10)
}

func xrayRuntimeLimits(sub *usermodel.Subscribe, product *subscribe.Subscribe) (int64, int64) {
	return xrayRuntimeLimitsWithCache(sub, product, nil)
}

func xrayRuntimeLimitsWithCache(sub *usermodel.Subscribe, product *subscribe.Subscribe, cache *xraySnapshotLimitsCache) (int64, int64) {
	speedLimit := product.SpeedLimit
	deviceLimit := product.DeviceLimit
	if snapshotSpeedLimit, snapshotDeviceLimit, ok := cache.Limits(sub.ProductSnapshot); ok {
		speedLimit = snapshotSpeedLimit
		deviceLimit = snapshotDeviceLimit
	}
	return speedLimit, deviceLimit
}

type xraySnapshotLimitsCacheEntry struct {
	raw         string
	speedLimit  int64
	deviceLimit int64
	ok          bool
}

type xraySnapshotLimitsCache struct {
	rows    []xraySnapshotLimitsCacheEntry
	byRaw   map[string]int
	capHint int
}

func newXraySnapshotLimitsCache(capHint int) xraySnapshotLimitsCache {
	if capHint <= 1 {
		return xraySnapshotLimitsCache{}
	}
	return xraySnapshotLimitsCache{
		capHint: capHint,
	}
}

func (c *xraySnapshotLimitsCache) Limits(raw string) (int64, int64, bool) {
	key := strings.TrimSpace(raw)
	if c == nil {
		return ordersnapshot.SubscribeLimits(key)
	}
	if c.byRaw != nil {
		row, ok := c.byRaw[key]
		if !ok {
			return c.add(key)
		}
		entry := c.rows[row]
		return entry.speedLimit, entry.deviceLimit, entry.ok
	}
	for _, entry := range c.rows {
		if entry.raw == key {
			return entry.speedLimit, entry.deviceLimit, entry.ok
		}
	}
	return c.add(key)
}

func (c *xraySnapshotLimitsCache) add(key string) (int64, int64, bool) {
	speedLimit, deviceLimit, ok := ordersnapshot.SubscribeLimits(key)
	c.ensureRows()
	c.rows = append(c.rows, xraySnapshotLimitsCacheEntry{
		raw:         key,
		speedLimit:  speedLimit,
		deviceLimit: deviceLimit,
		ok:          ok,
	})
	if c.byRaw != nil {
		c.byRaw[key] = len(c.rows) - 1
	} else if len(c.rows) > smallXrayUniqueThreshold {
		c.upgradeLookup()
	}
	return speedLimit, deviceLimit, ok
}

func (c *xraySnapshotLimitsCache) ensureRows() {
	if c.rows != nil || c.capHint <= 0 {
		return
	}
	c.rows = make([]xraySnapshotLimitsCacheEntry, 0, xrayInitialCap(c.capHint))
}

func (c *xraySnapshotLimitsCache) upgradeLookup() {
	if c.byRaw != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]xraySnapshotLimitsCacheEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.byRaw = make(map[string]int, nextCap)
	for row, entry := range c.rows {
		c.byRaw[entry.raw] = row
	}
}

func xrayRuntimeState(sub *usermodel.Subscribe, userInfo *usermodel.User, now time.Time) (bool, int64) {
	enabled := subscriptionstate.Available(subscriptionstate.StatusActive, sub.ExpireTime, sub.Traffic, sub.Download, sub.Upload, now)
	expiredAt := timex.UnixMilliOrZero(sub.ExpireTime)
	if userInfo.DeletedAt.Valid {
		enabled = false
	}
	if userInfo.Enable != nil && !*userInfo.Enable {
		enabled = false
	}
	return enabled, expiredAt
}

func buildXrayUsersResponse(clientVersion string, users []types.ServerXrayUser) *types.QueryServerXrayUsersResponse {
	if len(users) > 1 {
		sort.Slice(users, func(i, j int) bool {
			return users[i].UserSubscribeID < users[j].UserSubscribeID
		})
	}
	version := usersVersion(users)
	full := clientVersion == "" || clientVersion != version
	if !full {
		users = nil
	} else if users == nil {
		users = []types.ServerXrayUser{}
	}
	return &types.QueryServerXrayUsersResponse{
		Version: version,
		Full:    full,
		Users:   users,
	}
}

func collectXrayNodeScopes(nodes []node.Node) ([]int64, []string) {
	capHint := len(nodes)
	nodeIDs := make([]int64, 0, xrayUniqueInitialCap(capHint))
	nodeTags := make([]string, 0, xrayUniqueInitialCap(capHint))
	var seenIDs map[int64]struct{}
	var seenTags map[string]struct{}
	for _, item := range nodes {
		nodeIDs, seenIDs = appendUniqueXrayInt64(nodeIDs, item.Id, seenIDs, capHint)
		forEachXrayCSVItem(item.Tags, func(tag string) {
			nodeTags, seenTags = appendUniqueXrayString(nodeTags, tag, seenTags, capHint)
		})
	}
	return nodeIDs, nodeTags
}

type xrayProductIndex struct {
	products []*subscribe.Subscribe
	ids      []int64
	byID     map[int64]int
	capHint  int
}

func indexXrayProducts(products []*subscribe.Subscribe) xrayProductIndex {
	index := xrayProductIndex{
		capHint: len(products),
	}
	for _, product := range products {
		index.Add(product)
	}
	return index
}

func (i *xrayProductIndex) Add(product *subscribe.Subscribe) {
	if product == nil || product.Id <= 0 {
		return
	}
	if i.byID != nil {
		if _, ok := i.byID[product.Id]; ok {
			return
		}
		i.byID[product.Id] = len(i.products)
		i.products = append(i.products, product)
		i.ids = append(i.ids, product.Id)
		return
	}
	for _, existing := range i.products {
		if existing.Id == product.Id {
			return
		}
	}
	i.ensureRows()
	i.products = append(i.products, product)
	i.ids = append(i.ids, product.Id)
	if len(i.products) > smallXrayUniqueThreshold {
		i.upgradeLookup()
	}
}

func (i xrayProductIndex) Get(productID int64) *subscribe.Subscribe {
	if productID <= 0 {
		return nil
	}
	if i.byID != nil {
		row, ok := i.byID[productID]
		if !ok {
			return nil
		}
		return i.products[row]
	}
	for _, product := range i.products {
		if product.Id == productID {
			return product
		}
	}
	return nil
}

func (i xrayProductIndex) IDs() []int64 {
	if len(i.ids) == 0 {
		return []int64{}
	}
	return i.ids
}

func (i xrayProductIndex) Len() int {
	return len(i.products)
}

func (i *xrayProductIndex) ensureRows() {
	if i.products != nil || i.capHint <= 0 {
		return
	}
	capacity := xrayInitialCap(i.capHint)
	i.products = make([]*subscribe.Subscribe, 0, capacity)
	i.ids = make([]int64, 0, capacity)
}

func (i *xrayProductIndex) upgradeLookup() {
	if i.byID != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(i.products), i.capHint)
	if nextCap > cap(i.products) {
		products := make([]*subscribe.Subscribe, len(i.products), nextCap)
		copy(products, i.products)
		i.products = products
		ids := make([]int64, len(i.ids), nextCap)
		copy(ids, i.ids)
		i.ids = ids
	}
	i.byID = make(map[int64]int, nextCap)
	for row, existing := range i.products {
		i.byID[existing.Id] = row
	}
}

type xrayAllowedInboundTagCacheEntry struct {
	productID int64
	tags      []string
}

type xrayAllowedInboundTagCache struct {
	rows      []xrayAllowedInboundTagCacheEntry
	byProduct map[int64]int
	capHint   int
}

func newXrayAllowedInboundTagCache(capHint int) xrayAllowedInboundTagCache {
	if capHint <= 0 {
		return xrayAllowedInboundTagCache{}
	}
	return xrayAllowedInboundTagCache{
		capHint: capHint,
	}
}

func (c *xrayAllowedInboundTagCache) Get(productID int64) ([]string, bool) {
	if productID <= 0 {
		return nil, false
	}
	if c.byProduct != nil {
		row, ok := c.byProduct[productID]
		if !ok {
			return nil, false
		}
		return c.rows[row].tags, true
	}
	for _, row := range c.rows {
		if row.productID == productID {
			return row.tags, true
		}
	}
	return nil, false
}

func (c *xrayAllowedInboundTagCache) Set(productID int64, tags []string) {
	if productID <= 0 {
		return
	}
	if c.byProduct != nil {
		if row, ok := c.byProduct[productID]; ok {
			c.rows[row].tags = tags
			return
		}
		c.byProduct[productID] = len(c.rows)
		c.rows = append(c.rows, xrayAllowedInboundTagCacheEntry{productID: productID, tags: tags})
		return
	}
	for row := range c.rows {
		if c.rows[row].productID == productID {
			c.rows[row].tags = tags
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, xrayAllowedInboundTagCacheEntry{productID: productID, tags: tags})
	if len(c.rows) > smallXrayUniqueThreshold {
		c.upgradeLookup()
	}
}

func (c xrayAllowedInboundTagCache) Len() int {
	return len(c.rows)
}

func (c *xrayAllowedInboundTagCache) ensureRows() {
	if c.rows != nil || c.capHint <= 0 {
		return
	}
	c.rows = make([]xrayAllowedInboundTagCacheEntry, 0, xrayInitialCap(c.capHint))
}

func (c *xrayAllowedInboundTagCache) upgradeLookup() {
	if c.byProduct != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]xrayAllowedInboundTagCacheEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.byProduct = make(map[int64]int, nextCap)
	for row, item := range c.rows {
		c.byProduct[item.productID] = row
	}
}

func collectXrayUserIDs(subs []*usermodel.Subscribe) []int64 {
	capHint := len(subs)
	userIDs := make([]int64, 0, xrayUniqueInitialCap(capHint))
	var seenUsers map[int64]struct{}
	for _, sub := range subs {
		if sub == nil || sub.UserId <= 0 {
			continue
		}
		userIDs, seenUsers = appendUniqueXrayInt64(userIDs, sub.UserId, seenUsers, capHint)
	}
	return userIDs
}

func appendUniqueXrayString(values []string, value string, seen map[string]struct{}, capHint int) ([]string, map[string]struct{}) {
	if value == "" {
		return values, seen
	}
	if seen != nil {
		if _, ok := seen[value]; ok {
			return values, seen
		}
		seen[value] = struct{}{}
		return append(values, value), seen
	}
	for _, item := range values {
		if item == value {
			return values, seen
		}
	}
	values = append(values, value)
	if len(values) > smallXrayUniqueThreshold {
		nextCap := xrayUniqueLookupCap(len(values), capHint)
		if nextCap > cap(values) {
			upgraded := make([]string, len(values), nextCap)
			copy(upgraded, values)
			values = upgraded
		}
		seen = make(map[string]struct{}, nextCap)
		for _, item := range values {
			seen[item] = struct{}{}
		}
	}
	return values, seen
}

func appendUniqueXrayInt64(values []int64, value int64, seen map[int64]struct{}, capHint int) ([]int64, map[int64]struct{}) {
	if value <= 0 {
		return values, seen
	}
	if seen != nil {
		if _, ok := seen[value]; ok {
			return values, seen
		}
		seen[value] = struct{}{}
		return append(values, value), seen
	}
	for _, item := range values {
		if item == value {
			return values, seen
		}
	}
	values = append(values, value)
	if len(values) > smallXrayUniqueThreshold {
		nextCap := xrayUniqueLookupCap(len(values), capHint)
		if nextCap > cap(values) {
			upgraded := make([]int64, len(values), nextCap)
			copy(upgraded, values)
			values = upgraded
		}
		seen = make(map[int64]struct{}, nextCap)
		for _, item := range values {
			seen[item] = struct{}{}
		}
	}
	return values, seen
}

func xrayUniqueInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallXrayUniqueThreshold+1 {
		return smallXrayUniqueThreshold + 1
	}
	return capHint
}

func xrayUniqueLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (l *QueryServerXrayUsersLogic) queryProductsByNodes(nodeIDs []int64, nodeTags []string) ([]*subscribe.Subscribe, error) {
	if len(nodeIDs) == 0 && len(nodeTags) == 0 {
		return []*subscribe.Subscribe{}, nil
	}
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if len(nodeIDs) > 0 {
		values := make([]string, 0, len(nodeIDs))
		for _, id := range nodeIDs {
			values = append(values, strconv.FormatInt(id, 10))
		}
		clauses = append(clauses, "string_to_array(COALESCE(nodes, ''), ',') && ?")
		args = append(args, pq.Array(values))
	}
	if len(nodeTags) > 0 {
		clauses = append(clauses, "string_to_array(COALESCE(node_tags, ''), ',') && ?")
		args = append(args, pq.Array(nodeTags))
	}
	products := make([]*subscribe.Subscribe, 0)
	err := l.svcCtx.DB.WithContext(l.ctx).
		Model(&subscribe.Subscribe{}).
		Select(xrayUserProductColumns).
		Where("("+strings.Join(clauses, " OR ")+")", args...).
		Find(&products).Error
	return products, err
}

func (l *QueryServerXrayUsersLogic) queryActiveSubscriptions(productIDs []int64) ([]*usermodel.Subscribe, error) {
	if len(productIDs) == 0 {
		return []*usermodel.Subscribe{}, nil
	}
	subs := make([]*usermodel.Subscribe, 0)
	err := l.svcCtx.DB.WithContext(l.ctx).
		Model(&usermodel.Subscribe{}).
		Select(xrayUserSubscriptionColumns).
		Where("status = ?", 1).
		Where("subscribe_id IN ?", productIDs).
		Find(&subs).Error
	return subs, err
}

func (l *QueryServerXrayUsersLogic) queryUsers(userIDs []int64) (xrayUserIndex, error) {
	if len(userIDs) == 0 {
		return xrayUserIndex{}, nil
	}
	var users []*usermodel.User
	err := l.svcCtx.DB.WithContext(l.ctx).Unscoped().
		Model(&usermodel.User{}).
		Select(xrayUserColumns).
		Where("id IN ?", userIDs).
		Find(&users).Error
	if err != nil {
		return xrayUserIndex{}, err
	}
	return indexXrayUsers(users), nil
}

type xrayUserIndex struct {
	users   []*usermodel.User
	byID    map[int64]int
	capHint int
}

func indexXrayUsers(users []*usermodel.User) xrayUserIndex {
	index := xrayUserIndex{
		capHint: len(users),
	}
	for _, item := range users {
		index.Add(item)
	}
	return index
}

func (i *xrayUserIndex) Add(user *usermodel.User) {
	if user == nil || user.Id <= 0 {
		return
	}
	if i.byID != nil {
		if _, ok := i.byID[user.Id]; ok {
			return
		}
		i.byID[user.Id] = len(i.users)
		i.users = append(i.users, user)
		return
	}
	for _, existing := range i.users {
		if existing.Id == user.Id {
			return
		}
	}
	i.ensureRows()
	i.users = append(i.users, user)
	if len(i.users) > smallXrayUniqueThreshold {
		i.upgradeLookup()
	}
}

func (i xrayUserIndex) Get(userID int64) *usermodel.User {
	if userID <= 0 {
		return nil
	}
	if i.byID != nil {
		row, ok := i.byID[userID]
		if !ok {
			return nil
		}
		return i.users[row]
	}
	for _, user := range i.users {
		if user.Id == userID {
			return user
		}
	}
	return nil
}

func (i xrayUserIndex) Len() int {
	return len(i.users)
}

func (i *xrayUserIndex) ensureRows() {
	if i.users != nil || i.capHint <= 0 {
		return
	}
	i.users = make([]*usermodel.User, 0, xrayInitialCap(i.capHint))
}

func (i *xrayUserIndex) upgradeLookup() {
	if i.byID != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(i.users), i.capHint)
	if nextCap > cap(i.users) {
		users := make([]*usermodel.User, len(i.users), nextCap)
		copy(users, i.users)
		i.users = users
	}
	i.byID = make(map[int64]int, nextCap)
	for row, existing := range i.users {
		i.byID[existing.Id] = row
	}
}

var xrayUserNodeColumns = []string{
	"id",
	"tags",
	"protocol",
	"inbound_alias",
}

var xrayUserProductColumns = []string{
	"id",
	"nodes",
	"node_tags",
	"speed_limit",
	"device_limit",
}

var xrayUserSubscriptionColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_snapshot",
	"uuid",
	"expire_time",
	"traffic",
	"upload",
	"download",
}

var xrayUserColumns = []string{
	"id",
	"enable",
	"deleted_at",
}

type inboundTagRef struct {
	Tag  string
	Sort int
}

type inboundTagKey struct {
	NodeTag    string
	InboundTag string
}

func resolveInboundTagRefs(srv *node.Server, bindings []*node.ServerXrayTemplate) (*inboundTagRefIndex, error) {
	rendered, err := xraytemplate.RenderBindingConfigList(srv, bindings, xraytemplate.RenderBindingConfigOptions{
		TemplateType:            node.XrayTemplateInbound,
		SubscriptionEnabledOnly: true,
	})
	if err != nil {
		return nil, err
	}
	result := newInboundTagRefIndexWithHint(len(rendered) * 2)
	for _, item := range rendered {
		binding := item.Binding
		cfg := item.Config
		tag := strings.TrimSpace(configString(cfg, "tag"))
		if tag == "" {
			tag = strings.TrimSpace(binding.Alias)
		}
		if tag == "" {
			tag = strings.TrimSpace(binding.Template.Name)
		}
		if tag == "" {
			continue
		}
		ref := inboundTagRef{Tag: tag, Sort: binding.Sort}
		addAlias := func(alias string) {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				return
			}
			result.Set(alias, ref)
		}
		for _, alias := range xraytemplate.BindingInboundAliases(binding) {
			addAlias(alias)
		}
		addAlias(tag)
	}
	return result, nil
}

type inboundTagAliasRef struct {
	alias string
	ref   inboundTagRef
}

type inboundTagRefIndex struct {
	rows    []inboundTagAliasRef
	byAlias map[string]int
	capHint int
}

func newInboundTagRefIndex() *inboundTagRefIndex {
	return newInboundTagRefIndexWithHint(0)
}

func newInboundTagRefIndexWithHint(capHint int) *inboundTagRefIndex {
	if capHint <= 0 {
		return &inboundTagRefIndex{}
	}
	return &inboundTagRefIndex{
		capHint: capHint,
	}
}

func newInboundTagRefIndexFromMap(values map[string]inboundTagRef) *inboundTagRefIndex {
	index := newInboundTagRefIndexWithHint(len(values))
	for alias, ref := range values {
		index.Set(alias, ref)
	}
	return index
}

func (i *inboundTagRefIndex) Set(alias string, ref inboundTagRef) {
	if i == nil || ref.Tag == "" {
		return
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	if i.byAlias != nil {
		if row, ok := i.byAlias[alias]; ok {
			i.rows[row].ref = ref
			return
		}
		i.byAlias[alias] = len(i.rows)
		i.rows = append(i.rows, inboundTagAliasRef{alias: alias, ref: ref})
		return
	}
	for row := range i.rows {
		if i.rows[row].alias == alias {
			i.rows[row].ref = ref
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, inboundTagAliasRef{alias: alias, ref: ref})
	if len(i.rows) > smallXrayUniqueThreshold {
		i.upgradeLookup()
	}
}

func (i *inboundTagRefIndex) Find(alias string) (inboundTagRef, bool) {
	if i == nil {
		return inboundTagRef{}, false
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return inboundTagRef{}, false
	}
	if i.byAlias != nil {
		row, ok := i.byAlias[alias]
		if !ok {
			return inboundTagRef{}, false
		}
		return i.rows[row].ref, true
	}
	for _, item := range i.rows {
		if item.alias == alias {
			return item.ref, true
		}
	}
	return inboundTagRef{}, false
}

func (i *inboundTagRefIndex) Len() int {
	if i == nil {
		return 0
	}
	return len(i.rows)
}

func (i *inboundTagRefIndex) ensureRows() {
	if i.rows != nil || i.capHint <= 0 {
		return
	}
	i.rows = make([]inboundTagAliasRef, 0, xrayInitialCap(i.capHint))
}

func (i *inboundTagRefIndex) upgradeLookup() {
	if i.byAlias != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(i.rows), i.capHint)
	if nextCap > cap(i.rows) {
		rows := make([]inboundTagAliasRef, len(i.rows), nextCap)
		copy(rows, i.rows)
		i.rows = rows
	}
	i.byAlias = make(map[string]int, nextCap)
	for row, item := range i.rows {
		i.byAlias[item.alias] = row
	}
}

type xrayInboundNodeIndex struct {
	idRows     []xrayInboundNodeRef
	tagRows    []xrayInboundTagRefs
	byID       map[int64]int
	byTag      map[string]int
	capHintID  int
	capHintTag int
}

type xrayInboundNodeRef struct {
	nodeID int64
	ref    inboundTagRef
}

type xrayInboundTagRefs struct {
	tag  string
	refs []inboundTagRef
}

func newXrayInboundNodeIndex(nodes []node.Node, inboundTagsByAlias map[string]inboundTagRef) *xrayInboundNodeIndex {
	return newXrayInboundNodeIndexFromRefs(nodes, newInboundTagRefIndexFromMap(inboundTagsByAlias))
}

func newXrayInboundNodeIndexWithHint(capHint int) *xrayInboundNodeIndex {
	if capHint <= 0 {
		return &xrayInboundNodeIndex{}
	}
	return &xrayInboundNodeIndex{
		capHintID:  capHint,
		capHintTag: capHint,
	}
}

func newXrayInboundNodeIndexFromRefs(nodes []node.Node, inboundTagsByAlias *inboundTagRefIndex) *xrayInboundNodeIndex {
	if len(nodes) == 0 || inboundTagsByAlias.Len() == 0 {
		return &xrayInboundNodeIndex{}
	}
	index := newXrayInboundNodeIndexWithHint(len(nodes))
	seenTagRefs := newInboundTagKeySet(len(nodes))
	for _, item := range nodes {
		ref, ok := inboundTagsByAlias.Find(item.Inbound())
		if !ok || ref.Tag == "" {
			continue
		}
		index.SetNodeRef(item.Id, ref)
		forEachXrayCSVItem(item.Tags, func(tag string) {
			key := inboundTagKey{NodeTag: tag, InboundTag: ref.Tag}
			if !seenTagRefs.Add(key) {
				return
			}
			index.AddTagRef(tag, ref)
		})
	}
	return index
}

func (i *xrayInboundNodeIndex) Empty() bool {
	return i == nil || (len(i.idRows) == 0 && len(i.tagRows) == 0)
}

func (i *xrayInboundNodeIndex) SetNodeRef(nodeID int64, ref inboundTagRef) {
	if i == nil || nodeID <= 0 || ref.Tag == "" {
		return
	}
	if i.byID != nil {
		if row, ok := i.byID[nodeID]; ok {
			i.idRows[row].ref = ref
			return
		}
		i.byID[nodeID] = len(i.idRows)
		i.idRows = append(i.idRows, xrayInboundNodeRef{nodeID: nodeID, ref: ref})
		return
	}
	for row := range i.idRows {
		if i.idRows[row].nodeID == nodeID {
			i.idRows[row].ref = ref
			return
		}
	}
	i.ensureIDRows()
	i.idRows = append(i.idRows, xrayInboundNodeRef{nodeID: nodeID, ref: ref})
	if len(i.idRows) > smallXrayUniqueThreshold {
		i.upgradeIDLookup()
	}
}

func (i *xrayInboundNodeIndex) AddTagRef(tag string, ref inboundTagRef) {
	tag = strings.TrimSpace(tag)
	if i == nil || tag == "" || ref.Tag == "" {
		return
	}
	if i.byTag != nil {
		if row, ok := i.byTag[tag]; ok {
			i.tagRows[row].refs = appendXrayInboundTagRef(i.tagRows[row].refs, ref)
			return
		}
		i.byTag[tag] = len(i.tagRows)
		i.tagRows = append(i.tagRows, xrayInboundTagRefs{tag: tag, refs: []inboundTagRef{ref}})
		return
	}
	for row := range i.tagRows {
		if i.tagRows[row].tag == tag {
			i.tagRows[row].refs = appendXrayInboundTagRef(i.tagRows[row].refs, ref)
			return
		}
	}
	i.ensureTagRows()
	i.tagRows = append(i.tagRows, xrayInboundTagRefs{tag: tag, refs: []inboundTagRef{ref}})
	if len(i.tagRows) > smallXrayUniqueThreshold {
		i.upgradeTagLookup()
	}
}

func (i *xrayInboundNodeIndex) ensureIDRows() {
	if i.idRows != nil || i.capHintID <= 0 {
		return
	}
	i.idRows = make([]xrayInboundNodeRef, 0, xrayInitialCap(i.capHintID))
}

func (i *xrayInboundNodeIndex) ensureTagRows() {
	if i.tagRows != nil || i.capHintTag <= 0 {
		return
	}
	i.tagRows = make([]xrayInboundTagRefs, 0, xrayInitialCap(i.capHintTag))
}

func (i *xrayInboundNodeIndex) upgradeIDLookup() {
	if i.byID != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(i.idRows), i.capHintID)
	if nextCap > cap(i.idRows) {
		rows := make([]xrayInboundNodeRef, len(i.idRows), nextCap)
		copy(rows, i.idRows)
		i.idRows = rows
	}
	i.byID = make(map[int64]int, nextCap)
	for row, item := range i.idRows {
		i.byID[item.nodeID] = row
	}
}

func (i *xrayInboundNodeIndex) upgradeTagLookup() {
	if i.byTag != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(i.tagRows), i.capHintTag)
	if nextCap > cap(i.tagRows) {
		rows := make([]xrayInboundTagRefs, len(i.tagRows), nextCap)
		copy(rows, i.tagRows)
		i.tagRows = rows
	}
	i.byTag = make(map[string]int, nextCap)
	for row, item := range i.tagRows {
		i.byTag[item.tag] = row
	}
}

func appendXrayInboundTagRef(refs []inboundTagRef, ref inboundTagRef) []inboundTagRef {
	for row := range refs {
		if refs[row].Tag == ref.Tag {
			refs[row] = ref
			return refs
		}
	}
	return append(refs, ref)
}

func (i *xrayInboundNodeIndex) RefByID(nodeID int64) (inboundTagRef, bool) {
	if i == nil || nodeID <= 0 {
		return inboundTagRef{}, false
	}
	if i.byID != nil {
		row, ok := i.byID[nodeID]
		if !ok {
			return inboundTagRef{}, false
		}
		return i.idRows[row].ref, true
	}
	for _, item := range i.idRows {
		if item.nodeID == nodeID {
			return item.ref, true
		}
	}
	return inboundTagRef{}, false
}

func (i *xrayInboundNodeIndex) RefsByTag(tag string) []inboundTagRef {
	tag = strings.TrimSpace(tag)
	if i == nil || tag == "" {
		return nil
	}
	if i.byTag != nil {
		row, ok := i.byTag[tag]
		if !ok {
			return nil
		}
		return i.tagRows[row].refs
	}
	for _, item := range i.tagRows {
		if item.tag == tag {
			return item.refs
		}
	}
	return nil
}

type inboundTagKeySet struct {
	items   []inboundTagKey
	index   map[inboundTagKey]struct{}
	capHint int
}

func newInboundTagKeySet(capHint int) inboundTagKeySet {
	if capHint <= 0 {
		return inboundTagKeySet{}
	}
	return inboundTagKeySet{
		capHint: capHint,
	}
}

func (s *inboundTagKeySet) Add(key inboundTagKey) bool {
	if s == nil || key.NodeTag == "" || key.InboundTag == "" {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[key]; ok {
			return false
		}
		s.index[key] = struct{}{}
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
	if len(s.items) > smallXrayUniqueThreshold {
		s.upgradeLookup()
	}
	return true
}

func (s *inboundTagKeySet) ensureItems() {
	if s.items != nil || s.capHint <= 0 {
		return
	}
	s.items = make([]inboundTagKey, 0, xrayInitialCap(s.capHint))
}

func (s *inboundTagKeySet) upgradeLookup() {
	if s.index != nil {
		return
	}
	nextCap := xrayUniqueLookupCap(len(s.items), s.capHint)
	if nextCap > cap(s.items) {
		items := make([]inboundTagKey, len(s.items), nextCap)
		copy(items, s.items)
		s.items = items
	}
	s.index = make(map[inboundTagKey]struct{}, nextCap)
	for _, existing := range s.items {
		s.index[existing] = struct{}{}
	}
}

func resolveAllowedInboundTags(product *subscribe.Subscribe, index *xrayInboundNodeIndex) []string {
	if product == nil || index.Empty() {
		return nil
	}
	nodeIDs, nodeTags := subscribe.NormalizeNodeSelectors(product.Nodes, product.NodeTags)
	expectedRefs := len(nodeIDs) + len(nodeTags)
	if expectedRefs == 0 {
		return nil
	}
	var seenTags map[string]struct{}
	var refs []inboundTagRef
	addRef := func(ref inboundTagRef) {
		refs, seenTags = appendUniqueInboundRef(refs, ref, seenTags, expectedRefs)
	}
	for _, nodeID := range nodeIDs {
		if ref, ok := index.RefByID(nodeID); ok {
			addRef(ref)
		}
	}
	for _, tag := range nodeTags {
		for _, ref := range index.RefsByTag(tag) {
			addRef(ref)
		}
	}
	if len(refs) == 0 {
		return nil
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Sort == refs[j].Sort {
			return refs[i].Tag < refs[j].Tag
		}
		return refs[i].Sort < refs[j].Sort
	})
	tags := make([]string, 0, len(refs))
	for _, ref := range refs {
		tags = append(tags, ref.Tag)
	}
	return tags
}

func appendUniqueInboundRef(refs []inboundTagRef, ref inboundTagRef, seen map[string]struct{}, capHint int) ([]inboundTagRef, map[string]struct{}) {
	if ref.Tag == "" {
		return refs, seen
	}
	if seen != nil {
		if _, ok := seen[ref.Tag]; ok {
			return refs, seen
		}
		seen[ref.Tag] = struct{}{}
		return append(refs, ref), seen
	}
	for _, item := range refs {
		if item.Tag == ref.Tag {
			return refs, seen
		}
	}
	if refs == nil {
		refs = make([]inboundTagRef, 0, xrayUniqueInitialCap(capHint))
	}
	refs = append(refs, ref)
	if len(refs) > smallXrayUniqueThreshold {
		nextCap := xrayUniqueLookupCap(len(refs), capHint)
		if nextCap > cap(refs) {
			upgraded := make([]inboundTagRef, len(refs), nextCap)
			copy(upgraded, refs)
			refs = upgraded
		}
		seen = make(map[string]struct{}, nextCap)
		for _, item := range refs {
			seen[item.Tag] = struct{}{}
		}
	}
	return refs, seen
}

func xrayInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallXrayUniqueThreshold+1 {
		return smallXrayUniqueThreshold + 1
	}
	return capHint
}

func forEachXrayCSVItem(value string, visit func(string)) {
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

func usersVersion(users []types.ServerXrayUser) string {
	h := sha256.New()
	fieldSep := []byte{0}
	recordSep := []byte{1}
	buf := make([]byte, 0, 24)
	writeInt := func(value int64) {
		buf = strconv.AppendInt(buf[:0], value, 10)
		_, _ = h.Write(buf)
		_, _ = h.Write(fieldSep)
	}
	writeBool := func(value bool) {
		buf = strconv.AppendBool(buf[:0], value)
		_, _ = h.Write(buf)
		_, _ = h.Write(fieldSep)
	}
	writeString := func(value string) {
		buf = strconv.AppendInt(buf[:0], int64(len(value)), 10)
		_, _ = h.Write(buf)
		_, _ = h.Write(fieldSep)
		_, _ = io.WriteString(h, value)
		_, _ = h.Write(fieldSep)
	}
	for _, user := range users {
		writeInt(user.UserSubscribeID)
		writeInt(user.UserID)
		writeString(user.Email)
		writeString(user.UUID)
		writeString(user.Password)
		writeInt(int64(user.Level))
		writeBool(user.Enabled)
		writeInt(user.ExpiredAt)
		writeInt(user.SpeedLimit)
		writeInt(user.DeviceLimit)
		writeInt(int64(len(user.Inbounds)))
		for _, inbound := range user.Inbounds {
			writeString(inbound)
		}
		_, _ = h.Write(recordSep)
	}
	return hex.EncodeToString(h.Sum(nil))
}
