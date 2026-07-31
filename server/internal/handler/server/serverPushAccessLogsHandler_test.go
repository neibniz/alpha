package server

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestReadAccessRequestBodyGzip(t *testing.T) {
	payload := []byte(`{"events":[{"event_id":"00000000-0000-5000-8000-000000000001"}]}`)
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := readAccessRequestBody(&compressed, "gzip")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q", got)
	}
}

func TestReadAccessRequestBodyRejectsExpandedPayloadOverLimit(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte(strings.Repeat("x", int(maxAccessRequestBodyBytes)+1))); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readAccessRequestBody(&compressed, "gzip"); err == nil {
		t.Fatal("expected expanded payload limit error")
	}
}
