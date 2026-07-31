package order

import (
	"github.com/neibniz/alpha/server/internal/orderpricing"
	"github.com/neibniz/alpha/server/internal/types"
)

func getDiscount(discounts []types.SubscribeDiscount, inputMonths int64) float64 {
	return orderpricing.DiscountRate(discounts, inputMonths)
}
