package orderpricing

import (
	"errors"
	"math"

	"github.com/neibniz/alpha/server/internal/types"
)

const (
	MaxOrderAmount int64 = 2_147_483_647
	MaxQuantity    int64 = 1_000
)

var (
	ErrInvalidPrice    = errors.New("invalid order price")
	ErrInvalidQuantity = errors.New("invalid order quantity")
	ErrAmountTooLarge  = errors.New("order amount exceeds maximum")
)

type Quote struct {
	ListPrice int64
	Amount    int64
	Discount  int64
}

func DiscountRate(discounts []types.SubscribeDiscount, quantity int64) float64 {
	rate := float64(100)
	for _, discount := range discounts {
		if quantity >= discount.Quantity && discount.Discount >= 0 && discount.Discount < rate {
			rate = discount.Discount
		}
	}
	return rate / 100
}

func ProductQuote(unitPrice, quantity int64, discounts []types.SubscribeDiscount) (Quote, error) {
	if unitPrice < 0 {
		return Quote{}, ErrInvalidPrice
	}
	if quantity < 1 || quantity > MaxQuantity {
		return Quote{}, ErrInvalidQuantity
	}
	if unitPrice > math.MaxInt64/quantity {
		return Quote{}, ErrAmountTooLarge
	}
	listPrice := unitPrice * quantity
	if listPrice > MaxOrderAmount {
		return Quote{}, ErrAmountTooLarge
	}
	amount := int64(float64(listPrice) * DiscountRate(discounts, quantity))
	if amount < 0 || amount > MaxOrderAmount {
		return Quote{}, ErrAmountTooLarge
	}
	return Quote{
		ListPrice: listPrice,
		Amount:    amount,
		Discount:  listPrice - amount,
	}, nil
}
