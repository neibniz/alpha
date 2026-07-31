package console

import (
	"context"
	"strings"
	"testing"
	"time"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserTotalsQueryUsesSingleProjectedStatement(t *testing.T) {
	normalized := strings.ToLower(userTotalsSQL)
	if strings.Contains(normalized, "select *") {
		t.Fatal("user totals query must not select full rows")
	}
	for _, required := range []string{
		`from "user"`,
		`from "order"`,
		"cross join",
		"as today_register",
		"as monthly_register",
		"as all_register",
		"as today_new_order_users",
		"as monthly_renewal_order_users",
		"as all_membership_users",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("query missing %q: %s", required, userTotalsSQL)
		}
	}
	if got := strings.Count(normalized, `from "user"`); got != 1 {
		t.Fatalf("totals query user table scans = %d, want 1", got)
	}
	if got := strings.Count(normalized, `from "order"`); got != 1 {
		t.Fatalf("totals query order table scans = %d, want 1", got)
	}
}

func TestUserSeriesQueryUsesSingleUnionStatement(t *testing.T) {
	sql := userSeriesSQL("DATE_TRUNC('day', created_at)")
	normalized := strings.ToLower(sql)
	if strings.Contains(normalized, "select *") {
		t.Fatal("user series query must not select full rows")
	}
	for _, required := range []string{
		`from "user"`,
		`from "order"`,
		"union all",
		"group by bucket",
		"order by bucket asc",
		"as register",
		"as new_order_users",
		"as renewal_order_users",
		"as membership_users",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("series query missing %q: %s", required, sql)
		}
	}
	if got := strings.Count(normalized, `from "user"`); got != 1 {
		t.Fatalf("series query user table scans = %d, want 1", got)
	}
	if got := strings.Count(normalized, `from "order"`); got != 1 {
		t.Fatalf("series query order table scans = %d, want 1", got)
	}
}

func TestUserSeriesRowsToStatisticsFillsMissingBuckets(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	r := statisticsRange{
		Start:      start,
		End:        start.AddDate(0, 0, 3),
		Step:       24 * time.Hour,
		DateFormat: time.DateOnly,
	}
	got := userSeriesRowsToStatistics([]userSeriesRow{
		{
			Bucket:            start,
			Register:          2,
			NewOrderUsers:     1,
			RenewalOrderUsers: 1,
		},
		{
			Bucket:            start.Add(12 * time.Hour),
			Register:          99,
			NewOrderUsers:     99,
			RenewalOrderUsers: 99,
			MembershipUsers:   99,
		},
		{
			Bucket:          start.AddDate(0, 0, 2),
			MembershipUsers: 3,
		},
		{
			Bucket:          start.AddDate(0, 0, 3),
			MembershipUsers: 9,
		},
	}, r)

	if len(got) != 3 {
		t.Fatalf("series length = %d, want 3", len(got))
	}
	if got[0].Date != "2026-07-01" || got[0].Register != 2 || got[0].NewOrderUsers != 1 || got[0].RenewalOrderUsers != 1 || got[0].MembershipUsers != 0 {
		t.Fatalf("first bucket = %#v", got[0])
	}
	if got[1].Date != "2026-07-02" || got[1].Register != 0 || got[1].NewOrderUsers != 0 || got[1].RenewalOrderUsers != 0 || got[1].MembershipUsers != 0 {
		t.Fatalf("missing bucket = %#v", got[1])
	}
	if got[2].Date != "2026-07-03" || got[2].Register != 0 || got[2].NewOrderUsers != 0 || got[2].RenewalOrderUsers != 0 || got[2].MembershipUsers != 3 {
		t.Fatalf("last bucket = %#v", got[2])
	}
}

func TestQueryUserTotalsUseCombinedQuery(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelOrder.Order{}); err != nil {
		t.Fatal(err)
	}

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	enable := true
	users := []modelUser.User{
		{Id: 1, Password: "x", Enable: &enable, CreatedAt: day.Add(time.Hour)},
		{Id: 2, Password: "x", Enable: &enable, CreatedAt: day.Add(2 * time.Hour)},
		{Id: 3, Password: "x", Enable: &enable, CreatedAt: day.AddDate(0, 0, 2)},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}

	orders := []modelOrder.Order{
		{Id: 1, OrderNo: "o1", Type: orderTypeSubscribe, Status: orderStatusPaid, Amount: 100, CreatedAt: day.Add(time.Hour)},
		{Id: 2, OrderNo: "o2", Type: orderTypeRenewal, Status: orderStatusFinished, Amount: 200, CreatedAt: day.Add(2 * time.Hour)},
		{Id: 3, OrderNo: "o3", Type: orderTypeMembership, Status: orderStatusPaid, Amount: 300, CreatedAt: day.AddDate(0, 0, 2)},
		{Id: 4, OrderNo: "o4", Type: orderTypeResetTraffic, Status: 1, Amount: 400, CreatedAt: day.Add(3 * time.Hour)},
	}
	if err := db.Create(&orders).Error; err != nil {
		t.Fatal(err)
	}

	monthStart := day
	logic := &QueryUserStatisticsLogic{ctx: ctx, svcCtx: &svc.ServiceContext{DB: db}}
	totals, err := logic.queryUserTotals(day, day.AddDate(0, 0, 1), monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}

	todayStats := totals.today()
	if todayStats.Register != 2 || todayStats.NewOrderUsers != 1 || todayStats.RenewalOrderUsers != 1 || todayStats.MembershipUsers != 0 {
		t.Fatalf("today stats = %#v", todayStats)
	}

	monthlyStats := totals.monthly()
	if monthlyStats.Register != 3 || monthlyStats.NewOrderUsers != 1 || monthlyStats.RenewalOrderUsers != 1 || monthlyStats.MembershipUsers != 1 {
		t.Fatalf("monthly stats = %#v", monthlyStats)
	}

	allStats := totals.all()
	if allStats.Register != 3 || allStats.NewOrderUsers != 1 || allStats.RenewalOrderUsers != 1 || allStats.MembershipUsers != 1 {
		t.Fatalf("all stats = %#v", allStats)
	}
}
