package tool

import (
	"encoding/json"
	"testing"
)

func TestSystemLogLinesToEntriesNormalizesJSONObjectLines(t *testing.T) {
	got := systemLogLinesToEntries([]string{
		`{"timestamp":"2026-07-07 12:00:00","level":"info","caller":"svc.go:1","content":"started","count":2,"duration":"10.0ms"}`,
		`{"timestamp":"2026-07-07 12:00:01","level":"error","content":{"code":500}}`,
	}, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(got))
	}
	if got[0].Timestamp != "2026-07-07 12:00:00" || got[0].Level != "info" || got[0].Caller != "svc.go:1" || got[0].Duration != "10.0ms" {
		t.Fatalf("known fields not mapped: %+v", got[0])
	}
	if string(got[0].Content) != `"started"` {
		t.Fatalf("string content should be preserved as raw json: %s", got[0].Content)
	}
	if string(got[1].Content) != `{"code":500}` {
		t.Fatalf("object content should be preserved as raw json: %s", got[1].Content)
	}

	var extra struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(got[0].Fields, &extra); err != nil {
		t.Fatalf("extra fields should be json object: %v", err)
	}
	if extra.Count != 2 {
		t.Fatalf("extra field not preserved: %+v", extra)
	}

	payload, err := json.Marshal(struct {
		List any `json:"list"`
	}{List: got})
	if err != nil {
		t.Fatalf("marshal log entries: %v", err)
	}
	var decoded struct {
		List []struct {
			Content any            `json:"content"`
			Fields  map[string]any `json:"fields"`
		} `json:"list"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal marshaled log lines: %v", err)
	}
	if decoded.List[0].Fields["count"].(float64) != 2 {
		t.Fatalf("unexpected marshaled payload: %s", payload)
	}
	if decoded.List[1].Content.(map[string]any)["code"].(float64) != 500 {
		t.Fatalf("unexpected nested content: %s", payload)
	}
}

func TestSystemLogLinesToEntriesSkipsNonObjectLines(t *testing.T) {
	got := systemLogLinesToEntries([]string{
		``,
		`not-json`,
		`["valid","but","not","object"]`,
		`"valid string but not object"`,
		`{"ok":true}`,
	}, nil)
	if len(got) != 1 {
		t.Fatalf("expected only object log lines, got %d", len(got))
	}
	var extra struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(got[0].Fields, &extra); err != nil {
		t.Fatalf("extra fields should be preserved: %v", err)
	}
	if !extra.OK {
		t.Fatalf("unexpected extra fields: %s", got[0].Fields)
	}
}

func TestSystemLogLinesToEntriesMasksSensitiveFields(t *testing.T) {
	got := systemLogLinesToEntries([]string{
		`{"timestamp":"2026-07-07 12:00:00","content":{"token":"secret-token"},"ip":"203.0.113.42","user-agent":"Clash Verge/v1.7.7","query":"token=abcdef1234567890&flag=clash","request":"GET example.com/v1/subscribe/config?token=abcdef1234567890","email":"user@example.com"}`,
	}, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 log line, got %d", len(got))
	}
	if string(got[0].Content) != `{"token":"***"}` {
		t.Fatalf("content should be masked: %s", got[0].Content)
	}
	var fields map[string]string
	if err := json.Unmarshal(got[0].Fields, &fields); err != nil {
		t.Fatalf("extra fields should decode: %v", err)
	}
	if fields["ip"] != "203.0.113.*" || fields["user-agent"] != "Clash Verge/***" || fields["email"] != "u**r@example.com" {
		t.Fatalf("sensitive fields not masked: %#v", fields)
	}
	if fields["query"] != "flag=clash&token=%2A%2A%2A" {
		t.Fatalf("query not sanitized: %#v", fields)
	}
	if fields["request"] != "GET example.com/v1/subscribe/config?token=%2A%2A%2A" {
		t.Fatalf("request not sanitized: %#v", fields)
	}
}
