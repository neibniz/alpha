package tool

import (
	"encoding/json"
	"testing"

	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestSystemLogLinesToEntriesKeepsStableEmptyShape(t *testing.T) {
	got := systemLogLinesToEntries(nil, nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty system log entries = %#v, want non-nil empty slice", got)
	}

	got = systemLogLinesToEntries([]string{"", "  ", "{bad"}, nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("blank/invalid system log entries = %#v, want non-nil empty slice", got)
	}
}

func TestSystemLogLinesToEntriesSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	lines := make([]string, viewutil.SmallRowsCapacityThreshold+20)
	for i := range lines {
		lines[i] = "{bad"
	}
	lines[len(lines)-1] = `{"timestamp":"2026-07-08T12:00:00Z","level":"info"}`

	got := systemLogLinesToEntries(lines, nil)
	if len(got) != 1 {
		t.Fatalf("system log entries len = %d, want 1", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("invalid-heavy system log entries cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Timestamp != "2026-07-08T12:00:00Z" || got[0].Level != "info" {
		t.Fatalf("system log entry mismatch: %#v", got[0])
	}
}

func TestSystemLogLinesToEntriesBoundsGrowthAfterActualThreshold(t *testing.T) {
	lines := make([]string, viewutil.SmallRowsCapacityThreshold+20)
	for i := 0; i < viewutil.SmallRowsCapacityThreshold+2; i++ {
		lines[i] = `{"timestamp":"2026-07-08T12:00:00Z","level":"info"}`
	}

	got := systemLogLinesToEntries(lines, nil)
	if len(got) != viewutil.SmallRowsCapacityThreshold+2 {
		t.Fatalf("system log entries len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+2)
	}
	if cap(got) > len(got)*2 {
		t.Fatalf("system log entries cap = %d, want at most %d", cap(got), len(got)*2)
	}
}

func TestSystemLogLineToEntryMasksContentAndExtraFields(t *testing.T) {
	line := `{"timestamp":"2026-07-08T12:00:00Z","level":"error","caller":"auth/login.go:10","content":{"token":"abcdef","email":"user@example.com","ok":true},"duration":"3ms","trace":"trace-id","span":"span-id","truncated":true,"request":"GET /v1/subscribe/config?token=abcdef1234567890","ip":"203.0.113.42"}`

	entry, ok := systemLogLineToEntry(line)
	if !ok {
		t.Fatal("system log line should parse")
	}
	if entry.Timestamp != "2026-07-08T12:00:00Z" || entry.Level != "error" || entry.Caller != "auth/login.go:10" {
		t.Fatalf("basic fields not parsed: %#v", entry)
	}
	if entry.Duration != "3ms" || entry.Trace != "trace-id" || entry.Span != "span-id" || !entry.Truncated {
		t.Fatalf("metadata fields not parsed: %#v", entry)
	}

	var content map[string]any
	if err := json.Unmarshal(entry.Content, &content); err != nil {
		t.Fatalf("content should be valid json: %v", err)
	}
	if content["token"] != "***" || content["email"] != "***" || content["ok"] != true {
		t.Fatalf("content fields not masked: %#v", content)
	}

	var fields map[string]any
	if err := json.Unmarshal(entry.Fields, &fields); err != nil {
		t.Fatalf("extra fields should be valid json: %v", err)
	}
	if fields["ip"] != "203.0.113.*" {
		t.Fatalf("ip not masked: %#v", fields)
	}
	if fields["request"] != "GET /v1/subscribe/config?token=%2A%2A%2A" {
		t.Fatalf("request not sanitized: %#v", fields)
	}
}

func TestSystemLogLineToEntryParsesFixedScalarFieldsWithoutObjectDecode(t *testing.T) {
	entry, ok := systemLogLineToEntry(`{"timestamp":"2026-07-08T12:00:00Z","level":"info\nwarn","caller":12,"duration":null,"trace":"trace-id","span":"span-id","truncated":false}`)
	if !ok {
		t.Fatal("system log line should parse")
	}
	if entry.Level != "info\nwarn" || entry.Trace != "trace-id" || entry.Span != "span-id" {
		t.Fatalf("escaped string fields not parsed: %#v", entry)
	}
	if entry.Caller != "" || entry.Duration != "" || entry.Truncated {
		t.Fatalf("non-string or false fixed fields should keep zero values: %#v", entry)
	}
}

func TestSystemLogLineToEntryRejectsNonObjectLines(t *testing.T) {
	for _, line := range []string{"not json", "[]", `"message"`} {
		if entry, ok := systemLogLineToEntry(line); ok {
			t.Fatalf("line %q parsed as %#v, want rejected", line, entry)
		}
	}
}

func TestSystemLogLineToEntryPreservesNullExtraFieldShape(t *testing.T) {
	entry, ok := systemLogLineToEntry(`{"timestamp":"2026-07-08T12:00:00Z","optional":null}`)
	if !ok {
		t.Fatal("system log line should parse")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(entry.Fields, &fields); err != nil {
		t.Fatalf("extra fields should be valid json: %v", err)
	}
	if got := string(fields["optional"]); got != "null" {
		t.Fatalf("optional field = %s, want null", got)
	}
}
