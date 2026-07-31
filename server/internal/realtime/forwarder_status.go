package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const defaultForwarderStatusTTL = 30 * time.Second

type ForwarderRuleStatus struct {
	RuleID              string `json:"rule_id"`
	ListenIP            string `json:"listen_ip,omitempty"`
	ListenPort          uint16 `json:"listen_port"`
	TargetHost          string `json:"target_host"`
	TargetPort          uint16 `json:"target_port"`
	NodeID              int64  `json:"node_id,omitempty"`
	NodeName            string `json:"node_name,omitempty"`
	ServerID            int64  `json:"server_id,omitempty"`
	ServerName          string `json:"server_name,omitempty"`
	Inbound             string `json:"inbound,omitempty"`
	Status              string `json:"status"`
	Error               string `json:"error,omitempty"`
	ActiveConnections   int64  `json:"active_connections"`
	AcceptedConnections uint64 `json:"accepted_connections"`
	FailedConnections   uint64 `json:"failed_connections"`
	BlockedConnections  uint64 `json:"blocked_connections"`
	UploadBps           int64  `json:"upload_bps"`
	DownloadBps         int64  `json:"download_bps"`
	UploadBytes         uint64 `json:"upload_bytes"`
	DownloadBytes       uint64 `json:"download_bytes"`
}

type ForwarderStatus struct {
	Type                string                `json:"type,omitempty"`
	InstanceID          string                `json:"instance_id"`
	BootID              string                `json:"boot_id"`
	Online              bool                  `json:"online"`
	Version             string                `json:"version,omitempty"`
	Uptime              int64                 `json:"uptime"`
	ActiveConnections   int64                 `json:"active_connections"`
	AcceptedConnections uint64                `json:"accepted_connections"`
	FailedConnections   uint64                `json:"failed_connections"`
	BlockedConnections  uint64                `json:"blocked_connections"`
	UploadBps           int64                 `json:"upload_bps"`
	DownloadBps         int64                 `json:"download_bps"`
	UploadBytes         uint64                `json:"upload_bytes"`
	DownloadBytes       uint64                `json:"download_bytes"`
	DesiredVersion      string                `json:"desired_version,omitempty"`
	DesiredHash         string                `json:"desired_hash,omitempty"`
	AppliedVersion      string                `json:"applied_version,omitempty"`
	AppliedHash         string                `json:"applied_hash,omitempty"`
	ApplyStatus         string                `json:"apply_status"`
	ApplyError          string                `json:"apply_error,omitempty"`
	LastSeen            int64                 `json:"last_seen"`
	Rules               []ForwarderRuleStatus `json:"rules"`
}

type ForwarderSnapshot struct {
	Type string            `json:"type"`
	Data []ForwarderStatus `json:"data"`
}

type ForwarderManager struct {
	mu          sync.RWMutex
	states      map[string]ForwarderStatus
	subscribers map[*websocket.Conn]*forwarderSubscriber
	ttl         time.Duration
}

type forwarderSubscriber struct {
	send chan []byte
	stop chan struct{}
	once sync.Once
}

func NewForwarderManager(ttl time.Duration) *ForwarderManager {
	if ttl <= 0 {
		ttl = defaultForwarderStatusTTL
	}
	manager := &ForwarderManager{
		states:      make(map[string]ForwarderStatus),
		subscribers: make(map[*websocket.Conn]*forwarderSubscriber),
		ttl:         ttl,
	}
	go manager.expireLoop()
	return manager
}

func (m *ForwarderManager) Update(state ForwarderStatus) ForwarderStatus {
	state.Type = "forwarder_status"
	state.Online = true
	state.LastSeen = time.Now().UnixMilli()
	if state.Rules == nil {
		state.Rules = []ForwarderRuleStatus{}
	}
	m.mu.Lock()
	m.states[state.InstanceID] = state
	subscribers := m.subscriberSnapshotLocked()
	m.mu.Unlock()
	m.broadcast(subscribers, state)
	return state
}

func (m *ForwarderManager) Get(instanceID string) (ForwarderStatus, bool) {
	m.mu.RLock()
	state, ok := m.states[instanceID]
	m.mu.RUnlock()
	if !ok {
		return ForwarderStatus{}, false
	}
	return normalizeForwarderState(state, m.ttl), true
}

func (m *ForwarderManager) Snapshot() []ForwarderStatus {
	m.mu.RLock()
	list := make([]ForwarderStatus, 0, len(m.states))
	for _, state := range m.states {
		list = append(list, normalizeForwarderState(state, m.ttl))
	}
	m.mu.RUnlock()
	return list
}

func (m *ForwarderManager) Subscribe(conn *websocket.Conn) func() {
	subscriber := &forwarderSubscriber{
		send: make(chan []byte, 32),
		stop: make(chan struct{}),
	}
	m.mu.Lock()
	m.subscribers[conn] = subscriber
	list := make([]ForwarderStatus, 0, len(m.states))
	for _, state := range m.states {
		list = append(list, normalizeForwarderState(state, m.ttl))
	}
	if payload, err := json.Marshal(ForwarderSnapshot{Type: "forwarder_snapshot", Data: list}); err == nil {
		subscriber.send <- payload
	}
	m.mu.Unlock()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case payload := <-subscriber.send:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
					_ = conn.Close()
					return
				}
			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					_ = conn.Close()
					return
				}
			case <-subscriber.stop:
				return
			}
		}
	}()
	return func() {
		subscriber.once.Do(func() {
			m.mu.Lock()
			if current := m.subscribers[conn]; current == subscriber {
				delete(m.subscribers, conn)
			}
			m.mu.Unlock()
			close(subscriber.stop)
		})
	}
}

func (m *ForwarderManager) expireLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for now := range ticker.C {
		var expired []ForwarderStatus
		m.mu.Lock()
		for id, state := range m.states {
			if now.Sub(time.UnixMilli(state.LastSeen)) <= m.ttl || !state.Online {
				continue
			}
			state.Online = false
			state.Type = "forwarder_status"
			m.states[id] = state
			expired = append(expired, state)
		}
		subscribers := m.subscriberSnapshotLocked()
		m.mu.Unlock()
		for _, state := range expired {
			m.broadcast(subscribers, state)
		}
	}
}

func (m *ForwarderManager) subscriberSnapshotLocked() []*forwarderSubscriber {
	result := make([]*forwarderSubscriber, 0, len(m.subscribers))
	for _, subscriber := range m.subscribers {
		result = append(result, subscriber)
	}
	return result
}

func (m *ForwarderManager) broadcast(subscribers []*forwarderSubscriber, state ForwarderStatus) {
	payload, err := json.Marshal(state)
	if err != nil {
		return
	}
	for _, subscriber := range subscribers {
		select {
		case subscriber.send <- payload:
		case <-subscriber.stop:
		default:
		}
	}
}

func normalizeForwarderState(state ForwarderStatus, ttl time.Duration) ForwarderStatus {
	if time.Since(time.UnixMilli(state.LastSeen)) > ttl {
		state.Online = false
	}
	if state.Rules == nil {
		state.Rules = []ForwarderRuleStatus{}
	}
	return state
}
