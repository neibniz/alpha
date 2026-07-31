package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/neibniz/alpha/forwared/logger"
	"github.com/google/uuid"
)

type ForwardManager struct {
	sessions      map[uint]*ForwardSession
	mu            sync.Mutex
	metrics       *Metrics
	emit          func(ConnectionEvent)
	bootID        func() string
	configVersion func() string
}

type ForwardSession struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	done       chan struct{}
	ready      chan error
	targets    []ForwaredItem
	next       atomic.Uint64
}

type copyResult struct {
	initiator string
	err       error
}

type connectionCounters struct {
	upload   atomic.Uint64
	download atomic.Uint64
}

type countingReader struct {
	reader io.Reader
	add    func(uint64)
}

func (r countingReader) Read(payload []byte) (int, error) {
	count, err := r.reader.Read(payload)
	if count > 0 {
		r.add(uint64(count))
	}
	return count, err
}

func NewForwardManager(
	metrics *Metrics,
	emit func(ConnectionEvent),
	bootID func() string,
	configVersion func() string,
) *ForwardManager {
	return &ForwardManager{
		sessions:      make(map[uint]*ForwardSession),
		metrics:       metrics,
		emit:          emit,
		bootID:        bootID,
		configVersion: configVersion,
	}
}

func (m *ForwardManager) AddForward(localPort uint, targets []ForwaredItem) error {
	if localPort == 0 || localPort > 65535 || len(targets) == 0 {
		return fmt.Errorf("invalid forward rule")
	}
	m.mu.Lock()
	if _, exists := m.sessions[localPort]; exists {
		m.mu.Unlock()
		return fmt.Errorf("port %d already in use", localPort)
	}
	done := make(chan struct{})
	ready := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	session := &ForwardSession{
		ctx:        ctx,
		cancelFunc: cancel,
		done:       done,
		ready:      ready,
		targets:    append([]ForwaredItem(nil), targets...),
	}
	m.sessions[localPort] = session
	m.mu.Unlock()

	go m.runForwarder(ctx, localPort, session)
	if err := <-ready; err != nil {
		m.mu.Lock()
		if current := m.sessions[localPort]; current == session {
			delete(m.sessions, localPort)
		}
		m.mu.Unlock()
		cancel()
		<-done
		return err
	}
	return nil
}

func (m *ForwardManager) RemoveForward(localPort uint) error {
	m.mu.Lock()
	session, exists := m.sessions[localPort]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("port %d not found", localPort)
	}
	delete(m.sessions, localPort)
	session.cancelFunc()
	m.mu.Unlock()
	<-session.done
	return nil
}

func (m *ForwardManager) runForwarder(ctx context.Context, localPort uint, session *ForwardSession) {
	defer close(session.done)
	lc := net.ListenConfig{
		Control: func(_ string, _ string, conn syscall.RawConn) error {
			var optionErr error
			if err := conn.Control(func(fd uintptr) {
				optionErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			}); err != nil {
				return err
			}
			return optionErr
		},
	}
	listenIP := strings.TrimSpace(session.targets[0].ListenIP)
	address := net.JoinHostPort(listenIP, strconv.FormatUint(uint64(localPort), 10))
	listener, err := lc.Listen(ctx, "tcp", address)
	if err != nil {
		session.ready <- err
		logger.Logger.Error("Failed to listen", slog.String("address", address), slog.Any("error", err))
		return
	}
	defer listener.Close()
	session.ready <- nil
	logger.Logger.Info("Listening", slog.String("address", address), slog.Int("targets", len(session.targets)))

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Logger.Error("Accept error", slog.String("address", address), slog.Any("error", err))
			continue
		}
		target := session.pickTarget()
		if target.LocalPort == 0 {
			_ = clientConn.Close()
			continue
		}
		m.metrics.Accepted(target.RuleID)
		go m.handleConnection(ctx, clientConn, target)
	}
}

func (s *ForwardSession) pickTarget() ForwaredItem {
	if len(s.targets) == 0 {
		return ForwaredItem{}
	}
	index := s.next.Add(1)
	return s.targets[(index-1)%uint64(len(s.targets))]
}

func (m *ForwardManager) StopAll() {
	m.mu.Lock()
	sessions := make([]*ForwardSession, 0, len(m.sessions))
	for port, session := range m.sessions {
		session.cancelFunc()
		delete(m.sessions, port)
		sessions = append(sessions, session)
	}
	m.mu.Unlock()
	for _, session := range sessions {
		<-session.done
	}
}

func (m *ForwardManager) handleConnection(ctx context.Context, clientConn net.Conn, item ForwaredItem) {
	defer clientConn.Close()
	ruleMetrics := m.metrics.ensureRule(item.RuleID)
	connectionID := uuid.NewString()
	sourceIP, sourcePort := splitEndpoint(clientConn.RemoteAddr())
	base := m.baseEvent(connectionID, sourceIP, sourcePort, item)

	if !IsIPAllowed(sourceIP) {
		m.metrics.Blocked(item.RuleID)
		event := base
		event.EventID = uuid.NewString()
		event.EventTime = time.Now().UnixMilli()
		event.EventType = "blocked"
		event.CloseReason = "policy"
		m.emit(event)
		logger.Logger.Warn("Connection blocked",
			slog.String("connection_id", connectionID),
			slog.String("rule_id", item.RuleID),
			slog.Int("listen_port", int(item.LocalPort)))
		return
	}

	backendAddr := net.JoinHostPort(item.TargetHost, strconv.FormatUint(uint64(item.TargetPort), 10))
	dialStarted := time.Now()
	dialer := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	targetConn, err := dialer.DialContext(ctx, "tcp", backendAddr)
	if err != nil {
		m.metrics.Failed(item.RuleID)
		event := base
		event.EventID = uuid.NewString()
		event.EventTime = time.Now().UnixMilli()
		event.EventType = "dial_failed"
		event.DurationMS = time.Since(dialStarted).Milliseconds()
		event.CloseReason = classifyConnectionError(err)
		event.Error = boundedEventError(err)
		m.emit(event)
		logger.Logger.Warn("Failed to connect to target",
			slog.String("connection_id", connectionID),
			slog.String("rule_id", item.RuleID),
			slog.String("backend_addr", backendAddr),
			slog.Any("error", err))
		return
	}
	defer targetConn.Close()

	connectedAt := time.Now()
	resolvedIP, resolvedPort := splitEndpoint(targetConn.RemoteAddr())
	outboundIP, outboundPort := splitEndpoint(targetConn.LocalAddr())
	base.ResolvedTargetIP = resolvedIP
	base.ResolvedTargetPort = resolvedPort
	base.ForwarderOutboundIP = outboundIP
	base.ForwarderOutboundPort = outboundPort
	base.ConnectedAt = connectedAt.UnixMilli()

	connected := base
	connected.EventID = uuid.NewString()
	connected.EventTime = connectedAt.UnixMilli()
	connected.EventType = "connected"
	m.emit(connected)
	m.metrics.Connected(item.RuleID)
	defer m.metrics.Disconnected(item.RuleID)

	logger.Logger.Info("Connection established",
		slog.String("connection_id", connectionID),
		slog.String("rule_id", item.RuleID),
		slog.Int("listen_port", int(item.LocalPort)),
		slog.String("backend_addr", backendAddr))

	var closeOnce sync.Once
	closeConnections := func() {
		closeOnce.Do(func() {
			_ = clientConn.Close()
			_ = targetConn.Close()
		})
	}
	counters := &connectionCounters{}
	results := make(chan copyResult, 2)
	go func() {
		_, copyErr := io.Copy(targetConn, countingReader{
			reader: clientConn,
			add: func(count uint64) {
				counters.upload.Add(count)
				m.metrics.addUpload(ruleMetrics, count)
			},
		})
		results <- copyResult{initiator: "client", err: copyErr}
	}()
	go func() {
		_, copyErr := io.Copy(clientConn, countingReader{
			reader: targetConn,
			add: func(count uint64) {
				counters.download.Add(count)
				m.metrics.addDownload(ruleMetrics, count)
			},
		})
		results <- copyResult{initiator: "backend", err: copyErr}
	}()

	var first copyResult
	select {
	case first = <-results:
	case <-ctx.Done():
		first = copyResult{initiator: "process", err: ctx.Err()}
	}
	closeConnections()
	<-results
	if first.initiator == "process" {
		<-results
	}
	closedAt := time.Now()
	closeErr := first.err
	closed := base
	closed.EventID = uuid.NewString()
	closed.EventTime = closedAt.UnixMilli()
	closed.EventType = "closed"
	closed.ConnectedAt = connectedAt.UnixMilli()
	closed.ClosedAt = closedAt.UnixMilli()
	closed.DurationMS = closedAt.Sub(connectedAt).Milliseconds()
	closed.UploadBytes = counters.upload.Load()
	closed.DownloadBytes = counters.download.Load()
	closed.CloseInitiator = first.initiator
	closed.CloseReason = classifyConnectionError(closeErr)
	closed.Error = boundedEventError(closeErr)
	m.emit(closed)

	logger.Logger.Info("Connection closed",
		slog.String("connection_id", connectionID),
		slog.String("rule_id", item.RuleID),
		slog.String("initiator", first.initiator),
		slog.String("reason", closed.CloseReason),
		slog.Uint64("upload_bytes", closed.UploadBytes),
		slog.Uint64("download_bytes", closed.DownloadBytes),
		slog.Int64("duration_ms", closed.DurationMS))
}

func (m *ForwardManager) baseEvent(connectionID, sourceIP string, sourcePort uint16, item ForwaredItem) ConnectionEvent {
	return ConnectionEvent{
		ConnectionID:         connectionID,
		BootID:               m.bootID(),
		SourceIP:             sourceIP,
		SourcePort:           sourcePort,
		ListenIP:             item.ListenIP,
		ListenPort:           uint16(item.LocalPort),
		ConfiguredTargetHost: item.TargetHost,
		ConfiguredTargetPort: uint16(item.TargetPort),
		RuleID:               item.RuleID,
		ServerID:             item.ServerID,
		NodeID:               item.NodeID,
		Inbound:              item.Inbound,
		ConfigVersion:        m.configVersion(),
	}
}

func splitEndpoint(address net.Addr) (string, uint16) {
	if address == nil {
		return "", 0
	}
	host, portText, err := net.SplitHostPort(address.String())
	if err != nil {
		return "", 0
	}
	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil {
		return host, 0
	}
	return host, uint16(port)
}

func classifyConnectionError(err error) string {
	switch {
	case err == nil, errors.Is(err, io.EOF):
		return "eof"
	case errors.Is(err, context.Canceled):
		return "shutdown"
	case errors.Is(err, net.ErrClosed):
		return "closed"
	case errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.ECONNABORTED):
		return "reset"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	return "io_error"
}

func boundedEventError(err error) string {
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return ""
	}
	value := err.Error()
	if len(value) > 1024 {
		return value[:1024]
	}
	return value
}
