package order

import (
	"context"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/membership"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
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

func TestQueryMembershipCardReturnsUsageAndFeatureState(t *testing.T) {
	svcCtx := newMembershipTestService(t, true)
	if err := svcCtx.DB.AutoMigrate(&modelOrder.Order{}); err != nil {
		t.Fatalf("migrate order: %v", err)
	}
	now := time.Now()
	orders := []modelOrder.Order{
		{UserId: 9, OrderNo: "included", Type: OrderTypeSubscribe, Status: 1, Amount: 120, CreatedAt: now},
		{UserId: 9, OrderNo: "closed", Type: OrderTypeRenewal, Status: 3, Amount: 500, CreatedAt: now},
		{UserId: 9, OrderNo: "recharge", Type: OrderTypeRecharge, Status: 5, Amount: 700, CreatedAt: now},
		{UserId: 10, OrderNo: "other-user", Type: OrderTypeSubscribe, Status: 5, Amount: 900, CreatedAt: now},
	}
	if err := svcCtx.DB.Create(&orders).Error; err != nil {
		t.Fatalf("create orders: %v", err)
	}
	member := &user.User{Id: 9, MemberExpiredAt: now.Add(24 * time.Hour)}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, member)

	card, err := NewQueryMembershipCardLogic(ctx, svcCtx).QueryMembershipCard()
	if err != nil {
		t.Fatalf("query membership card: %v", err)
	}
	if !card.Enabled || !card.IsMember {
		t.Fatalf("membership state = enabled:%v member:%v, want true/true", card.Enabled, card.IsMember)
	}
	if card.Usage.MonthlyConsumption != 120 || card.Usage.MonthlyOrderCount != 1 {
		t.Fatalf("membership usage = %#v, want 120/1", card.Usage)
	}
}

func TestQueryMembershipCardPrefersEnabledPlan(t *testing.T) {
	svcCtx := newMembershipTestService(t, false)
	enabled := true
	if err := svcCtx.DB.Create(&membership.Plan{
		Name:      "active membership",
		Enabled:   &enabled,
		Sort:      100,
		UnitPrice: 2_000,
	}).Error; err != nil {
		t.Fatalf("create enabled membership plan: %v", err)
	}
	if err := svcCtx.DB.AutoMigrate(&modelOrder.Order{}); err != nil {
		t.Fatalf("migrate order: %v", err)
	}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &user.User{Id: 12})

	card, err := NewQueryMembershipCardLogic(ctx, svcCtx).QueryMembershipCard()
	if err != nil {
		t.Fatalf("query membership card: %v", err)
	}
	if !card.Enabled || card.Plan.Name != "active membership" {
		t.Fatalf("membership card = %#v, want enabled plan", card)
	}
}

func TestPreMembershipUsesWalletDeductionOrder(t *testing.T) {
	svcCtx := newMembershipTestService(t, true)
	if err := svcCtx.DB.Model(&membership.Plan{}).Where("name = ?", "membership").Update("unit_price", 1_000).Error; err != nil {
		t.Fatalf("update membership price: %v", err)
	}
	currentUser := &user.User{Id: 11, Commission: 300, GiftAmount: 900}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, currentUser)

	quote, err := NewPreMembershipLogic(ctx, svcCtx).PreMembership(&types.PreMembershipOrderRequest{})
	if err != nil {
		t.Fatalf("preview membership: %v", err)
	}
	if quote.Price != 1_000 || quote.CommissionDeduction != 300 || quote.GiftAmount != 700 || quote.Amount != 0 {
		t.Fatalf("membership quote = %#v", quote)
	}
}
