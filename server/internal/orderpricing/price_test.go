package orderpricing

import (
	"errors"
	"testing"

	"github.com/neibniz/alpha/server/internal/types"
)

func TestProductQuoteUsesBestEligibleDiscount(t *testing.T) {
	quote, err := ProductQuote(1_000, 6, []types.SubscribeDiscount{
		{Quantity: 3, Discount: 90},
		{Quantity: 6, Discount: 80},
		{Quantity: 12, Discount: 70},
	})
	if err != nil {
		t.Fatal(err)
	}
	if quote.ListPrice != 6_000 || quote.Amount != 4_800 || quote.Discount != 1_200 {
		t.Fatalf("unexpected quote: %#v", quote)
	}
}

func TestProductQuoteRejectsUnsafeAmount(t *testing.T) {
	if _, err := ProductQuote(MaxOrderAmount, 2, nil); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("error = %v, want ErrAmountTooLarge", err)
	}
}

func TestProductQuoteRejectsListPriceThatCannotBeStored(t *testing.T) {
	if _, err := ProductQuote(MaxOrderAmount, 2, []types.SubscribeDiscount{{Quantity: 1, Discount: 1}}); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("error = %v, want ErrAmountTooLarge", err)
	}
}
