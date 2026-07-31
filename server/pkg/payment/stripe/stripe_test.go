package stripe

import (
	"strings"
	"testing"

	"github.com/stripe/stripe-go/v81"
)

func TestClientKeepsSecretKeyPerInstance(t *testing.T) {
	original := stripe.Key
	stripe.Key = "global-sentinel"
	t.Cleanup(func() { stripe.Key = original })

	first := NewClient(Config{SecretKey: "sk_first"})
	second := NewClient(Config{SecretKey: "sk_second"})
	if first.api.PaymentIntents.Key != "sk_first" || second.api.PaymentIntents.Key != "sk_second" {
		t.Fatal("Stripe API clients do not retain independent secret keys")
	}
	if stripe.Key != "global-sentinel" {
		t.Fatalf("NewClient mutated package-global Stripe key: %q", stripe.Key)
	}
}

func TestStripeIdempotencyKeyIsStableAndBounded(t *testing.T) {
	first := stripeIdempotencyKey("order", "alpha-order-123")
	if first != stripeIdempotencyKey("order", "alpha-order-123") {
		t.Fatal("idempotency key is not deterministic")
	}
	if first == stripeIdempotencyKey("order", "alpha-order-124") {
		t.Fatal("different orders share an idempotency key")
	}
	if strings.Contains(first, "alpha-order-123") || len(first) > 255 {
		t.Fatalf("unsafe idempotency key %q", first)
	}
}

func TestStripeSearchLiteralEscapesQuerySyntax(t *testing.T) {
	if got, want := stripeSearchLiteral(`a'b\c@example.test`), `a\'b\\c@example.test`; got != want {
		t.Fatalf("stripeSearchLiteral() = %q, want %q", got, want)
	}
}

func TestStripeAlipay(t *testing.T) {
	t.Skipf("Skip TestStripeAlipay test")
	client := NewClient(Config{
		WebhookSecret: "",
	})
	order := Order{
		OrderNo:   "JS20210719123456789",
		Subscribe: "测试",
		Amount:    100,
		Currency:  string(stripe.CurrencyGBP),
		Payment:   "alipay",
	}
	user := User{
		UserId: 1,
		Email:  "tension@alpha.dev",
	}
	result, err := client.CreatePaymentSheet(&order, &user)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("TradeNo: %s\n", result.ClientSecret)
}

func TestStripeWechat(t *testing.T) {
	t.Skipf("Skip TestStripeWechat test")
	client := NewClient(Config{
		SecretKey:     "SecretKey",
		PublicKey:     "PublicKey",
		WebhookSecret: "",
	})
	order := Order{
		OrderNo:   "JS20210719123456789",
		Subscribe: "测试",
		Amount:    100,
		Currency:  string(stripe.CurrencyGBP),
		Payment:   "wechat_pay",
	}
	user := User{
		UserId: 1,
		Email:  "tension@alpha.dev",
	}
	result, err := client.CreatePaymentSheet(&order, &user)
	if err != nil {
		t.Error(err.Error())
	}
	t.Logf("TradeNo: %s\n", result.ClientSecret)
}
