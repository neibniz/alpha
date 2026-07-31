package payment

import (
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	paymentPlatform "github.com/neibniz/alpha/server/pkg/payment"
)

func hasOtherBalancePayment(rows []*modelPayment.Payment, excludeID int64, enabledOnly bool) bool {
	for _, row := range rows {
		if row == nil || row.Id == excludeID || paymentPlatform.ParsePlatform(row.Platform) != paymentPlatform.Balance {
			continue
		}
		if !enabledOnly || (row.Enable != nil && *row.Enable) {
			return true
		}
	}
	return false
}
