package console

import (
	"context"
	"strings"
	"testing"
	"time"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRevenueTotalsQueryUsesSingleProjectedStatement(t *testing.T) {
	normalized := strings.ToLower(revenueTotalsSQL)
	if strings.Contains(normalized, "select *") {
		t.Fatal("revenue totals query must not select full rows")
	}
	for _, required := range []string{
		`from "order"`,
		"status in",
		"type in",
		"as today_amount_total",
		"as monthly_amount_total",
		"as all_amount_total",
		"as today_new_order_amount",
		"as monthly_renewal_order_amount",
		"as all_reset_traffic_amount",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("revenue totals query missing %q: %s", required, revenueTotalsSQL)
		}
	}
}

func TestQueryRevenueTotalsAggregatesTodayMonthlyAndAll(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelOrder.Order{}); err != nil {
		t.Fatal(err)
	}

	today := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	monthStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	rows := []modelOrder.Order{
		{Id: 1, OrderNo: "o1", Type: orderTypeSubscribe, Status: orderStatusPaid, Amount: 100, CreatedAt: today.Add(time.Hour)},
		{Id: 2, OrderNo: "o2", Type: orderTypeRenewal, Status: orderStatusFinished, Amount: 200, CreatedAt: today.Add(2 * time.Hour)},
		{Id: 3, OrderNo: "o3", Type: orderTypeResetTraffic, Status: orderStatusPaid, Amount: 50, CreatedAt: today.Add(3 * time.Hour)},
		{Id: 4, OrderNo: "o4", Type: orderTypeMembership, Status: orderStatusPaid, Amount: 300, CreatedAt: monthStart.AddDate(0, 0, 2)},
		{Id: 5, OrderNo: "o5", Type: orderTypeSubscribe, Status: orderStatusPaid, Amount: 400, CreatedAt: monthStart.AddDate(0, -1, 0)},
		{Id: 6, OrderNo: "o6", Type: orderTypeRecharge, Status: orderStatusPaid, Amount: 500, CreatedAt: today.Add(4 * time.Hour)},
		{Id: 7, OrderNo: "o7", Type: orderTypeSubscribe, Status: 1, Amount: 600, CreatedAt: today.Add(5 * time.Hour)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	logic := &QueryRevenueStatisticsLogic{ctx: ctx, svcCtx: &svc.ServiceContext{DB: db}}
	totals, err := logic.queryRevenueTotals(today, today.AddDate(0, 0, 1), monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}

	todayStats := totals.today()
	if todayStats.AmountTotal != 350 || todayStats.NewOrderAmount != 100 || todayStats.RenewalOrderAmount != 200 || todayStats.ResetTrafficAmount != 50 || todayStats.MembershipAmount != 0 {
		t.Fatalf("today revenue = %#v", todayStats)
	}

	monthlyStats := totals.monthly()
	if monthlyStats.AmountTotal != 650 || monthlyStats.NewOrderAmount != 100 || monthlyStats.RenewalOrderAmount != 200 || monthlyStats.ResetTrafficAmount != 50 || monthlyStats.MembershipAmount != 300 {
		t.Fatalf("monthly revenue = %#v", monthlyStats)
	}

	allStats := totals.all()
	if allStats.AmountTotal != 1050 || allStats.NewOrderAmount != 500 || allStats.RenewalOrderAmount != 200 || allStats.ResetTrafficAmount != 50 || allStats.MembershipAmount != 300 {
		t.Fatalf("all revenue = %#v", allStats)
	}
}

func TestRevenueSeriesRowsToStatisticsFillsMissingBuckets(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	r := statisticsRange{
		Start:      start,
		End:        start.AddDate(0, 0, 3),
		Step:       24 * time.Hour,
		DateFormat: time.DateOnly,
	}
	got := revenueSeriesRowsToStatistics([]revenueSeriesRow{
		{
			Bucket:             start,
			AmountTotal:        100,
			NewOrderAmount:     60,
			RenewalOrderAmount: 40,
		},
		{
			Bucket:             start.Add(12 * time.Hour),
			AmountTotal:        999,
			NewOrderAmount:     999,
			RenewalOrderAmount: 999,
		},
		{
			Bucket:             start.AddDate(0, 0, 2),
			MembershipAmount:   30,
			ResetTrafficAmount: 10,
			RenewalOrderAmount: 20,
			NewOrderAmount:     5,
			AmountTotal:        65,
		},
	}, r)

	if len(got) != 3 {
		t.Fatalf("series length = %d, want 3", len(got))
	}
	if got[0].Date != "2026-07-01" || got[0].AmountTotal != 100 || got[0].NewOrderAmount != 60 || got[0].RenewalOrderAmount != 40 {
		t.Fatalf("first bucket = %#v", got[0])
	}
	if got[1].Date != "2026-07-02" || got[1].AmountTotal != 0 || got[1].NewOrderAmount != 0 || got[1].RenewalOrderAmount != 0 || got[1].MembershipAmount != 0 || got[1].ResetTrafficAmount != 0 {
		t.Fatalf("missing bucket = %#v", got[1])
	}
	if got[2].Date != "2026-07-03" || got[2].AmountTotal != 65 || got[2].NewOrderAmount != 5 || got[2].RenewalOrderAmount != 20 || got[2].MembershipAmount != 30 || got[2].ResetTrafficAmount != 10 {
		t.Fatalf("last bucket = %#v", got[2])
	}
}
