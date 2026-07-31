package notify

import (
	"testing"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/svc"
)

func TestParseYuanToCentsRejectsUnsafeAmounts(t *testing.T) {
	if got, err := parseYuanToCents("10.125"); err != nil || got != 1013 {
		t.Fatalf("parseYuanToCents() = %d, %v", got, err)
	}
	for _, value := range []string{"-1", "NaN", "+Inf", "1e100"} {
		if _, err := parseYuanToCents(value); err == nil {
			t.Fatalf("parseYuanToCents(%q) should fail", value)
		}
	}
}

func TestValidateNotifyAmountUsesSettlementSnapshot(t *testing.T) {
	service := &svc.ServiceContext{Config: config.Config{Currency: config.Currency{Unit: "USD"}}}
	legacyForeignOrder := &order.Order{Amount: 100}
	if validateNotifyAmount(service, legacyForeignOrder, 100) {
		t.Fatal("foreign-currency order without a settlement snapshot must fail closed")
	}

	settled := &order.Order{Amount: 100, SettlementAmount: 725, SettlementCurrency: "CNY"}
	service.SetCurrencyRuntime(config.Currency{Unit: "USD"}, 8.5)
	if !validateNotifyAmount(service, settled, 725) {
		t.Fatal("snapshotted settlement amount should be accepted after the runtime rate changes")
	}
	for _, paid := range []int64{0, 724, 726} {
		if validateNotifyAmount(service, settled, paid) {
			t.Fatalf("paid amount %d must not match exact settlement amount", paid)
		}
	}
	if validateNotifyAmount(service, &order.Order{Amount: 100, SettlementAmount: 725, SettlementCurrency: "USD"}, 725) {
		t.Fatal("non-CNY settlement snapshot must be rejected")
	}

	service.SetCurrencyRuntime(config.Currency{Unit: "CNY"}, 0)
	if !validateNotifyAmount(service, &order.Order{Amount: 100}, 100) {
		t.Fatal("legacy CNY order should remain verifiable without a settlement snapshot")
	}
}
