package order

import (
	"context"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRestoreSubscribeInventoryIncrementsFiniteInventoryOnly(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&subscribe.Subscribe{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*subscribe.Subscribe{
		{Id: 1, Name: "finite", Inventory: 3},
		{Id: 2, Name: "unlimited", Inventory: -1},
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := restoreSubscribeInventory(ctx, db, 1); err != nil {
		t.Fatal(err)
	}
	if err := restoreSubscribeInventory(ctx, db, 2); err != nil {
		t.Fatal(err)
	}

	var finite subscribe.Subscribe
	if err := db.First(&finite, 1).Error; err != nil {
		t.Fatal(err)
	}
	if finite.Inventory != 4 {
		t.Fatalf("finite inventory = %d, want 4", finite.Inventory)
	}
	var unlimited subscribe.Subscribe
	if err := db.First(&unlimited, 2).Error; err != nil {
		t.Fatal(err)
	}
	if unlimited.Inventory != -1 {
		t.Fatalf("unlimited inventory = %d, want -1", unlimited.Inventory)
	}
}

func TestResolveOrderPaymentAllowsZeroAmountWithoutPaymentMethod(t *testing.T) {
	payment, err := resolveOrderPayment(context.Background(), nil, 0, 0)
	if err != nil {
		t.Fatalf("zero amount should not require payment: %v", err)
	}
	if payment != nil {
		t.Fatalf("zero amount payment = %#v, want nil", payment)
	}
}

func TestResolveOrderPaymentRejectsMissingMethodForPaidOrder(t *testing.T) {
	if _, err := resolveOrderPayment(context.Background(), nil, 0, 100); err == nil {
		t.Fatal("paid order should require a payment method")
	}
}
