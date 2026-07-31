package xray

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	statscmd "github.com/xtls/xray-core/app/stats/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type TrafficDelta struct {
	UserSubscribeID int64
	Upload          int64
	Download        int64
}

type TrafficTotals struct {
	Upload   int64
	Download int64
}

type StatsCollector interface {
	Collect(ctx context.Context) ([]TrafficDelta, error)
	PendingCursor() map[string]int64
	RestoreCursor(map[string]int64) error
	Commit() error
}

type NoopStatsCollector struct{}

func (NoopStatsCollector) Collect(context.Context) ([]TrafficDelta, error) {
	return nil, nil
}

func (NoopStatsCollector) Commit() error {
	return nil
}

func (NoopStatsCollector) PendingCursor() map[string]int64 {
	return nil
}

func (NoopStatsCollector) RestoreCursor(map[string]int64) error {
	return nil
}

func ParseUserEmail(email string) (int64, bool) {
	if !strings.HasPrefix(email, "u_") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(email, "u_"), 10, 64)
	return id, err == nil && id > 0
}

type GRPCStatsCollector struct {
	address    string
	cursorFile string
	queryMu    sync.Mutex
	conn       *grpc.ClientConn
	client     statscmd.StatsServiceClient
	mu         sync.Mutex
	cursor     map[string]int64
	pending    map[string]int64
}

func NewGRPCStatsCollector(apiAddress string, cursorFile string) *GRPCStatsCollector {
	collector := &GRPCStatsCollector{
		address:    apiAddress,
		cursorFile: cursorFile,
		cursor:     map[string]int64{},
	}
	collector.loadCursor()
	return collector
}

func (c *GRPCStatsCollector) Collect(ctx context.Context) ([]TrafficDelta, error) {
	resp, err := c.queryUserStats(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	byUser := make(map[int64]*TrafficDelta)
	pending := make(map[string]int64)
	for _, stat := range resp.GetStat() {
		if stat == nil {
			continue
		}
		value := stat.GetValue()
		pending[stat.GetName()] = value
		userID, direction, ok := parseTrafficStatName(stat.GetName())
		if !ok {
			continue
		}
		previous := c.cursor[stat.GetName()]
		deltaValue := value - previous
		if deltaValue < 0 {
			deltaValue = value
		}
		if deltaValue <= 0 {
			continue
		}
		delta := byUser[userID]
		if delta == nil {
			delta = &TrafficDelta{UserSubscribeID: userID}
			byUser[userID] = delta
		}
		switch direction {
		case "uplink":
			delta.Upload += deltaValue
		case "downlink":
			delta.Download += deltaValue
		}
	}
	c.pending = pending
	result := make([]TrafficDelta, 0, len(byUser))
	for _, delta := range byUser {
		result = append(result, *delta)
	}
	return result, nil
}

func (c *GRPCStatsCollector) CollectTotals(ctx context.Context) (TrafficTotals, error) {
	resp, err := c.queryUserStats(ctx)
	if err != nil {
		return TrafficTotals{}, err
	}
	var totals TrafficTotals
	for _, stat := range resp.GetStat() {
		if stat == nil {
			continue
		}
		_, direction, ok := parseTrafficStatName(stat.GetName())
		if !ok {
			continue
		}
		switch direction {
		case "uplink":
			totals.Upload += stat.GetValue()
		case "downlink":
			totals.Download += stat.GetValue()
		}
	}
	return totals, nil
}

func (c *GRPCStatsCollector) Commit() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pending == nil {
		return nil
	}
	for name, value := range c.pending {
		c.cursor[name] = value
	}
	c.pending = nil
	return c.saveCursor()
}

func (c *GRPCStatsCollector) PendingCursor() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return copyStatsCursor(c.pending)
}

func (c *GRPCStatsCollector) RestoreCursor(cursor map[string]int64) error {
	if len(cursor) == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cursor = copyStatsCursor(cursor)
	c.pending = nil
	return c.saveCursor()
}

func (c *GRPCStatsCollector) Close() error {
	c.queryMu.Lock()
	defer c.queryMu.Unlock()
	return c.closeConnLocked()
}

func (c *GRPCStatsCollector) loadCursor() {
	if c.cursorFile == "" {
		return
	}
	data, err := os.ReadFile(c.cursorFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &c.cursor)
}

func (c *GRPCStatsCollector) saveCursor() error {
	if c.cursorFile == "" {
		return nil
	}
	data, err := json.MarshalIndent(c.cursor, "", "  ")
	if err != nil {
		return err
	}
	tempPath := c.cursorFile + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tempPath, c.cursorFile)
}

func copyStatsCursor(cursor map[string]int64) map[string]int64 {
	if len(cursor) == 0 {
		return nil
	}
	result := make(map[string]int64, len(cursor))
	for name, value := range cursor {
		result[name] = value
	}
	return result
}

func parseTrafficStatName(name string) (int64, string, bool) {
	parts := strings.Split(name, ">>>")
	if len(parts) != 4 || parts[0] != "user" || parts[2] != "traffic" {
		return 0, "", false
	}
	id, ok := ParseUserEmail(parts[1])
	if !ok {
		return 0, "", false
	}
	if parts[3] != "uplink" && parts[3] != "downlink" {
		return 0, "", false
	}
	return id, parts[3], true
}

func (c *GRPCStatsCollector) queryUserStats(ctx context.Context) (*statscmd.QueryStatsResponse, error) {
	c.queryMu.Lock()
	defer c.queryMu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := c.statsClientLocked(callCtx)
	if err != nil {
		return nil, c.friendlyError(err)
	}

	resp, err := client.QueryStats(callCtx, &statscmd.QueryStatsRequest{
		Pattern: "user>>>",
		Reset_:  false,
	})
	if err != nil {
		_ = c.closeConnLocked()
		return nil, c.friendlyError(err)
	}
	return resp, nil
}

func (c *GRPCStatsCollector) statsClientLocked(ctx context.Context) (statscmd.StatsServiceClient, error) {
	if c.client != nil {
		return c.client, nil
	}
	if err := c.checkReachable(ctx); err != nil {
		return nil, err
	}
	conn, err := grpc.DialContext(ctx, c.address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	c.conn = conn
	c.client = statscmd.NewStatsServiceClient(conn)
	return c.client, nil
}

func (c *GRPCStatsCollector) checkReachable(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(checkCtx, "tcp", c.address)
	if err != nil {
		return err
	}
	return conn.Close()
}

func (c *GRPCStatsCollector) closeConnLocked() error {
	if c.conn == nil {
		c.client = nil
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	c.client = nil
	return err
}

func (c *GRPCStatsCollector) friendlyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
		return fmt.Errorf("xray stats api %s timed out; check xray.api_address and xray.api_port", c.address)
	}
	return fmt.Errorf("xray stats api %s unavailable; check xray.api_address and xray.api_port: %w", c.address, err)
}
