package subscribeview

import (
	"encoding/json"
	"strings"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

var (
	emptyDiscounts = []types.SubscribeDiscount{}
	emptyNodeIDs   = []int64{}
	emptyNodeTags  = []string{}
)

const smallSubscribeViewCacheLimit = 8

type SubscribeOptions struct {
	IncludeTopology bool
}

func SubscribeFromModel(item *modelSubscribe.Subscribe, opts SubscribeOptions) (types.Subscribe, error) {
	return subscribeFromModelWithCaches(item, opts, nil, nil)
}

func SubscribeFromModelLenient(item *modelSubscribe.Subscribe, opts SubscribeOptions) types.Subscribe {
	resp, _ := subscribeFromModelWithCaches(item, opts, nil, nil)
	return resp
}

func SubscribeRowsLenient(rows []*modelSubscribe.Subscribe, opts SubscribeOptions) []types.Subscribe {
	if len(rows) == 0 {
		return []types.Subscribe{}
	}
	collector := viewutil.NewRowsCollector[types.Subscribe](len(rows))
	discountCache := newDiscountParseCache(len(rows))
	topologyCache := newTopologyParseCache(len(rows), opts.IncludeTopology)
	for _, row := range rows {
		if row == nil {
			continue
		}
		subscribe, _ := subscribeFromModelWithCaches(row, opts, discountCache, topologyCache)
		collector.Add(subscribe)
	}
	return collector.Rows()
}

func subscribeFromModelWithCaches(
	item *modelSubscribe.Subscribe,
	opts SubscribeOptions,
	discountCache *discountParseCache,
	topologyCache *topologyParseCache,
) (types.Subscribe, error) {
	if item == nil {
		return types.Subscribe{}, nil
	}
	discount, err := discountCache.Parse(item.Discount)
	if err != nil {
		return subscribeFromModelWithParsedValues(item, opts, emptyDiscounts, topologyCache), err
	}
	return subscribeFromModelWithParsedValues(item, opts, discount, topologyCache), nil
}

func subscribeFromModelWithParsedValues(
	item *modelSubscribe.Subscribe,
	opts SubscribeOptions,
	discount []types.SubscribeDiscount,
	topologyCache *topologyParseCache,
) types.Subscribe {
	if discount == nil {
		discount = emptyDiscounts
	}
	resp := types.Subscribe{
		Id:                item.Id,
		Name:              item.Name,
		Language:          item.Language,
		Description:       item.Description,
		UnitPrice:         item.UnitPrice,
		UnitTime:          item.UnitTime,
		Discount:          discount,
		Replacement:       item.Replacement,
		Inventory:         item.Inventory,
		Traffic:           item.Traffic,
		SpeedLimit:        item.SpeedLimit,
		DeviceLimit:       item.DeviceLimit,
		Quota:             item.Quota,
		Show:              BoolValue(item.Show),
		UserVisible:       BoolValue(item.UserVisible),
		Sell:              BoolValue(item.Sell),
		Sort:              item.Sort,
		DeductionRatio:    item.DeductionRatio,
		AllowDeduction:    BoolValue(item.AllowDeduction),
		AllowRenewal:      BoolValue(item.AllowRenewal),
		AllowResetTraffic: BoolValue(item.AllowResetTraffic),
		ResetCycle:        item.ResetCycle,
		RenewalReset:      BoolValue(item.RenewalReset),
		ShowOriginalPrice: item.ShowOriginalPrice,
		DisplayTemplate:   item.DisplayTemplate,
		Badge:             item.Badge,
		MinQuantity:       item.MinQuantity,
		MaxQuantity:       item.MaxQuantity,
		SaleStartsAt:      timex.UnixMilliPointerOrZero(item.SaleStartsAt),
		SaleEndsAt:        timex.UnixMilliPointerOrZero(item.SaleEndsAt),
		CreatedAt:         timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:         timex.UnixMilliOrZero(item.UpdatedAt),
		Nodes:             emptyNodeIDs,
		NodeTags:          emptyNodeTags,
	}
	if opts.IncludeTopology {
		nodes, nodeTags := topologyCache.Parse(item.Nodes, item.NodeTags)
		resp.Nodes = nonNilNodeIDs(nodes)
		resp.NodeTags = nonNilNodeTags(nodeTags)
	}
	return resp
}

type discountParseCache struct {
	rows    []discountParseCacheEntry
	index   map[string]int
	capHint int
}

type discountParseCacheEntry struct {
	raw      string
	discount []types.SubscribeDiscount
	err      error
}

func newDiscountParseCache(capHint int) *discountParseCache {
	if capHint <= 1 {
		return nil
	}
	return &discountParseCache{capHint: capHint}
}

func (c *discountParseCache) Parse(raw string) ([]types.SubscribeDiscount, error) {
	if c == nil {
		return ParseDiscount(raw)
	}
	if c.index != nil {
		if idx, ok := c.index[raw]; ok {
			entry := c.rows[idx]
			return entry.discount, entry.err
		}
		return c.add(raw)
	}
	for _, entry := range c.rows {
		if entry.raw == raw {
			return entry.discount, entry.err
		}
	}
	return c.add(raw)
}

func (c *discountParseCache) add(raw string) ([]types.SubscribeDiscount, error) {
	discount, err := ParseDiscount(raw)
	entry := discountParseCacheEntry{
		raw:      raw,
		discount: discount,
		err:      err,
	}
	c.ensureRows()
	c.rows = append(c.rows, entry)
	if c.index != nil {
		c.index[raw] = len(c.rows) - 1
	} else if len(c.rows) > smallSubscribeViewCacheLimit {
		nextCap := subscribeViewCacheCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]discountParseCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return discount, err
}

type topologyParseCache struct {
	rows    []topologyParseCacheEntry
	index   map[topologyCacheKey]int
	capHint int
}

type topologyParseCacheEntry struct {
	key      topologyCacheKey
	nodes    []int64
	nodeTags []string
}

type topologyCacheKey struct {
	nodes    string
	nodeTags string
}

func newTopologyParseCache(capHint int, enabled bool) *topologyParseCache {
	if !enabled || capHint <= 1 {
		return nil
	}
	return &topologyParseCache{capHint: capHint}
}

func (c *topologyParseCache) Parse(rawNodes, rawNodeTags string) ([]int64, []string) {
	if c == nil {
		return modelSubscribe.NormalizeNodeSelectors(rawNodes, rawNodeTags)
	}
	key := topologyCacheKey{nodes: rawNodes, nodeTags: rawNodeTags}
	if c.index != nil {
		if idx, ok := c.index[key]; ok {
			entry := c.rows[idx]
			return entry.nodes, entry.nodeTags
		}
		return c.add(key)
	}
	for _, entry := range c.rows {
		if entry.key == key {
			return entry.nodes, entry.nodeTags
		}
	}
	return c.add(key)
}

func (c *topologyParseCache) add(key topologyCacheKey) ([]int64, []string) {
	nodes, nodeTags := modelSubscribe.NormalizeNodeSelectors(key.nodes, key.nodeTags)
	entry := topologyParseCacheEntry{
		key:      key,
		nodes:    nodes,
		nodeTags: nodeTags,
	}
	c.ensureRows()
	c.rows = append(c.rows, entry)
	if c.index != nil {
		c.index[key] = len(c.rows) - 1
	} else if len(c.rows) > smallSubscribeViewCacheLimit {
		nextCap := subscribeViewCacheCap(len(c.rows), c.capHint)
		c.index = make(map[topologyCacheKey]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]topologyParseCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.key] = i
		}
	}
	return nodes, nodeTags
}

func (c *discountParseCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallSubscribeViewCacheLimit
	} else if capHint > smallSubscribeViewCacheLimit+1 {
		capHint = smallSubscribeViewCacheLimit + 1
	}
	c.rows = make([]discountParseCacheEntry, 0, capHint)
}

func (c *topologyParseCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallSubscribeViewCacheLimit
	} else if capHint > smallSubscribeViewCacheLimit+1 {
		capHint = smallSubscribeViewCacheLimit + 1
	}
	c.rows = make([]topologyParseCacheEntry, 0, capHint)
}

func subscribeViewCacheCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func nonNilNodeIDs(nodes []int64) []int64 {
	if nodes == nil {
		return emptyNodeIDs
	}
	return nodes
}

func nonNilNodeTags(tags []string) []string {
	if tags == nil {
		return emptyNodeTags
	}
	return tags
}

func ParseDiscount(raw string) ([]types.SubscribeDiscount, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return emptyDiscounts, nil
	}
	var discount []types.SubscribeDiscount
	if err := json.Unmarshal([]byte(raw), &discount); err != nil {
		return nil, err
	}
	if discount == nil {
		return emptyDiscounts, nil
	}
	return discount, nil
}

func ParseDiscountLenient(raw string) []types.SubscribeDiscount {
	discount, err := ParseDiscount(raw)
	if err != nil {
		return emptyDiscounts
	}
	return discount
}

func SplitCommaList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	_, tags := modelSubscribe.NormalizeNodeSelectors("", raw)
	return tags
}

func BoolValue(value *bool) bool {
	return value != nil && *value
}
