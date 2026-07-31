package payment

import (
	"strings"

	"github.com/neibniz/alpha/server/internal/types"
)

type Platform int

const (
	Stripe Platform = iota
	AlipayF2F
	EPay
	Balance
	CryptoSaaS
	UNSUPPORTED Platform = -1
)

var platformNames = map[string]Platform{
	"CryptoSaaS":  CryptoSaaS,
	"Stripe":      Stripe,
	"AlipayF2F":   AlipayF2F,
	"EPay":        EPay,
	"balance":     Balance,
	"unsupported": UNSUPPORTED,
}

var platformAliases = map[string]Platform{
	"stripe":      Stripe,
	"alipayf2f":   AlipayF2F,
	"alipay_f2f":  AlipayF2F,
	"epay":        EPay,
	"balance":     Balance,
	"cryptosaas":  CryptoSaaS,
	"crypto_saas": CryptoSaaS,
	"unsupported": UNSUPPORTED,
}

func (p Platform) String() string {
	switch p {
	case Stripe:
		return "Stripe"
	case AlipayF2F:
		return "AlipayF2F"
	case EPay:
		return "EPay"
	case Balance:
		return "balance"
	case CryptoSaaS:
		return "CryptoSaaS"
	case UNSUPPORTED:
		return "unsupported"
	default:
		return "unsupported"
	}
}

func ParsePlatform(s string) Platform {
	value := strings.TrimSpace(s)
	if p, ok := platformNames[value]; ok {
		return p
	}
	if p, ok := platformAliases[strings.ToLower(value)]; ok {
		return p
	}
	return UNSUPPORTED
}

// NormalizePlatform keeps persisted and API platform names stable while
// accepting legacy lowercase aliases already present in older installations.
func NormalizePlatform(s string) (string, bool) {
	platform := ParsePlatform(s)
	if platform == UNSUPPORTED {
		return "", false
	}
	return platform.String(), true
}

// CompatiblePlatformNames returns canonical and historical values that may be
// present in persisted rows. It is intended for indexed equality filters.
func CompatiblePlatformNames(s string) []string {
	platform := ParsePlatform(s)
	switch platform {
	case Stripe:
		return []string{"Stripe", "stripe"}
	case AlipayF2F:
		return []string{"AlipayF2F", "alipayf2f", "alipay_f2f"}
	case EPay:
		return []string{"EPay", "epay"}
	case Balance:
		return []string{"balance", "Balance"}
	case CryptoSaaS:
		return []string{"CryptoSaaS", "cryptosaas", "crypto_saas"}
	default:
		return []string{strings.TrimSpace(s)}
	}
}

func GetSupportedPlatforms() []types.PlatformInfo {
	return []types.PlatformInfo{
		{
			Platform:    Stripe.String(),
			PlatformUrl: "https://stripe.com",
			PlatformFieldDescription: map[string]string{
				"public_key":     "Publishable key",
				"secret_key":     "Secret key",
				"webhook_secret": "Webhook secret",
				"payment":        "Payment Method, only supported card/alipay/wechat_pay",
			},
		},
		{
			Platform:    AlipayF2F.String(),
			PlatformUrl: "https://alipay.com",
			PlatformFieldDescription: map[string]string{
				"app_id":       "App ID",
				"private_key":  "Private Key",
				"public_key":   "Public Key",
				"invoice_name": "Invoice Name",
				"sandbox":      "Sandbox Mode",
			},
		},
		{
			Platform:    EPay.String(),
			PlatformUrl: "",
			PlatformFieldDescription: map[string]string{
				"pid":  "PID",
				"url":  "URL",
				"key":  "Key",
				"type": "Type",
			},
		},
		{
			Platform:    CryptoSaaS.String(),
			PlatformUrl: "https://t.me/CryptoSaaSBot",
			PlatformFieldDescription: map[string]string{
				"endpoint":   "API Endpoint",
				"account_id": "Account ID",
				"secret_key": "Secret Key",
				"type":       "Payment Type",
			},
		},
	}
}
