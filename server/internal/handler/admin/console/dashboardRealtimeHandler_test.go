package console

import (
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/order"
	trafficModel "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDashboardActivitySelectColumnsStayProjected(t *testing.T) {
	assertDashboardColumns(t, dashboardSystemActivitySelectColumns, []string{
		"id",
		"type",
		"object_id AS user_id",
		"method",
		"method AS auth_method",
		"identifier",
		"ip AS client_ip",
		"success",
		"event_at AS timestamp",
		"created_at",
	}, []string{"content"})
	assertDashboardColumnsAvoidJSONContent(t, dashboardSystemActivitySelectColumns)

	assertDashboardColumns(t, dashboardSubscribeActivitySelectColumns, []string{
		"id",
		"user_id",
		"user_subscribe_id",
		"client_ip",
		"country_short",
		"region",
		"location",
		"success",
		"created_at",
	}, []string{"token_hash"})
}

func assertDashboardColumnsAvoidJSONContent(t *testing.T, columns []string) {
	t.Helper()
	for _, column := range columns {
		if strings.Contains(column, "content::jsonb") {
			t.Fatalf("dashboard realtime columns must not parse content JSON: %s", column)
		}
	}
}

func TestDashboardOrderActivitySelectColumnsStayProjected(t *testing.T) {
	assertDashboardColumns(t, dashboardOrderActivitySelectColumns, []string{
		`"order".id`,
		`"order".user_id`,
		`"order".type`,
		`"order".status`,
		`"order".amount`,
		`"order".subscribe_id`,
		`COALESCE(NULLIF("order".product_name, ''), subscribe.name, '') AS product_name`,
		`"order".client_ip`,
		`"order".country_short`,
		`"order".region`,
		`"order".location`,
		`"order".created_at`,
	}, []string{
		`"order".product_snapshot`,
		`"order".subscribe_token`,
		`"order".coupon`,
		`"order".trade_no`,
		`"order".payment_id`,
	})
}

func TestDashboardNetworkCandidatesSQLStaysProjectedUnion(t *testing.T) {
	normalized := strings.ToLower(dashboardActivityNetworkCandidatesSQL)
	for _, required := range []string{
		"union all",
		"from system_logs",
		"from subscribe_access_logs",
		"object_id as user_id",
		"client_ip as ip",
		"order by created_at desc, id desc",
		"limit ?",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("network candidate query missing %q: %s", required, dashboardActivityNetworkCandidatesSQL)
		}
	}
	for _, forbidden := range []string{"content", "token_hash", "token_prefix", "user_agent"} {
		if strings.Contains(normalized, forbidden) {
			t.Fatalf("network candidate query must not project %q: %s", forbidden, dashboardActivityNetworkCandidatesSQL)
		}
	}
}

func TestQueryActivityNetworkCandidatesUsesUnionRows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&log.SystemLog{}, &log.SubscribeAccessLog{}); err != nil {
		t.Fatalf("migrate logs: %v", err)
	}

	now := time.Now()
	if err := db.Create([]log.SystemLog{
		{Id: 1, ObjectID: 7, IP: "198.51.100.1", CountryShort: "US", Region: "CA", CreatedAt: now.Add(-2 * time.Hour)},
		{Id: 2, ObjectID: 8, IP: "198.51.100.2", CountryShort: "US", Region: "NY", CreatedAt: now.Add(-30 * time.Minute)},
		{Id: 3, ObjectID: 7, IP: "", CountryShort: "US", Region: "WA", CreatedAt: now.Add(-10 * time.Minute)},
	}).Error; err != nil {
		t.Fatalf("seed system logs: %v", err)
	}
	if err := db.Create([]log.SubscribeAccessLog{
		{Id: 10, UserId: 7, ClientIP: "203.0.113.7", Location: "cached location", CreatedAt: now.Add(-time.Hour)},
		{Id: 11, UserId: 9, ClientIP: "203.0.113.9", CountryShort: "JP", Region: "13", CreatedAt: now.Add(-20 * time.Minute)},
	}).Error; err != nil {
		t.Fatalf("seed subscribe logs: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/dashboard/realtime", nil)

	got := queryActivityNetworkCandidates(
		c,
		&svc.ServiceContext{DB: db},
		[]int64{7},
		now.Add(-3*time.Hour),
		now,
	)
	if len(got) != 2 {
		t.Fatalf("network candidates len = %d, want 2: %#v", len(got), got)
	}
	if got[0].IP != "203.0.113.*" || got[0].Location != "cached location" {
		t.Fatalf("latest candidate = %#v", got[0])
	}
	if got[1].IP != "198.51.100.*" || got[1].Location != "US · CA" {
		t.Fatalf("older candidate = %#v", got[1])
	}

	empty := queryActivityNetworkCandidates(
		c,
		&svc.ServiceContext{DB: db},
		[]int64{7},
		now.Add(-24*time.Hour),
		now.Add(-23*time.Hour),
	)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty network candidates = %#v, want non-nil empty slice", empty)
	}
}

func TestQueryActivityOrderNetworksReturnsNilWithoutLookupData(t *testing.T) {
	orders := []dashboardOrderActivity{
		{Id: 1, UserId: 0, CreatedAt: time.Now()},
		{Id: 2, UserId: -1, CreatedAt: time.Now().Add(time.Hour)},
	}
	if got := queryActivityOrderNetworks(nil, nil, orders); got != nil {
		t.Fatalf("queryActivityOrderNetworks without lookup data = %#v, want nil", got)
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = queryActivityOrderNetworks(nil, nil, orders)
	})
	if allocs != 0 {
		t.Fatalf("queryActivityOrderNetworks no lookup allocations = %f, want 0", allocs)
	}
}

func TestQueryActivityOrderNetworksUsesOrderClientIP(t *testing.T) {
	now := time.Now()
	orders := []dashboardOrderActivity{{
		Id:           9,
		UserId:       7,
		ClientIP:     "203.0.113.9",
		CountryShort: "US",
		Region:       "CA",
		CreatedAt:    now,
	}}

	got := queryActivityOrderNetworks(nil, &svc.ServiceContext{}, orders)
	network, ok := got[9]
	if !ok {
		t.Fatalf("order network missing: %#v", got)
	}
	if network.IP != "203.0.113.9" {
		t.Fatalf("network IP = %q", network.IP)
	}
	if network.Location != "US · CA" {
		t.Fatalf("network Location = %q", network.Location)
	}
	if !network.CreatedAt.Equal(now) {
		t.Fatalf("network CreatedAt = %v, want %v", network.CreatedAt, now)
	}
}

func TestDashboardActivityNetworkCandidateIndexUsesLinearPathForSmallInput(t *testing.T) {
	index := newDashboardActivityNetworkCandidateIndex([]int64{7, 8, 7})
	if index.byUser != nil {
		t.Fatal("small activity network candidate index should not allocate map")
	}
	index.Add(dashboardActivityNetworkCandidate{UserID: 7, IP: "203.0.113.7"})
	index.Add(dashboardActivityNetworkCandidate{UserID: 9, IP: "203.0.113.9"})

	got := index.Get(7)
	if len(got) != 1 || got[0].IP != "203.0.113.7" {
		t.Fatalf("candidate index get(7) = %#v", got)
	}
	if got := index.Get(8); got == nil || len(got) != 0 {
		t.Fatalf("candidate index get(8) = %#v, want non-nil empty slice for known user", got)
	}
	if got := index.Get(9); got != nil {
		t.Fatalf("candidate index should ignore unknown user: %#v", got)
	}
}

func TestDashboardActivityNetworkCandidateIndexUpgradesLargeInputToMap(t *testing.T) {
	userIDs := make([]int64, 0, dashboardSmallInt64SetLimit+1)
	for id := int64(1); id <= dashboardSmallInt64SetLimit+1; id++ {
		userIDs = append(userIDs, id)
	}
	index := newDashboardActivityNetworkCandidateIndex(userIDs)
	if index.byUser == nil {
		t.Fatal("large activity network candidate index should allocate map")
	}
	index.Add(dashboardActivityNetworkCandidate{UserID: 1, IP: "203.0.113.1"})
	index.Add(dashboardActivityNetworkCandidate{UserID: int64(dashboardSmallInt64SetLimit + 1), IP: "203.0.113.9"})

	if got := index.Get(1); len(got) != 1 || got[0].IP != "203.0.113.1" {
		t.Fatalf("candidate index get(first) = %#v", got)
	}
	if got := index.Get(int64(dashboardSmallInt64SetLimit + 1)); len(got) != 1 || got[0].IP != "203.0.113.9" {
		t.Fatalf("candidate index get(last) = %#v", got)
	}
	if got := index.Get(2); got == nil || len(got) != 0 {
		t.Fatalf("candidate index get(empty large) = %#v, want non-nil empty slice", got)
	}
	if cap(index.buckets) != dashboardSmallInt64SetLimit+1 {
		t.Fatalf("candidate index bucket cap = %d, want %d", cap(index.buckets), dashboardSmallInt64SetLimit+1)
	}
}

func TestDashboardActivityNetworkCandidateIndexAvoidsMapForLargeDuplicateInput(t *testing.T) {
	userIDs := make([]int64, 0, dashboardSmallInt64SetLimit+20)
	for i := 0; i < dashboardSmallInt64SetLimit+20; i++ {
		userIDs = append(userIDs, 7)
	}
	index := newDashboardActivityNetworkCandidateIndex(userIDs)

	if index.byUser != nil {
		t.Fatal("duplicate-heavy activity network candidate index should not allocate map")
	}
	if len(index.buckets) != 1 {
		t.Fatalf("candidate index bucket len = %d, want 1", len(index.buckets))
	}
	if cap(index.buckets) != dashboardSmallInt64SetLimit+1 {
		t.Fatalf("candidate index bucket cap = %d, want %d", cap(index.buckets), dashboardSmallInt64SetLimit+1)
	}
	index.Add(dashboardActivityNetworkCandidate{UserID: 7, IP: "203.0.113.7"})
	if got := index.Get(7); len(got) != 1 || got[0].IP != "203.0.113.7" {
		t.Fatalf("candidate index get(7) = %#v", got)
	}
}

func TestDashboardActivityNetworkCandidateIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	userIDs := make([]int64, 0, dashboardSmallInt64SetLimit+20)
	for id := int64(1); id <= dashboardSmallInt64SetLimit+1; id++ {
		userIDs = append(userIDs, id)
	}
	for len(userIDs) < dashboardSmallInt64SetLimit+20 {
		userIDs = append(userIDs, 1)
	}
	index := newDashboardActivityNetworkCandidateIndex(userIDs)

	if index.byUser == nil {
		t.Fatal("large activity network candidate index should allocate map")
	}
	if len(index.buckets) != dashboardSmallInt64SetLimit+1 {
		t.Fatalf("candidate index bucket len = %d, want %d", len(index.buckets), dashboardSmallInt64SetLimit+1)
	}
	if maxCap := len(index.buckets) * 2; cap(index.buckets) > maxCap {
		t.Fatalf("candidate index bucket cap = %d, want <= %d", cap(index.buckets), maxCap)
	}
}

func TestDashboardPresentationMasksSensitiveValues(t *testing.T) {
	if got := dashboardDisplayIP("203.0.113.42"); got != "203.0.113.*" {
		t.Fatalf("dashboardDisplayIP() = %q", got)
	}
	if got := dashboardDisplayIdentifier("user@example.com"); got != "u**r@example.com" {
		t.Fatalf("dashboardDisplayIdentifier(email) = %q", got)
	}
	if got := dashboardDisplayIdentifier("device-fingerprint-abcdef"); got != "devi...cdef" {
		t.Fatalf("dashboardDisplayIdentifier(device) = %q", got)
	}
}

func TestDisplaySubscriptionUsesNameBeforeFallback(t *testing.T) {
	if got := displaySubscription(12, " Premium "); got != "Premium" {
		t.Fatalf("displaySubscription(named) = %q", got)
	}
	if got := displaySubscription(12, ""); got != "Subscription #12" {
		t.Fatalf("displaySubscription(fallback) = %q", got)
	}
	if got := displaySubscription(0, " "); got != "Subscription #0" {
		t.Fatalf("displaySubscription(zero) = %q", got)
	}
}

func TestDisplaySubscriptionNamedInputDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = displaySubscription(12, "Premium")
	})
	if allocs != 0 {
		t.Fatalf("displaySubscription named allocations = %f, want 0", allocs)
	}
}

func TestDashboardPrefixedIntBuildsStableLabels(t *testing.T) {
	if got := dashboardPrefixedInt("log-", 42); got != "log-42" {
		t.Fatalf("dashboardPrefixedInt(log) = %q", got)
	}
	if got := dashboardPrefixedInt("User #", 7); got != "User #7" {
		t.Fatalf("dashboardPrefixedInt(user) = %q", got)
	}
	if got := dashboardPrefixedInt("Subscription #", 0); got != "Subscription #0" {
		t.Fatalf("dashboardPrefixedInt(subscription) = %q", got)
	}
}

func TestDashboardPrefixedIntAllocatesAtMostOnce(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = dashboardPrefixedInt("subscribe-", 123456)
	})
	if allocs > 1 {
		t.Fatalf("dashboardPrefixedInt allocations = %f, want <= 1", allocs)
	}
}

func TestActivityDetailTrimsSkipsAndJoinsValues(t *testing.T) {
	if got := activityDetail("", "  email  "); got != "email" {
		t.Fatalf("activityDetail(single) = %q", got)
	}
	if got := activityDetail(" US ", "", " CA ", " San Francisco "); got != "US · CA · San Francisco" {
		t.Fatalf("activityDetail(multiple) = %q", got)
	}
	if got := activityDetail("", "  "); got != "" {
		t.Fatalf("activityDetail(empty) = %q", got)
	}
}

func TestActivityDetailEmptyAndSingleInputsDoNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = activityDetail("", "  ")
		_ = activityDetail("", "  email  ")
	})
	if allocs != 0 {
		t.Fatalf("activityDetail empty/single allocations = %f, want 0", allocs)
	}
}

func TestDashboardActivityLocationUsesCachedLocationFirst(t *testing.T) {
	if got := dashboardActivityLocation(nil, "203.0.113.10", " cached location ", "", ""); got != "cached location" {
		t.Fatalf("dashboardActivityLocation cached = %q", got)
	}
	if got := dashboardActivityLocation(nil, "203.0.113.10", "", " US ", " CA "); got != "US · CA" {
		t.Fatalf("dashboardActivityLocation region = %q", got)
	}
	if got := dashboardActivityLocation(nil, "203.0.113.10", "", "", ""); got != "" {
		t.Fatalf("dashboardActivityLocation empty = %q", got)
	}
}

func TestDashboardActivityLocationCachedInputDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = dashboardActivityLocation(nil, "203.0.113.10", " cached location ", "", "")
	})
	if allocs != 0 {
		t.Fatalf("dashboardActivityLocation cached allocations = %f, want 0", allocs)
	}
}

func TestQueryUserLabelsCachedReusesKnownIDsAndMarksMisses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&user.AuthMethods{}); err != nil {
		t.Fatalf("migrate auth methods: %v", err)
	}
	if err := db.Create([]user.AuthMethods{
		{UserId: 1, AuthType: "email", AuthIdentifier: "db@example.com"},
		{UserId: 2, AuthType: "email", AuthIdentifier: "fresh@example.com"},
		{UserId: 3, AuthType: "mobile", AuthIdentifier: "+8613800000000"},
	}).Error; err != nil {
		t.Fatalf("seed auth methods: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/dashboard/realtime", nil)

	cache := map[int64]string{1: "cached-user"}
	got := queryUserLabelsCached(c, &svc.ServiceContext{DB: db}, []int64{1, 2, 2, 3}, cache)
	if got[1] != "cached-user" {
		t.Fatalf("cached label was not reused: %#v", got)
	}
	if got[2] != "f***h@example.com" {
		t.Fatalf("fresh label = %q, want masked email", got[2])
	}
	if label, ok := got[3]; ok || label != "" {
		t.Fatalf("non-email user should not be returned: %#v", got)
	}
	if label, ok := cache[3]; !ok || label != "" {
		t.Fatalf("missing email should be remembered in cache: %#v", cache)
	}

	if err := db.Model(&user.AuthMethods{}).
		Where("user_id = ? AND auth_type = ?", 2, "email").
		Update("auth_identifier", "changed@example.com").Error; err != nil {
		t.Fatalf("change cached row: %v", err)
	}
	got = queryUserLabelsCached(c, &svc.ServiceContext{DB: db}, []int64{2, 3}, cache)
	if got[2] != "f***h@example.com" {
		t.Fatalf("cached label was overwritten after DB change: %#v", got)
	}
	if _, ok := got[3]; ok {
		t.Fatalf("cached miss should remain omitted from result: %#v", got)
	}
}

func TestDashboardLookupHelpersReturnNilForEmptyInputs(t *testing.T) {
	if got := queryServerNames(nil, nil, nil); got != nil {
		t.Fatalf("queryServerNames empty = %#v, want nil", got)
	}
	if got := queryActivityUsers(nil, nil, []dashboardOrderActivity{{UserId: 0}}, nil); got != nil {
		t.Fatalf("queryActivityUsers without positive ids = %#v, want nil", got)
	}
	if got := queryActivityMembershipPlans(nil, nil, []dashboardOrderActivity{{Type: orderTypeSubscribe, SubscribeId: 1}}); got != nil {
		t.Fatalf("queryActivityMembershipPlans without membership ids = %#v, want nil", got)
	}
	if got := queryUserSubscribeNames(nil, nil, nil); got != nil {
		t.Fatalf("queryUserSubscribeNames empty = %#v, want nil", got)
	}
	if got := queryUserLabelsCached(nil, nil, nil, nil); got != nil {
		t.Fatalf("queryUserLabelsCached empty = %#v, want nil", got)
	}
	if got := queryUserLabelsCached(nil, nil, []int64{3}, map[int64]string{3: ""}); got != nil {
		t.Fatalf("queryUserLabelsCached cached miss = %#v, want nil", got)
	}
}

func TestDashboardLookupHelpersEmptyInputsDoNotAllocate(t *testing.T) {
	ordersWithoutUsers := []dashboardOrderActivity{{UserId: 0}}
	ordersWithoutMembership := []dashboardOrderActivity{{Type: orderTypeSubscribe, SubscribeId: 1}}
	cachedMissIDs := []int64{3}
	cacheMiss := map[int64]string{3: ""}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = queryServerNames(nil, nil, nil)
		_ = queryActivityUsers(nil, nil, ordersWithoutUsers, nil)
		_ = queryActivityMembershipPlans(nil, nil, ordersWithoutMembership)
		_ = queryUserSubscribeNames(nil, nil, nil)
		_ = queryUserLabelsCached(nil, nil, nil, nil)
		_ = queryUserLabelsCached(nil, nil, cachedMissIDs, cacheMiss)
	})
	if allocs != 0 {
		t.Fatalf("dashboard lookup helpers empty input allocations = %f, want 0", allocs)
	}
}

func TestQueryDashboardBusinessAggregatesAmountsAndCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&order.Order{}, &user.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now()
	todayStart := startOfDay(now)
	yesterday := todayStart.Add(-time.Hour)
	if err := db.Create([]order.Order{
		{Id: 1, OrderNo: "subscribe", Type: orderTypeSubscribe, Status: orderStatusPaid, Amount: 100, CreatedAt: todayStart.Add(time.Hour)},
		{Id: 2, OrderNo: "renewal", Type: orderTypeRenewal, Status: orderStatusFinished, Amount: 200, CreatedAt: todayStart.Add(2 * time.Hour)},
		{Id: 3, OrderNo: "membership", Type: orderTypeMembership, Status: orderStatusPaid, Amount: 300, CreatedAt: todayStart.Add(3 * time.Hour)},
		{Id: 4, OrderNo: "reset", Type: orderTypeResetTraffic, Status: orderStatusPaid, Amount: 40, CreatedAt: todayStart.Add(4 * time.Hour)},
		{Id: 5, OrderNo: "recharge", Type: orderTypeRecharge, Status: orderStatusPaid, Amount: 900, CreatedAt: todayStart.Add(5 * time.Hour)},
		{Id: 6, OrderNo: "pending", Type: orderTypeSubscribe, Status: 1, Amount: 700, CreatedAt: todayStart.Add(6 * time.Hour)},
		{Id: 7, OrderNo: "old", Type: orderTypeSubscribe, Status: orderStatusPaid, Amount: 800, CreatedAt: yesterday},
	}).Error; err != nil {
		t.Fatalf("seed orders: %v", err)
	}

	enabled := true
	admin := false
	if err := db.Create([]user.User{
		{Id: 1, Password: "secret", Enable: &enabled, IsAdmin: &admin, CreatedAt: todayStart.Add(time.Hour)},
		{Id: 2, Password: "secret", Enable: &enabled, IsAdmin: &admin, CreatedAt: todayStart.Add(2 * time.Hour)},
		{Id: 3, Password: "secret", Enable: &enabled, IsAdmin: &admin, CreatedAt: yesterday},
	}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/dashboard/realtime", nil)

	got := queryDashboardBusiness(c, &svc.ServiceContext{DB: db})
	if got.RevenueToday != 640 {
		t.Fatalf("RevenueToday = %d, want 640", got.RevenueToday)
	}
	if got.NewOrderAmount != 100 || got.RenewalOrderAmount != 200 || got.MembershipAmount != 300 || got.ResetTrafficAmount != 40 {
		t.Fatalf("amount split = %+v", got)
	}
	if got.NewOrderCount != 1 || got.RenewalOrderCount != 1 || got.MembershipOrderCount != 1 {
		t.Fatalf("order count split = %+v", got)
	}
	if got.RegisterToday != 2 {
		t.Fatalf("RegisterToday = %d, want 2", got.RegisterToday)
	}
}

func TestDashboardTrafficTotalsQueryUsesSingleProjectedStatement(t *testing.T) {
	normalized := strings.ToLower(dashboardTrafficTotalsSQL)
	if strings.Contains(normalized, "select *") {
		t.Fatal("dashboard traffic totals query must not select full rows")
	}
	for _, required := range []string{
		`from "daily_traffic_stat"`,
		"case when date = ?",
		"as today_upload",
		"as today_download",
		"as monthly_upload",
		"as monthly_download",
		"where date >= ? and date <= ?",
	} {
		if !strings.Contains(normalized, required) {
			t.Fatalf("traffic totals query missing %q: %s", required, dashboardTrafficTotalsSQL)
		}
	}
	if got := strings.Count(normalized, `from "daily_traffic_stat"`); got != 1 {
		t.Fatalf("traffic totals table scans = %d, want 1", got)
	}
}

func TestQueryDashboardTrafficTotalsAggregatesTodayAndMonth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&trafficModel.DailyTrafficStat{}); err != nil {
		t.Fatalf("migrate daily traffic stat: %v", err)
	}
	if err := db.Create([]trafficModel.DailyTrafficStat{
		{Date: "2026-07-01", Upload: 10, Download: 20},
		{Date: "2026-07-06", Upload: 100, Download: 200},
		{Date: "2026-06-30", Upload: 1_000, Download: 2_000},
	}).Error; err != nil {
		t.Fatalf("seed daily traffic stats: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/dashboard/realtime", nil)

	got := queryDashboardTrafficTotalsForDates(c, &svc.ServiceContext{DB: db}, "2026-07-06", "2026-07-01")
	if got.TodayUpload != 100 || got.TodayDownload != 200 {
		t.Fatalf("today traffic = upload %d download %d, want 100/200", got.TodayUpload, got.TodayDownload)
	}
	if got.MonthlyUpload != 110 || got.MonthlyDownload != 220 {
		t.Fatalf("monthly traffic = upload %d download %d, want 110/220", got.MonthlyUpload, got.MonthlyDownload)
	}
}

func TestCompactValuePointsKeepsLastPointPerBucket(t *testing.T) {
	points := []dashboardRealtimeValuePoint{
		{Timestamp: 0, Value: 1},
		{Timestamp: 10_000, Value: 2},
		{Timestamp: 60_000, Value: 3},
		{Timestamp: 70_000, Value: 4},
		{Timestamp: 120_000, Value: 5},
	}
	want := []dashboardRealtimeValuePoint{
		{Timestamp: 10_000, Value: 2},
		{Timestamp: 70_000, Value: 4},
		{Timestamp: 120_000, Value: 5},
	}
	if got := compactValuePoints(points, time.Minute); !reflect.DeepEqual(got, want) {
		t.Fatalf("compactValuePoints() = %+v, want %+v", got, want)
	} else if &got[0] != &points[0] {
		t.Fatal("compactValuePoints should compact into the input slice")
	}
}

func TestCompactTrafficPointsKeepsLastPointPerBucket(t *testing.T) {
	points := []dashboardRealtimeTrafficPoint{
		{Timestamp: 0, Upload: 1, Download: 2, Total: 3},
		{Timestamp: 30_000, Upload: 4, Download: 5, Total: 9},
		{Timestamp: 60_000, Upload: 6, Download: 7, Total: 13},
	}
	want := []dashboardRealtimeTrafficPoint{
		{Timestamp: 30_000, Upload: 4, Download: 5, Total: 9},
		{Timestamp: 60_000, Upload: 6, Download: 7, Total: 13},
	}
	if got := compactTrafficPoints(points, time.Minute); !reflect.DeepEqual(got, want) {
		t.Fatalf("compactTrafficPoints() = %+v, want %+v", got, want)
	} else if &got[0] != &points[0] {
		t.Fatal("compactTrafficPoints should compact into the input slice")
	}
}

func TestCompactNetworkPointsKeepsLastPointPerBucket(t *testing.T) {
	points := []dashboardRealtimeNetworkPoint{
		{Timestamp: 0, SystemRxBps: 1},
		{Timestamp: 15_000, SystemRxBps: 2},
		{Timestamp: 60_000, SystemRxBps: 3},
	}
	want := []dashboardRealtimeNetworkPoint{
		{Timestamp: 15_000, SystemRxBps: 2},
		{Timestamp: 60_000, SystemRxBps: 3},
	}
	if got := compactNetworkPoints(points, time.Minute); !reflect.DeepEqual(got, want) {
		t.Fatalf("compactNetworkPoints() = %+v, want %+v", got, want)
	} else if &got[0] != &points[0] {
		t.Fatal("compactNetworkPoints should compact into the input slice")
	}
}

func TestFilterValuePointsSinceReturnsTailView(t *testing.T) {
	points := []dashboardRealtimeValuePoint{
		{Timestamp: 1_000, Value: 1},
		{Timestamp: 2_000, Value: 2},
		{Timestamp: 3_000, Value: 3},
	}

	got := filterValuePointsSince(points, time.UnixMilli(2_500))
	want := []dashboardRealtimeValuePoint{{Timestamp: 3_000, Value: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterValuePointsSince() = %+v, want %+v", got, want)
	}
	if &got[0] != &points[2] {
		t.Fatal("filterValuePointsSince should return a subslice of the input")
	}
}

func TestBuildDashboardOnlineUserSeriesKeepsSeriesIndependent(t *testing.T) {
	now := time.UnixMilli((48 * time.Hour).Milliseconds())
	points := []dashboardRealtimeValuePoint{
		{Timestamp: 0, Value: 1},
		{Timestamp: (24 * time.Hour).Milliseconds(), Value: 2},
		{Timestamp: (24*time.Hour + 5*time.Minute).Milliseconds(), Value: 3},
		{Timestamp: (25 * time.Hour).Milliseconds(), Value: 4},
		{Timestamp: (25*time.Hour + 5*time.Minute).Milliseconds(), Value: 5},
	}

	got := buildDashboardOnlineUserSeries(points, now)
	wantLastDay := []dashboardRealtimeValuePoint{
		{Timestamp: (24 * time.Hour).Milliseconds(), Value: 2},
		{Timestamp: (24*time.Hour + 5*time.Minute).Milliseconds(), Value: 3},
		{Timestamp: (25 * time.Hour).Milliseconds(), Value: 4},
		{Timestamp: (25*time.Hour + 5*time.Minute).Milliseconds(), Value: 5},
	}
	wantLast7Days := []dashboardRealtimeValuePoint{
		{Timestamp: 0, Value: 1},
		{Timestamp: (24*time.Hour + 5*time.Minute).Milliseconds(), Value: 3},
		{Timestamp: (25*time.Hour + 5*time.Minute).Milliseconds(), Value: 5},
	}

	if !reflect.DeepEqual(got.LastDay, wantLastDay) {
		t.Fatalf("LastDay = %+v, want %+v", got.LastDay, wantLastDay)
	}
	if !reflect.DeepEqual(got.Last7Days, wantLast7Days) {
		t.Fatalf("Last7Days = %+v, want %+v", got.Last7Days, wantLast7Days)
	}
	if &got.LastDay[0] == &got.Last7Days[0] {
		t.Fatal("LastDay and Last7Days should not share the same compacted prefix")
	}
}

func TestBuildDashboardOnlineUserSeriesKeepsEmptyLastDayAsEmptySlice(t *testing.T) {
	now := time.UnixMilli((48 * time.Hour).Milliseconds())
	points := []dashboardRealtimeValuePoint{
		{Timestamp: 0, Value: 1},
		{Timestamp: time.Hour.Milliseconds(), Value: 2},
	}

	got := buildDashboardOnlineUserSeries(points, now)
	if got.LastDay == nil {
		t.Fatal("LastDay should remain an empty slice when historical points exist outside the last day")
	}
	if len(got.LastDay) != 0 {
		t.Fatalf("LastDay len = %d, want 0", len(got.LastDay))
	}
}

func TestBuildDashboardOnlineUserSeriesKeepsEmptyInputAsEmptySlices(t *testing.T) {
	got := buildDashboardOnlineUserSeries(nil, time.UnixMilli((48 * time.Hour).Milliseconds()))
	if got.LastDay == nil || len(got.LastDay) != 0 {
		t.Fatalf("LastDay = %#v, want non-nil empty slice", got.LastDay)
	}
	if got.Last7Days == nil || len(got.Last7Days) != 0 {
		t.Fatalf("Last7Days = %#v, want non-nil empty slice", got.Last7Days)
	}
}

func TestQueryDashboardTrafficKeepsEmptyRankingsAsEmptySlices(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&trafficModel.DailyTrafficStat{}, &trafficModel.DailyServerTraffic{}, &trafficModel.DailyUserTraffic{}); err != nil {
		t.Fatalf("migrate traffic tables: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/dashboard/realtime", nil)

	got := queryDashboardTraffic(c, &svc.ServiceContext{
		DB:              db,
		TrafficLogModel: trafficModel.NewModel(db),
	})
	if got.TodaySeries == nil {
		t.Fatal("TodaySeries should be a non-nil empty slice")
	}
	if got.ServerRankingToday == nil || got.UserRankingToday == nil || got.ServerRankingMonthly == nil || got.UserRankingMonthly == nil {
		t.Fatalf("traffic rankings should be non-nil empty slices: %#v", got)
	}
}

func TestDashboardPointCompactionDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		valuePoints := []dashboardRealtimeValuePoint{
			{Timestamp: 0, Value: 1},
			{Timestamp: 10_000, Value: 2},
			{Timestamp: 60_000, Value: 3},
		}
		trafficPoints := []dashboardRealtimeTrafficPoint{
			{Timestamp: 0, Upload: 1},
			{Timestamp: 10_000, Upload: 2},
			{Timestamp: 60_000, Upload: 3},
		}
		networkPoints := []dashboardRealtimeNetworkPoint{
			{Timestamp: 0, SystemRxBps: 1},
			{Timestamp: 10_000, SystemRxBps: 2},
			{Timestamp: 60_000, SystemRxBps: 3},
		}

		_ = compactValuePoints(valuePoints, time.Minute)
		_ = compactTrafficPoints(trafficPoints, time.Minute)
		_ = compactNetworkPoints(networkPoints, time.Minute)
		_ = filterValuePointsSince(valuePoints, time.UnixMilli(30_000))
	})
	if allocs != 0 {
		t.Fatalf("dashboard point compaction allocations = %f, want 0", allocs)
	}
}

func TestDashboardRedisInt64ParsesNativeRedisValues(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int64
		ok   bool
	}{
		{name: "string", in: "42", want: 42, ok: true},
		{name: "bytes", in: []byte("43"), want: 43, ok: true},
		{name: "int64", in: int64(44), want: 44, ok: true},
		{name: "uint", in: uint(45), want: 45, ok: true},
		{name: "blank", in: "  ", ok: false},
		{name: "json", in: `{"value":1}`, ok: false},
		{name: "unsupported", in: struct{}{}, ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := dashboardRedisInt64(tc.in)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("dashboardRedisInt64(%#v) = %d/%v, want %d/%v", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestDashboardRedisInt64ParsesByteValuesWithoutAllocation(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want int64
		ok   bool
	}{
		{name: "positive", in: []byte("  +43 "), want: 43, ok: true},
		{name: "negative", in: []byte("\n-44\t"), want: -44, ok: true},
		{name: "min int64", in: []byte("-9223372036854775808"), want: -9223372036854775808, ok: true},
		{name: "max int64", in: []byte("9223372036854775807"), want: 9223372036854775807, ok: true},
		{name: "positive overflow", in: []byte("9223372036854775808"), ok: false},
		{name: "negative overflow", in: []byte("-9223372036854775809"), ok: false},
		{name: "sign only", in: []byte("-"), ok: false},
		{name: "invalid", in: []byte("42x"), ok: false},
		{name: "blank", in: []byte(" \n\t "), ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := dashboardRedisInt64(tc.in)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("dashboardRedisInt64(%q) = %d/%v, want %d/%v", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}

	allocs := testing.AllocsPerRun(1000, func() {
		got, ok := dashboardRedisInt64([]byte(" 123456 "))
		if !ok || got != 123456 {
			t.Fatalf("dashboardRedisInt64 bytes = %d/%v, want 123456/true", got, ok)
		}
	})
	if allocs != 0 {
		t.Fatalf("dashboardRedisInt64 byte parsing allocations = %f, want 0", allocs)
	}
}

func TestDashboardRedisJSONParsesStringAndBytesOnly(t *testing.T) {
	for _, value := range []any{
		`{"upload":10,"download":20,"total":30}`,
		[]byte(`{"upload":10,"download":20,"total":30}`),
	} {
		var point dashboardRealtimeTrafficPoint
		if !dashboardRedisJSON(value, &point) {
			t.Fatalf("dashboardRedisJSON(%T) failed", value)
		}
		if point.Upload != 10 || point.Download != 20 || point.Total != 30 {
			t.Fatalf("decoded traffic point = %+v", point)
		}
	}

	var point dashboardRealtimeTrafficPoint
	if dashboardRedisJSON(12, &point) {
		t.Fatal("dashboardRedisJSON should reject numeric values")
	}
	if dashboardRedisJSON("{bad", &point) {
		t.Fatal("dashboardRedisJSON should reject malformed JSON")
	}
}

func TestUniqueDashboardInt64sFiltersAndDeduplicatesInPlace(t *testing.T) {
	values := []int64{3, 0, 2, 3, -1, 4, 2}
	got := uniqueDashboardInt64s(values)
	want := []int64{3, 2, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueDashboardInt64s() = %#v, want %#v", got, want)
	}
	if &got[0] != &values[0] {
		t.Fatal("uniqueDashboardInt64s should compact into the input slice")
	}
}

func TestUniqueDashboardInt64sSmallInputDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		values := []int64{1, 2, 2, 3, 0, 4}
		_ = uniqueDashboardInt64s(values)
	})
	if allocs != 0 {
		t.Fatalf("uniqueDashboardInt64s small input allocations = %f, want 0", allocs)
	}
}

func TestDashboardSystemActivityUserIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]dashboardSystemActivityLog, viewutil.SmallRowsCapacityThreshold+20)
	rows[2] = dashboardSystemActivityLog{UserID: 7}

	got := dashboardSystemActivityUserIDs(rows)
	want := []int64{7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dashboardSystemActivityUserIDs() = %#v, want %#v", got, want)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("dashboardSystemActivityUserIDs cap = %d, want small initial cap", cap(got))
	}
}

func TestDashboardSubscribeActivityTargetsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]log.SubscribeAccessLog, viewutil.SmallRowsCapacityThreshold+20)
	rows[1] = log.SubscribeAccessLog{UserId: 7, UserSubscribeId: 10}

	userIDs, subscribeIDs := dashboardSubscribeActivityTargets(rows)
	if want := []int64{7}; !reflect.DeepEqual(userIDs, want) {
		t.Fatalf("user ids = %#v, want %#v", userIDs, want)
	}
	if want := []int64{10}; !reflect.DeepEqual(subscribeIDs, want) {
		t.Fatalf("subscribe ids = %#v, want %#v", subscribeIDs, want)
	}
	if cap(userIDs) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("user ids cap = %d, want small initial cap", cap(userIDs))
	}
	if cap(subscribeIDs) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe ids cap = %d, want small initial cap", cap(subscribeIDs))
	}
}

func assertDashboardColumns(t *testing.T, columns, required, forbidden []string) {
	t.Helper()
	columnSet := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		columnSet[column] = struct{}{}
	}
	for _, column := range required {
		if _, ok := columnSet[column]; !ok {
			t.Fatalf("dashboard select columns must include %q", column)
		}
	}
	for _, column := range forbidden {
		if _, ok := columnSet[column]; ok {
			t.Fatalf("dashboard select columns must not include %q", column)
		}
	}
}
