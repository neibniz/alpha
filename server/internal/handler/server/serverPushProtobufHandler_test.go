package server

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neibniz/alpha/server/internal/agentprotocol"
	agentv1 "github.com/neibniz/alpha/server/internal/protocol/alpha/agent/v1"
	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

func TestBindAgentProtobufReadsGzipPayload(t *testing.T) {
	payload, err := proto.Marshal(&agentv1.PushOnlineUsersRequest{Mode: "delta", Sequence: 9, Active: []int64{12, 13}})
	if err != nil {
		t.Fatal(err)
	}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v2/server/online-users", bytes.NewReader(compressed.Bytes()))
	context.Request.Header.Set("Content-Type", agentprotocol.ContentType)
	context.Request.Header.Set("Content-Encoding", "gzip")
	var message agentv1.PushOnlineUsersRequest
	if err := bindAgentProtobuf(context, &message, 4<<20, "online users"); err != nil {
		t.Fatal(err)
	}
	if message.GetSequence() != 9 || len(message.GetActive()) != 2 {
		t.Fatalf("message = %+v", &message)
	}
}

func TestBindAgentProtobufRejectsMediaTypeAndOversizedBody(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v2/server/online-users", bytes.NewReader(nil))
	context.Request.Header.Set("Content-Type", "application/json")
	if status := protobufBindStatus(bindAgentProtobuf(context, &agentv1.PushOnlineUsersRequest{}, 32, "online users")); status != http.StatusUnsupportedMediaType {
		t.Fatalf("media type status = %d", status)
	}

	context, _ = gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/v2/server/online-users", bytes.NewReader(make([]byte, 33)))
	context.Request.Header.Set("Content-Type", agentprotocol.ContentType)
	if status := protobufBindStatus(bindAgentProtobuf(context, &agentv1.PushOnlineUsersRequest{}, 32, "online users")); status != http.StatusBadRequest {
		t.Fatalf("oversized body status = %d", status)
	}
}
