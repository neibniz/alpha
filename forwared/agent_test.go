package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendEventsUsesAuthenticatedGzipRequest(t *testing.T) {
	events := []ConnectionEvent{{EventID: "event-1", ConnectionID: "connection-1"}}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer instance-token" {
			t.Errorf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		if request.Header.Get("X-Alpha-Forwarder-ID") != "instance-1" {
			t.Errorf("unexpected instance header: %q", request.Header.Get("X-Alpha-Forwarder-ID"))
		}
		if request.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("unexpected content encoding: %q", request.Header.Get("Content-Encoding"))
		}
		reader, err := gzip.NewReader(request.Body)
		if err != nil {
			t.Error(err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		defer reader.Close()
		var payload struct {
			Events []ConnectionEvent `json:"events"`
		}
		if err := json.NewDecoder(reader).Decode(&payload); err != nil {
			t.Error(err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(payload.Events) != 1 || payload.Events[0].EventID != events[0].EventID {
			t.Errorf("unexpected event payload: %#v", payload.Events)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	agent := &Agent{
		cfg:      &Config{BaseURL: server.URL},
		client:   server.Client(),
		identity: agentIdentity{InstanceID: "instance-1"},
		token:    "instance-token",
	}
	if err := agent.sendEvents(context.Background(), events); err != nil {
		t.Fatal(err)
	}
}
