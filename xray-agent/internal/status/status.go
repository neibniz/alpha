package status

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/config"
	"github.com/neibniz/alpha/xray-agent/internal/supervisor"
)

type Payload struct {
	ServerID                  int64   `json:"server_id"`
	CPU                       float64 `json:"cpu"`
	Mem                       float64 `json:"mem"`
	Disk                      float64 `json:"disk"`
	Load1                     float64 `json:"load1,omitempty"`
	Load5                     float64 `json:"load5,omitempty"`
	Load15                    float64 `json:"load15,omitempty"`
	Uptime                    int64   `json:"uptime,omitempty"`
	XrayRunning               bool    `json:"xray_running"`
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
	MemoryUsed                int64   `json:"memory_used,omitempty"`
	MemoryTotal               int64   `json:"memory_total,omitempty"`
	DiskUsed                  int64   `json:"disk_used,omitempty"`
	DiskTotal                 int64   `json:"disk_total,omitempty"`
	AgentVersion              string  `json:"agent_version,omitempty"`
	XrayVersion               string  `json:"xray_version,omitempty"`
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

type Collector struct {
	cfg        config.Config
	supervisor *supervisor.Supervisor
	runtime    *RuntimeState
	prevCPU    cpuSample
	prevNet    netSample
	prevAt     time.Time
}

type RuntimeState struct {
	mu       sync.RWMutex
	snapshot RuntimeSnapshot
}

type RuntimeSnapshot struct {
	XrayRxBps         int64
	XrayTxBps         int64
	XrayRxBytes       int64
	XrayTxBytes       int64
	XrayStatsError    string
	OnlineUsers       int64
	ConfigSyncStatus  string
	ConfigApplyStatus string
	LastConfigHash    string
	RunningConfigHash string
	PendingConfigHash string
	LastSyncAt        int64
	LastApplyAt       int64
	LastConfigError   string
	LastApplyError    string
}

type cpuSample struct {
	idle  uint64
	total uint64
}

type netSample struct {
	rx uint64
	tx uint64
}

func NewRuntimeState() *RuntimeState {
	return &RuntimeState{snapshot: RuntimeSnapshot{
		ConfigSyncStatus:  "pending",
		ConfigApplyStatus: "pending",
	}}
}

func (s *RuntimeState) Snapshot() RuntimeSnapshot {
	if s == nil {
		return RuntimeSnapshot{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func (s *RuntimeState) SetXrayTraffic(rxBps int64, txBps int64, rxBytes int64, txBytes int64, errText string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.XrayRxBps = rxBps
	s.snapshot.XrayTxBps = txBps
	s.snapshot.XrayRxBytes = rxBytes
	s.snapshot.XrayTxBytes = txBytes
	s.snapshot.XrayStatsError = errText
}

func (s *RuntimeState) SetOnlineUsers(count int64) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.OnlineUsers = count
}

func (s *RuntimeState) SetSyncFailed(hash string, err error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.ConfigSyncStatus = "sync_failed"
	s.snapshot.ConfigApplyStatus = "sync_failed"
	s.snapshot.LastConfigHash = hash
	s.snapshot.PendingConfigHash = hash
	s.snapshot.LastSyncAt = time.Now().UnixMilli()
	s.snapshot.LastConfigError = errString(err)
	s.snapshot.LastApplyError = errString(err)
}

func (s *RuntimeState) SetSyncOK(hash string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.ConfigSyncStatus = "ok"
	s.snapshot.LastConfigHash = hash
	s.snapshot.PendingConfigHash = hash
	s.snapshot.LastSyncAt = time.Now().UnixMilli()
	s.snapshot.LastConfigError = ""
}

func (s *RuntimeState) SetApplyPending(hash string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.ConfigApplyStatus = "pending"
	s.snapshot.PendingConfigHash = hash
}

func (s *RuntimeState) SetApplyFailed(status string, hash string, err error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.ConfigApplyStatus = status
	s.snapshot.PendingConfigHash = hash
	s.snapshot.LastApplyError = errString(err)
}

func (s *RuntimeState) SetApplyOK(hash string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot.ConfigSyncStatus = "ok"
	s.snapshot.ConfigApplyStatus = "ok"
	s.snapshot.RunningConfigHash = hash
	s.snapshot.PendingConfigHash = ""
	s.snapshot.LastApplyAt = time.Now().UnixMilli()
	s.snapshot.LastConfigError = ""
	s.snapshot.LastApplyError = ""
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func NewCollector(cfg config.Config, supervisor *supervisor.Supervisor, runtime *RuntimeState) *Collector {
	c := &Collector{cfg: cfg, supervisor: supervisor, runtime: runtime}
	c.prevCPU = readCPU()
	c.prevNet = readNet(cfg.Report.InterfaceName)
	c.prevAt = time.Now()
	return c
}

func (c *Collector) Collect() Payload {
	now := time.Now()
	cpu := c.cpuPercent()
	memPercent, memUsed, memTotal := memory()
	diskPercent, diskUsed, diskTotal := disk(c.cfg.Xray.WorkDir)
	net := readNet(c.cfg.Report.InterfaceName)
	elapsed := now.Sub(c.prevAt).Seconds()
	var rxBps, txBps int64
	if elapsed > 0 && net.rx >= c.prevNet.rx && net.tx >= c.prevNet.tx {
		rxBps = int64(float64(net.rx-c.prevNet.rx) * 8 / elapsed)
		txBps = int64(float64(net.tx-c.prevNet.tx) * 8 / elapsed)
	}
	c.prevNet = net
	c.prevAt = now
	load1, load5, load15 := loadAvg()
	runtime := c.runtime.Snapshot()
	connectionStats := tcpConnections(c.supervisor.PID(), inboundPorts(c.cfg.Xray.ConfigFile))
	return Payload{
		ServerID:                  c.cfg.Panel.ServerID,
		CPU:                       cpu,
		Mem:                       memPercent,
		Disk:                      diskPercent,
		Load1:                     load1,
		Load5:                     load5,
		Load15:                    load15,
		Uptime:                    uptime(),
		XrayRunning:               c.supervisor.Running(),
		Connections:               connectionStats.Total,
		SystemConnections:         connectionStats.SystemInbound + connectionStats.SystemOutbound,
		SystemInboundConnections:  connectionStats.SystemInbound,
		SystemOutboundConnections: connectionStats.SystemOutbound,
		XrayConnections:           connectionStats.XrayInbound + connectionStats.XrayOutbound,
		XrayInboundConnections:    connectionStats.XrayInbound,
		XrayOutboundConnections:   connectionStats.XrayOutbound,
		OnlineUsers:               runtime.OnlineUsers,
		NetRxBps:                  rxBps,
		NetTxBps:                  txBps,
		NetRxBytes:                int64(net.rx),
		NetTxBytes:                int64(net.tx),
		XrayRxBps:                 runtime.XrayRxBps,
		XrayTxBps:                 runtime.XrayTxBps,
		XrayRxBytes:               runtime.XrayRxBytes,
		XrayTxBytes:               runtime.XrayTxBytes,
		XrayStatsError:            runtime.XrayStatsError,
		UpdatedAt:                 now.UnixMilli(),
		MemoryUsed:                int64(memUsed),
		MemoryTotal:               int64(memTotal),
		DiskUsed:                  int64(diskUsed),
		DiskTotal:                 int64(diskTotal),
		AgentVersion:              config.Version,
		XrayVersion:               c.supervisor.XrayVersion(),
		ConfigVersion:             c.supervisor.ConfigVersion(),
		ConfigSyncStatus:          runtime.ConfigSyncStatus,
		ConfigApplyStatus:         runtime.ConfigApplyStatus,
		LastConfigHash:            runtime.LastConfigHash,
		RunningConfigHash:         runtime.RunningConfigHash,
		PendingConfigHash:         runtime.PendingConfigHash,
		LastSyncAt:                runtime.LastSyncAt,
		LastApplyAt:               runtime.LastApplyAt,
		LastConfigError:           runtime.LastConfigError,
		LastApplyError:            runtime.LastApplyError,
	}
}

func (c *Collector) cpuPercent() float64 {
	current := readCPU()
	idle := current.idle - c.prevCPU.idle
	total := current.total - c.prevCPU.total
	c.prevCPU = current
	if total == 0 {
		return 0
	}
	return clamp(float64(total-idle) * 100 / float64(total))
}

func readCPU() cpuSample {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuSample{}
	}
	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
	if len(fields) < 5 {
		return cpuSample{}
	}
	var values []uint64
	for _, field := range fields[1:] {
		value, _ := strconv.ParseUint(field, 10, 64)
		values = append(values, value)
	}
	var total uint64
	for _, value := range values {
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuSample{idle: idle, total: total}
}

func memory() (float64, uint64, uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return 0, m.Alloc, m.Sys
	}
	defer file.Close()
	values := map[string]uint64{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			values[strings.TrimSuffix(fields[0], ":")], _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}
	total := values["MemTotal"] * 1024
	available := values["MemAvailable"] * 1024
	if total == 0 {
		return 0, 0, 0
	}
	used := total - available
	return clamp(float64(used) * 100 / float64(total)), used, total
}

func disk(path string) (float64, uint64, uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	if total == 0 {
		return 0, used, total
	}
	return clamp(float64(used) * 100 / float64(total)), used, total
}

func readNet(interfaceName string) netSample {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return netSample{}
	}
	defer file.Close()
	var total netSample
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		if name == "lo" || (interfaceName != "" && name != interfaceName) {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		total.rx += rx
		total.tx += tx
	}
	return total
}

func loadAvg() (float64, float64, float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0
	}
	v1, _ := strconv.ParseFloat(fields[0], 64)
	v5, _ := strconv.ParseFloat(fields[1], 64)
	v15, _ := strconv.ParseFloat(fields[2], 64)
	return v1, v5, v15
}

func uptime() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fields[0], 64)
	return int64(value)
}

type connectionStats struct {
	Total          int64
	SystemInbound  int64
	SystemOutbound int64
	XrayInbound    int64
	XrayOutbound   int64
}

type tcpEntry struct {
	localPort uint16
	state     string
	inode     string
}

func tcpConnections(xrayPID int, inboundPorts map[uint16]bool) connectionStats {
	inodes := processSocketInodes(xrayPID)
	stats := connectionStats{}
	entries := append(readTCP("/proc/net/tcp"), readTCP("/proc/net/tcp6")...)
	listenPorts := listeningPorts(entries)
	for _, entry := range entries {
		if entry.state != "01" {
			continue
		}
		stats.Total++
		if listenPorts[entry.localPort] {
			stats.SystemInbound++
		} else {
			stats.SystemOutbound++
		}
		if !inodes[entry.inode] {
			continue
		}
		if inboundPorts[entry.localPort] {
			stats.XrayInbound++
		} else {
			stats.XrayOutbound++
		}
	}
	return stats
}

func listeningPorts(entries []tcpEntry) map[uint16]bool {
	result := map[uint16]bool{}
	for _, entry := range entries {
		if entry.state == "0A" {
			result[entry.localPort] = true
		}
	}
	return result
}

func readTCP(path string) []tcpEntry {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	var entries []tcpEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "sl") || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		localPort, ok := parseProcNetPort(fields[1])
		if !ok {
			continue
		}
		entries = append(entries, tcpEntry{
			localPort: localPort,
			state:     fields[3],
			inode:     fields[9],
		})
	}
	return entries
}

func parseProcNetPort(value string) (uint16, bool) {
	_, portHex, ok := strings.Cut(value, ":")
	if !ok {
		return 0, false
	}
	raw, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(raw), true
}

func processSocketInodes(pid int) map[string]bool {
	result := map[string]bool{}
	if pid <= 0 {
		return result
	}
	matches, err := filepath.Glob(fmt.Sprintf("/proc/%d/fd/*", pid))
	if err != nil {
		return result
	}
	for _, item := range matches {
		target, err := os.Readlink(item)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(target, "socket:[") || !strings.HasSuffix(target, "]") {
			continue
		}
		inode := strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")
		if inode != "" {
			result[inode] = true
		}
	}
	return result
}

func inboundPorts(configPath string) map[uint16]bool {
	result := map[uint16]bool{}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return result
	}
	var cfg struct {
		Inbounds []struct {
			Port interface{} `json:"port"`
		} `json:"inbounds"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return result
	}
	for _, inbound := range cfg.Inbounds {
		port, ok := uint16Port(inbound.Port)
		if ok {
			result[port] = true
		}
	}
	return result
}

func uint16Port(value interface{}) (uint16, bool) {
	switch typed := value.(type) {
	case float64:
		if typed > 0 && typed <= 65535 {
			return uint16(typed), true
		}
	case string:
		raw, err := strconv.ParseUint(typed, 10, 16)
		if err == nil && raw > 0 {
			return uint16(raw), true
		}
	case json.Number:
		raw, err := strconv.ParseUint(string(typed), 10, 16)
		if err == nil && raw > 0 {
			return uint16(raw), true
		}
	}
	return 0, false
}

func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	rounded, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return rounded
}
