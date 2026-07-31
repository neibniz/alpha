package payment

import (
	"context"
	"encoding/json"
	"testing"

	paymentPlatform "github.com/neibniz/alpha/server/pkg/payment"
)

func TestParsePaymentPlatformConfigNormalizesRawJSON(t *testing.T) {
	got, err := parsePaymentPlatformConfig(context.Background(), paymentPlatform.Stripe, json.RawMessage(`{"public_key":"pk","secret_key":"sk","payment":"card"}`))
	if err != nil {
		t.Fatalf("parsePaymentPlatformConfig returned error: %v", err)
	}

	var payload struct {
		PublicKey string `json:"public_key"`
		SecretKey string `json:"secret_key"`
		Payment   string `json:"payment"`
	}
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("normalized config should be json: %v", err)
	}
	if payload.PublicKey != "pk" || payload.SecretKey != "sk" || payload.Payment != "card" {
		t.Fatalf("normalized config mismatch: %+v", payload)
	}
}

func TestParsePaymentPlatformConfigRejectsInvalidJSON(t *testing.T) {
	if _, err := parsePaymentPlatformConfig(context.Background(), paymentPlatform.Stripe, json.RawMessage(`{invalid`)); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestPaymentConfigRawNormalizesBlankAndNull(t *testing.T) {
	for _, raw := range []json.RawMessage{
		nil,
		json.RawMessage("  "),
		json.RawMessage("null"),
		json.RawMessage(" \n null\t"),
	} {
		if got := string(paymentConfigRaw(raw)); got != "{}" {
			t.Fatalf("paymentConfigRaw(%q) = %q, want {}", raw, got)
		}
	}
}

func TestPaymentConfigRawValidInputAvoidsAllocation(t *testing.T) {
	raw := json.RawMessage(` {"public_key":"pk"} `)
	allocs := testing.AllocsPerRun(1000, func() {
		got := paymentConfigRaw(raw)
		if string(got) != `{"public_key":"pk"}` {
			t.Fatalf("paymentConfigRaw() = %q", got)
		}
	})
	if allocs != 0 {
		t.Fatalf("paymentConfigRaw allocations = %f, want 0", allocs)
	}
}

func TestParsePaymentPlatformConfigHandlesBalanceAsEmptyObject(t *testing.T) {
	got, err := parsePaymentPlatformConfig(context.Background(), paymentPlatform.Balance, nil)
	if err != nil {
		t.Fatalf("balance config should not require fields: %v", err)
	}
	if got != "{}" {
		t.Fatalf("balance config = %q, want {}", got)
	}
}
