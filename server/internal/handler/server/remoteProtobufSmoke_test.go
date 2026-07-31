package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/agentprotocol"
	"github.com/neibniz/alpha/server/internal/nodeauth"
	agentv1 "github.com/neibniz/alpha/server/internal/protocol/alpha/agent/v1"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func TestRemoteProtobufCompatibility(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("ALPHA_SMOKE_BASE_URL"), "/")
	configPath := os.Getenv("ALPHA_SMOKE_SERVER_CONFIG")
	if baseURL == "" || configPath == "" {
		t.Skip("remote Protobuf smoke test is not configured")
	}
	serverID := int64(1)
	if value := os.Getenv("ALPHA_SMOKE_SERVER_ID"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			t.Fatalf("invalid ALPHA_SMOKE_SERVER_ID")
		}
		serverID = parsed
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var runtimeConfig struct {
		Node struct {
			NodeSecret string `yaml:"NodeSecret"`
		} `yaml:"Node"`
	}
	if err := yaml.Unmarshal(configData, &runtimeConfig); err != nil {
		t.Fatal(err)
	}
	token, err := nodeauth.DeriveToken(runtimeConfig.Node.NodeSecret, serverID)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	protobufPayload, err := proto.Marshal(&agentv1.PushAccessEventsRequest{Protocol: "xray"})
	if err != nil {
		t.Fatal(err)
	}
	protobufBody := smokeRequest(t, client, baseURL+"/v2/server/access-events", agentprotocol.ContentType, protobufPayload, serverID, token)
	var protobufResponse agentv1.PushAccessEventsResponse
	if err := proto.Unmarshal(protobufBody, &protobufResponse); err != nil {
		t.Fatalf("decode v2 response: %v", err)
	}
	if protobufResponse.GetAccepted() != 0 || protobufResponse.GetRejected() != 0 {
		t.Fatalf("unexpected v2 response: %+v", &protobufResponse)
	}
	trafficPayload, err := proto.Marshal(&agentv1.PushTrafficReportsRequest{
		Protocol: "xray",
		Reports: []*agentv1.TrafficReport{{
			ReportId:        "00000000-0000-4000-8000-00000000f001",
			SampledAtMs:     time.Now().UnixMilli(),
			IntervalSeconds: 5,
			Sequence:        1,
			Traffic:         []*agentv1.UserTraffic{{}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	smokeRequest(t, client, baseURL+"/v2/server/traffic-reports", agentprotocol.ContentType, trafficPayload, serverID, token)

	jsonBody := smokeRequest(t, client, baseURL+"/v1/server/access-logs?protocol=xray", "application/json", []byte(`{"events":[]}`), serverID, token)
	if !bytes.Contains(jsonBody, []byte(`"code":200`)) {
		t.Fatalf("unexpected v1 response: %s", string(jsonBody))
	}
}

func smokeRequest(t *testing.T, client *http.Client, endpoint, contentType string, payload []byte, serverID int64, token string) []byte {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Alpha-Server-ID", strconv.FormatInt(serverID, 10))
	request.Header.Set("Content-Type", contentType)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64<<10))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		t.Fatalf("%s returned HTTP %d", endpoint, response.StatusCode)
	}
	if strings.Contains(contentType, "protobuf") && response.Header.Get("X-Alpha-Agent-Protocol") != "protobuf-v1" {
		t.Fatal(fmt.Errorf("v2 response is missing the protocol version header"))
	}
	return body
}
