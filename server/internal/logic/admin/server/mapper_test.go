package server

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestServerListItemFromModelMapsListShapeWithoutReflection(t *testing.T) {
	lastReportedAt := time.UnixMilli(1716192000000)
	createdAt := time.UnixMilli(1716195600000)
	updatedAt := time.UnixMilli(1716199200000)

	got := serverListItemFromModel(&node.Server{
		Id:             11,
		Name:           "server-a",
		Country:        "US",
		City:           "Denver",
		Address:        "192.0.2.10",
		Sort:           3,
		LastReportedAt: &lastReportedAt,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	})

	if got.Id != 11 || got.Name != "server-a" || got.Address != "192.0.2.10" || got.Sort != 3 {
		t.Fatalf("server mapping mismatch: %+v", got)
	}
	if got.LastReportedAt != lastReportedAt.UnixMilli() || got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("server timestamp mapping mismatch: %+v", got)
	}
	if got.Status.Status != "offline" {
		t.Fatalf("default status should be offline: %+v", got.Status)
	}
}

func TestServerRowsToListItemsSkipsNilAndAppliesRealtimeStatus(t *testing.T) {
	rows := []*node.Server{
		nil,
		{Id: 7, Name: "server-a"},
		{Id: 8, Name: "server-b"},
	}
	got := serverRowsToListItems(rows, func(serverID int64) (realtime.NodeStatus, bool) {
		if serverID != 7 {
			return realtime.NodeStatus{}, false
		}
		return realtime.NodeStatus{Status: "online", CPU: 1.5, OnlineUsers: 3}, true
	})

	if len(got) != 2 {
		t.Fatalf("server rows len = %d, want 2", len(got))
	}
	if got[0].Id != 7 || got[0].Status.Status != "online" || got[0].Status.Cpu != 1.5 || got[0].Status.OnlineUsers != 3 {
		t.Fatalf("first server status not applied: %#v", got[0])
	}
	if got[1].Id != 8 || got[1].Status.Status != "offline" {
		t.Fatalf("second server should keep default offline status: %#v", got[1])
	}

	empty := serverRowsToListItems(nil, nil)
	if empty == nil {
		t.Fatal("empty server rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.ServerListItem `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty server rows marshal = %s, want list []", payload)
	}
}

func TestServerRowsToListItemsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*node.Server, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &node.Server{Id: 1, Name: "first"}
	rows[len(rows)-1] = &node.Server{Id: 2, Name: "second"}

	got := serverRowsToListItems(rows, nil)
	if len(got) != 2 {
		t.Fatalf("server rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy server rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("server row order changed: %#v", got)
	}
}

func TestServerRowsToListItemsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*node.Server, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &node.Server{Id: int64(index + 1), Name: "server"}
	}

	got := serverRowsToListItems(rows, nil)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("server rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("server rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("server row order changed: %#v", got)
	}
}

func TestNodeRowsToListItemsSkipsNilAndNormalizesResponse(t *testing.T) {
	enabled := true
	createdAt := time.UnixMilli(1716192000000)
	updatedAt := time.UnixMilli(1716195600000)
	rows := []*node.Node{
		{
			Id:           9,
			Name:         "node-a",
			Tags:         " edge, premium,,edge ",
			Port:         443,
			Address:      "node.example.com",
			ServerId:     3,
			Protocol:     "vless",
			InboundAlias: "VLESS",
			Enabled:      &enabled,
			Sort:         2,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		},
		nil,
		{
			Id:       10,
			Name:     "node-b",
			Tags:     " edge, premium,,edge ",
			Protocol: "trojan",
		},
	}

	got := nodeRowsToListItems(rows)
	if len(got) != 2 {
		t.Fatalf("node rows = %d, want 2: %#v", len(got), got)
	}
	if got[0].Id != 9 || got[0].Name != "node-a" || got[0].Port != 443 || got[0].Address != "node.example.com" || got[0].ServerId != 3 {
		t.Fatalf("node fields not mapped: %#v", got[0])
	}
	if got[0].InboundAlias != "VLESS" || got[0].Protocol != "VLESS" {
		t.Fatalf("inbound alias should be normalized from model inbound: %#v", got[0])
	}
	if !reflect.DeepEqual(got[0].Tags, []string{"edge", "premium"}) {
		t.Fatalf("tags = %#v, want normalized unique tags", got[0].Tags)
	}
	if !got[0].Enabled || got[0].Sort != 2 || got[0].CreatedAt != createdAt.UnixMilli() || got[0].UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("node metadata not mapped: %#v", got[0])
	}
	if got[1].InboundAlias != "trojan" || got[1].Protocol != "trojan" {
		t.Fatalf("protocol should fall back to model protocol: %#v", got[1])
	}
	if !reflect.DeepEqual(got[1].Tags, got[0].Tags) {
		t.Fatalf("duplicate node tags = %#v, want %#v", got[1].Tags, got[0].Tags)
	}
	if !got[1].Enabled {
		t.Fatalf("nil node enabled should default true in API response: %#v", got[1])
	}
	if got[1].CreatedAt != 0 || got[1].UpdatedAt != 0 {
		t.Fatalf("zero timestamps should stay zero: %#v", got[1])
	}
}

func TestNodeRowsToListItemsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*node.Node, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &node.Node{Id: 1, Name: "first"}
	rows[len(rows)-1] = &node.Node{Id: 2, Name: "second"}

	got := nodeRowsToListItems(rows)
	if len(got) != 2 {
		t.Fatalf("node rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy node rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("node row order changed: %#v", got)
	}
}

func TestNodeRowsToListItemsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*node.Node, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &node.Node{Id: int64(index + 1), Name: "node"}
	}

	got := nodeRowsToListItems(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("node rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("node rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("node row order changed: %#v", got)
	}
}

func TestNodeRowsToListItemsKeepsEmptyListShape(t *testing.T) {
	got := nodeRowsToListItems(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty node rows should return non-nil empty slice: %#v", got)
	}
}

func TestNodeTagsCacheReusesRawTags(t *testing.T) {
	cache := newNodeTagsCache(2)

	first := cache.Tags("edge, premium,,edge")
	second := cache.Tags("edge, premium,,edge")

	if !reflect.DeepEqual(first, []string{"edge", "premium"}) || !reflect.DeepEqual(second, first) {
		t.Fatalf("cached node tags mismatch: first=%#v second=%#v", first, second)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate raw node tags should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small node tags cache should not allocate map")
	}
}

func TestNodeTagsCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newNodeTagsCache(smallNodeTagsCacheThreshold + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large node tags cache should not preallocate storage from hint alone: %#v", cache)
	}
	first := cache.Tags("edge, premium,,edge")
	second := cache.Tags("edge, premium,,edge")
	if !reflect.DeepEqual(first, second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate node tags should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newNodeTagsCache(smallNodeTagsCacheThreshold + 1)
	for i := 0; i <= smallNodeTagsCacheThreshold; i++ {
		cache.Tags(string(rune('a' + i)))
	}
	cache.Tags("a")

	if len(cache.rows) != smallNodeTagsCacheThreshold+1 || len(cache.index) != smallNodeTagsCacheThreshold+1 {
		t.Fatalf("node tags cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), smallNodeTagsCacheThreshold+1)
	}
	if !reflect.DeepEqual(cache.rows[0].tags, []string{"a"}) {
		t.Fatalf("node tags cache did not normalize first row: %#v", cache.rows[0].tags)
	}
}

func TestNodeTagsCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newNodeTagsCache(smallNodeTagsCacheThreshold + 20)
	for i := 0; i <= smallNodeTagsCacheThreshold; i++ {
		cache.Tags(string(rune('a' + i)))
	}
	if cache.index == nil {
		t.Fatal("node tags cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("node tags cache row cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallNodeTagsCacheThreshold+20; i++ {
		cache.Tags("a")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("node tags cache row cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestServerStatusFromRealtimeMapsRuntimeMetrics(t *testing.T) {
	got := serverStatusFromRealtime(realtime.NodeStatus{
		CPU:              1.5,
		Mem:              2.5,
		Disk:             3.5,
		Status:           "online",
		NetRxBps:         10,
		NetTxBps:         11,
		XrayRunning:      true,
		XrayVersion:      "25.1.1",
		OnlineUsers:      42,
		LastSeen:         1716192000000,
		Load1:            0.1,
		ConfigSyncStatus: "synced",
		LastConfigError:  "none",
	})
	if got.Status != "online" || got.Cpu != 1.5 || got.NetRxBps != 10 || got.OnlineUsers != 42 {
		t.Fatalf("server status mapping mismatch: %+v", got)
	}
	if !got.XrayRunning || got.XrayVersion != "25.1.1" || got.ConfigSyncStatus != "synced" {
		t.Fatalf("xray/config status mapping mismatch: %+v", got)
	}
	if got.LastSeen != 1716192000000 || got.Load1 != 0.1 || got.LastConfigError != "none" {
		t.Fatalf("extended status mapping mismatch: %+v", got)
	}
}
