package realtime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNodeStatusMarshalIncludesEmptyClearableStatusFields(t *testing.T) {
	data, err := json.Marshal(NodeStatus{ServerID: 1})
	if err != nil {
		t.Fatalf("marshal node status: %v", err)
	}
	if !strings.Contains(string(data), `"xray_stats_error":""`) {
		t.Fatalf("node status does not clear xray stats error: %s", data)
	}
	if !strings.Contains(string(data), `"pending_config_hash":""`) {
		t.Fatalf("node status does not clear pending config hash: %s", data)
	}
	if !strings.Contains(string(data), `"last_config_error":""`) {
		t.Fatalf("node status does not clear last config error: %s", data)
	}
	if !strings.Contains(string(data), `"last_apply_error":""`) {
		t.Fatalf("node status does not clear last apply error: %s", data)
	}
}

func TestNodeStatusClearsXrayStatsErrorWhenOffline(t *testing.T) {
	manager := NewManager()
	state := manager.Update(NodeStatus{
		ServerID:         1,
		XrayRunning:      true,
		XrayStatsError:   "stats timeout",
		UpdatedAt:        time.Now().UnixMilli(),
		ConfigSyncStatus: "ok",
	})
	if state.XrayStatsError == "" {
		t.Fatalf("xray stats error should be preserved while node is online")
	}

	manager.MarkOffline(1)
	state, ok := manager.Get(1)
	if !ok {
		t.Fatalf("node status missing")
	}
	if state.XrayStatsError != "" {
		t.Fatalf("offline node should clear xray stats error, got %q", state.XrayStatsError)
	}
}

func TestNodeStatusClearsXrayStatsErrorWhenXrayStopped(t *testing.T) {
	manager := NewManager()
	state := manager.Update(NodeStatus{
		ServerID:       1,
		XrayRunning:    false,
		XrayStatsError: "stats timeout",
		UpdatedAt:      time.Now().UnixMilli(),
	})
	if state.XrayStatsError != "" {
		t.Fatalf("stopped xray should clear xray stats error, got %q", state.XrayStatsError)
	}
}

func TestNodeStatusUpdateUsesServerTimestamp(t *testing.T) {
	manager := NewManager()
	before := time.Now().UnixMilli()
	state := manager.Update(NodeStatus{ServerID: 1, UpdatedAt: 1})
	if state.UpdatedAt < before {
		t.Fatalf("updated_at = %d, want server timestamp >= %d", state.UpdatedAt, before)
	}
}

func TestNodeStatusUpdateUsesSubscriberServerIndex(t *testing.T) {
	manager := newTestNodeStatusManager()
	target := newTestNodeStatusSubscriber(&websocket.Conn{}, 1)
	global := newTestNodeStatusSubscriber(&websocket.Conn{})
	other := newTestNodeStatusSubscriber(&websocket.Conn{}, 2)

	manager.mu.Lock()
	manager.addSubscriberLocked(target)
	manager.addSubscriberLocked(global)
	manager.addSubscriberLocked(other)
	manager.mu.Unlock()

	manager.Update(NodeStatus{ServerID: 1})

	assertNodeStatusPayload(t, target, 1)
	assertNodeStatusPayload(t, global, 1)
	assertNoNodeStatusPayload(t, other)
}

func TestNodeStatusSetSubscriptionMaintainsIndexes(t *testing.T) {
	manager := newTestNodeStatusManager()
	conn := &websocket.Conn{}
	sub := newTestNodeStatusSubscriber(conn, 1)

	manager.mu.Lock()
	manager.addSubscriberLocked(sub)
	manager.mu.Unlock()

	manager.SetSubscription(conn, []int64{2})

	manager.mu.Lock()
	serverOneSubscribers := manager.subscribersForServerLocked(1)
	serverTwoSubscribers := manager.subscribersForServerLocked(2)
	manager.mu.Unlock()

	if containsNodeStatusSubscriber(serverOneSubscribers, sub) {
		t.Fatal("subscriber remained indexed under old server id")
	}
	if !containsNodeStatusSubscriber(serverTwoSubscribers, sub) {
		t.Fatal("subscriber was not indexed under new server id")
	}
}

func TestNodeStatusSubscriberSnapshotAvoidsSliceForSingleSubscriber(t *testing.T) {
	manager := newTestNodeStatusManager()
	sub := newTestNodeStatusSubscriber(&websocket.Conn{}, 1)

	manager.mu.Lock()
	manager.addSubscriberLocked(sub)
	snapshot := manager.subscribersForServerLocked(1)
	manager.mu.Unlock()

	if snapshot.Len() != 1 {
		t.Fatalf("snapshot len = %d, want 1", snapshot.Len())
	}
	if snapshot.single != sub {
		t.Fatalf("snapshot single subscriber = %#v, want %#v", snapshot.single, sub)
	}
	if snapshot.items != nil {
		t.Fatalf("single subscriber snapshot should not allocate slice: %#v", snapshot.items)
	}
	if !containsNodeStatusSubscriber(snapshot, sub) {
		t.Fatal("single subscriber snapshot should contain subscriber")
	}
}

func TestNodeStatusSubscriberSnapshotAllocatesOnlyForMultipleSubscribers(t *testing.T) {
	manager := newTestNodeStatusManager()
	global := newTestNodeStatusSubscriber(&websocket.Conn{})
	target := newTestNodeStatusSubscriber(&websocket.Conn{}, 1)
	other := newTestNodeStatusSubscriber(&websocket.Conn{}, 2)

	manager.mu.Lock()
	manager.addSubscriberLocked(global)
	manager.addSubscriberLocked(target)
	manager.addSubscriberLocked(other)
	snapshot := manager.subscribersForServerLocked(1)
	manager.mu.Unlock()

	if snapshot.single != nil {
		t.Fatalf("multi subscriber snapshot should not use single field: %#v", snapshot.single)
	}
	if snapshot.Len() != 2 {
		t.Fatalf("snapshot len = %d, want 2", snapshot.Len())
	}
	if cap(snapshot.items) != 2 {
		t.Fatalf("snapshot items cap = %d, want 2", cap(snapshot.items))
	}
	if !containsNodeStatusSubscriber(snapshot, global) || !containsNodeStatusSubscriber(snapshot, target) {
		t.Fatalf("snapshot missing expected subscribers: %#v", snapshot.items)
	}
	if containsNodeStatusSubscriber(snapshot, other) {
		t.Fatal("snapshot should not contain unrelated server subscriber")
	}
}

func TestNodeStatusNotificationListAvoidsAllocationForEmptyAndSingle(t *testing.T) {
	var empty nodeStatusNotificationList
	if empty.Len() != 0 {
		t.Fatalf("empty notification list len = %d, want 0", empty.Len())
	}
	visited := 0
	empty.ForEach(func(nodeStatusNotification) {
		visited++
	})
	if visited != 0 {
		t.Fatalf("empty notification list visited = %d, want 0", visited)
	}

	list := newNodeStatusNotificationList(8)
	list.Add(nodeStatusNotification{state: NodeStatus{ServerID: 1}})
	if list.Len() != 1 {
		t.Fatalf("single notification list len = %d, want 1", list.Len())
	}
	if !list.hasSingle || list.single.state.ServerID != 1 {
		t.Fatalf("single notification not stored inline: %#v", list)
	}
	if list.items != nil {
		t.Fatalf("single notification should not allocate slice: %#v", list.items)
	}
}

func TestNodeStatusNotificationListAllocatesOnlyForMultipleNotifications(t *testing.T) {
	list := newNodeStatusNotificationList(3)
	list.Add(nodeStatusNotification{state: NodeStatus{ServerID: 1}})
	list.Add(nodeStatusNotification{state: NodeStatus{ServerID: 2}})
	list.Add(nodeStatusNotification{state: NodeStatus{ServerID: 3}})

	if list.hasSingle {
		t.Fatalf("multi notification list should not keep inline single: %#v", list.single)
	}
	if list.Len() != 3 {
		t.Fatalf("notification list len = %d, want 3", list.Len())
	}
	if cap(list.items) != 3 {
		t.Fatalf("notification list cap = %d, want 3", cap(list.items))
	}
	var ids []int64
	list.ForEach(func(notification nodeStatusNotification) {
		ids = append(ids, notification.state.ServerID)
	})
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Fatalf("notification ids = %#v, want [1 2 3]", ids)
	}
}

func TestNodeStatusSendSnapshotUsesConnectionIndex(t *testing.T) {
	manager := newTestNodeStatusManager()
	conn := &websocket.Conn{}
	sub := newTestNodeStatusSubscriber(conn, 1)
	manager.states[1] = NodeStatus{ServerID: 1, LastSeen: time.Now().UnixMilli(), Online: true}

	manager.mu.Lock()
	manager.addSubscriberLocked(sub)
	manager.mu.Unlock()

	manager.SendSnapshot(conn, []int64{1})

	select {
	case payload := <-sub.send:
		snapshot, ok := payload.(NodeStatusSnapshot)
		if !ok {
			t.Fatalf("payload = %T, want NodeStatusSnapshot", payload)
		}
		if len(snapshot.Data) != 1 || snapshot.Data[0].ServerID != 1 {
			t.Fatalf("snapshot data = %#v, want server 1", snapshot.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("snapshot was not enqueued")
	}
}

func TestNodeStatusSnapshotUsesRequestedIDOrderAndDeduplicates(t *testing.T) {
	manager := newTestNodeStatusManager()
	now := time.Now().UnixMilli()
	manager.states[1] = NodeStatus{ServerID: 1, LastSeen: now}
	manager.states[2] = NodeStatus{ServerID: 2, LastSeen: now}
	manager.states[3] = NodeStatus{ServerID: 3, LastSeen: now}

	snapshot := manager.Snapshot([]int64{3, 1, 3, 99, -1})
	if len(snapshot) != 2 {
		t.Fatalf("snapshot len = %d, want 2", len(snapshot))
	}
	if snapshot[0].ServerID != 3 || snapshot[1].ServerID != 1 {
		t.Fatalf("snapshot server ids = [%d %d], want [3 1]", snapshot[0].ServerID, snapshot[1].ServerID)
	}
	if !snapshot[0].Online || !snapshot[1].Online {
		t.Fatalf("snapshot should normalize recent states online: %#v", snapshot)
	}
}

func BenchmarkNodeStatusSnapshotFiltered(b *testing.B) {
	manager := newTestNodeStatusManager()
	now := time.Now().UnixMilli()
	for id := int64(1); id <= 10_000; id++ {
		manager.states[id] = NodeStatus{ServerID: id, LastSeen: now}
	}
	serverIDs := []int64{9_999, 5_000, 1}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := manager.Snapshot(serverIDs); len(got) != len(serverIDs) {
			b.Fatalf("snapshot len = %d, want %d", len(got), len(serverIDs))
		}
	}
}

func BenchmarkNodeStatusSubscriberSnapshot(b *testing.B) {
	manager := newTestNodeStatusManager()
	for i := 0; i < 1_000; i++ {
		manager.addSubscriberLocked(newTestNodeStatusSubscriber(&websocket.Conn{}, 1))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := manager.subscribersForServerLocked(1); got.Len() != 1_000 {
			b.Fatalf("subscriber snapshot len = %d, want 1000", got.Len())
		}
	}
}

func TestMakeServerIDSetUsesEmptySetForGlobalSubscription(t *testing.T) {
	if got := makeServerIDSet(nil); got.Len() != 0 {
		t.Fatalf("empty server id set = %#v, want empty", got)
	}
	if got := makeServerIDSet([]int64{0, -1}); got.Len() != 0 {
		t.Fatalf("invalid-only server id set = %#v, want empty", got)
	}

	got := makeServerIDSet([]int64{3, 3, -1, 4})
	if got.Len() != 2 {
		t.Fatalf("server id set length = %d, want 2", got.Len())
	}
	if !got.Contains(3) {
		t.Fatal("server id 3 missing from set")
	}
	if !got.Contains(4) {
		t.Fatal("server id 4 missing from set")
	}
}

func TestMakeServerIDSetUsesLinearPathForSmallInput(t *testing.T) {
	got := makeServerIDSet([]int64{3, 3, -1, 4})
	if got.index != nil {
		t.Fatal("small server id set should not allocate map")
	}
	if got.Len() != 2 || !got.Contains(3) || !got.Contains(4) || got.Contains(5) {
		t.Fatalf("small server id set mismatch: %#v", got)
	}
}

func TestMakeServerIDSetUpgradesLargeInputToMap(t *testing.T) {
	ids := make([]int64, 0, smallServerIDSetThreshold+1)
	for id := int64(1); id <= smallServerIDSetThreshold+1; id++ {
		ids = append(ids, id)
	}
	got := makeServerIDSet(ids)
	if cap(got.ids) != smallServerIDSetThreshold+1 {
		t.Fatalf("large server id set cap = %d, want %d", cap(got.ids), smallServerIDSetThreshold+1)
	}
	if got.index == nil {
		t.Fatal("large server id set should allocate map")
	}
	if got.Len() != smallServerIDSetThreshold+1 || !got.Contains(1) || !got.Contains(int64(smallServerIDSetThreshold+1)) {
		t.Fatalf("large server id set mismatch: %#v", got)
	}
}

func TestServerIDSetBoundsSparseUpgradeCapacity(t *testing.T) {
	set := serverIDSet{capHint: smallServerIDSetThreshold + 20}
	if set.ids != nil || set.index != nil {
		t.Fatalf("hinted server id set should not allocate before first id: %#v", set)
	}
	set.Add(1)
	set.Add(1)
	if set.index != nil || set.Len() != 1 {
		t.Fatalf("duplicate first id should stay on linear path: %#v", set)
	}
	for id := int64(2); id <= int64(smallServerIDSetThreshold+1); id++ {
		set.Add(id)
	}
	if set.index == nil {
		t.Fatal("server id set should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := set.Len() * 2; cap(set.ids) > maxCap {
		t.Fatalf("server id set cap = %d, want <= %d", cap(set.ids), maxCap)
	}
}

func newTestNodeStatusManager() *Manager {
	return &Manager{
		states:      make(map[int64]NodeStatus),
		subscribers: make(map[*subscriber]struct{}),
	}
}

func newTestNodeStatusSubscriber(conn *websocket.Conn, serverIDs ...int64) *subscriber {
	return &subscriber{
		conn:      conn,
		serverIDs: makeServerIDSet(serverIDs),
		send:      make(chan nodeStatusPayload, 4),
		done:      make(chan struct{}),
	}
}

func assertNodeStatusPayload(t *testing.T, sub *subscriber, serverID int64) {
	t.Helper()
	select {
	case payload := <-sub.send:
		state, ok := payload.(NodeStatus)
		if !ok {
			t.Fatalf("payload = %T, want NodeStatus", payload)
		}
		if state.ServerID != serverID {
			t.Fatalf("payload server_id = %d, want %d", state.ServerID, serverID)
		}
	case <-time.After(time.Second):
		t.Fatalf("subscriber did not receive server %d update", serverID)
	}
}

func assertNoNodeStatusPayload(t *testing.T, sub *subscriber) {
	t.Helper()
	select {
	case payload := <-sub.send:
		t.Fatalf("unexpected payload = %#v", payload)
	default:
	}
}

func containsNodeStatusSubscriber(subscribers nodeStatusSubscriberSnapshot, target *subscriber) bool {
	found := false
	subscribers.ForEach(func(sub *subscriber) {
		if sub == target {
			found = true
		}
	})
	return found
}
