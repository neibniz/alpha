package user

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	userModel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"gorm.io/gorm"
)

func TestPublicUserFromModelMasksAndSortsAuthMethods(t *testing.T) {
	trueValue := true
	falseValue := false
	now := time.UnixMilli(1700000000000)
	expiresAt := now.Add(time.Hour)
	createdAt := now.Add(-2 * time.Hour)
	updatedAt := now.Add(-time.Hour)
	deletedAt := now.Add(-30 * time.Minute)
	item := &userModel.User{
		Id:                    7,
		Avatar:                "avatar",
		Balance:               100,
		ReferCode:             "REF",
		RefererId:             3,
		Commission:            20,
		ReferralPercentage:    15,
		OnlyFirstPurchase:     &trueValue,
		GiftAmount:            30,
		MemberExpiredAt:       expiresAt,
		Enable:                &trueValue,
		IsAdmin:               &falseValue,
		EnableBalanceNotify:   &trueValue,
		EnableLoginNotify:     &falseValue,
		EnableSubscribeNotify: &trueValue,
		EnableTradeNotify:     &falseValue,
		Rules:                 `[" block ", "", "direct", "block"]`,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
		DeletedAt:             gorm.DeletedAt{Time: deletedAt, Valid: true},
		AuthMethods: []userModel.AuthMethods{
			{AuthType: "github", AuthIdentifier: "github-open-id-123456", Verified: true},
			{AuthType: "mobile", AuthIdentifier: "+8613800138000", Verified: true},
			{AuthType: "device", AuthIdentifier: "device-id", Verified: true},
			{AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true},
		},
	}

	got := publicUserFromModel(item, now)
	if got.Id != item.Id || got.Balance != item.Balance || got.ReferCode != item.ReferCode || got.MemberExpiredAt != expiresAt.UnixMilli() || !got.IsMember {
		t.Fatalf("public user fields not mapped: %#v", got)
	}
	if !got.OnlyFirstPurchase || !got.Enable || got.IsAdmin || !got.EnableBalanceNotify || got.EnableLoginNotify || !got.EnableSubscribeNotify || got.EnableTradeNotify {
		t.Fatalf("boolean fields not mapped: %#v", got)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() || got.DeletedAt != deletedAt.UnixMilli() {
		t.Fatalf("timestamps not mapped: %#v", got)
	}
	if !reflect.DeepEqual(got.Rules, []string{"block", "direct"}) {
		t.Fatalf("rules not normalized: %#v", got.Rules)
	}
	if len(got.AuthMethods) != 2 {
		t.Fatalf("auth methods = %d, want 2", len(got.AuthMethods))
	}
	if got.AuthMethods[0].AuthType != "email" || got.AuthMethods[1].AuthType != "github" {
		t.Fatalf("auth methods not sorted by priority: %#v", got.AuthMethods)
	}
	if got.AuthMethods[0].AuthIdentifier != "alpha@example.com" {
		t.Fatalf("email should stay visible: %#v", got.AuthMethods[0])
	}
	if got.AuthMethods[1].AuthIdentifier != "git***************456" {
		t.Fatalf("open id not masked as expected: %#v", got.AuthMethods[1])
	}
}

func TestPublicUserFromModelKeepsZeroTimesAtZero(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	got := publicUserFromModel(&userModel.User{Id: 8}, now)
	if got.CreatedAt != 0 || got.UpdatedAt != 0 || got.DeletedAt != 0 {
		t.Fatalf("zero user timestamps should stay zero: %#v", got)
	}
	if got.AuthMethods == nil || len(got.AuthMethods) != 0 {
		t.Fatalf("empty auth methods should stay a non-nil empty slice: %#v", got.AuthMethods)
	}
	if got.Rules == nil || len(got.Rules) != 0 {
		t.Fatalf("empty rules should stay a non-nil empty slice: %#v", got.Rules)
	}
}

func TestPublicAuthMethodsHelpersKeepEmptyArrayShape(t *testing.T) {
	if got := publicAuthMethodsFromPointers(nil, true); got == nil || len(got) != 0 {
		t.Fatalf("empty pointer auth methods should be a non-nil empty slice: %#v", got)
	}
	if got := publicAuthMethodsFromValues(nil, true); got == nil || len(got) != 0 {
		t.Fatalf("empty value auth methods should be a non-nil empty slice: %#v", got)
	}
}

func TestPublicAuthMethodsFromPointersCanKeepDatabaseOrder(t *testing.T) {
	methods := []*userModel.AuthMethods{
		{AuthType: "github", AuthIdentifier: "github-open-id-123456", Verified: true},
		{AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true},
	}

	got := publicAuthMethodsFromPointers(methods, false)
	if len(got) != 2 {
		t.Fatalf("auth methods = %d, want 2", len(got))
	}
	if got[0].AuthType != "github" || got[1].AuthType != "email" {
		t.Fatalf("auth methods order changed: %#v", got)
	}
	if got[0].AuthIdentifier != "git***************456" {
		t.Fatalf("open id not masked: %#v", got[0])
	}
}

func TestPublicAuthMethodsDeduplicatesByAuthType(t *testing.T) {
	pointers := []*userModel.AuthMethods{
		nil,
		{AuthType: "", AuthIdentifier: "ignored", Verified: true},
		{AuthType: "email", AuthIdentifier: "first@example.com", Verified: true},
		{AuthType: "github", AuthIdentifier: "github-open-id-123456", Verified: true},
		{AuthType: "email", AuthIdentifier: "second@example.com", Verified: true},
	}

	got := publicAuthMethodsFromPointers(pointers, false)
	if len(got) != 2 {
		t.Fatalf("deduped pointer auth methods = %d, want 2: %#v", len(got), got)
	}
	if got[0].AuthType != "email" || got[0].AuthIdentifier != "first@example.com" {
		t.Fatalf("first email auth method should be preserved: %#v", got)
	}
	if got[1].AuthType != "github" {
		t.Fatalf("github auth method should remain in database order: %#v", got)
	}
}

func TestPublicAuthMethodsDeduplicatesLargeInputsWithLookup(t *testing.T) {
	methods := make([]userModel.AuthMethods, 0, smallPublicAuthMethodLookupThreshold+3)
	methods = append(methods,
		userModel.AuthMethods{AuthType: "github", AuthIdentifier: "github-open-id-123456", Verified: true},
		userModel.AuthMethods{AuthType: "email", AuthIdentifier: "first@example.com", Verified: true},
	)
	for index := 0; index <= smallPublicAuthMethodLookupThreshold; index++ {
		methods = append(methods, userModel.AuthMethods{
			AuthType:       "provider-" + string(rune('a'+index)),
			AuthIdentifier: "opaque-" + string(rune('a'+index)),
			Verified:       true,
		})
	}
	methods = append(methods,
		userModel.AuthMethods{AuthType: "email", AuthIdentifier: "second@example.com", Verified: true},
		userModel.AuthMethods{AuthType: "", AuthIdentifier: "ignored", Verified: true},
	)

	got := publicAuthMethodsFromValues(methods, true)
	if len(got) != smallPublicAuthMethodLookupThreshold+3 {
		t.Fatalf("deduped large auth methods = %d, want %d: %#v", len(got), smallPublicAuthMethodLookupThreshold+3, got)
	}
	if got[0].AuthType != "email" || got[0].AuthIdentifier != "first@example.com" {
		t.Fatalf("large auth methods should keep first email after sorting: %#v", got)
	}
}

func TestPublicAuthMethodsLargeDuplicateInputAvoidsFullCapAndMap(t *testing.T) {
	methods := make([]userModel.AuthMethods, smallPublicAuthMethodLookupThreshold+20)
	for index := range methods {
		methods[index] = userModel.AuthMethods{AuthType: "email", AuthIdentifier: "first@example.com", Verified: true}
	}
	got := publicAuthMethodsFromValues(methods, false)
	if len(got) != 1 || got[0].AuthType != "email" {
		t.Fatalf("deduped duplicate auth methods = %#v, want one email", got)
	}
	if cap(got) != smallPublicAuthMethodLookupThreshold+1 {
		t.Fatalf("deduped duplicate auth methods cap = %d, want %d", cap(got), smallPublicAuthMethodLookupThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = publicAuthMethodsFromValues(methods, false)
	})
	if allocs > 1 {
		t.Fatalf("deduped duplicate auth methods allocations = %f, want at most 1", allocs)
	}
}

func TestPublicAuthMethodCollectorUpgradesOnlyAfterActualUniqueThreshold(t *testing.T) {
	collector := newPublicAuthMethodCollector(smallPublicAuthMethodLookupThreshold + 20)
	for index := 0; index < smallPublicAuthMethodLookupThreshold+20; index++ {
		collector.Add(types.UserAuthMethod{AuthType: "email", AuthIdentifier: "first@example.com"})
	}
	if collector.seen != nil {
		t.Fatal("duplicate-heavy public auth method collector should not allocate map")
	}
	if len(collector.methods) != 1 || cap(collector.methods) != smallPublicAuthMethodLookupThreshold+1 {
		t.Fatalf("collector len/cap = %d/%d, want 1/%d", len(collector.methods), cap(collector.methods), smallPublicAuthMethodLookupThreshold+1)
	}

	collector = newPublicAuthMethodCollector(smallPublicAuthMethodLookupThreshold + 20)
	for index := 0; index <= smallPublicAuthMethodLookupThreshold; index++ {
		collector.Add(types.UserAuthMethod{AuthType: "provider-" + string(rune('a'+index))})
	}
	if collector.seen == nil {
		t.Fatal("public auth method collector should allocate map after actual unique auth types exceed threshold")
	}
	if len(collector.methods) != smallPublicAuthMethodLookupThreshold+1 {
		t.Fatalf("collector len = %d, want %d", len(collector.methods), smallPublicAuthMethodLookupThreshold+1)
	}
	if maxCap := len(collector.methods) * 2; cap(collector.methods) > maxCap {
		t.Fatalf("collector cap = %d, want <= %d", cap(collector.methods), maxCap)
	}
}

func TestWithdrawalRowsToTypesMapsFieldsAndKeepsListShape(t *testing.T) {
	createdAt := time.UnixMilli(1700000000000)
	updatedAt := createdAt.Add(time.Minute)
	rows := []userModel.Withdrawal{
		{
			Id:        11,
			UserId:    7,
			Amount:    1200,
			Content:   "account",
			Status:    2,
			Reason:    "missing details",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
	}

	got := withdrawalRowsToTypes(rows)
	if len(got) != 1 {
		t.Fatalf("withdrawal rows = %d, want 1: %#v", len(got), got)
	}
	if cap(got) != len(rows) {
		t.Fatalf("withdrawal rows capacity = %d, want %d", cap(got), len(rows))
	}
	if got[0].Id != 11 || got[0].UserId != 7 || got[0].Amount != 1200 || got[0].Content != "account" || got[0].Status != 2 || got[0].Reason != "missing details" {
		t.Fatalf("withdrawal fields not mapped: %#v", got[0])
	}
	if got[0].CreatedAt != createdAt.UnixMilli() || got[0].UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("withdrawal times not mapped: %#v", got[0])
	}

	empty := withdrawalRowsToTypes(nil)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty withdrawal rows should return non-nil empty slice: %#v", empty)
	}
}

func TestCommissionWithdrawUserColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "commission"}
	if !reflect.DeepEqual(commissionWithdrawUserColumns, want) {
		t.Fatalf("commission withdraw user columns = %#v, want %#v", commissionWithdrawUserColumns, want)
	}
}

func TestUnsubscribeRefundUserColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "balance", "commission", "gift_amount"}
	if !reflect.DeepEqual(unsubscribeRefundUserColumns, want) {
		t.Fatalf("unsubscribe refund user columns = %#v, want %#v", unsubscribeRefundUserColumns, want)
	}
}

func TestUserSubscribeFromDetailsAppliesProductSnapshot(t *testing.T) {
	trueValue := true
	now := time.UnixMilli(1700000000000)
	finishedAt := now.Add(2 * time.Hour)
	snapshot := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:              ordersnapshot.KindSubscribe,
		ID:                88,
		Name:              "Snapshot Plan",
		Description:       "snapshot description",
		UnitPrice:         1299,
		UnitTime:          "month",
		Discount:          []types.SubscribeDiscount{{Quantity: 3, Discount: 0.85}},
		Replacement:       1999,
		Traffic:           4096,
		SpeedLimit:        50,
		DeviceLimit:       3,
		Quota:             2,
		Nodes:             []int64{7, 9},
		NodeTags:          []string{"edge", "home"},
		AllowRenewal:      true,
		AllowResetTraffic: true,
		ResetCycle:        2,
		RenewalReset:      true,
	})
	item := &userModel.SubscribeDetails{
		Id:              11,
		UserId:          7,
		OrderId:         19,
		SubscribeId:     22,
		ProductSnapshot: snapshot,
		StartTime:       now,
		ExpireTime:      now.Add(24 * time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         2048,
		Download:        100,
		Upload:          50,
		Token:           "subscribe-token-for-public",
		Status:          1,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now,
		Subscribe: &modelSubscribe.Subscribe{
			Id:             22,
			Name:           "Live Plan",
			Discount:       `[{"quantity":1,"discount":0.95}]`,
			Nodes:          "1,2",
			NodeTags:       "live",
			Show:           &trueValue,
			Sell:           &trueValue,
			AllowDeduction: &trueValue,
			CreatedAt:      now.Add(-2 * time.Hour),
			UpdatedAt:      now.Add(-time.Hour),
		},
	}

	got := userSubscribeFromDetails(item, false)
	if got.Id != item.Id {
		t.Fatalf("subscription id not mapped: %#v", got)
	}
	if got.ExpireTime != item.ExpireTime.UnixMilli() {
		t.Fatalf("subscription expire time not mapped: %#v", got)
	}
	if got.Short == "" || got.Short == item.Token {
		t.Fatalf("short token not generated: %#v", got.Short)
	}
	if got.ProductName != "Snapshot Plan" {
		t.Fatalf("snapshot product name not applied: %#v", got.ProductName)
	}
	if got.Subscribe.Id != 88 || got.Subscribe.Name != "Snapshot Plan" || got.Subscribe.UnitPrice != 1299 || got.Subscribe.UnitTime != "month" {
		t.Fatalf("snapshot product fields not applied: %#v", got.Subscribe)
	}
	if got.Subscribe.Replacement != 1999 || got.Subscribe.Traffic != 4096 || got.Subscribe.SpeedLimit != 50 || got.Subscribe.DeviceLimit != 3 {
		t.Fatalf("snapshot resource fields not applied: %#v", got.Subscribe)
	}
	if len(got.Subscribe.Discount) != 1 || got.Subscribe.Discount[0].Quantity != 3 || got.Subscribe.Discount[0].Discount != 0.85 {
		t.Fatalf("snapshot discount not applied: %#v", got.Subscribe.Discount)
	}
	if !got.Subscribe.AllowRenewal || !got.Subscribe.AllowResetTraffic || got.Subscribe.AllowDeduction || got.Subscribe.ShowOriginalPrice {
		t.Fatalf("snapshot policy fields not applied: %#v", got.Subscribe)
	}
}

func TestUserSubscribeRowsFromDetailsAtSkipsNilAndKeepsEmptySlice(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.Local)
	expireAt := time.Date(2026, 7, 10, 0, 0, 0, 0, time.Local)
	snapshot := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:       ordersnapshot.KindSubscribe,
		ID:         88,
		Name:       "Snapshot Plan",
		Traffic:    4096,
		ResetCycle: 2,
	})
	got := userSubscribeRowsFromDetailsAt([]*userModel.SubscribeDetails{
		nil,
		{
			Id:              7,
			ProductSnapshot: snapshot,
			ExpireTime:      expireAt,
			Subscribe:       &modelSubscribe.Subscribe{ResetCycle: 2},
		},
		{
			Id:              8,
			ProductSnapshot: snapshot,
			ExpireTime:      expireAt,
			Subscribe:       &modelSubscribe.Subscribe{ResetCycle: 2},
		},
	}, true, now)
	if len(got) != 2 {
		t.Fatalf("user subscribe rows len = %d, want 2", len(got))
	}
	if got[0].Id != 7 || got[0].ResetTime != expireAt.UnixMilli() || got[0].ProductName != "Snapshot Plan" {
		t.Fatalf("user subscribe row mismatch: %#v", got[0])
	}
	if got[1].Id != 8 || got[1].ProductName != "Snapshot Plan" || got[1].Subscribe.Id != 88 {
		t.Fatalf("duplicate snapshot row mismatch: %#v", got[1])
	}

	empty := userSubscribeRowsFromDetailsAt(nil, true, now)
	if empty == nil {
		t.Fatal("empty user subscribe rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.PublicUserSubscribe `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty user subscribe rows marshal = %s, want list []", payload)
	}
}

func TestUserSubscribeRowsFromDetailsAtSkipNilWithoutInputSizedCapacity(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.Local)
	rows := make([]*userModel.SubscribeDetails, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &userModel.SubscribeDetails{Id: 1}
	rows[len(rows)-1] = &userModel.SubscribeDetails{Id: 2}

	got := userSubscribeRowsFromDetailsAt(rows, true, now)
	if len(got) != 2 {
		t.Fatalf("user subscribe rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy user subscribe rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("user subscribe row order changed: %#v", got)
	}
}

func TestUserSubscribeRowsFromDetailsAtUseCapacityHintAfterActualThreshold(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.Local)
	rows := make([]*userModel.SubscribeDetails, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &userModel.SubscribeDetails{Id: int64(index + 1)}
	}

	got := userSubscribeRowsFromDetailsAt(rows, false, now)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("user subscribe rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("user subscribe rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("user subscribe row order changed: %#v", got)
	}
}

func TestUserSubscribeFromDetailsKeepsZeroTimesAtZero(t *testing.T) {
	got := userSubscribeFromDetails(&userModel.SubscribeDetails{
		Subscribe: &modelSubscribe.Subscribe{},
	}, false)
	if got.ExpireTime != 0 {
		t.Fatalf("zero subscription times should stay zero: %#v", got)
	}
	if got.Subscribe.Id != 0 || got.Subscribe.Name != "" || got.Subscribe.UnitPrice != 0 || len(got.Subscribe.Discount) != 0 {
		t.Fatalf("zero product summary should stay zero-valued with an empty discount array: %#v", got.Subscribe)
	}
}

func TestUserSubscribeFromDetailsAtUsesSharedNowForResetTime(t *testing.T) {
	location := time.Local
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, location)
	expireAt := time.Date(2026, 7, 10, 0, 0, 0, 0, location)
	got := userSubscribeFromDetailsAt(&userModel.SubscribeDetails{
		ExpireTime: expireAt,
		Subscribe:  &modelSubscribe.Subscribe{ResetCycle: 2},
	}, true, now)

	want := time.Date(2026, 7, 10, 0, 0, 0, 0, location).UnixMilli()
	if got.ResetTime != want {
		t.Fatalf("reset time = %d, want %d", got.ResetTime, want)
	}
}
