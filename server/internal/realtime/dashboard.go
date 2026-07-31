package realtime

import (
	"sync"
	"sync/atomic"
	"time"
)

type DashboardEvent string

const (
	DashboardEventGeneral     DashboardEvent = "general"
	DashboardEventActivity    DashboardEvent = "activity"
	DashboardEventBusiness    DashboardEvent = "business"
	DashboardEventNodeStatus  DashboardEvent = "node_status"
	DashboardEventOnlineUsers DashboardEvent = "online_users"
	DashboardEventTraffic     DashboardEvent = "traffic"
	DashboardEventTicket      DashboardEvent = "ticket"
	DashboardEventServer      DashboardEvent = "server"
)

type DashboardMessage struct {
	Reason    DashboardEvent `json:"reason"`
	Seq       uint64         `json:"seq"`
	UpdatedAt int64          `json:"updated_at"`
}

type DashboardManager struct {
	mu          sync.RWMutex
	seq         atomic.Uint64
	input       chan DashboardEvent
	subscribers dashboardSubscriberSet
}

type DashboardSubscriber struct {
	C chan DashboardMessage
}

const smallDashboardSubscriberSetThreshold = 8

type dashboardSubscriberSet struct {
	items []*DashboardSubscriber
	index map[*DashboardSubscriber]struct{}
}

func (s *dashboardSubscriberSet) Add(sub *DashboardSubscriber) {
	if sub == nil {
		return
	}
	if s.index != nil {
		s.index[sub] = struct{}{}
		return
	}
	for _, item := range s.items {
		if item == sub {
			return
		}
	}
	s.items = append(s.items, sub)
	if len(s.items) > smallDashboardSubscriberSetThreshold {
		s.index = make(map[*DashboardSubscriber]struct{}, len(s.items))
		for _, item := range s.items {
			s.index[item] = struct{}{}
		}
		s.items = nil
	}
}

func (s *dashboardSubscriberSet) Remove(sub *DashboardSubscriber) {
	if sub == nil {
		return
	}
	if s.index != nil {
		delete(s.index, sub)
		if len(s.index) == 0 {
			s.items = nil
			s.index = nil
		}
		return
	}
	for i, item := range s.items {
		if item != sub {
			continue
		}
		copy(s.items[i:], s.items[i+1:])
		s.items[len(s.items)-1] = nil
		s.items = s.items[:len(s.items)-1]
		return
	}
}

func (s dashboardSubscriberSet) ForEach(visit func(*DashboardSubscriber)) {
	if s.index != nil {
		for sub := range s.index {
			visit(sub)
		}
		return
	}
	for _, sub := range s.items {
		visit(sub)
	}
}

func NewDashboardManager() *DashboardManager {
	manager := &DashboardManager{
		input: make(chan DashboardEvent, 1024),
	}
	go manager.dispatchLoop()
	return manager
}

func (m *DashboardManager) Publish(reason DashboardEvent) {
	if m == nil {
		return
	}
	if reason == "" {
		reason = DashboardEventGeneral
	}
	select {
	case m.input <- reason:
	default:
		select {
		case <-m.input:
		default:
		}
		select {
		case m.input <- DashboardEventGeneral:
		default:
		}
	}
}

func (m *DashboardManager) Subscribe() (*DashboardSubscriber, func()) {
	sub := &DashboardSubscriber{
		C: make(chan DashboardMessage, 32),
	}
	m.mu.Lock()
	m.subscribers.Add(sub)
	m.mu.Unlock()
	return sub, func() {
		m.mu.Lock()
		m.subscribers.Remove(sub)
		m.mu.Unlock()
	}
}

func (m *DashboardManager) CurrentSeq() uint64 {
	if m == nil {
		return 0
	}
	return m.seq.Load()
}

func (m *DashboardManager) dispatchLoop() {
	for reason := range m.input {
		msg := DashboardMessage{
			Reason:    reason,
			Seq:       m.seq.Add(1),
			UpdatedAt: time.Now().UnixMilli(),
		}
		m.mu.RLock()
		m.subscribers.ForEach(func(sub *DashboardSubscriber) {
			deliverDashboardMessage(sub, msg)
		})
		m.mu.RUnlock()
	}
}

func deliverDashboardMessage(sub *DashboardSubscriber, msg DashboardMessage) {
	select {
	case sub.C <- msg:
		return
	default:
	}
	select {
	case <-sub.C:
	default:
	}
	select {
	case sub.C <- msg:
	default:
	}
}
