package main

import (
	"sync"
	"sync/atomic"
	"time"
)

type ruleCounters struct {
	active   atomic.Int64
	accepted atomic.Uint64
	failed   atomic.Uint64
	blocked  atomic.Uint64
	upload   atomic.Uint64
	download atomic.Uint64
}

type Metrics struct {
	active   atomic.Int64
	accepted atomic.Uint64
	failed   atomic.Uint64
	blocked  atomic.Uint64
	upload   atomic.Uint64
	download atomic.Uint64

	mu           sync.Mutex
	rules        map[string]*ruleCounters
	lastAt       time.Time
	lastUpload   uint64
	lastDownload uint64
	lastRules    map[string][2]uint64
}

type MetricsSnapshot struct {
	ActiveConnections   int64
	AcceptedConnections uint64
	FailedConnections   uint64
	BlockedConnections  uint64
	UploadBps           int64
	DownloadBps         int64
	UploadBytes         uint64
	DownloadBytes       uint64
	Rules               []RuleStatus
}

type RuleStatus struct {
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

func NewMetrics() *Metrics {
	return &Metrics{
		rules:     make(map[string]*ruleCounters),
		lastAt:    time.Now(),
		lastRules: make(map[string][2]uint64),
	}
}

func (m *Metrics) ensureRule(ruleID string) *ruleCounters {
	m.mu.Lock()
	defer m.mu.Unlock()
	counter := m.rules[ruleID]
	if counter == nil {
		counter = &ruleCounters{}
		m.rules[ruleID] = counter
	}
	return counter
}

func (m *Metrics) Accepted(ruleID string) {
	m.accepted.Add(1)
	m.ensureRule(ruleID).accepted.Add(1)
}

func (m *Metrics) Connected(ruleID string) {
	m.active.Add(1)
	m.ensureRule(ruleID).active.Add(1)
}

func (m *Metrics) Disconnected(ruleID string) {
	m.active.Add(-1)
	m.ensureRule(ruleID).active.Add(-1)
}

func (m *Metrics) Failed(ruleID string) {
	m.failed.Add(1)
	m.ensureRule(ruleID).failed.Add(1)
}

func (m *Metrics) Blocked(ruleID string) {
	m.blocked.Add(1)
	m.ensureRule(ruleID).blocked.Add(1)
}

func (m *Metrics) AddUpload(ruleID string, count uint64) {
	m.addUpload(m.ensureRule(ruleID), count)
}

func (m *Metrics) AddDownload(ruleID string, count uint64) {
	m.addDownload(m.ensureRule(ruleID), count)
}

func (m *Metrics) addUpload(rule *ruleCounters, count uint64) {
	m.upload.Add(count)
	rule.upload.Add(count)
}

func (m *Metrics) addDownload(rule *ruleCounters, count uint64) {
	m.download.Add(count)
	rule.download.Add(count)
}

func (m *Metrics) Snapshot(items []ForwaredItem) MetricsSnapshot {
	now := time.Now()
	upload := m.upload.Load()
	download := m.download.Load()

	m.mu.Lock()
	elapsed := now.Sub(m.lastAt).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}
	snapshot := MetricsSnapshot{
		ActiveConnections:   m.active.Load(),
		AcceptedConnections: m.accepted.Load(),
		FailedConnections:   m.failed.Load(),
		BlockedConnections:  m.blocked.Load(),
		UploadBps:           int64(float64(upload-m.lastUpload) / elapsed),
		DownloadBps:         int64(float64(download-m.lastDownload) / elapsed),
		UploadBytes:         upload,
		DownloadBytes:       download,
		Rules:               make([]RuleStatus, 0, len(items)),
	}
	for _, item := range items {
		counter := m.rules[item.RuleID]
		if counter == nil {
			counter = &ruleCounters{}
			m.rules[item.RuleID] = counter
		}
		ruleUpload := counter.upload.Load()
		ruleDownload := counter.download.Load()
		previous := m.lastRules[item.RuleID]
		snapshot.Rules = append(snapshot.Rules, RuleStatus{
			RuleID:              item.RuleID,
			ListenIP:            item.ListenIP,
			ListenPort:          uint16(item.LocalPort),
			TargetHost:          item.TargetHost,
			TargetPort:          uint16(item.TargetPort),
			NodeID:              item.NodeID,
			NodeName:            item.NodeName,
			ServerID:            item.ServerID,
			ServerName:          item.ServerName,
			Inbound:             item.Inbound,
			Status:              "running",
			ActiveConnections:   counter.active.Load(),
			AcceptedConnections: counter.accepted.Load(),
			FailedConnections:   counter.failed.Load(),
			BlockedConnections:  counter.blocked.Load(),
			UploadBps:           int64(float64(ruleUpload-previous[0]) / elapsed),
			DownloadBps:         int64(float64(ruleDownload-previous[1]) / elapsed),
			UploadBytes:         ruleUpload,
			DownloadBytes:       ruleDownload,
		})
		m.lastRules[item.RuleID] = [2]uint64{ruleUpload, ruleDownload}
	}
	m.lastAt = now
	m.lastUpload = upload
	m.lastDownload = download
	m.mu.Unlock()
	return snapshot
}
