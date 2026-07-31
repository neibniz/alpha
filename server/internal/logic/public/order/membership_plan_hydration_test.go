package order

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/membershipview"
	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestHydrateOrderMembershipPlansNormalizesPlanIDs(t *testing.T) {
	var queries []string
	db := setupMembershipPlanHydrationDB(t, &queries)
	enabled := true
	if err := db.Create(&membership.Plan{
		Id:            3,
		Name:          "Member",
		UnitPrice:     9900,
		DurationUnit:  "Month",
		DurationValue: 1,
		Enabled:       &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}

	orders := []types.OrderDetail{
		{Type: OrderTypeMembership, SubscribeId: 3, Subscribe: types.OrderSubscribeProduct{Id: 99, Name: "legacy"}},
		{Type: OrderTypeMembership, SubscribeId: 3},
		{Type: OrderTypeMembership},
		{Type: OrderTypeSubscribe, SubscribeId: 3},
		{Type: OrderTypeMembership, SubscribeId: 4, MembershipPlan: &types.MembershipPlan{Id: 4, Name: "snapshot"}},
	}

	if err := hydrateOrderMembershipPlans(context.Background(), &svc.ServiceContext{DB: db}, orders); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 {
		t.Fatalf("membership plan queries = %d, want 1: %#v", len(queries), queries)
	}
	if strings.Contains(queries[0], "3,3") {
		t.Fatalf("membership plan query kept duplicate ids: %s", queries[0])
	}
	if orders[0].Subscribe.Id != 0 || orders[0].Subscribe.Name != "" {
		t.Fatalf("membership order should clear subscribe payload: %#v", orders[0].Subscribe)
	}
	if orders[0].MembershipPlan == nil || orders[0].MembershipPlan.Id != 3 || orders[0].MembershipPlan.Name != "Member" {
		t.Fatalf("first membership plan not hydrated: %#v", orders[0].MembershipPlan)
	}
	if orders[1].MembershipPlan == nil || orders[1].MembershipPlan.Id != 3 {
		t.Fatalf("duplicate membership plan not hydrated: %#v", orders[1].MembershipPlan)
	}
	if orders[3].MembershipPlan != nil {
		t.Fatalf("non-membership order should not hydrate membership plan: %#v", orders[3].MembershipPlan)
	}
	if orders[4].MembershipPlan == nil || orders[4].MembershipPlan.Id != 4 || orders[4].MembershipPlan.Name != "snapshot" {
		t.Fatalf("existing membership snapshot should be preserved: %#v", orders[4].MembershipPlan)
	}
}

func TestHydratePublicOrderMembershipPlansNormalizesPlanIDs(t *testing.T) {
	var queries []string
	db := setupMembershipPlanHydrationDB(t, &queries)
	enabled := true
	if err := db.Create(&membership.Plan{
		Id:            3,
		Name:          "Member",
		UnitPrice:     9900,
		DurationUnit:  "Month",
		DurationValue: 1,
		Enabled:       &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}

	orders := []types.PublicOrder{
		{Type: OrderTypeMembership, SubscribeId: 3},
		{Type: OrderTypeMembership, SubscribeId: 3},
		{Type: OrderTypeSubscribe, SubscribeId: 3},
		{Type: OrderTypeMembership, SubscribeId: 4, MembershipPlan: &types.MembershipPlan{Id: 4, Name: "snapshot"}},
	}

	if err := hydratePublicOrderMembershipPlans(context.Background(), &svc.ServiceContext{DB: db}, orders); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 {
		t.Fatalf("public membership plan queries = %d, want 1: %#v", len(queries), queries)
	}
	if strings.Contains(queries[0], "3,3") {
		t.Fatalf("public membership plan query kept duplicate ids: %s", queries[0])
	}
	if orders[0].MembershipPlan == nil || orders[0].MembershipPlan.Id != 3 || orders[0].MembershipPlan.Name != "Member" {
		t.Fatalf("first public membership plan not hydrated: %#v", orders[0].MembershipPlan)
	}
	if orders[1].MembershipPlan == nil || orders[1].MembershipPlan.Id != 3 {
		t.Fatalf("duplicate public membership plan not hydrated: %#v", orders[1].MembershipPlan)
	}
	if orders[2].MembershipPlan != nil {
		t.Fatalf("non-membership public order should not hydrate membership plan: %#v", orders[2].MembershipPlan)
	}
	if orders[3].MembershipPlan == nil || orders[3].MembershipPlan.Id != 4 || orders[3].MembershipPlan.Name != "snapshot" {
		t.Fatalf("existing public membership snapshot should be preserved: %#v", orders[3].MembershipPlan)
	}
}

func TestNewMembershipPlanRowsUsesUniquePlanCapacity(t *testing.T) {
	rows := membershipview.NewPlanRows([]int64{3, 4, 5})
	if rows == nil || len(rows) != 0 || cap(rows) != 3 {
		t.Fatalf("membership plan rows len/cap = %d/%d, want 0/3", len(rows), cap(rows))
	}
	empty := membershipview.NewPlanRows(nil)
	if empty == nil || len(empty) != 0 || cap(empty) != 0 {
		t.Fatalf("empty membership plan rows len/cap = %d/%d, want 0/0 non-nil", len(empty), cap(empty))
	}
}

func setupMembershipPlanHydrationDB(t *testing.T, queries *[]string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: membershipPlanQueryLogger{queries: queries},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&membership.Plan{}); err != nil {
		t.Fatal(err)
	}
	return db
}

type membershipPlanQueryLogger struct {
	queries *[]string
}

func (l membershipPlanQueryLogger) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (l membershipPlanQueryLogger) Info(context.Context, string, ...interface{}) {}

func (l membershipPlanQueryLogger) Warn(context.Context, string, ...interface{}) {}

func (l membershipPlanQueryLogger) Error(context.Context, string, ...interface{}) {}

func (l membershipPlanQueryLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	if strings.Contains(sql, "FROM `membership_plan`") || strings.Contains(sql, `FROM "membership_plan"`) {
		*l.queries = append(*l.queries, sql)
	}
}
