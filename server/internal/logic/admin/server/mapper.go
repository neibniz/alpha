package server

import (
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/tool"
)

const smallNodeTagsCacheThreshold = 8

func serverListItemFromModel(item *node.Server) types.ServerListItem {
	if item == nil {
		return types.ServerListItem{}
	}
	resp := types.ServerListItem{
		Id:            item.Id,
		Name:          item.Name,
		Country:       item.Country,
		City:          item.City,
		Address:       item.Address,
		Sort:          item.Sort,
		NodeCount:     item.NodeCount,
		TemplateCount: item.TemplateCount,
		Status:        types.ServerStatus{Status: "offline"},
		CreatedAt:     timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:     timex.UnixMilliOrZero(item.UpdatedAt),
	}
	if item.LastReportedAt != nil {
		resp.LastReportedAt = timex.UnixMilliOrZero(*item.LastReportedAt)
	}
	return resp
}

type serverStatusLookup func(serverID int64) (realtime.NodeStatus, bool)

func serverRowsToListItems(rows []*node.Server, lookup serverStatusLookup) []types.ServerListItem {
	if len(rows) == 0 {
		return []types.ServerListItem{}
	}
	collector := viewutil.NewRowsCollector[types.ServerListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		item := serverListItemFromModel(row)
		if lookup != nil {
			if status, ok := lookup(row.Id); ok {
				item.Status = serverStatusFromRealtime(status)
			}
		}
		collector.Add(item)
	}
	return collector.Rows()
}

func nodeListItemFromModel(item *node.Node) types.NodeListItem {
	return nodeListItemFromModelWithTagsCache(item, nil)
}

func nodeListItemFromModelWithTagsCache(item *node.Node, tags *nodeTagsCache) types.NodeListItem {
	if item == nil {
		return types.NodeListItem{}
	}
	inbound := item.Inbound()
	return types.NodeListItem{
		Id:           item.Id,
		Name:         item.Name,
		Tags:         tags.Tags(item.Tags),
		Port:         item.Port,
		Address:      item.Address,
		ServerId:     item.ServerId,
		InboundAlias: inbound,
		Protocol:     inbound,
		Enabled:      boolPtrOrDefault(item.Enabled, true),
		Sort:         item.Sort,
		CreatedAt:    timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:    timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func nodeRowsToListItems(rows []*node.Node) []types.NodeListItem {
	if len(rows) == 0 {
		return []types.NodeListItem{}
	}
	collector := viewutil.NewRowsCollector[types.NodeListItem](len(rows))
	tags := newNodeTagsCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(nodeListItemFromModelWithTagsCache(row, tags))
	}
	return collector.Rows()
}

func serverStatusFromRealtime(state realtime.NodeStatus) types.ServerStatus {
	return types.ServerStatus{
		Cpu:                       state.CPU,
		Mem:                       state.Mem,
		Disk:                      state.Disk,
		Status:                    state.Status,
		NetRxBps:                  state.NetRxBps,
		NetTxBps:                  state.NetTxBps,
		NetRxBytes:                state.NetRxBytes,
		NetTxBytes:                state.NetTxBytes,
		XrayRxBps:                 state.XrayRxBps,
		XrayTxBps:                 state.XrayTxBps,
		XrayRxBytes:               state.XrayRxBytes,
		XrayTxBytes:               state.XrayTxBytes,
		XrayStatsError:            state.XrayStatsError,
		Connections:               state.Connections,
		SystemConnections:         state.SystemConnections,
		SystemInboundConnections:  state.SystemInboundConnections,
		SystemOutboundConnections: state.SystemOutboundConnections,
		XrayConnections:           state.XrayConnections,
		XrayInboundConnections:    state.XrayInboundConnections,
		XrayOutboundConnections:   state.XrayOutboundConnections,
		OnlineUsers:               state.OnlineUsers,
		XrayRunning:               state.XrayRunning,
		XrayVersion:               state.XrayVersion,
		UpdatedAt:                 state.UpdatedAt,
		LastSeen:                  state.LastSeen,
		Load1:                     state.Load1,
		Load5:                     state.Load5,
		Load15:                    state.Load15,
		Uptime:                    state.Uptime,
		AgentVersion:              state.AgentVersion,
		ConfigVersion:             state.ConfigVersion,
		ConfigSyncStatus:          state.ConfigSyncStatus,
		ConfigApplyStatus:         state.ConfigApplyStatus,
		LastConfigHash:            state.LastConfigHash,
		RunningConfigHash:         state.RunningConfigHash,
		PendingConfigHash:         state.PendingConfigHash,
		LastSyncAt:                state.LastSyncAt,
		LastApplyAt:               state.LastApplyAt,
		LastConfigError:           state.LastConfigError,
		LastApplyError:            state.LastApplyError,
	}
}

type nodeTagsCacheEntry struct {
	raw  string
	tags []string
}

type nodeTagsCache struct {
	rows    []nodeTagsCacheEntry
	index   map[string]int
	capHint int
}

func newNodeTagsCache(capHint int) *nodeTagsCache {
	if capHint <= 1 {
		return nil
	}
	return &nodeTagsCache{capHint: capHint}
}

func (c *nodeTagsCache) Tags(raw string) []string {
	if c == nil {
		return tool.StringMergeAndRemoveDuplicates(raw)
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

func (c *nodeTagsCache) add(raw string) []string {
	tags := tool.StringMergeAndRemoveDuplicates(raw)
	c.ensureRows()
	c.rows = append(c.rows, nodeTagsCacheEntry{raw: raw, tags: tags})
	if c.index != nil {
		c.index[raw] = len(c.rows) - 1
	} else if len(c.rows) > smallNodeTagsCacheThreshold {
		nextCap := adminServerLookupCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]nodeTagsCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return tags
}

func (c *nodeTagsCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallNodeTagsCacheThreshold
	} else if capHint > smallNodeTagsCacheThreshold+1 {
		capHint = smallNodeTagsCacheThreshold + 1
	}
	c.rows = make([]nodeTagsCacheEntry, 0, capHint)
}

func adminServerLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
