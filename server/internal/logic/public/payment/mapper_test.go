package payment

import (
	"testing"

	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
)

func TestPaymentMethodFromModelMapsPublicShape(t *testing.T) {
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

	got := paymentMethodFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.Platform != item.Platform || got.Description != item.Description || got.Icon != item.Icon {
		t.Fatalf("public payment fields not mapped: %#v", got)
	}
	if got.FeeMode != item.FeeMode || got.FeePercent != item.FeePercent || got.FeeAmount != item.FeeAmount {
		t.Fatalf("fee fields not mapped: %#v", got)
	}
}
