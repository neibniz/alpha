package syncer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/neibniz/alpha/xray-agent/internal/config"
	"github.com/neibniz/alpha/xray-agent/internal/panel"
	"github.com/neibniz/alpha/xray-agent/internal/status"
	"github.com/neibniz/alpha/xray-agent/internal/supervisor"
)

func TestReportStatusWSSendsInitialStatusImmediately(t *testing.T) {
	received := make(chan status.Payload, 1)
	serverErrors := make(chan error, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			serverErrors <- err
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		var payload status.Payload
		if err := conn.ReadJSON(&payload); err != nil {
			serverErrors <- err
			return
		}
		received <- payload
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.Server = config.ServerConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"}
	cfg.Panel = cfg.Server
	cfg.Report.StatusInterval = time.Hour
	cfg.Xray.WorkDir = t.TempDir()
	cfg.Xray.BinPath = "/does-not-exist"
	xraySupervisor := supervisor.New(cfg.Xray)
	collector := status.NewCollector(cfg, xraySupervisor, status.NewRuntimeState())
	runner := &Runner{cfg: cfg, panel: panel.NewClient(cfg.Panel)}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- runner.reportStatusWS(ctx, collector) }()

	select {
	case payload := <-received:
		if payload.ServerID != 7 {
			t.Fatalf("server id = %d, want 7", payload.ServerID)
		}
	case err := <-serverErrors:
		t.Fatalf("status websocket server error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("initial status waited for the periodic report ticker")
	}
	cancel()
	select {
	case <-result:
	case <-time.After(2 * time.Second):
		t.Fatal("reportStatusWS did not stop after cancellation")
	}
}

func TestEffectiveUsersUsesCacheForUnchangedVersion(t *testing.T) {
	runner := &Runner{
		userVersion: "v1",
		userCache: []panel.XrayUser{
			{Email: "u_1", UUID: "uuid-1", Inbounds: []string{"main"}},
		},
	}

	users, err := runner.effectiveUsers(panel.XrayUsersResponse{
		Version: "v1",
		Full:    false,
	})
	if err != nil {
		t.Fatalf("effectiveUsers() error = %v", err)
	}
	if len(users) != 1 || users[0].Email != "u_1" {
		t.Fatalf("effectiveUsers() = %#v", users)
	}

	users[0].Inbounds[0] = "mutated"
	if runner.userCache[0].Inbounds[0] != "main" {
		t.Fatalf("effectiveUsers() returned mutable cache reference")
	}
}

func TestEffectiveUsersRefreshesCacheForFullResponse(t *testing.T) {
	runner := &Runner{
		userVersion: "v1",
		userCache: []panel.XrayUser{
			{Email: "u_1", UUID: "uuid-1"},
		},
	}

	users, err := runner.effectiveUsers(panel.XrayUsersResponse{
		Version: "v2",
		Full:    true,
		Users: []panel.XrayUser{
			{Email: "u_2", UUID: "uuid-2"},
		},
	})
	if err != nil {
		t.Fatalf("effectiveUsers() error = %v", err)
	}
	if len(users) != 1 || users[0].Email != "u_2" {
		t.Fatalf("effectiveUsers() = %#v", users)
	}
	if len(runner.userCache) != 1 || runner.userCache[0].Email != "u_2" {
		t.Fatalf("userCache = %#v", runner.userCache)
	}
}
