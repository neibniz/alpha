package user

import (
	"reflect"
	"testing"
)

func TestNormalizeUserRulesTrimsAndDeduplicates(t *testing.T) {
	got := normalizeUserRules([]string{" rule-a ", "", "rule-b", "rule-a", " rule-c "})
	want := []string{"rule-a", "rule-b", "rule-c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeUserRules() = %#v, want %#v", got, want)
	}
}

func TestNormalizeUserRulesDropsEmptyRules(t *testing.T) {
	got := normalizeUserRules([]string{"", "  "})
	if got == nil || len(got) != 0 {
		t.Fatalf("normalizeUserRules() = %#v, want non-nil empty", got)
	}
}

func TestMarshalUserRulesUsesCompactJSONArray(t *testing.T) {
	got, err := marshalUserRules([]string{"rule-a", "rule-b"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `["rule-a","rule-b"]`; got != want {
		t.Fatalf("marshalUserRules() = %q, want %q", got, want)
	}
}

func TestMarshalUserRulesUsesEmptyJSONArray(t *testing.T) {
	got, err := marshalUserRules(normalizeUserRules([]string{"", "  "}))
	if err != nil {
		t.Fatal(err)
	}
	if want := `[]`; got != want {
		t.Fatalf("marshalUserRules(empty) = %q, want %q", got, want)
	}
}
