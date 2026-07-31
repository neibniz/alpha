package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/internal/xraytemplate"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type QueryServerForwardTableLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

const smallForwardUniqueThreshold = 8

func NewQueryServerForwardTableLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryServerForwardTableLogic {
	return &QueryServerForwardTableLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryServerForwardTableLogic) QueryServerForwardTable(req *types.QueryServerForwardTableRequest) (*types.QueryServerForwardTableResponse, error) {
	forwarderIP := strings.TrimSpace(req.IP)
	if net.ParseIP(forwarderIP) == nil {
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "valid forwarder ip is required"), "invalid forwarder ip: %s", req.IP)
	}

	nodes, err := l.queryForwardNodes()
	if err != nil {
		l.Errorf("[QueryServerForwardTable] query nodes error: %v", err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query nodes error: %v", err)
	}

	resp := &types.QueryServerForwardTableResponse{
		IP:       forwarderIP,
		Warnings: []string{},
	}
	items := viewutil.NewRowsCollector[types.ServerForwardItem](len(nodes))
	portOwners := newForwardPortOwnerCache(len(nodes))
	inboundCache := newForwardInboundPortCache(len(nodes))
	serverIPCache := newForwardHostIPCache(len(nodes))
	nodeIPCache := newForwardHostIPCache(len(nodes))

	for _, item := range nodes {
		if item == nil || item.Server == nil || item.Port == 0 || strings.TrimSpace(item.Address) == "" {
			continue
		}
		nodeIPs, err := resolveHostIPsCached(l.ctx, &nodeIPCache, item.Address)
		if err != nil {
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("node %d address %q resolve failed: %v", item.Id, item.Address, err))
			continue
		}
		if !containsString(nodeIPs, forwarderIP) {
			continue
		}

		inbound := strings.TrimSpace(item.Inbound())
		if inbound == "" {
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("node %d has empty inbound alias", item.Id))
			continue
		}
		inboundPorts, ok := inboundCache.Get(item.ServerId)
		if !ok {
			inboundPorts, err = l.queryServerInboundPorts(item.Server)
			if err != nil {
				resp.Warnings = append(resp.Warnings, fmt.Sprintf("server %d inbound render failed: %v", item.ServerId, err))
				continue
			}
			inboundCache.Set(item.ServerId, inboundPorts)
		}
		targetPort, ok := inboundPorts.Get(inbound)
		if !ok || targetPort == 0 {
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("node %d inbound %q has no rendered port", item.Id, inbound))
			continue
		}

		targetIPs, err := resolveHostIPsCached(l.ctx, &serverIPCache, item.Server.Address)
		if err != nil {
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("server %d address %q resolve failed: %v", item.ServerId, item.Server.Address, err))
			continue
		}
		owner := fmt.Sprintf("node %d inbound %s", item.Id, inbound)
		for _, targetIP := range targetIPs {
			if existing, exists := portOwners.Get(uint(item.Port)); exists && existing != owner {
				resp.Warnings = append(resp.Warnings, fmt.Sprintf("local port %d conflict: %s skipped, existing %s", item.Port, owner, existing))
				continue
			}
			portOwners.Set(uint(item.Port), owner)
			items.Add(types.ServerForwardItem{
				LocalPort:  uint(item.Port),
				TargetHost: targetIP,
				TargetPort: targetPort,
				NodeID:     item.Id,
				NodeName:   item.Name,
				ServerID:   item.ServerId,
				ServerName: item.Server.Name,
				Inbound:    inbound,
			})
		}
	}

	resp.Items = items.Rows()
	sort.Slice(resp.Items, func(i, j int) bool {
		if resp.Items[i].LocalPort == resp.Items[j].LocalPort {
			if resp.Items[i].TargetHost == resp.Items[j].TargetHost {
				return resp.Items[i].TargetPort < resp.Items[j].TargetPort
			}
			return resp.Items[i].TargetHost < resp.Items[j].TargetHost
		}
		return resp.Items[i].LocalPort < resp.Items[j].LocalPort
	})
	sort.Strings(resp.Warnings)
	return resp, nil
}

func (l *QueryServerForwardTableLogic) queryForwardNodes() ([]*node.Node, error) {
	var nodes []*node.Node
	err := l.svcCtx.DB.WithContext(l.ctx).
		Model(&node.Node{}).
		Select(forwardTableNodeColumns).
		Preload("Server", func(db *gorm.DB) *gorm.DB {
			return db.Select(forwardTableServerColumns)
		}).
		Where("enabled = ?", true).
		Where("port > ?", 0).
		Where("address <> ''").
		Order("sort ASC, id ASC").
		Find(&nodes).Error
	return nodes, err
}

var forwardTableNodeColumns = []string{
	"id",
	"name",
	"port",
	"address",
	"server_id",
	"protocol",
	"inbound_alias",
}

var forwardTableServerColumns = []string{
	"id",
	"name",
	"address",
}

func (l *QueryServerForwardTableLogic) queryServerInboundPorts(srv *node.Server) (*forwardInboundPortIndex, error) {
	if srv == nil {
		return &forwardInboundPortIndex{}, nil
	}
	bindings, err := l.svcCtx.NodeModel.QueryServerXrayTemplates(l.ctx, srv.Id, true)
	if err != nil {
		return nil, err
	}
	rendered, err := xraytemplate.RenderBindingConfigList(srv, bindings, xraytemplate.RenderBindingConfigOptions{
		TemplateType: node.XrayTemplateInbound,
	})
	if err != nil {
		return nil, err
	}
	out := newForwardInboundPortIndexWithHint(len(rendered) * 2)
	for index, item := range rendered {
		binding := item.Binding
		addForwardInboundPortAliases(out, binding, item.Config, fmt.Sprintf("inbound_%d", index+1))
	}
	return out, nil
}

func addForwardInboundPortAliases(out *forwardInboundPortIndex, binding *node.ServerXrayTemplate, cfg xraytemplate.ConfigObject, fallback string) {
	port := configUint(cfg, "port")
	if out == nil || port == 0 {
		return
	}
	added := false
	for _, alias := range xraytemplate.BindingInboundAliasesFromConfig(binding, cfg) {
		if strings.TrimSpace(alias) == "" {
			continue
		}
		out.Set(alias, port)
		added = true
	}
	if !added {
		out.Set(fallback, port)
	}
}

func countForwardInboundBindings(bindings []*node.ServerXrayTemplate) int {
	count := 0
	for _, binding := range bindings {
		if binding == nil || binding.Template == nil || binding.Template.Type != node.XrayTemplateInbound {
			continue
		}
		count++
	}
	return count
}

type forwardInboundPortEntry struct {
	alias string
	port  uint
}

type forwardInboundPortIndex struct {
	rows    []forwardInboundPortEntry
	byAlias map[string]int
	capHint int
}

func newForwardInboundPortIndexWithHint(capHint int) *forwardInboundPortIndex {
	if capHint <= 0 {
		return &forwardInboundPortIndex{}
	}
	return &forwardInboundPortIndex{
		capHint: capHint,
	}
}

func newForwardInboundPortIndexFromMap(values map[string]uint) *forwardInboundPortIndex {
	index := newForwardInboundPortIndexWithHint(len(values))
	for alias, port := range values {
		index.Set(alias, port)
	}
	return index
}

func (i *forwardInboundPortIndex) Get(alias string) (uint, bool) {
	if i == nil {
		return 0, false
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return 0, false
	}
	if i.byAlias != nil {
		row, ok := i.byAlias[alias]
		if !ok {
			return 0, false
		}
		return i.rows[row].port, true
	}
	for _, row := range i.rows {
		if row.alias == alias {
			return row.port, true
		}
	}
	return 0, false
}

func (i *forwardInboundPortIndex) Set(alias string, port uint) {
	if i == nil || port == 0 {
		return
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	if i.byAlias != nil {
		if row, ok := i.byAlias[alias]; ok {
			i.rows[row].port = port
			return
		}
		i.byAlias[alias] = len(i.rows)
		i.rows = append(i.rows, forwardInboundPortEntry{alias: alias, port: port})
		return
	}
	for row := range i.rows {
		if i.rows[row].alias == alias {
			i.rows[row].port = port
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, forwardInboundPortEntry{alias: alias, port: port})
	if len(i.rows) > smallForwardUniqueThreshold {
		i.upgradeLookup()
	}
}

func (i *forwardInboundPortIndex) Len() int {
	if i == nil {
		return 0
	}
	return len(i.rows)
}

func (i *forwardInboundPortIndex) ensureRows() {
	if i.rows != nil || i.capHint <= 0 {
		return
	}
	i.rows = make([]forwardInboundPortEntry, 0, forwardInitialCap(i.capHint))
}

func (i *forwardInboundPortIndex) upgradeLookup() {
	if i.byAlias != nil {
		return
	}
	nextCap := forwardLookupCap(len(i.rows), i.capHint)
	if nextCap > cap(i.rows) {
		rows := make([]forwardInboundPortEntry, len(i.rows), nextCap)
		copy(rows, i.rows)
		i.rows = rows
	}
	i.byAlias = make(map[string]int, nextCap)
	for row, item := range i.rows {
		i.byAlias[item.alias] = row
	}
}

type forwardInboundPortCacheEntry struct {
	serverID int64
	ports    *forwardInboundPortIndex
}

type forwardInboundPortCache struct {
	rows     []forwardInboundPortCacheEntry
	byServer map[int64]int
	capHint  int
}

func newForwardInboundPortCache(capHint int) forwardInboundPortCache {
	if capHint <= 0 {
		return forwardInboundPortCache{}
	}
	return forwardInboundPortCache{
		capHint: capHint,
	}
}

func (c *forwardInboundPortCache) Get(serverID int64) (*forwardInboundPortIndex, bool) {
	if serverID <= 0 {
		return nil, false
	}
	if c.byServer != nil {
		row, ok := c.byServer[serverID]
		if !ok {
			return nil, false
		}
		return c.rows[row].ports, true
	}
	for _, row := range c.rows {
		if row.serverID == serverID {
			return row.ports, true
		}
	}
	return nil, false
}

func (c *forwardInboundPortCache) Set(serverID int64, ports *forwardInboundPortIndex) {
	if serverID <= 0 {
		return
	}
	if c.byServer != nil {
		if row, ok := c.byServer[serverID]; ok {
			c.rows[row].ports = ports
			return
		}
		c.byServer[serverID] = len(c.rows)
		c.rows = append(c.rows, forwardInboundPortCacheEntry{serverID: serverID, ports: ports})
		return
	}
	for row := range c.rows {
		if c.rows[row].serverID == serverID {
			c.rows[row].ports = ports
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, forwardInboundPortCacheEntry{serverID: serverID, ports: ports})
	if len(c.rows) > smallForwardUniqueThreshold {
		c.upgradeLookup()
	}
}

func (c forwardInboundPortCache) Len() int {
	return len(c.rows)
}

func (c *forwardInboundPortCache) ensureRows() {
	if c.rows != nil || c.capHint <= 0 {
		return
	}
	c.rows = make([]forwardInboundPortCacheEntry, 0, forwardInitialCap(c.capHint))
}

func (c *forwardInboundPortCache) upgradeLookup() {
	if c.byServer != nil {
		return
	}
	nextCap := forwardLookupCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]forwardInboundPortCacheEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.byServer = make(map[int64]int, nextCap)
	for row, item := range c.rows {
		c.byServer[item.serverID] = row
	}
}

type forwardPortOwnerEntry struct {
	port  uint
	owner string
}

type forwardPortOwnerCache struct {
	rows    []forwardPortOwnerEntry
	byPort  map[uint]int
	capHint int
}

func newForwardPortOwnerCache(capHint int) forwardPortOwnerCache {
	if capHint <= 0 {
		return forwardPortOwnerCache{}
	}
	return forwardPortOwnerCache{
		capHint: capHint,
	}
}

func (c *forwardPortOwnerCache) Get(port uint) (string, bool) {
	if port == 0 {
		return "", false
	}
	if c.byPort != nil {
		row, ok := c.byPort[port]
		if !ok {
			return "", false
		}
		return c.rows[row].owner, true
	}
	for _, row := range c.rows {
		if row.port == port {
			return row.owner, true
		}
	}
	return "", false
}

func (c *forwardPortOwnerCache) Set(port uint, owner string) {
	if port == 0 {
		return
	}
	if c.byPort != nil {
		if row, ok := c.byPort[port]; ok {
			c.rows[row].owner = owner
			return
		}
		c.byPort[port] = len(c.rows)
		c.rows = append(c.rows, forwardPortOwnerEntry{port: port, owner: owner})
		return
	}
	for row := range c.rows {
		if c.rows[row].port == port {
			c.rows[row].owner = owner
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, forwardPortOwnerEntry{port: port, owner: owner})
	if len(c.rows) > smallForwardUniqueThreshold {
		c.upgradeLookup()
	}
}

func (c forwardPortOwnerCache) Len() int {
	return len(c.rows)
}

func (c *forwardPortOwnerCache) ensureRows() {
	if c.rows != nil || c.capHint <= 0 {
		return
	}
	c.rows = make([]forwardPortOwnerEntry, 0, forwardInitialCap(c.capHint))
}

func (c *forwardPortOwnerCache) upgradeLookup() {
	if c.byPort != nil {
		return
	}
	nextCap := forwardLookupCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]forwardPortOwnerEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.byPort = make(map[uint]int, nextCap)
	for row, item := range c.rows {
		c.byPort[item.port] = row
	}
}

type forwardHostIPCacheEntry struct {
	host string
	ips  []string
}

type forwardHostIPCache struct {
	rows    []forwardHostIPCacheEntry
	byHost  map[string]int
	capHint int
}

func newForwardHostIPCache(capHint int) forwardHostIPCache {
	if capHint <= 0 {
		return forwardHostIPCache{}
	}
	return forwardHostIPCache{
		capHint: capHint,
	}
}

func (c *forwardHostIPCache) Get(host string) ([]string, bool) {
	if host == "" {
		return nil, false
	}
	if c.byHost != nil {
		row, ok := c.byHost[host]
		if !ok {
			return nil, false
		}
		return c.rows[row].ips, true
	}
	for _, row := range c.rows {
		if row.host == host {
			return row.ips, true
		}
	}
	return nil, false
}

func (c *forwardHostIPCache) Set(host string, ips []string) {
	if host == "" {
		return
	}
	if c.byHost != nil {
		if row, ok := c.byHost[host]; ok {
			c.rows[row].ips = ips
			return
		}
		c.byHost[host] = len(c.rows)
		c.rows = append(c.rows, forwardHostIPCacheEntry{host: host, ips: ips})
		return
	}
	for row := range c.rows {
		if c.rows[row].host == host {
			c.rows[row].ips = ips
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, forwardHostIPCacheEntry{host: host, ips: ips})
	if len(c.rows) > smallForwardUniqueThreshold {
		c.upgradeLookup()
	}
}

func (c forwardHostIPCache) Len() int {
	return len(c.rows)
}

func (c *forwardHostIPCache) ensureRows() {
	if c.rows != nil || c.capHint <= 0 {
		return
	}
	c.rows = make([]forwardHostIPCacheEntry, 0, forwardInitialCap(c.capHint))
}

func (c *forwardHostIPCache) upgradeLookup() {
	if c.byHost != nil {
		return
	}
	nextCap := forwardLookupCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]forwardHostIPCacheEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.byHost = make(map[string]int, nextCap)
	for row, item := range c.rows {
		c.byHost[item.host] = row
	}
}

func resolveHostIPsCached(ctx context.Context, cache *forwardHostIPCache, host string) ([]string, error) {
	normalized := normalizeHost(host)
	if normalized == "" {
		return nil, fmt.Errorf("empty host")
	}
	if ips, ok := cache.Get(normalized); ok {
		return ips, nil
	}
	ips, err := resolveHostIPs(ctx, normalized)
	if err != nil {
		return nil, err
	}
	cache.Set(normalized, ips)
	return ips, nil
}

func resolveHostIPs(ctx context.Context, host string) ([]string, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []string{ip.String()}, nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(addrs))
	var seen map[string]struct{}
	for _, addr := range addrs {
		ip := addr.IP.String()
		ips, seen = appendUniqueForwardIP(ips, ip, seen)
	}
	sort.Strings(ips)
	if len(ips) == 0 {
		return nil, fmt.Errorf("no ip records")
	}
	return ips, nil
}

func appendUniqueForwardIP(values []string, value string, seen map[string]struct{}) ([]string, map[string]struct{}) {
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
	for _, existing := range values {
		if existing == value {
			return values, seen
		}
	}
	values = append(values, value)
	if len(values) > smallForwardUniqueThreshold {
		seen = make(map[string]struct{}, len(values))
		for _, existing := range values {
			seen[existing] = struct{}{}
		}
	}
	return values, seen
}

func forwardInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallForwardUniqueThreshold+1 {
		return smallForwardUniqueThreshold + 1
	}
	return capHint
}

func forwardLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func normalizeHost(value string) string {
	host := strings.TrimSpace(value)
	if host == "" {
		return ""
	}
	if strings.Contains(host, "://") {
		if parsed, err := url.Parse(host); err == nil && parsed.Host != "" {
			host = parsed.Host
		}
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	return strings.Trim(strings.TrimSpace(host), "[]")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func configString(cfg xraytemplate.ConfigObject, key string) string {
	value, ok := cfg[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func configUint(cfg xraytemplate.ConfigObject, key string) uint {
	value, ok := cfg[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case uint:
		return v
	case uint16:
		return uint(v)
	case uint32:
		return uint(v)
	case uint64:
		return uint(v)
	case int:
		if v < 0 {
			return 0
		}
		return uint(v)
	case int32:
		if v < 0 {
			return 0
		}
		return uint(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint(v)
	case float64:
		if v < 0 {
			return 0
		}
		return uint(v)
	case json.Number:
		n, _ := strconv.ParseUint(string(v), 10, 64)
		return uint(n)
	default:
		return 0
	}
}
