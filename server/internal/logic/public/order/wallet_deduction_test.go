package order

import (
	"testing"

	"github.com/neibniz/alpha/server/internal/model/user"
)

func TestQuoteOrderWalletDeductionUsesCommissionThenGift(t *testing.T) {
	tests := []struct {
		name       string
		amount     int64
		user       *user.User
		commission int64
		gift       int64
		provider   int64
	}{
		{name: "commission covers order", amount: 500, user: &user.User{Commission: 700, GiftAmount: 900}, commission: 500},
		{name: "gift covers remainder", amount: 1000, user: &user.User{Commission: 300, GiftAmount: 900}, commission: 300, gift: 700},
		{name: "provider covers remainder", amount: 1000, user: &user.User{Commission: 200, GiftAmount: 300}, commission: 200, gift: 300, provider: 500},
		{name: "negative balances ignored", amount: 1000, user: &user.User{Commission: -1, GiftAmount: -2}, provider: 1000},
		{name: "negative amount clamped", amount: -1, user: &user.User{Commission: 100, GiftAmount: 100}},
		{name: "nil user", amount: 1000, provider: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteOrderWalletDeduction(tt.amount, tt.user)
			if got.Commission != tt.commission || got.Gift != tt.gift || got.Provider != tt.provider {
				t.Fatalf("deduction = %#v, want commission=%d gift=%d provider=%d", got, tt.commission, tt.gift, tt.provider)
			}
		})
	}
}
