package user

import (
	"testing"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
)

func TestRefundableWalletFundingIncludesPaidRenewals(t *testing.T) {
	details := &modelOrder.Details{
		CommissionDeduction: 100,
		GiftAmount:          200,
		SubOrders: []*modelOrder.Order{
			{Status: 2, CommissionDeduction: 30, GiftAmount: 40},
			{Status: 5, CommissionDeduction: 50, GiftAmount: 60},
			{Status: 1, CommissionDeduction: 700, GiftAmount: 800},
			{Status: 3, CommissionDeduction: 900, GiftAmount: 1000},
		},
	}

	commission, gift := refundableWalletFunding(details)
	if commission != 180 || gift != 300 {
		t.Fatalf("funding = commission:%d gift:%d, want commission:180 gift:300", commission, gift)
	}
}

func TestRefundableWalletFundingIgnoresNegativeAndNilRows(t *testing.T) {
	details := &modelOrder.Details{
		CommissionDeduction: -10,
		GiftAmount:          -20,
		SubOrders: []*modelOrder.Order{
			nil,
			{Status: 2, CommissionDeduction: -30, GiftAmount: -40},
		},
	}

	commission, gift := refundableWalletFunding(details)
	if commission != 0 || gift != 0 {
		t.Fatalf("funding = commission:%d gift:%d, want zero", commission, gift)
	}
}
