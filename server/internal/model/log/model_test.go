package log

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSystemLogListColumnsStayExplicit(t *testing.T) {
	required := []string{
		"id",
		"type",
		"date",
		"object_id",
		"ip",
		"country_short",
		"region",
		"location",
		"method",
		"identifier",
		"user_agent",
		"success",
		"event_at",
		"content",
		"created_at",
	}
	if !reflect.DeepEqual(systemLogListColumns, required) {
		t.Fatalf("system log list columns = %#v, want %#v", systemLogListColumns, required)
	}
}

func TestBalanceLogSelectColumnsStayProjected(t *testing.T) {
	assertProjectedLogColumns(t, balanceLogSelectColumns, []string{
		"object_id AS user_id",
		"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
		"COALESCE(NULLIF(content::jsonb->>'amount', ''), '0')::bigint AS amount",
		"COALESCE(content::jsonb->>'order_no', '') AS order_no",
		"COALESCE(NULLIF(content::jsonb->>'balance', ''), '0')::bigint AS balance",
		"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
	})
}

func TestCommissionLogSelectColumnsStayProjected(t *testing.T) {
	assertProjectedLogColumns(t, commissionLogSelectColumns, []string{
		"object_id AS user_id",
		"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
		"COALESCE(NULLIF(content::jsonb->>'amount', ''), '0')::bigint AS amount",
		"COALESCE(content::jsonb->>'order_no', '') AS order_no",
		"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
	})
}

func TestLoginLogSelectColumnsStayProjected(t *testing.T) {
	assertProjectedLogColumns(t, loginLogSelectColumns, []string{
		"id",
		"object_id AS user_id",
		"country_short",
		"region",
		"method",
		"ip AS login_ip",
		"user_agent",
		"success",
		"created_at",
	})
	assertColumnsAvoidJSONContent(t, loginLogSelectColumns)
}

func TestRegisterLogSelectColumnsStayProjected(t *testing.T) {
	assertProjectedLogColumns(t, registerLogSelectColumns, []string{
		"object_id AS user_id",
		"country_short",
		"region",
		"method AS auth_method",
		"identifier",
		"ip AS register_ip",
		"user_agent",
		"event_at AS timestamp",
	})
	assertColumnsAvoidJSONContent(t, registerLogSelectColumns)
}

func TestMessageLogSelectColumnsStayProjected(t *testing.T) {
	assertProjectedLogColumns(t, messageLogSelectColumns, []string{
		"id",
		"type",
		"COALESCE(content::jsonb->>'platform', '') AS platform",
		"COALESCE(content::jsonb->>'to', '') AS to",
		"COALESCE(content::jsonb->>'subject', '') AS subject",
		"(content::jsonb->'content')::text AS content",
		"COALESCE(NULLIF(content::jsonb->>'status', ''), '0')::int AS status",
		"created_at",
	})
}

func TestResetSubscribeLogSelectColumnsStayProjected(t *testing.T) {
	assertProjectedLogColumns(t, resetSubscribeLogSelectColumns, []string{
		"id",
		"object_id AS user_subscribe_id",
		"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
		"COALESCE(NULLIF(content::jsonb->>'user_id', ''), '0')::bigint AS user_id",
		"COALESCE(content::jsonb->>'order_no', '') AS order_no",
		"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
	})
}

func TestGiftLogSelectColumnsStayProjected(t *testing.T) {
	assertProjectedLogColumns(t, giftLogSelectColumns, []string{
		"object_id AS user_id",
		"COALESCE(NULLIF(content::jsonb->>'type', ''), '0')::int AS type",
		"COALESCE(content::jsonb->>'order_no', '') AS order_no",
		"COALESCE(NULLIF(content::jsonb->>'subscribe_id', ''), '0')::bigint AS subscribe_id",
		"COALESCE(NULLIF(content::jsonb->>'amount', ''), '0')::bigint AS amount",
		"COALESCE(NULLIF(content::jsonb->>'balance', ''), '0')::bigint AS balance",
		"COALESCE(content::jsonb->>'remark', '') AS remark",
		"COALESCE(NULLIF(content::jsonb->>'timestamp', ''), '0')::bigint AS timestamp",
	})
}

func TestProjectedLogFiltersForceExpectedType(t *testing.T) {
	filter := &FilterParams{Type: TypeBalance.Uint8(), Page: 2, Size: 20, ObjectID: 9}

	login := filterWithType(filter, TypeLogin)
	if login.Type != TypeLogin.Uint8() || login.Page != 2 || login.Size != 20 || login.ObjectID != 9 {
		t.Fatalf("login filter = %#v", login)
	}
	if filter.Type != TypeBalance.Uint8() {
		t.Fatalf("filterWithType mutated source filter: %#v", filter)
	}

	register := filterWithType(nil, TypeRegister)
	if register.Type != TypeRegister.Uint8() {
		t.Fatalf("register filter type = %d, want %d", register.Type, TypeRegister.Uint8())
	}

	gift := filterWithType(filter, TypeGift)
	if gift.Type != TypeGift.Uint8() || gift.Page != 2 || gift.Size != 20 || gift.ObjectID != 9 {
		t.Fatalf("gift filter = %#v", gift)
	}

	resetSubscribe := filterWithType(filter, TypeResetSubscribe)
	if resetSubscribe.Type != TypeResetSubscribe.Uint8() || resetSubscribe.Page != 2 || resetSubscribe.Size != 20 || resetSubscribe.ObjectID != 9 {
		t.Fatalf("reset subscribe filter = %#v", resetSubscribe)
	}
}

func TestNormalizeSystemLogFilterUsesSharedPaginationRules(t *testing.T) {
	filter := &FilterParams{Page: -1, Size: 200, MaxSize: 60, Type: TypeBalance.Uint8()}
	got := normalizeSystemLogFilter(filter)

	if got != filter {
		t.Fatal("normalizeSystemLogFilter should update the supplied filter")
	}
	if filter.Page != 1 || filter.Size != 60 || filter.Type != TypeBalance.Uint8() {
		t.Fatalf("filter = %#v, want page 1 size 60 with type preserved", filter)
	}

	empty := normalizeSystemLogFilter(nil)
	if empty == nil || empty.Page != 1 || empty.Size != 10 {
		t.Fatalf("nil filter normalized to %#v, want default page 1 size 10", empty)
	}
}

func TestNormalizeSubscribeAccessLogFilterUsesSharedPaginationRules(t *testing.T) {
	filter := &SubscribeAccessFilterParams{Page: -1, Size: 0, UserId: 7, UserSubscribeId: 11}
	got := normalizeSubscribeAccessLogFilter(filter)

	if got != filter {
		t.Fatal("normalizeSubscribeAccessLogFilter should update the supplied filter")
	}
	if filter.Page != 1 || filter.Size != 10 || filter.UserId != 7 || filter.UserSubscribeId != 11 {
		t.Fatalf("filter = %#v, want default pagination with IDs preserved", filter)
	}
}

func TestPrepareSubscribeAccessLogQueryFilterParsesDateRangeOnce(t *testing.T) {
	filter := &SubscribeAccessFilterParams{Page: -1, Size: 0, Date: "2026-07-08", UserId: 7}
	got, err := prepareSubscribeAccessLogQueryFilter(filter)
	if err != nil {
		t.Fatal(err)
	}
	if got.params != filter {
		t.Fatal("prepared subscribe access filter should retain normalized source filter")
	}
	if filter.Page != 1 || filter.Size != 10 || got.params.UserId != 7 {
		t.Fatalf("normalized params = %#v", got.params)
	}
	if !got.hasDate {
		t.Fatal("prepared subscribe access filter missing date range")
	}
	if got.dateUntil.Sub(got.dateFrom) != 24*time.Hour {
		t.Fatalf("date range = %s, want 24h", got.dateUntil.Sub(got.dateFrom))
	}
	if got.dateFrom.Format(time.DateOnly) != "2026-07-08" {
		t.Fatalf("date start = %s, want 2026-07-08", got.dateFrom.Format(time.DateOnly))
	}
}

func TestFilterLoginLogsPreallocatesCurrentPageRows(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SystemLog{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	rows := []SystemLog{
		{Id: 1, Type: TypeLogin.Uint8(), ObjectID: 7, IP: "203.0.113.1", Method: "email", UserAgent: "agent-a", Success: true, CreatedAt: now},
		{Id: 2, Type: TypeLogin.Uint8(), ObjectID: 8, IP: "203.0.113.2", Method: "mobile", UserAgent: "agent-b", Success: false, CreatedAt: now.Add(time.Minute)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	list, total, err := NewModel(db).FilterLoginLogs(ctx, &FilterParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("login logs total/len = %d/%d, want 2/2", total, len(list))
	}
	if cap(list) != 2 {
		t.Fatalf("login logs capacity = %d, want 2", cap(list))
	}
	if list[0].UserId != 8 || list[1].UserId != 7 {
		t.Fatalf("login logs order = %#v, want id desc", list)
	}
}

func TestFilterSystemLogPreallocatesCurrentPageRows(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SystemLog{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	rows := []SystemLog{
		{Id: 1, Type: TypeBalance.Uint8(), Date: "2023-11-14", ObjectID: 7, IP: "203.0.113.1", CountryShort: "US", Region: "CA", Location: "US · CA", Method: "email", Identifier: "user@example.com", UserAgent: "agent-a", Success: true, EventAt: 1700000000000, Content: `{"amount":100}`, CreatedAt: now},
		{Id: 2, Type: TypeCommission.Uint8(), Date: "2023-11-15", ObjectID: 8, IP: "203.0.113.2", CountryShort: "JP", Region: "13", Location: "JP · 13", Method: "mobile", Identifier: "+8613800000000", UserAgent: "agent-b", Success: false, EventAt: 1700000060000, Content: `{"amount":200}`, CreatedAt: now.Add(time.Minute)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	list, total, err := NewModel(db).FilterSystemLog(ctx, &FilterParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("system logs total/len = %d/%d, want 2/2", total, len(list))
	}
	if cap(list) != 2 {
		t.Fatalf("system logs capacity = %d, want 2", cap(list))
	}
	if list[0].Id != 2 || list[0].Content != `{"amount":200}` || list[0].Location != "JP · 13" || list[0].Identifier != "+861...0000" || list[0].IP != "203.0.113.*" || list[0].EventAt != 1700000060000 {
		t.Fatalf("system logs projected fields missing: %#v", list[0])
	}

	empty, emptyTotal, err := NewModel(db).FilterSystemLog(ctx, &FilterParams{Type: 99, Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if emptyTotal != 0 || empty == nil || len(empty) != 0 || cap(empty) != 0 {
		t.Fatalf("empty system logs = total %d list %#v cap %d, want non-nil empty", emptyTotal, empty, cap(empty))
	}
}

func TestSubscribeAccessLogListColumnsStayLightweight(t *testing.T) {
	assertProjectedLogColumns(t, subscribeAccessLogListColumns, []string{
		"id",
		"user_id",
		"user_subscribe_id",
		"token_prefix",
		"user_agent",
		"client_ip",
		"country_short",
		"region",
		"location",
		"created_at",
	})

	forbidden := []string{"token_hash", "success"}
	columnSet := make(map[string]struct{}, len(subscribeAccessLogListColumns))
	for _, column := range subscribeAccessLogListColumns {
		columnSet[column] = struct{}{}
	}
	for _, column := range forbidden {
		if _, ok := columnSet[column]; ok {
			t.Fatalf("subscribe access log list columns must not include %q", column)
		}
	}
}

func TestFilterSubscribeAccessLogSelectsOnlyListColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&SubscribeAccessLog{}); err != nil {
		t.Fatal(err)
	}

	createdAt := time.Unix(1700000000, 0)
	rows := []SubscribeAccessLog{
		{
			Id:              1,
			UserId:          7,
			UserSubscribeId: 11,
			TokenHash:       "private-token-hash",
			TokenPrefix:     "token-pre",
			UserAgent:       "Clash",
			ClientIP:        "203.0.113.1",
			CountryShort:    "US",
			Region:          "CA",
			Location:        "US · CA",
			Success:         true,
			CreatedAt:       createdAt,
		},
		{
			Id:              2,
			UserId:          7,
			UserSubscribeId: 12,
			TokenHash:       "other-token-hash",
			TokenPrefix:     "other",
			UserAgent:       "SingBox",
			ClientIP:        "203.0.113.2",
			Success:         true,
			CreatedAt:       createdAt.Add(time.Minute),
		},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db)
	list, total, err := model.FilterSubscribeAccessLog(ctx, &SubscribeAccessFilterParams{
		Page:            1,
		Size:            10,
		UserId:          7,
		UserSubscribeId: 11,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(list) != 1 {
		t.Fatalf("list length = %d, want 1", len(list))
	}
	if cap(list) != 1 {
		t.Fatalf("subscribe access log capacity = %d, want 1", cap(list))
	}
	item := list[0]
	if item.Id != 1 || item.UserId != 7 || item.UserSubscribeId != 11 || item.TokenPrefix != "token-pre" {
		t.Fatalf("projected subscribe access log missing response fields: %#v", item)
	}
	if item.UserAgent != "Clash" || item.ClientIP != "203.0.113.*" || item.Location != "US · CA" {
		t.Fatalf("projected subscribe access log missing client fields: %#v", item)
	}
	if item.TokenHash != "" {
		t.Fatalf("subscribe access log list loaded token hash: %q", item.TokenHash)
	}
	if item.Success {
		t.Fatal("subscribe access log list loaded success flag unexpectedly")
	}
}

func assertProjectedLogColumns(t *testing.T, columns, required []string) {
	t.Helper()
	columnSet := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		columnSet[column] = struct{}{}
	}
	for _, column := range required {
		if _, ok := columnSet[column]; !ok {
			t.Fatalf("projected log columns must include %q", column)
		}
	}
	for _, column := range columns {
		if column == "content" || column == `"content"` || column == "system_logs.content" {
			t.Fatal("projected log list columns must not include full content")
		}
	}
}

func assertColumnsAvoidJSONContent(t *testing.T, columns []string) {
	t.Helper()
	for _, column := range columns {
		if strings.Contains(column, "content::jsonb") {
			t.Fatalf("hot log list columns must not parse content JSON: %s", column)
		}
	}
}
