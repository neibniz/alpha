package order

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMembershipPolicySelectColumnsStayPolicyOnly(t *testing.T) {
	for _, column := range []string{"name", "description", "unit_price", "duration_unit", "duration_value", "enabled"} {
		if slices.Contains(membershipPolicySelectColumns, column) {
			t.Fatalf("membership policy query should not load %q", column)
		}
	}
	for _, column := range []string{
		"id",
		"monthly_consumption_limit_enabled",
		"monthly_consumption_limit",
		"monthly_order_limit_enabled",
		"monthly_order_limit",
		"subscription_expiry_limit_enabled",
		"subscription_expiry_grace_days",
	} {
		if !slices.Contains(membershipPolicySelectColumns, column) {
			t.Fatalf("membership policy query is missing %q", column)
		}
	}
}

func TestEnforceMembershipSubscriptionExpiryPolicy(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&membership.Plan{}); err != nil {
		t.Fatalf("migrate membership plan: %v", err)
	}
	if err := db.Create(&membership.Plan{
		Name:                           "member",
		SubscriptionExpiryLimitEnabled: true,
		SubscriptionExpiryGraceDays:    2,
	}).Error; err != nil {
		t.Fatalf("create membership plan: %v", err)
	}

	svcCtx := &svc.ServiceContext{DB: db}
	now := time.Now()
	currentExpireAt := now.Add(time.Hour)
	member := &user.User{Id: 1, MemberExpiredAt: currentExpireAt.AddDate(0, 0, 5)}
	sub := &subscribe.Subscribe{UnitTime: "Day"}

	if err := EnforceMembershipSubscriptionExpiryPolicy(ctx, svcCtx, member, sub, 7, &currentExpireAt); err != nil {
		t.Fatalf("expected subscription ending within grace to pass: %v", err)
	}
	if err := EnforceMembershipSubscriptionExpiryPolicy(ctx, svcCtx, member, sub, 8, &currentExpireAt); err == nil {
		t.Fatal("expected subscription ending after grace to fail")
	}
}

func TestEnforceMembershipSubscriptionExpiryPolicyRejectsNoLimit(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&membership.Plan{}); err != nil {
		t.Fatalf("migrate membership plan: %v", err)
	}
	if err := db.Create(&membership.Plan{
		Name:                           "member",
		SubscriptionExpiryLimitEnabled: true,
	}).Error; err != nil {
		t.Fatalf("create membership plan: %v", err)
	}

	svcCtx := &svc.ServiceContext{DB: db}
	member := &user.User{Id: 1, MemberExpiredAt: time.Now().AddDate(0, 0, 30)}
	sub := &subscribe.Subscribe{UnitTime: "NoLimit"}

	if err := EnforceMembershipSubscriptionExpiryPolicy(ctx, svcCtx, member, sub, 1, nil); err == nil {
		t.Fatal("expected no-limit subscription to fail when policy is enabled")
	}
}
