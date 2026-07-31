package panel

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/config"
	agentv1 "github.com/neibniz/alpha/xray-agent/internal/protocol/alpha/agent/v1"
	"google.golang.org/protobuf/proto"
)

func TestPostAccessEventsUsesCompressedProtobuf(t *testing.T) {
	events := make([]AccessEvent, 100)
	for index := range events {
		events[index] = AccessEvent{
			EventID:         "00000000-0000-5000-8000-000000000001",
			Timestamp:       time.Now().UnixMilli(),
			UserSubscribeID: 12,
			SourceIP:        "203.0.113.7",
			DestinationHost: "repeated-destination.example.com",
			Status:          "accepted",
			Reason:          strings.Repeat("route-observation-", 4),
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v2/server/access-events" {
			t.Errorf("path = %q", request.URL.Path)
		}
		assertAgentAuthentication(t, request)
		if request.URL.RawQuery != "" {
			t.Errorf("credentials or protocol leaked into query = %q", request.URL.RawQuery)
		}
		if request.Header.Get("Content-Type") != agentProtobufContentType || request.Header.Get("Accept") != agentProtobufContentType {
			t.Errorf("protobuf headers are incomplete")
		}
		if request.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("content encoding = %q", request.Header.Get("Content-Encoding"))
		}
		var payload agentv1.PushAccessEventsRequest
		decodeProtobufRequest(t, request, &payload)
		if payload.GetProtocol() != "xray" || len(payload.GetEvents()) != len(events) {
			t.Fatalf("payload protocol=%q events=%d", payload.GetProtocol(), len(payload.GetEvents()))
		}
		response, err := proto.Marshal(&agentv1.PushAccessEventsResponse{
			Accepted: uint32(len(events)),
			IpLimitDecisions: []*agentv1.IPLimitDecision{{
				EventId:         events[0].EventID,
				UserSubscribeId: 12,
				SourceNetwork:   "203.0.113.7/32",
				Blocked:         true,
				ExpiresAtMs:     1_784_059_230_000,
				Active:          2,
				Limit:           2,
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		writer.Header().Set("Content-Type", agentProtobufContentType)
		_, _ = writer.Write(response)
	}))
	defer server.Close()

	client := NewClient(config.PanelConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"})
	result, err := client.PostAccessEvents(context.Background(), events)
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted != len(events) || len(result.IPLimitDecisions) != 1 || !result.IPLimitDecisions[0].Blocked {
		t.Fatalf("access result = %+v", result)
	}
}

func TestPostTrafficReportsUsesProtobufSamplingMetadata(t *testing.T) {
	reports := sampleTrafficReports(1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v2/server/traffic-reports" || request.URL.RawQuery != "" {
			t.Errorf("request endpoint = %s?%s", request.URL.Path, request.URL.RawQuery)
		}
		assertAgentAuthentication(t, request)
		var payload agentv1.PushTrafficReportsRequest
		decodeProtobufRequest(t, request, &payload)
		if payload.GetProtocol() != "xray" || len(payload.GetReports()) != 1 {
			t.Fatalf("payload = %+v", &payload)
		}
		report := payload.GetReports()[0]
		if report.GetReportId() != reports[0].ReportID || report.GetSequence() != 17 || report.GetIntervalSeconds() != 5 {
			t.Errorf("report = %+v", report)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(config.PanelConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"})
	if err := client.PostTrafficReports(context.Background(), "xray", reports); err != nil {
		t.Fatal(err)
	}
}

func TestPostOnlineUsersUsesProtobuf(t *testing.T) {
	update := OnlineUsersUpdate{Mode: "delta", InstanceID: "agent-a", Seq: 9, Active: []int64{12, 13}, Inactive: []int64{14}, TTLSeconds: 30}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v2/server/online-users" {
			t.Errorf("path = %q", request.URL.Path)
		}
		var payload agentv1.PushOnlineUsersRequest
		decodeProtobufRequest(t, request, &payload)
		if payload.GetMode() != update.Mode || payload.GetInstanceId() != update.InstanceID || payload.GetSequence() != update.Seq || len(payload.GetActive()) != 2 {
			t.Fatalf("payload = %+v", &payload)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(config.PanelConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"})
	if err := client.PostOnlineUsers(context.Background(), update); err != nil {
		t.Fatal(err)
	}
}

func TestTrafficReportsFallbackToJSONAndCacheUnsupportedServer(t *testing.T) {
	protobufRequests := 0
	jsonRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/server/traffic-reports":
			protobufRequests++
			writer.WriteHeader(http.StatusUnsupportedMediaType)
		case "/v1/server/push":
			jsonRequests++
			if request.URL.Query().Get("protocol") != "xray" || request.Header.Get("Content-Type") != "application/json" {
				t.Errorf("invalid JSON fallback request")
			}
			var payload struct {
				Reports []TrafficReport `json:"reports"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Reports) != 1 {
				t.Errorf("reports = %d", len(payload.Reports))
			}
			writer.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path %q", request.URL.Path)
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(config.PanelConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"})
	for range 2 {
		if err := client.PostTrafficReports(context.Background(), "xray", sampleTrafficReports(1)); err != nil {
			t.Fatal(err)
		}
	}
	if protobufRequests != 1 || jsonRequests != 2 {
		t.Fatalf("protobuf requests=%d JSON requests=%d", protobufRequests, jsonRequests)
	}
}

func TestAccessAndOnlineUsersFallbackToJSON(t *testing.T) {
	accessJSON := 0
	onlineJSON := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v2/server/access-events":
			writer.WriteHeader(http.StatusNotFound)
		case "/v1/server/access-logs":
			accessJSON++
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"code":200,"msg":"ok","data":{"accepted":1,"rejected":0}}`))
		case "/v2/server/online-users":
			writer.WriteHeader(http.StatusMethodNotAllowed)
		case "/v1/server/online-users":
			onlineJSON++
			writer.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path %q", request.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(config.PanelConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"})
	result, err := client.PostAccessEvents(context.Background(), []AccessEvent{{EventID: "00000000-0000-5000-8000-000000000001", Timestamp: time.Now().UnixMilli(), Status: "accepted"}})
	if err != nil || result.Accepted != 1 {
		t.Fatalf("access result=%+v error=%v", result, err)
	}
	if err := client.PostOnlineUsers(context.Background(), OnlineUsersUpdate{Mode: "full", Seq: 1}); err != nil {
		t.Fatal(err)
	}
	if accessJSON != 1 || onlineJSON != 1 {
		t.Fatalf("access JSON=%d online JSON=%d", accessJSON, onlineJSON)
	}
}

func TestAccessJSONRequiresExplicitCompleteAcknowledgement(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing data", body: `{"code":200,"msg":"ok"}`},
		{name: "null data", body: `{"code":200,"msg":"ok","data":null}`},
		{name: "partial count", body: `{"code":200,"msg":"ok","data":{"accepted":0,"rejected":0}}`},
		{name: "negative count", body: `{"code":200,"msg":"ok","data":{"accepted":-1,"rejected":2}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/v2/server/access-events" {
					writer.WriteHeader(http.StatusNotFound)
					return
				}
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()

			client := NewClient(config.PanelConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"})
			_, err := client.PostAccessEvents(context.Background(), []AccessEvent{{
				EventID: "00000000-0000-5000-8000-000000000001", Timestamp: time.Now().UnixMilli(), Status: "accepted",
			}})
			if err == nil {
				t.Fatal("missing or invalid acknowledgement should keep the event queued")
			}
		})
	}
}

func TestProtobufServerErrorDoesNotFallback(t *testing.T) {
	jsonRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/v1/server/push" {
			jsonRequests++
		}
		payload, err := proto.Marshal(&agentv1.ErrorResponse{Code: 500, Message: "ingest unavailable"})
		if err != nil {
			t.Fatal(err)
		}
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write(payload)
	}))
	defer server.Close()

	client := NewClient(config.PanelConfig{BaseURL: server.URL, ServerID: 7, Token: "node-token"})
	err := client.PostTrafficReports(context.Background(), "xray", sampleTrafficReports(1))
	if err == nil || !strings.Contains(err.Error(), "ingest unavailable") {
		t.Fatalf("error = %v", err)
	}
	if jsonRequests != 0 {
		t.Fatalf("JSON fallback requests = %d", jsonRequests)
	}
}

func TestProtobufTrafficPayloadIsSmallerThanJSON(t *testing.T) {
	reports := sampleTrafficReports(1_000)
	jsonPayload, err := json.Marshal(struct {
		Reports []TrafficReport `json:"reports"`
	}{Reports: reports})
	if err != nil {
		t.Fatal(err)
	}
	protobufPayload, err := proto.Marshal(sampleTrafficProtobuf(reports))
	if err != nil {
		t.Fatal(err)
	}
	if len(protobufPayload) >= len(jsonPayload) {
		t.Fatalf("protobuf bytes=%d JSON bytes=%d", len(protobufPayload), len(jsonPayload))
	}
	t.Logf("protobuf bytes=%d JSON bytes=%d reduction=%.1f%%", len(protobufPayload), len(jsonPayload), 100*(1-float64(len(protobufPayload))/float64(len(jsonPayload))))
}

func BenchmarkTrafficReportEncoding(b *testing.B) {
	reports := sampleTrafficReports(1_000)
	protobufMessage := sampleTrafficProtobuf(reports)
	jsonMessage := struct {
		Reports []TrafficReport `json:"reports"`
	}{Reports: reports}
	b.Run("json", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = json.Marshal(jsonMessage)
		}
	})
	b.Run("protobuf", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = proto.Marshal(protobufMessage)
		}
	})
}

func assertAgentAuthentication(t *testing.T, request *http.Request) {
	t.Helper()
	if request.Header.Get("Authorization") != "Bearer node-token" || request.Header.Get("X-Alpha-Server-ID") != "7" {
		t.Errorf("authentication headers are incomplete")
	}
	if request.URL.Query().Get("secret_key") != "" || request.URL.Query().Get("server_id") != "" {
		t.Errorf("credentials leaked into query = %q", request.URL.RawQuery)
	}
}

func decodeProtobufRequest(t *testing.T, request *http.Request, message proto.Message) {
	t.Helper()
	var reader io.Reader = request.Body
	if request.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}
	payload, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := proto.Unmarshal(payload, message); err != nil {
		t.Fatal(err)
	}
}

func sampleTrafficReports(rows int) []TrafficReport {
	traffic := make([]UserTraffic, rows)
	for index := range traffic {
		traffic[index] = UserTraffic{SID: int64(index + 1), Upload: int64(index + 100), Download: int64(index + 200)}
	}
	return []TrafficReport{{
		ReportID:        "00000000-0000-4000-8000-000000000001",
		SampledAt:       1_720_000_000_000,
		IntervalSeconds: 5,
		Sequence:        17,
		Traffic:         traffic,
	}}
}

func sampleTrafficProtobuf(reports []TrafficReport) *agentv1.PushTrafficReportsRequest {
	message := &agentv1.PushTrafficReportsRequest{Protocol: "xray", Reports: make([]*agentv1.TrafficReport, 0, len(reports))}
	for _, report := range reports {
		traffic := make([]*agentv1.UserTraffic, 0, len(report.Traffic))
		for _, row := range report.Traffic {
			traffic = append(traffic, &agentv1.UserTraffic{UserSubscribeId: row.SID, UploadBytes: row.Upload, DownloadBytes: row.Download})
		}
		message.Reports = append(message.Reports, &agentv1.TrafficReport{
			ReportId:        report.ReportID,
			SampledAtMs:     report.SampledAt,
			IntervalSeconds: report.IntervalSeconds,
			Sequence:        report.Sequence,
			Traffic:         traffic,
		})
	}
	return message
}
