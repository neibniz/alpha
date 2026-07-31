package paymentview

import (
	"encoding/json"
	"testing"

	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestMethodFromModelMapsPublicShape(t *testing.T) {
	item := &modelPayment.Payment{
		Id:          9,
		Name:        "Stripe",
		Platform:    "stripe",
		Description: "card",
		Icon:        "icon",
		FeeMode:     1,
		FeePercent:  5,
		FeeAmount:   10,
		Domain:      "https://private.example.com",
		Config:      `{"secret":"value"}`,
		Token:       "secret-token",
	}

	got := MethodFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.Platform != item.Platform || got.Description != item.Description || got.Icon != item.Icon {
		t.Fatalf("public payment fields not mapped: %#v", got)
	}
	if got.FeeMode != item.FeeMode || got.FeePercent != item.FeePercent || got.FeeAmount != item.FeeAmount {
		t.Fatalf("fee fields not mapped: %#v", got)
	}
}

func TestMethodRowsSkipsNilRowsWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelPayment.Payment, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelPayment.Payment{Id: 1, Name: "Stripe", Platform: "stripe"}
	rows[len(rows)-1] = &modelPayment.Payment{Id: 2, Name: "Balance", Platform: "balance"}

	got := MethodRows(rows)
	if len(got) != 2 {
		t.Fatalf("method rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy method rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("method rows order/ids not preserved: %#v", got)
	}
}

func TestMethodRowsUsesCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelPayment.Payment, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelPayment.Payment{Id: int64(index + 1), Name: "method"}
	}
	got := MethodRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("method rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("method rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("method rows order/ids not preserved: %#v", got)
	}
}

func TestMethodRowsReturnsEmptySlice(t *testing.T) {
	got := MethodRows(nil)
	if got == nil {
		t.Fatal("empty method rows must return an empty slice, not nil")
	}
	if len(got) != 0 {
		t.Fatalf("empty method rows len = %d, want 0", len(got))
	}
	payload, err := json.Marshal(struct {
		List []types.PaymentMethod `json:"list"`
	}{List: got})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("test sanity marshal = %s", payload)
	}
}

func TestConfigFromModelMapsAdminConfig(t *testing.T) {
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

	got := ConfigFromModel(item)
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

func TestConfigFromModelInvalidConfigFallsBackToEmptyObject(t *testing.T) {
	got := ConfigFromModel(&modelPayment.Payment{Config: `invalid`})
	if got.Enable {
		t.Fatalf("nil enable should map to false")
	}
	payload, err := json.Marshal(struct {
		Config json.RawMessage `json:"config"`
	}{Config: got.Config})
	if err != nil {
		t.Fatalf("marshal config response: %v", err)
	}
	if string(payload) != `{"config":{}}` {
		t.Fatalf("invalid config should fall back to empty object, got %s", payload)
	}
}

func TestListItemFromModelMapsAdminListShape(t *testing.T) {
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

	got := ListItemFromModel(item, "https://notify.example.com")
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

func TestListItemFromModelDefaultsNilEnable(t *testing.T) {
	got := ListItemFromModel(&modelPayment.Payment{}, "")
	if got.Enable {
		t.Fatalf("nil enable should map to false")
	}
}
