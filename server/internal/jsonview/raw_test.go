package jsonview

import (
	"encoding/json"
	"testing"
)

func TestRawValueOrNull(t *testing.T) {
	if got := RawValueOrNull(` {"ok":true} `); string(got) != `{"ok":true}` {
		t.Fatalf("raw object mismatch: %s", got)
	}
	if got := RawValueOrNull(`["ok"]`); string(got) != `["ok"]` {
		t.Fatalf("raw array mismatch: %s", got)
	}
	if got := RawValueOrNull(`invalid`); string(got) != "null" {
		t.Fatalf("invalid json should be null, got %s", got)
	}
}

func TestRawValueReportsValidity(t *testing.T) {
	raw, ok := RawValue(` null `)
	if !ok || string(raw) != "null" {
		t.Fatalf("RawValue(null) = %s/%v, want null/true", raw, ok)
	}
	raw, ok = RawValue(`{"ok":true}`)
	if !ok || string(raw) != `{"ok":true}` {
		t.Fatalf("RawValue(object) = %s/%v, want object/true", raw, ok)
	}
	for _, input := range []string{"", "  ", "{bad"} {
		if raw, ok := RawValue(input); ok || raw != nil {
			t.Fatalf("RawValue(%q) = %s/%v, want nil/false", input, raw, ok)
		}
	}
}

func TestRawObject(t *testing.T) {
	raw, ok := RawObject(` {"ok":true} `)
	if !ok || string(raw) != `{"ok":true}` {
		t.Fatalf("object should be accepted, raw=%s ok=%v", raw, ok)
	}
	for _, input := range []string{``, `invalid`, `["ok"]`, `"ok"`} {
		if raw, ok := RawObject(input); ok || raw != nil {
			t.Fatalf("%q should not be accepted as object, raw=%s ok=%v", input, raw, ok)
		}
	}
}

func TestRawObjectOrEmptyMarshalsAsObject(t *testing.T) {
	payload, err := json.Marshal(struct {
		Config json.RawMessage `json:"config"`
	}{Config: RawObjectOrEmpty(`invalid`)})
	if err != nil {
		t.Fatalf("marshal empty object raw: %v", err)
	}
	if string(payload) != `{"config":{}}` {
		t.Fatalf("unexpected payload: %s", payload)
	}
}
