package order

import "testing"

func TestNormalizeOrderRequestID(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	if got, err := normalizeOrderRequestID(valid); err != nil || got != valid {
		t.Fatalf("normalize valid request id = %q/%v", got, err)
	}
	if got, err := normalizeOrderRequestID(""); err != nil || got != "" {
		t.Fatalf("empty request id = %q/%v", got, err)
	}
	for _, invalid := range []string{"short", "contains whitespace 123", "../../unsafe-request-id"} {
		if _, err := normalizeOrderRequestID(invalid); err == nil {
			t.Fatalf("request id %q should be rejected", invalid)
		}
	}
}
