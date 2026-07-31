package productpolicy

import (
	"errors"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
)

func boolPtr(value bool) *bool { return &value }

func TestNewPurchaseAvailability(t *testing.T) {
	now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	startsLater := now.Add(time.Hour)
	ended := now.Add(-time.Minute)
	tests := []struct {
		name    string
		product *subscribe.Subscribe
		want    string
	}{
		{name: "disabled", product: &subscribe.Subscribe{Sell: boolPtr(false)}, want: AvailabilityNotForSale},
		{name: "scheduled", product: &subscribe.Subscribe{Sell: boolPtr(true), Inventory: -1, SaleStartsAt: &startsLater}, want: AvailabilityComingSoon},
		{name: "ended", product: &subscribe.Subscribe{Sell: boolPtr(true), Inventory: -1, SaleEndsAt: &ended}, want: AvailabilityEnded},
		{name: "stock", product: &subscribe.Subscribe{Sell: boolPtr(true), Inventory: 0}, want: AvailabilityOutOfStock},
		{name: "available", product: &subscribe.Subscribe{Sell: boolPtr(true), Inventory: -1}, want: AvailabilityAvailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NewPurchaseAvailability(test.product, now); got != test.want {
				t.Fatalf("availability = %q, want %q", got, test.want)
			}
		})
	}
}

func TestValidateQuantityUsesProductAndSystemLimits(t *testing.T) {
	product := &subscribe.Subscribe{MinQuantity: 3, MaxQuantity: 12}
	if err := ValidateQuantity(product, 2, 1000); !errors.Is(err, ErrQuantityBelowMinimum) {
		t.Fatalf("minimum error = %v", err)
	}
	if err := ValidateQuantity(product, 13, 1000); !errors.Is(err, ErrQuantityAboveMaximum) {
		t.Fatalf("maximum error = %v", err)
	}
	if err := ValidateQuantity(&subscribe.Subscribe{MinQuantity: 1}, 1001, 1000); !errors.Is(err, ErrQuantityAboveMaximum) {
		t.Fatalf("system maximum error = %v", err)
	}
	if err := ValidateQuantity(product, 6, 1000); err != nil {
		t.Fatalf("valid quantity error = %v", err)
	}
}
