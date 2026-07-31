package order

import (
	"context"
	"testing"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCouponUsageReachedCountsOnlyApplicableOrders(t *testing.T) {
	db := newOrderLogicTestDB(t)
	rows := []modelOrder.Order{
		{Id: 1, UserId: 7, OrderNo: "pending", Coupon: "SAVE", Status: 1},
		{Id: 2, UserId: 7, OrderNo: "paid", Coupon: "SAVE", Status: 2},
		{Id: 3, UserId: 7, OrderNo: "closed", Coupon: "SAVE", Status: 3},
		{Id: 4, UserId: 7, OrderNo: "other-coupon", Coupon: "OTHER", Status: 5},
		{Id: 5, UserId: 8, OrderNo: "other-user", Coupon: "SAVE", Status: 5},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	reached, err := couponUsageReached(context.Background(), db, 7, "SAVE", 2)
	if err != nil || !reached {
		t.Fatalf("limit 2 reached/error = %v/%v, want true/nil", reached, err)
	}
	reached, err = couponUsageReached(context.Background(), db, 7, "SAVE", 3)
	if err != nil || reached {
		t.Fatalf("limit 3 reached/error = %v/%v, want false/nil", reached, err)
	}
}

func TestCouponUsageReachedSkipsUnlimitedCouponQuery(t *testing.T) {
	reached, err := couponUsageReached(context.Background(), nil, 7, "SAVE", 0)
	if err != nil || reached {
		t.Fatalf("unlimited coupon reached/error = %v/%v, want false/nil", reached, err)
	}
}

func newOrderLogicTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelOrder.Order{}); err != nil {
		t.Fatal(err)
	}
	return db
}
