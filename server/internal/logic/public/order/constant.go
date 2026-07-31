package order

import "github.com/neibniz/alpha/server/internal/orderpricing"

const (
	Epay            = "epay"
	AlipayF2f       = "alipay_f2f"
	StripeAlipay    = "stripe_alipay"
	StripeWeChatPay = "stripe_wechat_pay"
	Balance         = "balance"

	// MaxOrderAmount Order amount limits
	MaxOrderAmount    = orderpricing.MaxOrderAmount
	MaxRechargeAmount = 2000000000 // 2 billion, slightly lower for safety
	MaxQuantity       = orderpricing.MaxQuantity
)
