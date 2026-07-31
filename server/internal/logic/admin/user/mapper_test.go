package user

import (
	"encoding/json"
	"testing"
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	userModel "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"gorm.io/gorm"
)

func TestUserListItemFromModelMapsListResponse(t *testing.T) {
	trueValue := true
	falseValue := false
	now := time.UnixMilli(1700000000000)
	createdAt := now.Add(-2 * time.Hour)
	updatedAt := now.Add(-time.Hour)
	deletedAt := now.Add(-30 * time.Minute)
	expiresAt := now.Add(time.Hour)
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
		Rules:                 `[" admin ", "", "direct", "admin"]`,
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
		DeletedAt:             gorm.DeletedAt{Time: deletedAt, Valid: true},
		AuthMethods: []userModel.AuthMethods{
			{AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true},
			{AuthType: "mobile", AuthIdentifier: "13800138000", Verified: false},
		},
	}

	got := userListItemFromModel(item, now)
	if got.Id != item.Id || got.Balance != item.Balance || got.ReferCode != item.ReferCode || got.RefererId != item.RefererId {
		t.Fatalf("basic fields not mapped: %#v", got)
	}
	if got.Commission != item.Commission || got.GiftAmount != item.GiftAmount {
		t.Fatalf("amount fields not mapped: %#v", got)
	}
	if !got.Enable {
		t.Fatalf("boolean fields not mapped: %#v", got)
	}
	if got.MemberExpiredAt != expiresAt.UnixMilli() || !got.IsMember {
		t.Fatalf("membership fields = %d/%v, want %d/true", got.MemberExpiredAt, got.IsMember, expiresAt.UnixMilli())
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.DeletedAt != deletedAt.UnixMilli() {
		t.Fatalf("timestamps = %d/%d", got.CreatedAt, got.DeletedAt)
	}
	if got.PrimaryAuthMethod.AuthType != "email" || got.PrimaryAuthMethod.AuthIdentifier != "alpha@example.com" || !got.PrimaryAuthMethod.Verified {
		t.Fatalf("primary auth method not mapped: %#v", got.PrimaryAuthMethod)
	}
}

func TestUserListItemFromModelHandlesNilBooleansAndExpiredMember(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	got := userListItemFromModel(&userModel.User{Id: 8, MemberExpiredAt: now.Add(-time.Hour)}, now)
	if got.Enable || got.IsMember {
		t.Fatalf("nil booleans and expired member should map to false: %#v", got)
	}
	if got.CreatedAt != 0 || got.DeletedAt != 0 {
		t.Fatalf("zero user timestamps should stay zero: %#v", got)
	}
	if got.PrimaryAuthMethod != (types.UserAuthMethod{}) {
		t.Fatalf("empty primary auth method should stay zero-value: %#v", got.PrimaryAuthMethod)
	}
}

func TestUserListRowsFromModelsSkipsNilAndKeepsEmptySlice(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	got := userListRowsFromModels([]*userModel.User{
		nil,
		{Id: 7, MemberExpiredAt: now.Add(time.Hour)},
		{Id: 8, MemberExpiredAt: now.Add(time.Hour)},
	}, now)
	if len(got) != 2 {
		t.Fatalf("user rows len = %d, want 2", len(got))
	}
	if got[0].Id != 7 || !got[0].IsMember {
		t.Fatalf("user row mismatch: %#v", got[0])
	}
	empty := userListRowsFromModels(nil, now)
	if empty == nil {
		t.Fatal("empty user rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.UserListItem `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty user rows marshal = %s, want list []", payload)
	}
}

func TestUserListRowsFromModelsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	rows := make([]*userModel.User, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &userModel.User{Id: 1}
	rows[len(rows)-1] = &userModel.User{Id: 2}

	got := userListRowsFromModels(rows, now)
	if len(got) != 2 {
		t.Fatalf("user rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy user rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("user row order changed: %#v", got)
	}
}

func TestUserListRowsFromModelsUseCapacityHintAfterActualThreshold(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	rows := make([]*userModel.User, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &userModel.User{Id: int64(index + 1)}
	}

	got := userListRowsFromModels(rows, now)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("user rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("user rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("user row order changed: %#v", got)
	}
}

func TestUserDetailFromModelHidesRemovedAuthMethods(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	item := &userModel.User{
		Id: 9,
		AuthMethods: []userModel.AuthMethods{
			{AuthType: "mobile", AuthIdentifier: "8613800138000", Verified: true},
			{AuthType: "device", AuthIdentifier: "device-id", Verified: true},
			{AuthType: "github", AuthIdentifier: "github-open-id-123456", Verified: true},
		},
	}

	got := userDetailFromModel(item, now)
	if len(got.AuthMethods) != 1 {
		t.Fatalf("auth methods = %d, want 1", len(got.AuthMethods))
	}
	if got.AuthMethods[0].AuthType != "github" || got.AuthMethods[0].AuthIdentifier != "github-open-id-123456" {
		t.Fatalf("admin detail should expose only the GitHub method: %#v", got.AuthMethods)
	}
}

func TestUserAuthMethodsFromModelsFiltersRemovedMethods(t *testing.T) {
	if got := userAuthMethodsFromModels(nil, true); got == nil || len(got) != 0 {
		t.Fatalf("empty auth methods should be a non-nil empty slice: %#v", got)
	}

	methods := []*userModel.AuthMethods{
		{AuthType: "mobile", AuthIdentifier: "8613800138000", Verified: true},
		{AuthType: "device", AuthIdentifier: "device-id", Verified: true},
		{AuthType: "email", AuthIdentifier: "user@example.com", Verified: true},
	}

	raw := userAuthMethodsFromModels(methods, false)
	if len(raw) != 1 || raw[0].AuthType != "email" || raw[0].AuthIdentifier != "user@example.com" {
		t.Fatalf("removed auth methods were not filtered: %#v", raw)
	}

	formatted := userAuthMethodsFromModels(methods, true)
	if len(formatted) != 1 || formatted[0].AuthType != "email" || formatted[0].AuthIdentifier != "user@example.com" {
		t.Fatalf("removed auth methods were not filtered from list output: %#v", formatted)
	}
}

func TestUserAuthMethodsFromModelsNilHeavyInputAvoidsFullCap(t *testing.T) {
	methods := make([]*userModel.AuthMethods, smallAdminUserAuthMethodCapacityThreshold+20)
	methods[len(methods)-1] = &userModel.AuthMethods{AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true}

	got := userAuthMethodsFromModels(methods, false)
	if len(got) != 1 || got[0].AuthType != "email" || got[0].AuthIdentifier != "alpha@example.com" {
		t.Fatalf("nil-heavy auth methods = %#v, want one email", got)
	}
	if cap(got) != smallAdminUserAuthMethodCapacityThreshold+1 {
		t.Fatalf("nil-heavy auth method cap = %d, want %d", cap(got), smallAdminUserAuthMethodCapacityThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = userAuthMethodsFromModels(methods, false)
	})
	if allocs > 1 {
		t.Fatalf("nil-heavy auth methods allocations = %f, want at most 1", allocs)
	}
}

func TestAdminUserAuthMethodCollectorBoundsSparseCapacity(t *testing.T) {
	collector := newAdminUserAuthMethodCollector(smallAdminUserAuthMethodCapacityThreshold + 20)
	for index := 0; index <= smallAdminUserAuthMethodCapacityThreshold; index++ {
		collector.Add(types.UserAuthMethod{AuthType: "provider-" + string(rune('a'+index))})
	}
	if len(collector.methods) != smallAdminUserAuthMethodCapacityThreshold+1 {
		t.Fatalf("collector len = %d, want %d", len(collector.methods), smallAdminUserAuthMethodCapacityThreshold+1)
	}
	if maxCap := len(collector.methods) * 2; cap(collector.methods) > maxCap {
		t.Fatalf("collector cap = %d, want <= %d", cap(collector.methods), maxCap)
	}
}

func TestUserAuthMethodsFromValuesPreservesDuplicateAdminRecords(t *testing.T) {
	methods := []userModel.AuthMethods{
		{AuthType: "email", AuthIdentifier: "first@example.com", Verified: true},
		{AuthType: "email", AuthIdentifier: "second@example.com", Verified: false},
	}
	got := userAuthMethodsFromValues(methods, false)
	if len(got) != 2 {
		t.Fatalf("auth methods len = %d, want duplicate records preserved: %#v", len(got), got)
	}
	if got[0].AuthIdentifier != "first@example.com" || got[1].AuthIdentifier != "second@example.com" {
		t.Fatalf("duplicate auth methods order changed: %#v", got)
	}
}

func TestUserSubscribeFromDetailsMapsLiveProduct(t *testing.T) {
	trueValue := true
	falseValue := false
	now := time.UnixMilli(1700000000000)
	finishedAt := now.Add(2 * time.Hour)
	item := &userModel.SubscribeDetails{
		Id:          11,
		UserId:      7,
		OrderId:     19,
		SubscribeId: 22,
		ProductName: "Live Plan",
		StartTime:   now,
		ExpireTime:  now.Add(24 * time.Hour),
		FinishedAt:  &finishedAt,
		Traffic:     2048,
		Download:    100,
		Upload:      50,
		Token:       "subscribe-token-for-admin",
		Status:      1,
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now,
		Subscribe: &modelSubscribe.Subscribe{
			Id:                22,
			Name:              "Live Plan",
			Description:       "live description",
			UnitPrice:         999,
			UnitTime:          "month",
			Discount:          `[{"quantity":2,"discount":0.9}]`,
			Replacement:       1299,
			Inventory:         100,
			Traffic:           4096,
			SpeedLimit:        50,
			DeviceLimit:       3,
			Quota:             2,
			Nodes:             "7, 9",
			NodeTags:          "edge, home,",
			Show:              &trueValue,
			UserVisible:       &trueValue,
			Sell:              &trueValue,
			AllowDeduction:    &falseValue,
			AllowRenewal:      &trueValue,
			AllowResetTraffic: &trueValue,
			ResetCycle:        2,
			RenewalReset:      &trueValue,
			ShowOriginalPrice: true,
			CreatedAt:         now.Add(-2 * time.Hour),
			UpdatedAt:         now.Add(-time.Hour),
		},
	}

	got := userSubscribeFromDetails(item)
	if got.Id != item.Id || got.UserId != item.UserId || got.ProductName != "Live Plan" {
		t.Fatalf("subscription fields not mapped: %#v", got)
	}
	if got.StartTime != item.StartTime.UnixMilli() || got.ExpireTime != item.ExpireTime.UnixMilli() || got.FinishedAt != finishedAt.UnixMilli() {
		t.Fatalf("subscription times not mapped: %#v", got)
	}
	if got.Short == "" || got.Short == item.Token {
		t.Fatalf("short token not generated: %#v", got.Short)
	}
	if got.Subscribe.Id != 22 || got.Subscribe.Name != "Live Plan" || got.Subscribe.UnitPrice != 999 || got.Subscribe.Inventory != 100 {
		t.Fatalf("live product fields not mapped: %#v", got.Subscribe)
	}
	if len(got.Subscribe.Discount) != 1 || got.Subscribe.Discount[0].Quantity != 2 || got.Subscribe.Discount[0].Discount != 0.9 {
		t.Fatalf("live product discount not parsed: %#v", got.Subscribe.Discount)
	}
	if len(got.Subscribe.Nodes) != 2 || got.Subscribe.Nodes[0] != 7 || got.Subscribe.Nodes[1] != 9 {
		t.Fatalf("live product nodes not parsed: %#v", got.Subscribe.Nodes)
	}
	if len(got.Subscribe.NodeTags) != 2 || got.Subscribe.NodeTags[0] != "edge" || got.Subscribe.NodeTags[1] != "home" {
		t.Fatalf("live product node tags not parsed: %#v", got.Subscribe.NodeTags)
	}
	if !got.Subscribe.Show || !got.Subscribe.UserVisible || !got.Subscribe.Sell || got.Subscribe.AllowDeduction || !got.Subscribe.AllowRenewal || !got.Subscribe.AllowResetTraffic {
		t.Fatalf("live product booleans not mapped: %#v", got.Subscribe)
	}
}

func TestUserSubscribeRowsFromDetailsSkipsNilAndKeepsEmptySlice(t *testing.T) {
	snapshot := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:    ordersnapshot.KindSubscribe,
		ID:      88,
		Name:    "Snapshot Plan",
		Traffic: 4096,
	})
	got := userSubscribeRowsFromDetails([]*userModel.SubscribeDetails{
		nil,
		{
			Id:              7,
			ProductSnapshot: snapshot,
			Subscribe:       &modelSubscribe.Subscribe{Name: "Live Plan"},
		},
		{
			Id:              8,
			ProductSnapshot: snapshot,
			Subscribe:       &modelSubscribe.Subscribe{Name: "Live Plan"},
		},
	})
	if len(got) != 2 {
		t.Fatalf("user subscribe rows len = %d, want 2", len(got))
	}
	if got[0].Id != 7 || got[0].ProductName != "Snapshot Plan" || got[0].Subscribe.Id != 88 {
		t.Fatalf("user subscribe row mismatch: %#v", got[0])
	}
	if got[1].Id != 8 || got[1].ProductName != "Snapshot Plan" || got[1].Subscribe.Traffic != 4096 {
		t.Fatalf("duplicate snapshot row mismatch: %#v", got[1])
	}

	empty := userSubscribeRowsFromDetails(nil)
	if empty == nil {
		t.Fatal("empty user subscribe rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.UserSubscribe `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty user subscribe rows marshal = %s, want list []", payload)
	}
}

func TestUserSubscribeRowsFromDetailsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*userModel.SubscribeDetails, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &userModel.SubscribeDetails{Id: 1}
	rows[len(rows)-1] = &userModel.SubscribeDetails{Id: 2}

	got := userSubscribeRowsFromDetails(rows)
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

func TestUserSubscribeRowsFromDetailsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*userModel.SubscribeDetails, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &userModel.SubscribeDetails{Id: int64(index + 1)}
	}

	got := userSubscribeRowsFromDetails(rows)
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

func TestUserSubscribeListRowsFromDetailsOmitsAccessAndFullProduct(t *testing.T) {
	snapshot := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:        ordersnapshot.KindSubscribe,
		ID:          88,
		Name:        "Snapshot Plan",
		SpeedLimit:  1048576,
		DeviceLimit: 3,
		Traffic:     4096,
	})
	got := userSubscribeListRowsFromDetails([]*userModel.SubscribeDetails{
		nil,
		{
			Id:              7,
			UserId:          9,
			OrderId:         10,
			SubscribeId:     11,
			ProductSnapshot: snapshot,
			Token:           "private-token",
			UUID:            "private-uuid",
			Subscribe: &modelSubscribe.Subscribe{
				Id:          99,
				Name:        "Live Plan",
				UnitPrice:   1000,
				Discount:    `[{"quantity":2,"discount":0.9}]`,
				Nodes:       `[7,9]`,
				SpeedLimit:  2048,
				DeviceLimit: 5,
			},
		},
	})
	if len(got) != 1 {
		t.Fatalf("user subscribe list rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].UserId != 9 || got[0].SubscribeId != 11 || got[0].ProductName != "Snapshot Plan" {
		t.Fatalf("user subscribe list row mismatch: %#v", got[0])
	}
	if got[0].Subscribe.Id != 88 || got[0].Subscribe.Name != "Snapshot Plan" || got[0].Subscribe.SpeedLimit != 1048576 || got[0].Subscribe.DeviceLimit != 3 {
		t.Fatalf("user subscribe list product summary mismatch: %#v", got[0].Subscribe)
	}

	payload, err := json.Marshal(struct {
		List []types.UserSubscribeListItem `json:"list"`
	}{List: got})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string][]map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	row := decoded["list"][0]
	for _, key := range []string{"order_id", "finished_at", "token", "short"} {
		if _, ok := row[key]; ok {
			t.Fatalf("user subscribe list row must not expose %q: %s", key, payload)
		}
	}
	subscribe, ok := row["subscribe"].(map[string]any)
	if !ok {
		t.Fatalf("user subscribe list row missing subscribe summary: %s", payload)
	}
	for _, key := range []string{"unit_price", "unit_time", "discount", "replacement", "traffic", "nodes", "node_tags", "allow_deduction", "allow_renewal", "allow_reset_traffic", "reset_cycle", "show_original_price"} {
		if _, ok := subscribe[key]; ok {
			t.Fatalf("user subscribe list product summary must not expose %q: %s", key, payload)
		}
	}
}

func TestUserSubscribeListRowsFromDetailsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*userModel.SubscribeDetails, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &userModel.SubscribeDetails{Id: 1}
	rows[len(rows)-1] = &userModel.SubscribeDetails{Id: 2}

	got := userSubscribeListRowsFromDetails(rows)
	if len(got) != 2 {
		t.Fatalf("user subscribe list rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy user subscribe list rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("user subscribe list row order changed: %#v", got)
	}
}

func TestUserSubscribeDetailFromDetailsAppliesSnapshotAndKeepsRawUserAuth(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	snapshot := ordersnapshot.Marshal(ordersnapshot.Snapshot{
		Kind:         ordersnapshot.KindSubscribe,
		ID:           88,
		Name:         "Snapshot Plan",
		UnitPrice:    1299,
		Discount:     []types.SubscribeDiscount{{Quantity: 3, Discount: 0.85}},
		Traffic:      4096,
		Nodes:        []int64{7, 9},
		NodeTags:     []string{"edge", "home"},
		ResetCycle:   2,
		RenewalReset: true,
	})
	item := &userModel.SubscribeDetails{
		Id:              11,
		UserId:          7,
		OrderId:         19,
		SubscribeId:     22,
		ProductSnapshot: snapshot,
		StartTime:       now,
		ExpireTime:      now.Add(24 * time.Hour),
		Token:           "subscribe-token-for-admin-detail",
		Status:          1,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now,
		Subscribe:       &modelSubscribe.Subscribe{Id: 22, Name: "Live Plan"},
		User: &userModel.User{
			Id: 7,
			AuthMethods: []userModel.AuthMethods{
				{AuthType: "mobile", AuthIdentifier: "8613800138000", Verified: true},
				{AuthType: "github", AuthIdentifier: "github-open-id-123456", Verified: true},
			},
		},
	}

	got := userSubscribeDetailFromDetails(item, now)
	if got.ProductName != "Snapshot Plan" || got.Subscribe.Id != 88 || got.Subscribe.Name != "Snapshot Plan" {
		t.Fatalf("snapshot product not applied to detail: %#v", got)
	}
	if len(got.Subscribe.Discount) != 1 || got.Subscribe.Discount[0].Quantity != 3 {
		t.Fatalf("snapshot discount not applied to detail: %#v", got.Subscribe.Discount)
	}
	if got.User.Id != 7 || len(got.User.AuthMethods) != 1 || got.User.AuthMethods[0].AuthIdentifier != "github-open-id-123456" {
		t.Fatalf("admin detail should exclude removed auth methods: %#v", got.User)
	}
}

func TestUserSubscribeFromDetailsKeepsZeroTimesAtZero(t *testing.T) {
	got := userSubscribeFromDetails(&userModel.SubscribeDetails{
		Subscribe: &modelSubscribe.Subscribe{},
	})
	if got.StartTime != 0 || got.ExpireTime != 0 || got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero subscription times should stay zero: %#v", got)
	}
	if got.Subscribe.CreatedAt != 0 || got.Subscribe.UpdatedAt != 0 {
		t.Fatalf("zero product times should stay zero: %#v", got.Subscribe)
	}
}
