package payment

import (
	"testing"

	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
)

func TestHasOtherBalancePayment(t *testing.T) {
	enabled := true
	disabled := false
	rows := []*modelPayment.Payment{
		{Id: -1, Platform: "balance", Enable: &enabled},
		{Id: 2, Platform: "Balance", Enable: &disabled},
		{Id: 3, Platform: "Stripe", Enable: &enabled},
	}

	if !hasOtherBalancePayment(rows, 2, false) {
		t.Fatal("expected the default balance payment to be detected")
	}
	if hasOtherBalancePayment(rows, -1, true) {
		t.Fatal("disabled legacy balance payment must not block enable checks")
	}
	if !hasOtherBalancePayment(rows, 2, true) {
		t.Fatal("enabled default balance payment must block a second enabled method")
	}
}
