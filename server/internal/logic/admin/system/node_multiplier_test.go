package system

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/neibniz/alpha/server/internal/types"
)

func TestNodeMultiplierPeriodsFromRawKeepsEmptyArrayShape(t *testing.T) {
	for _, raw := range []string{"", "null", "[]"} {
		got, err := nodeMultiplierPeriodsFromRaw(raw)
		if err != nil {
			t.Fatalf("nodeMultiplierPeriodsFromRaw(%q) error = %v", raw, err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("nodeMultiplierPeriodsFromRaw(%q) = %#v, want non-nil empty slice", raw, got)
		}
		payload, err := json.Marshal(struct {
			Periods []types.TimePeriod `json:"periods"`
		}{Periods: got})
		if err != nil {
			t.Fatalf("marshal periods: %v", err)
		}
		if string(payload) != `{"periods":[]}` {
			t.Fatalf("periods payload = %s, want []", payload)
		}
	}
}

func TestNodeMultiplierPeriodsFromRawParsesConfiguredPeriods(t *testing.T) {
	got, err := nodeMultiplierPeriodsFromRaw(`[{"start_time":"00:00","end_time":"12:00","multiplier":1.5}]`)
	if err != nil {
		t.Fatalf("nodeMultiplierPeriodsFromRaw configured error = %v", err)
	}
	want := []types.TimePeriod{{StartTime: "00:00", EndTime: "12:00", Multiplier: 1.5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nodeMultiplierPeriodsFromRaw configured = %#v, want %#v", got, want)
	}
}

func TestNodeMultiplierPeriodsFromRawRejectsInvalidJSON(t *testing.T) {
	if _, err := nodeMultiplierPeriodsFromRaw(`{"start_time":"00:00"}`); err == nil {
		t.Fatal("object periods json should be rejected")
	}
	if _, err := nodeMultiplierPeriodsFromRaw(`{invalid`); err == nil {
		t.Fatal("invalid periods json should be rejected")
	}
}

func TestNormalizeNodeMultiplierPeriodsKeepsEmptyArrayShape(t *testing.T) {
	got := normalizeNodeMultiplierPeriods(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("normalizeNodeMultiplierPeriods(nil) = %#v, want non-nil empty slice", got)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal normalized periods: %v", err)
	}
	if string(payload) != `[]` {
		t.Fatalf("normalized nil periods marshal = %s, want []", payload)
	}
}
