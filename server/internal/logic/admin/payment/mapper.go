package payment

import (
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/paymentview"
	"github.com/neibniz/alpha/server/internal/types"
)

func paymentConfigFromModel(item *modelPayment.Payment) types.PaymentConfig {
	return paymentview.ConfigFromModel(item)
}

func paymentMethodListItemFromModel(item *modelPayment.Payment, notifyURL string) types.PaymentMethodListItem {
	return paymentview.ListItemFromModel(item, notifyURL)
}
