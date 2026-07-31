package ordersnapshot

import (
	"encoding/json"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/tool"
)

const (
	KindSubscribe  = "subscribe"
	KindMembership = "membership"
	KindRecharge   = "recharge"
)

const smallSelectorMatchThreshold = 8
const parseCacheLinearLimit = 8

var (
	emptyDiscounts = []types.SubscribeDiscount{}
	emptyNodeIDs   = []int64{}
	emptyNodeTags  = []string{}
)

type Snapshot struct {
	Version           int                       `json:"version"`
	Kind              string                    `json:"kind"`
	ID                int64                     `json:"id,omitempty"`
	Name              string                    `json:"name"`
	Description       string                    `json:"description,omitempty"`
	UnitPrice         int64                     `json:"unit_price,omitempty"`
	UnitTime          string                    `json:"unit_time,omitempty"`
	Discount          []types.SubscribeDiscount `json:"discount,omitempty"`
	Replacement       int64                     `json:"replacement,omitempty"`
	Traffic           int64                     `json:"traffic,omitempty"`
	SpeedLimit        int64                     `json:"speed_limit,omitempty"`
	DeviceLimit       int64                     `json:"device_limit,omitempty"`
	Quota             int64                     `json:"quota,omitempty"`
	Nodes             []int64                   `json:"nodes,omitempty"`
	NodeTags          []string                  `json:"node_tags,omitempty"`
	ShowOriginalPrice bool                      `json:"show_original_price,omitempty"`
	ResetCycle        int64                     `json:"reset_cycle,omitempty"`
	RenewalReset      bool                      `json:"renewal_reset,omitempty"`
	AllowRenewal      bool                      `json:"allow_renewal,omitempty"`
	AllowResetTraffic bool                      `json:"allow_reset_traffic,omitempty"`
	DurationUnit      string                    `json:"duration_unit,omitempty"`
	DurationValue     int64                     `json:"duration_value,omitempty"`
	MembershipPolicy  types.MembershipPolicy    `json:"membership_policy,omitempty"`
}

func FromSubscribe(sub *subscribe.Subscribe) Snapshot {
	if sub == nil {
		return Snapshot{}
	}
	nodes, nodeTags := subscribe.NormalizeNodeSelectors(sub.Nodes, sub.NodeTags)
	return Snapshot{
		Version:           1,
		Kind:              KindSubscribe,
		ID:                sub.Id,
		Name:              sub.Name,
		Description:       sub.Description,
		UnitPrice:         sub.UnitPrice,
		UnitTime:          sub.UnitTime,
		Discount:          parseSubscribeDiscount(sub.Discount),
		Replacement:       sub.Replacement,
		Traffic:           sub.Traffic,
		SpeedLimit:        sub.SpeedLimit,
		DeviceLimit:       sub.DeviceLimit,
		Quota:             sub.Quota,
		Nodes:             nodes,
		NodeTags:          nodeTags,
		ShowOriginalPrice: sub.ShowOriginalPrice,
		ResetCycle:        sub.ResetCycle,
		RenewalReset:      boolValue(sub.RenewalReset),
		AllowRenewal:      boolValue(sub.AllowRenewal),
		AllowResetTraffic: boolValue(sub.AllowResetTraffic),
	}
}

func parseSubscribeDiscount(raw string) []types.SubscribeDiscount {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var discounts []types.SubscribeDiscount
	if err := json.Unmarshal([]byte(raw), &discounts); err != nil || len(discounts) == 0 {
		return nil
	}
	return discounts
}

func FromMembershipPlan(plan *membership.Plan) Snapshot {
	if plan == nil {
		return Snapshot{}
	}
	return Snapshot{
		Version:       1,
		Kind:          KindMembership,
		ID:            plan.Id,
		Name:          plan.Name,
		Description:   plan.Description,
		UnitPrice:     plan.UnitPrice,
		DurationUnit:  plan.DurationUnit,
		DurationValue: plan.DurationValue,
		MembershipPolicy: types.MembershipPolicy{
			MonthlyConsumptionLimitEnabled: plan.MonthlyConsumptionLimitEnabled,
			MonthlyConsumptionLimit:        plan.MonthlyConsumptionLimit,
			MonthlyOrderLimitEnabled:       plan.MonthlyOrderLimitEnabled,
			MonthlyOrderLimit:              plan.MonthlyOrderLimit,
			SubscriptionExpiryLimitEnabled: plan.SubscriptionExpiryLimitEnabled,
			SubscriptionExpiryGraceDays:    plan.SubscriptionExpiryGraceDays,
		},
	}
}

func FromRecharge(amount int64) Snapshot {
	return Snapshot{
		Version:   1,
		Kind:      KindRecharge,
		Name:      "Balance Recharge",
		UnitPrice: amount,
	}
}

func Marshal(snapshot Snapshot) string {
	if snapshot.Version == 0 {
		snapshot.Version = 1
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return ""
	}
	return string(data)
}

func Parse(raw string) (Snapshot, bool) {
	if raw == "" {
		return Snapshot{}, false
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return Snapshot{}, false
	}
	return snapshot, snapshot.Kind != "" && snapshot.Name != ""
}

type ParseCache struct {
	rows    []parseCacheEntry
	index   map[string]int
	capHint int
}

type parseCacheEntry struct {
	raw      string
	snapshot Snapshot
	ok       bool
}

func NewParseCache(capHint int) *ParseCache {
	if capHint <= 1 {
		return nil
	}
	return &ParseCache{capHint: capHint}
}

func (c *ParseCache) Parse(raw string) (Snapshot, bool) {
	if c == nil {
		return Parse(raw)
	}
	if c.index != nil {
		if idx, ok := c.index[raw]; ok {
			entry := c.rows[idx]
			return entry.snapshot, entry.ok
		}
		return c.add(raw)
	}
	for _, entry := range c.rows {
		if entry.raw == raw {
			return entry.snapshot, entry.ok
		}
	}
	return c.add(raw)
}

func (c *ParseCache) add(raw string) (Snapshot, bool) {
	snapshot, ok := Parse(raw)
	entry := parseCacheEntry{raw: raw, snapshot: snapshot, ok: ok}
	c.ensureRows()
	c.rows = append(c.rows, entry)
	if c.index != nil {
		c.index[raw] = len(c.rows) - 1
	} else if len(c.rows) > parseCacheLinearLimit {
		nextCap := snapshotLookupCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]parseCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return snapshot, ok
}

func (c *ParseCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = parseCacheLinearLimit
	} else if capHint > parseCacheLinearLimit+1 {
		capHint = parseCacheLinearLimit + 1
	}
	c.rows = make([]parseCacheEntry, 0, capHint)
}

type snapshotIdentity struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

func parseSnapshotIdentity(raw string) (snapshotIdentity, bool) {
	if raw == "" {
		return snapshotIdentity{}, false
	}
	var snapshot snapshotIdentity
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return snapshotIdentity{}, false
	}
	return snapshot, snapshot.Kind != "" && snapshot.Name != ""
}

func ProductName(raw string) string {
	snapshot, ok := parseSnapshotIdentity(raw)
	if !ok {
		return ""
	}
	return snapshot.Name
}

func IsSubscribe(raw string) bool {
	snapshot, ok := parseSnapshotIdentity(raw)
	return ok && snapshot.Kind == KindSubscribe
}

func ApplyToSubscribeType(raw string, target *types.Subscribe) bool {
	snapshot, ok := Parse(raw)
	if !ok || snapshot.Kind != KindSubscribe || target == nil {
		return false
	}
	*target = SubscribeType(snapshot)
	return true
}

func SubscribeType(snapshot Snapshot) types.Subscribe {
	if snapshot.Kind != KindSubscribe {
		return types.Subscribe{}
	}
	nodes, nodeTags := normalizeSnapshotTopology(snapshot)
	return types.Subscribe{
		Id:                snapshot.ID,
		Name:              snapshot.Name,
		Description:       snapshot.Description,
		UnitPrice:         snapshot.UnitPrice,
		UnitTime:          snapshot.UnitTime,
		Discount:          nonNilDiscounts(snapshot.Discount),
		Replacement:       snapshot.Replacement,
		Traffic:           snapshot.Traffic,
		SpeedLimit:        snapshot.SpeedLimit,
		DeviceLimit:       snapshot.DeviceLimit,
		Quota:             snapshot.Quota,
		Nodes:             nonNilNodeIDs(nodes),
		NodeTags:          nonNilNodeTags(nodeTags),
		AllowRenewal:      snapshot.AllowRenewal,
		AllowResetTraffic: snapshot.AllowResetTraffic,
		ResetCycle:        snapshot.ResetCycle,
		RenewalReset:      snapshot.RenewalReset,
		ShowOriginalPrice: snapshot.ShowOriginalPrice,
	}
}

func ApplyToMembershipPlanType(raw string, target **types.MembershipPlan) bool {
	snapshot, ok := Parse(raw)
	if !ok || snapshot.Kind != KindMembership || target == nil {
		return false
	}
	*target = MembershipPlanType(snapshot)
	return true
}

func MembershipPlanType(snapshot Snapshot) *types.MembershipPlan {
	if snapshot.Kind != KindMembership {
		return nil
	}
	return &types.MembershipPlan{
		Id:            snapshot.ID,
		Name:          snapshot.Name,
		Description:   snapshot.Description,
		UnitPrice:     snapshot.UnitPrice,
		DurationUnit:  snapshot.DurationUnit,
		DurationValue: snapshot.DurationValue,
		Enabled:       true,
		Policy:        snapshot.MembershipPolicy,
	}
}

func ApplyToMembershipPlanModel(raw string, target *membership.Plan) bool {
	snapshot, ok := Parse(raw)
	if !ok || snapshot.Kind != KindMembership || target == nil {
		return false
	}
	target.Id = snapshot.ID
	target.Name = snapshot.Name
	target.Description = snapshot.Description
	target.UnitPrice = snapshot.UnitPrice
	target.DurationUnit = snapshot.DurationUnit
	target.DurationValue = snapshot.DurationValue
	target.MonthlyConsumptionLimitEnabled = snapshot.MembershipPolicy.MonthlyConsumptionLimitEnabled
	target.MonthlyConsumptionLimit = snapshot.MembershipPolicy.MonthlyConsumptionLimit
	target.MonthlyOrderLimitEnabled = snapshot.MembershipPolicy.MonthlyOrderLimitEnabled
	target.MonthlyOrderLimit = snapshot.MembershipPolicy.MonthlyOrderLimit
	target.SubscriptionExpiryLimitEnabled = snapshot.MembershipPolicy.SubscriptionExpiryLimitEnabled
	target.SubscriptionExpiryGraceDays = snapshot.MembershipPolicy.SubscriptionExpiryGraceDays
	enabled := true
	target.Enabled = &enabled
	return true
}

func ApplyToSubscribeModel(raw string, target *subscribe.Subscribe) bool {
	snapshot, ok := Parse(raw)
	if !ok || snapshot.Kind != KindSubscribe || target == nil {
		return false
	}
	nodes, nodeTags := normalizeSnapshotTopology(snapshot)
	target.Id = snapshot.ID
	target.Name = snapshot.Name
	target.Description = snapshot.Description
	target.UnitPrice = snapshot.UnitPrice
	target.UnitTime = snapshot.UnitTime
	target.Replacement = snapshot.Replacement
	target.Traffic = snapshot.Traffic
	target.SpeedLimit = snapshot.SpeedLimit
	target.DeviceLimit = snapshot.DeviceLimit
	target.Quota = snapshot.Quota
	target.Nodes = tool.Int64SliceToString(nodes)
	target.NodeTags = tool.StringSliceToString(nodeTags)
	target.ShowOriginalPrice = snapshot.ShowOriginalPrice
	target.ResetCycle = snapshot.ResetCycle
	target.RenewalReset = boolPtr(snapshot.RenewalReset)
	target.AllowRenewal = boolPtr(snapshot.AllowRenewal)
	target.AllowResetTraffic = boolPtr(snapshot.AllowResetTraffic)
	target.Discount = ""
	if len(snapshot.Discount) > 0 {
		data, _ := json.Marshal(snapshot.Discount)
		target.Discount = string(data)
	}
	return true
}

func SubscribeModel(raw string) (*subscribe.Subscribe, bool) {
	var product subscribe.Subscribe
	if !ApplyToSubscribeModel(raw, &product) {
		return nil, false
	}
	return &product, true
}

func SubscribeTopology(raw string) ([]int64, []string, bool) {
	if raw == "" {
		return nil, nil, false
	}
	var snapshot struct {
		Kind     string   `json:"kind"`
		Name     string   `json:"name"`
		Nodes    []int64  `json:"nodes,omitempty"`
		NodeTags []string `json:"node_tags,omitempty"`
	}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, nil, false
	}
	if snapshot.Kind != KindSubscribe || snapshot.Name == "" {
		return nil, nil, false
	}
	return nilIfEmptyInt64(tool.NormalizePositiveInt64Slice(snapshot.Nodes)), nilIfEmptyString(tool.NormalizeStringSlice(snapshot.NodeTags)), true
}

func SubscribeLimits(raw string) (speedLimit int64, deviceLimit int64, ok bool) {
	if raw == "" {
		return 0, 0, false
	}
	var snapshot struct {
		Kind        string `json:"kind"`
		Name        string `json:"name"`
		SpeedLimit  int64  `json:"speed_limit,omitempty"`
		DeviceLimit int64  `json:"device_limit,omitempty"`
	}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return 0, 0, false
	}
	if snapshot.Kind != KindSubscribe || snapshot.Name == "" {
		return 0, 0, false
	}
	return snapshot.SpeedLimit, snapshot.DeviceLimit, true
}

func MatchesNodes(raw string, nodeIDs []int64, nodeTags []string) bool {
	snapshot, ok := Parse(raw)
	if !ok || snapshot.Kind != KindSubscribe {
		return false
	}
	snapshotNodes, snapshotTags := normalizeSnapshotTopology(snapshot)
	nodeIDs = tool.NormalizePositiveInt64Slice(nodeIDs)
	if hasAnyInt64Selector(nodeIDs, snapshotNodes) {
		return true
	}
	nodeTags = tool.NormalizeStringSlice(nodeTags)
	return hasAnyStringSelector(nodeTags, snapshotTags)
}

func hasAnyInt64Selector(left []int64, right []int64) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	if len(left) > len(right) {
		left, right = right, left
	}
	if len(left) <= smallSelectorMatchThreshold {
		for _, value := range left {
			for _, candidate := range right {
				if candidate == value {
					return true
				}
			}
		}
		return false
	}
	values := newSelectorMatchSet[int64](len(left))
	for _, value := range left {
		values.Add(value)
	}
	for _, candidate := range right {
		if values.Contains(candidate) {
			return true
		}
	}
	return false
}

func hasAnyStringSelector(left []string, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	if len(left) > len(right) {
		left, right = right, left
	}
	if len(left) <= smallSelectorMatchThreshold {
		for _, value := range left {
			for _, candidate := range right {
				if candidate == value {
					return true
				}
			}
		}
		return false
	}
	values := newSelectorMatchSet[string](len(left))
	for _, value := range left {
		values.Add(value)
	}
	for _, candidate := range right {
		if values.Contains(candidate) {
			return true
		}
	}
	return false
}

type selectorMatchSet[T comparable] struct {
	values  []T
	index   map[T]struct{}
	capHint int
}

func newSelectorMatchSet[T comparable](capHint int) selectorMatchSet[T] {
	return selectorMatchSet[T]{capHint: capHint}
}

func (s *selectorMatchSet[T]) Add(value T) bool {
	if s.index != nil {
		if _, ok := s.index[value]; ok {
			return false
		}
		s.index[value] = struct{}{}
		s.values = append(s.values, value)
		return true
	}
	for _, existing := range s.values {
		if existing == value {
			return false
		}
	}
	if s.values == nil {
		s.values = make([]T, 0, selectorMatchInitialCap(s.capHint))
	}
	s.values = append(s.values, value)
	if len(s.values) > smallSelectorMatchThreshold {
		s.upgradeIndex()
	}
	return true
}

func (s selectorMatchSet[T]) Contains(value T) bool {
	if s.index != nil {
		_, ok := s.index[value]
		return ok
	}
	for _, existing := range s.values {
		if existing == value {
			return true
		}
	}
	return false
}

func (s *selectorMatchSet[T]) upgradeIndex() {
	if s.index != nil {
		return
	}
	nextCap := snapshotLookupCap(len(s.values), s.capHint)
	if nextCap > cap(s.values) {
		values := make([]T, len(s.values), nextCap)
		copy(values, s.values)
		s.values = values
	}
	s.index = make(map[T]struct{}, nextCap)
	for _, value := range s.values {
		s.index[value] = struct{}{}
	}
}

func snapshotLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func selectorMatchInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallSelectorMatchThreshold+1 {
		return smallSelectorMatchThreshold + 1
	}
	return capHint
}

func normalizeSnapshotTopology(snapshot Snapshot) ([]int64, []string) {
	return nilIfEmptyInt64(tool.NormalizePositiveInt64Slice(snapshot.Nodes)), nilIfEmptyString(tool.NormalizeStringSlice(snapshot.NodeTags))
}

func nilIfEmptyInt64(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	return values
}

func nilIfEmptyString(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return values
}

func nonNilDiscounts(discounts []types.SubscribeDiscount) []types.SubscribeDiscount {
	if discounts == nil {
		return emptyDiscounts
	}
	return discounts
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

func boolValue(value *bool) bool {
	return value != nil && *value
}

func boolPtr(value bool) *bool {
	return &value
}
