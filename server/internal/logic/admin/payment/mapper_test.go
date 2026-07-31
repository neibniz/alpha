package payment

import (
	"encoding/json"
	"testing"

	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
)

func TestPaymentConfigFromModelMapsAdminConfig(t *testing.T) {
	enabled := true
	item := &modelPayment.Payment{
		Id:          7,
		Name:        "Stripe",
		Platform:    "stripe",
		Description: "card",
		Icon:        "icon",
		Domain:      "https://pay.example.com",
		Config:      `{"public_key":"pk_test","sandbox":true}`,
		FeeMode:     1,
		FeePercent:  5,
		FeeAmount:   10,
		Enable:      &enabled,
		Token:       "secret-token",
	}

	got := paymentConfigFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.Platform != item.Platform || got.Description != item.Description || got.Domain != item.Domain {
		t.Fatalf("basic config fields not mapped: %#v", got)
	}
	if got.FeeMode != item.FeeMode || got.FeePercent != item.FeePercent || got.FeeAmount != item.FeeAmount {
		t.Fatalf("fee fields not mapped: %#v", got)
	}
	if got.Enable != enabled {
		t.Fatalf("enable not mapped: %#v", got.Enable)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(got.Config, &config); err != nil {
		t.Fatalf("config should marshal as json object: %v", err)
	}
	if config["public_key"] != "pk_test" || config["sandbox"] != true {
		t.Fatalf("config not parsed: %#v", config)
	}
}

func TestPaymentMethodListItemFromModelMapsListShape(t *testing.T) {
	enabled := true
	item := &modelPayment.Payment{
		Id:          8,
		Name:        "Wallet",
		Platform:    "balance",
		Description: "internal",
		Icon:        "wallet",
		Domain:      "https://pay.example.com",
		FeeMode:     2,
		FeePercent:  0,
		FeeAmount:   3,
		Enable:      &enabled,
		Config:      `{"private":"secret"}`,
	}

	got := paymentMethodListItemFromModel(item, "https://notify.example.com")
	if got.Id != item.Id || got.Platform != item.Platform || got.Description != item.Description || got.NotifyURL != "https://notify.example.com" {
		t.Fatalf("list detail fields not mapped: %#v", got)
	}
	if !got.Enable {
		t.Fatalf("enable = false, want true")
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"domain", "config"} {
		if _, ok := decoded[key]; ok {
			t.Fatalf("payment method list item must not expose %q: %s", key, payload)
		}
	}
}

func TestPaymentMethodListItemFromModelDefaultsNilEnable(t *testing.T) {
	got := paymentMethodListItemFromModel(&modelPayment.Payment{}, "")
	if got.Enable {
		t.Fatalf("nil enable should map to false")
	}
}
