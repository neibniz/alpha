package portal

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/config"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBalancePaymentUserColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "balance"}
	if !reflect.DeepEqual(balancePaymentUserColumns, want) {
		t.Fatalf("balance payment user columns = %#v, want %#v", balancePaymentUserColumns, want)
	}
}

func TestSubscriptionExpiryProductFromSnapshotKeepsOnlyPolicyFields(t *testing.T) {
	raw := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:        ordersnapshot.KindSubscribe,
		ID:          17,
		Name:        "Snapshot Plan",
		Description: "large presentation field",
		UnitTime:    "Month",
		Traffic:     4096,
		Nodes:       []int64{1, 2},
		NodeTags:    []string{"edge"},
	})

	got, ok := subscriptionExpiryProductFromSnapshot(raw)
	if !ok {
		t.Fatal("subscriptionExpiryProductFromSnapshot rejected subscribe snapshot")
	}
	if got.Id != 17 || got.UnitTime != "Month" {
		t.Fatalf("expiry product fields = %#v, want id/unit_time only", got)
	}
	if got.Name != "" || got.Description != "" || got.Traffic != 0 || got.Nodes != "" || got.NodeTags != "" {
		t.Fatalf("expiry product should not hydrate presentation/topology fields: %#v", got)
	}

	if got, ok := subscriptionExpiryProductFromSnapshot(ordersnapshot.Marshal(ordersnapshot.Snapshot{Kind: ordersnapshot.KindMembership, Name: "Member"})); ok || got != nil {
		t.Fatalf("non-subscribe snapshot = %#v, %v; want nil false", got, ok)
	}
	if got, ok := subscriptionExpiryProductFromSnapshot(`{"kind":"subscribe","unit_time":"Month"}`); ok || got != nil {
		t.Fatalf("snapshot without name = %#v, %v; want nil false", got, ok)
	}
}

func TestPaymentURLsIgnoreUntrustedRequestHostAndReturnOrigin(t *testing.T) {
	ctx := context.WithValue(context.Background(), constant.CtxKeyRequestHost, "attacker.example.test")
	logic := NewPurchaseCheckoutLogic(ctx, &svc.ServiceContext{Config: config.Config{
		Site: config.SiteConfig{Host: "https://user.example.test"},
		HTTP: config.HTTPConfig{AllowedOrigins: []string{
			"https://admin.example.test",
		}},
	}})

	notifyURL, err := logic.paymentNotifyURL(&modelPayment.Payment{Platform: "EPay", Token: "secure-token"})
	if err != nil {
		t.Fatal(err)
	}
	if notifyURL != "https://user.example.test/v1/notify/EPay/secure-token" {
		t.Fatalf("notify URL = %q", notifyURL)
	}

	returnURL, err := logic.paymentReturnURL("https://user.example.test/orders/1?paid=1")
	if err != nil {
		t.Fatal(err)
	}
	if returnURL != "https://user.example.test/orders/1?paid=1" {
		t.Fatalf("return URL = %q", returnURL)
	}
	if _, err := logic.paymentReturnURL("https://attacker.example.test/receipt"); err == nil {
		t.Fatal("untrusted payment return URL should be rejected")
	}
}

func TestQueryExchangeRateRejectsMissingOrUnsafeRate(t *testing.T) {
	service := &svc.ServiceContext{}
	logic := NewPurchaseCheckoutLogic(context.Background(), service)
	service.SetCurrencyRuntime(config.Currency{Unit: "USD"}, 7.25)
	if got, err := logic.queryExchangeRate("CNY", 100); err != nil || got != 7.25 {
		t.Fatalf("queryExchangeRate() = %v, %v", got, err)
	}
	service.SetCurrencyRuntime(config.Currency{Unit: "USD"}, math.Inf(1))
	if _, err := logic.queryExchangeRate("CNY", 100); err == nil {
		t.Fatal("infinite cached rate should be rejected")
	}
	service.SetCurrencyRuntime(config.Currency{Unit: "USD"}, 0)
	if _, err := logic.queryExchangeRate("CNY", 100); err == nil {
		t.Fatal("missing rate and API key should be rejected")
	}
}

func TestSettlementAmountCNYPersistsAndReusesSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelOrder.Order{}); err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rds.Close() })

	info := &modelOrder.Order{Id: 91, OrderNo: "alpha-order-91", Amount: 100, Status: 1}
	if err := db.Create(info).Error; err != nil {
		t.Fatal(err)
	}
	service := &svc.ServiceContext{OrderModel: modelOrder.NewModel(db, rds)}
	service.SetCurrencyRuntime(config.Currency{Unit: "USD"}, 7.25)
	logic := NewPurchaseCheckoutLogic(context.Background(), service)

	if got, err := logic.settlementAmountCNY(info); err != nil || got != 725 {
		t.Fatalf("first settlement = %d, %v", got, err)
	}
	service.SetCurrencyRuntime(config.Currency{Unit: "USD"}, 8.5)
	reloaded, err := service.OrderModel.FindOneForCheckout(context.Background(), info.OrderNo)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := logic.settlementAmountCNY(reloaded); err != nil || got != 725 {
		t.Fatalf("reused settlement = %d, %v", got, err)
	}
	stale := &modelOrder.Order{Id: info.Id, OrderNo: info.OrderNo, Amount: info.Amount, Status: 1}
	if _, err := logic.settlementAmountCNY(stale); err == nil {
		t.Fatal("a checkout using a conflicting current rate must not replace the settlement snapshot")
	}
}
