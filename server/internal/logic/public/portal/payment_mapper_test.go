package portal

import (
	"testing"

	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
)

func TestPaymentMethodFromModelMapsPortalPublicShape(t *testing.T) {
	item := &modelPayment.Payment{
		Id:          10,
		Name:        "Alipay",
		Platform:    "alipay",
		Description: "qr",
		Icon:        "alipay",
		FeeMode:     2,
		FeePercent:  0,
		FeeAmount:   6,
		Domain:      "https://private.example.com",
		Config:      `{"secret":"value"}`,
		Token:       "secret-token",
	}

	got := paymentMethodFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.Platform != item.Platform || got.Description != item.Description || got.Icon != item.Icon {
		t.Fatalf("portal payment fields not mapped: %#v", got)
	}
	if got.FeeMode != item.FeeMode || got.FeePercent != item.FeePercent || got.FeeAmount != item.FeeAmount {
		t.Fatalf("portal fee fields not mapped: %#v", got)
	}
}
