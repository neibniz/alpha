package payment

import (
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/paymentview"
	"github.com/neibniz/alpha/server/internal/types"
)

func paymentMethodFromModel(item *modelPayment.Payment) types.PaymentMethod {
	return paymentview.MethodFromModel(item)
}
