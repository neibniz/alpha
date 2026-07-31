package order

import (
	"context"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMembershipTestService(t *testing.T, enabled bool) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&membership.Plan{}); err != nil {
		t.Fatalf("migrate membership plan: %v", err)
	}
	if err := db.Create(&membership.Plan{
		Name:                           "membership",
		Enabled:                        &enabled,
		MonthlyConsumptionLimitEnabled: true,
		MonthlyConsumptionLimit:        1,
		MonthlyOrderLimitEnabled:       true,
		MonthlyOrderLimit:              1,
		SubscriptionExpiryLimitEnabled: true,
	}).Error; err != nil {
		t.Fatalf("create membership plan: %v", err)
	}
	return &svc.ServiceContext{DB: db}
}

func TestRequireActiveMemberSkipsWhenMembershipFeatureDisabled(t *testing.T) {
	ctx := context.Background()
	svcCtx := newMembershipTestService(t, false)

	if err := requireActiveMember(ctx, svcCtx, &user.User{Id: 1}); err != nil {
		t.Fatalf("disabled membership feature should not require membership: %v", err)
	}
}

func TestRequireActiveMemberRejectsNonMemberWhenFeatureEnabled(t *testing.T) {
	ctx := context.Background()
	svcCtx := newMembershipTestService(t, true)

	if err := requireActiveMember(ctx, svcCtx, &user.User{Id: 1}); err == nil {
		t.Fatal("enabled membership feature should reject a non-member")
	}
}

func TestDisabledMembershipFeatureSkipsAllPolicies(t *testing.T) {
	ctx := context.Background()
	svcCtx := newMembershipTestService(t, false)
	nonMember := &user.User{Id: 1}

	if err := enforceMembershipOrderPolicies(ctx, svcCtx, nonMember, 100); err != nil {
		t.Fatalf("disabled membership feature should skip order policies: %v", err)
	}
	if err := EnforceMembershipSubscriptionExpiryPolicy(
		ctx,
		svcCtx,
		nonMember,
		&subscribe.Subscribe{UnitTime: "NoLimit"},
		1,
		nil,
	); err != nil {
		t.Fatalf("disabled membership feature should skip expiry policy: %v", err)
	}
}
