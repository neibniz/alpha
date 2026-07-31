package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const NodeStatusTTL = 30 * time.Second
const nodeStatusExpireInterval = 5 * time.Second

type NodeStatus struct {
	Type                      string  `json:"type,omitempty"`
	ServerID                  int64   `json:"server_id"`
	Online                    bool    `json:"online"`
	CPU                       float64 `json:"cpu"`
	Mem                       float64 `json:"mem"`
	Disk                      float64 `json:"disk"`
	Load1                     float64 `json:"load1,omitempty"`
	Load5                     float64 `json:"load5,omitempty"`
	Load15                    float64 `json:"load15,omitempty"`
	Uptime                    int64   `json:"uptime,omitempty"`
	XrayRunning               bool    `json:"xray_running"`
	XrayVersion               string  `json:"xray_version,omitempty"`
	Connections               int64   `json:"connections,omitempty"`
	SystemConnections         int64   `json:"system_connections,omitempty"`
	SystemInboundConnections  int64   `json:"system_inbound_connections,omitempty"`
	SystemOutboundConnections int64   `json:"system_outbound_connections,omitempty"`
	XrayConnections           int64   `json:"xray_connections,omitempty"`
	XrayInboundConnections    int64   `json:"xray_inbound_connections,omitempty"`
	XrayOutboundConnections   int64   `json:"xray_outbound_connections,omitempty"`
	OnlineUsers               int64   `json:"online_users,omitempty"`
	NetRxBps                  int64   `json:"net_rx_bps,omitempty"`
	NetTxBps                  int64   `json:"net_tx_bps,omitempty"`
	NetRxBytes                int64   `json:"net_rx_bytes,omitempty"`
	NetTxBytes                int64   `json:"net_tx_bytes,omitempty"`
	XrayRxBps                 int64   `json:"xray_rx_bps,omitempty"`
	XrayTxBps                 int64   `json:"xray_tx_bps,omitempty"`
	XrayRxBytes               int64   `json:"xray_rx_bytes,omitempty"`
	XrayTxBytes               int64   `json:"xray_tx_bytes,omitempty"`
	XrayStatsError            string  `json:"xray_stats_error"`
	UpdatedAt                 int64   `json:"updated_at"`
	LastSeen                  int64   `json:"last_seen"`
	Status                    string  `json:"status"`
	MemoryUsed                int64   `json:"memory_used,omitempty"`
	MemoryTotal               int64   `json:"memory_total,omitempty"`
	DiskUsed                  int64   `json:"disk_used,omitempty"`
	DiskTotal                 int64   `json:"disk_total,omitempty"`
	AgentVersion              string  `json:"agent_version,omitempty"`
	ConfigVersion             string  `json:"config_version,omitempty"`
	ConfigSyncStatus          string  `json:"config_sync_status,omitempty"`
	ConfigApplyStatus         string  `json:"config_apply_status,omitempty"`
	LastConfigHash            string  `json:"last_config_hash,omitempty"`
	RunningConfigHash         string  `json:"running_config_hash,omitempty"`
	PendingConfigHash         string  `json:"pending_config_hash"`
	LastSyncAt                int64   `json:"last_sync_at,omitempty"`
	LastApplyAt               int64   `json:"last_apply_at,omitempty"`
	LastConfigError           string  `json:"last_config_error"`
	LastApplyError            string  `json:"last_apply_error"`
}

func (NodeStatus) realtimePayload() {}

type NodeStatusSnapshot struct {
	Type string       `json:"type"`
	Data []NodeStatus `json:"data"`
}

func (NodeStatusSnapshot) realtimePayload() {}

type nodeStatusPayload interface {
	realtimePayload()
}

type Manager struct {
	mu                    sync.RWMutex
	states                map[int64]NodeStatus
	subscribers           map[*subscriber]struct{}
	subscribersByConn     map[*websocket.Conn]*subscriber
	subscribersByServerID map[int64]map[*subscriber]struct{}
	globalSubscribers     map[*subscriber]struct{}
	onChange              func(NodeStatus)
}

type subscriber struct {
	conn      *websocket.Conn
	serverIDs serverIDSet
	send      chan nodeStatusPayload
	done      chan struct{}
	closeOnce sync.Once
}

const smallServerIDSetThreshold = 8

type serverIDSet struct {
	ids     []int64
	index   map[int64]struct{}
	capHint int
}

func makeServerIDSet(serverIDs []int64) serverIDSet {
	if len(serverIDs) == 0 {
		return serverIDSet{}
	}
	set := serverIDSet{
		capHint: len(serverIDs),
	}
	for _, id := range serverIDs {
		set.Add(id)
	}
	return set
}

func (s *serverIDSet) Add(id int64) bool {
	if id <= 0 {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[id]; ok {
			return false
		}
		s.index[id] = struct{}{}
		s.ensureIDs()
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
	if len(s.ids) > smallServerIDSetThreshold {
		nextCap := serverIDSetLookupCap(len(s.ids), s.capHint)
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
	return true
}

func (s *serverIDSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	capHint := s.capHint
	if capHint <= 0 {
		capHint = smallServerIDSetThreshold
	} else if capHint > smallServerIDSetThreshold+1 {
		capHint = smallServerIDSetThreshold + 1
	}
	s.ids = make([]int64, 0, capHint)
}

func serverIDSetLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (s serverIDSet) Len() int {
	return len(s.ids)
}

func (s serverIDSet) Contains(id int64) bool {
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

func (s serverIDSet) ForEach(visit func(int64)) {
	for _, id := range s.ids {
		visit(id)
	}
}

func NewManager() *Manager {
	manager := &Manager{
		states:                make(map[int64]NodeStatus),
		subscribers:           make(map[*subscriber]struct{}),
		subscribersByConn:     make(map[*websocket.Conn]*subscriber),
		subscribersByServerID: make(map[int64]map[*subscriber]struct{}),
		globalSubscribers:     make(map[*subscriber]struct{}),
	}
	go manager.expireLoop()
	return manager
}

func (m *Manager) SetChangeHook(hook func(NodeStatus)) {
	m.mu.Lock()
	m.onChange = hook
	m.mu.Unlock()
}

func (m *Manager) Update(state NodeStatus) NodeStatus {
	now := time.Now().UnixMilli()
	state.UpdatedAt = now
	if !state.XrayRunning {
		state.XrayStatsError = ""
	}
	state.LastSeen = now
	state.Online = true
	state.Status = "online"
	state.Type = "node_status"

	m.mu.Lock()
	m.states[state.ServerID] = state
	subscribers := m.subscribersForServerLocked(state.ServerID)
	m.mu.Unlock()

	m.notifyChange(state)
	subscribers.ForEach(func(sub *subscriber) {
		sub.enqueue(state)
	})
	return state
}

func (m *Manager) Get(serverID int64) (NodeStatus, bool) {
	m.mu.RLock()
	state, ok := m.states[serverID]
	m.mu.RUnlock()
	if !ok {
		return NodeStatus{}, false
	}
	return normalizeState(state), true
}

func (m *Manager) Snapshot(serverIDs []int64) []NodeStatus {
	filter := makeServerIDSet(serverIDs)
	now := time.Now()

	m.mu.RLock()
	defer m.mu.RUnlock()

	if filter.Len() > 0 {
		list := make([]NodeStatus, 0, filter.Len())
		filter.ForEach(func(id int64) {
			if state, ok := m.states[id]; ok {
				list = append(list, normalizeStateAt(state, now))
			}
		})
		return list
	}

	list := make([]NodeStatus, 0, len(m.states))
	for _, state := range m.states {
		list = append(list, normalizeStateAt(state, now))
	}
	return list
}

func (m *Manager) Subscribe(conn *websocket.Conn, serverIDs []int64) func() {
	sub := &subscriber{
		conn:      conn,
		serverIDs: makeServerIDSet(serverIDs),
		send:      make(chan nodeStatusPayload, 64),
		done:      make(chan struct{}),
	}
	m.mu.Lock()
	m.addSubscriberLocked(sub)
	m.mu.Unlock()
	go sub.writeLoop()
	return func() {
		m.mu.Lock()
		m.removeSubscriberLocked(sub)
		m.mu.Unlock()
		sub.close()
	}
}

func (m *Manager) SetSubscription(conn *websocket.Conn, serverIDs []int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureSubscriberIndexesLocked()
	sub := m.subscribersByConn[conn]
	if sub != nil {
		m.setSubscriberServerIDsLocked(sub, makeServerIDSet(serverIDs))
	}
}

func (m *Manager) SendSnapshot(conn *websocket.Conn, serverIDs []int64) {
	snapshot := m.Snapshot(serverIDs)
	m.mu.Lock()
	m.ensureSubscriberIndexesLocked()
	sub := m.subscribersByConn[conn]
	m.mu.Unlock()
	if sub != nil {
		sub.enqueue(NodeStatusSnapshot{Type: "snapshot", Data: snapshot})
	}
}

func (m *Manager) MarkOffline(serverID int64) {
	m.MarkOfflineIfLastSeen(serverID, 0)
}

func (m *Manager) SetOnlineUsers(serverID int64, onlineUsers int64) {
	m.mu.Lock()
	state, ok := m.states[serverID]
	if ok {
		state.OnlineUsers = onlineUsers
		state.Type = "node_status"
		m.states[serverID] = state
	}
	var subscribers nodeStatusSubscriberSnapshot
	if ok {
		subscribers = m.subscribersForServerLocked(serverID)
	}
	m.mu.Unlock()
	if !ok {
		return
	}
	m.notifyChange(state)
	subscribers.ForEach(func(sub *subscriber) {
		sub.enqueue(state)
	})
}

func (m *Manager) MarkOfflineIfLastSeen(serverID int64, lastSeen int64) {
	m.mu.Lock()
	state, ok := m.states[serverID]
	if ok {
		if lastSeen > 0 && state.LastSeen != lastSeen {
			m.mu.Unlock()
			return
		}
		state.Online = false
		state.Status = "offline"
		state.XrayStatsError = ""
		state.Type = "node_status"
		state.UpdatedAt = time.Now().UnixMilli()
		m.states[serverID] = state
	}
	var subscribers nodeStatusSubscriberSnapshot
	if ok {
		subscribers = m.subscribersForServerLocked(serverID)
	}
	m.mu.Unlock()
	if !ok {
		return
	}
	m.notifyChange(state)
	subscribers.ForEach(func(sub *subscriber) {
		sub.enqueue(state)
	})
}

func (m *Manager) expireLoop() {
	ticker := time.NewTicker(nodeStatusExpireInterval)
	defer ticker.Stop()
	for range ticker.C {
		m.MarkExpiredOffline()
	}
}

func (m *Manager) MarkExpiredOffline() {
	now := time.Now()

	m.mu.Lock()
	notifications := newNodeStatusNotificationList(len(m.states))
	for serverID, state := range m.states {
		if !state.Online || state.LastSeen == 0 || now.Sub(time.UnixMilli(state.LastSeen)) <= NodeStatusTTL {
			continue
		}
		state.Online = false
		state.Status = "offline"
		state.XrayStatsError = ""
		state.Type = "node_status"
		state.UpdatedAt = now.UnixMilli()
		m.states[serverID] = state
		notifications.Add(nodeStatusNotification{
			state:       state,
			subscribers: m.subscribersForServerLocked(serverID),
		})
	}
	m.mu.Unlock()

	notifications.ForEach(func(notification nodeStatusNotification) {
		m.notifyChange(notification.state)
		notification.subscribers.ForEach(func(sub *subscriber) {
			sub.enqueue(notification.state)
		})
	})
}

func (m *Manager) notifyChange(state NodeStatus) {
	m.mu.RLock()
	hook := m.onChange
	m.mu.RUnlock()
	if hook != nil {
		hook(state)
	}
}

type nodeStatusNotification struct {
	state       NodeStatus
	subscribers nodeStatusSubscriberSnapshot
}

type nodeStatusNotificationList struct {
	single    nodeStatusNotification
	hasSingle bool
	items     []nodeStatusNotification
	capHint   int
}

func newNodeStatusNotificationList(capHint int) nodeStatusNotificationList {
	if capHint <= 0 {
		return nodeStatusNotificationList{}
	}
	return nodeStatusNotificationList{capHint: capHint}
}

func (l *nodeStatusNotificationList) Add(notification nodeStatusNotification) {
	if !l.hasSingle && len(l.items) == 0 {
		l.single = notification
		l.hasSingle = true
		return
	}
	if l.hasSingle {
		l.ensureItems()
		l.items = append(l.items, l.single, notification)
		l.single = nodeStatusNotification{}
		l.hasSingle = false
		return
	}
	l.items = append(l.items, notification)
}

func (l *nodeStatusNotificationList) ensureItems() {
	if l.items != nil {
		return
	}
	capHint := l.capHint
	if capHint < 2 {
		capHint = 2
	}
	l.items = make([]nodeStatusNotification, 0, capHint)
}

func (l nodeStatusNotificationList) ForEach(visit func(nodeStatusNotification)) {
	if visit == nil {
		return
	}
	if l.hasSingle {
		visit(l.single)
		return
	}
	for _, notification := range l.items {
		visit(notification)
	}
}

func (l nodeStatusNotificationList) Len() int {
	if l.hasSingle {
		return 1
	}
	return len(l.items)
}

func (m *Manager) ensureSubscriberIndexesLocked() {
	if m.subscribers == nil {
		m.subscribers = make(map[*subscriber]struct{})
	}
	if m.subscribersByConn != nil && m.subscribersByServerID != nil && m.globalSubscribers != nil {
		return
	}

	m.subscribersByConn = make(map[*websocket.Conn]*subscriber, len(m.subscribers))
	m.subscribersByServerID = make(map[int64]map[*subscriber]struct{})
	m.globalSubscribers = make(map[*subscriber]struct{})
	for sub := range m.subscribers {
		if sub == nil {
			continue
		}
		m.subscribersByConn[sub.conn] = sub
		m.indexSubscriberLocked(sub)
	}
}

func (m *Manager) addSubscriberLocked(sub *subscriber) {
	m.ensureSubscriberIndexesLocked()
	m.subscribers[sub] = struct{}{}
	m.subscribersByConn[sub.conn] = sub
	m.indexSubscriberLocked(sub)
}

func (m *Manager) removeSubscriberLocked(sub *subscriber) {
	m.ensureSubscriberIndexesLocked()
	delete(m.subscribers, sub)
	if m.subscribersByConn[sub.conn] == sub {
		delete(m.subscribersByConn, sub.conn)
	}
	m.unindexSubscriberLocked(sub)
}

func (m *Manager) setSubscriberServerIDsLocked(sub *subscriber, serverIDs serverIDSet) {
	m.ensureSubscriberIndexesLocked()
	m.unindexSubscriberLocked(sub)
	sub.serverIDs = serverIDs
	m.indexSubscriberLocked(sub)
}

func (m *Manager) indexSubscriberLocked(sub *subscriber) {
	if sub.serverIDs.Len() == 0 {
		m.globalSubscribers[sub] = struct{}{}
		return
	}
	sub.serverIDs.ForEach(func(serverID int64) {
		subscribers := m.subscribersByServerID[serverID]
		if subscribers == nil {
			subscribers = make(map[*subscriber]struct{})
			m.subscribersByServerID[serverID] = subscribers
		}
		subscribers[sub] = struct{}{}
	})
}

func (m *Manager) unindexSubscriberLocked(sub *subscriber) {
	delete(m.globalSubscribers, sub)
	sub.serverIDs.ForEach(func(serverID int64) {
		subscribers := m.subscribersByServerID[serverID]
		if subscribers == nil {
			return
		}
		delete(subscribers, sub)
		if len(subscribers) == 0 {
			delete(m.subscribersByServerID, serverID)
		}
	})
}

func (m *Manager) subscribersForServerLocked(serverID int64) nodeStatusSubscriberSnapshot {
	m.ensureSubscriberIndexesLocked()
	serverSubscribers := m.subscribersByServerID[serverID]
	subscribers := newNodeStatusSubscriberSnapshot(len(m.globalSubscribers) + len(serverSubscribers))
	for sub := range m.globalSubscribers {
		subscribers.Add(sub)
	}
	for sub := range serverSubscribers {
		subscribers.Add(sub)
	}
	return subscribers
}

type nodeStatusSubscriberSnapshot struct {
	single  *subscriber
	items   []*subscriber
	capHint int
}

func newNodeStatusSubscriberSnapshot(capHint int) nodeStatusSubscriberSnapshot {
	if capHint <= 0 {
		return nodeStatusSubscriberSnapshot{}
	}
	return nodeStatusSubscriberSnapshot{capHint: capHint}
}

func (s *nodeStatusSubscriberSnapshot) Add(sub *subscriber) {
	if sub == nil {
		return
	}
	if s.single == nil && len(s.items) == 0 {
		s.single = sub
		return
	}
	if s.single != nil {
		s.ensureItems()
		s.items = append(s.items, s.single, sub)
		s.single = nil
		return
	}
	s.items = append(s.items, sub)
}

func (s *nodeStatusSubscriberSnapshot) ensureItems() {
	if s.items != nil {
		return
	}
	capHint := s.capHint
	if capHint < 2 {
		capHint = 2
	}
	s.items = make([]*subscriber, 0, capHint)
}

func (s nodeStatusSubscriberSnapshot) ForEach(visit func(*subscriber)) {
	if visit == nil {
		return
	}
	if s.single != nil {
		visit(s.single)
		return
	}
	for _, sub := range s.items {
		visit(sub)
	}
}

func (s nodeStatusSubscriberSnapshot) Len() int {
	if s.single != nil {
		return 1
	}
	return len(s.items)
}

func (s *subscriber) enqueue(v nodeStatusPayload) {
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.send <- v:
		return
	default:
	}
	select {
	case <-s.send:
	default:
	}
	select {
	case s.send <- v:
	default:
	}
}

func (s *subscriber) writeLoop() {
	for {
		select {
		case <-s.done:
			return
		case v := <-s.send:
			if err := s.conn.WriteJSON(v); err != nil {
				s.close()
				return
			}
		}
	}
}

func (s *subscriber) close() {
	s.closeOnce.Do(func() {
		close(s.done)
		_ = s.conn.Close()
	})
}

func normalizeState(state NodeStatus) NodeStatus {
	return normalizeStateAt(state, time.Now())
}

func normalizeStateAt(state NodeStatus, now time.Time) NodeStatus {
	if now.Sub(time.UnixMilli(state.LastSeen)) > NodeStatusTTL {
		state.Online = false
		state.Status = "offline"
		state.XrayStatsError = ""
	} else {
		state.Online = true
		state.Status = "online"
	}
	state.Type = "node_status"
	return state
}

func DecodeNodeStatus(data []byte) (NodeStatus, error) {
	var state NodeStatus
	err := json.Unmarshal(data, &state)
	return state, err
}
