package user

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userListColumns))
	for _, column := range userListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"balance",
		"commission",
		"gift_amount",
		"refer_code",
		"referer_id",
		"member_expired_at",
		"enable",
		"created_at",
		"deleted_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user list columns must include %q", column)
		}
	}

	forbidden := []string{
		"password",
		"algo",
		"salt",
		"avatar",
		"referral_percentage",
		"only_first_purchase",
		"is_admin",
		"enable_balance_notify",
		"enable_login_notify",
		"enable_subscribe_notify",
		"enable_trade_notify",
		"rules",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user list columns must not include private/heavy column %q", column)
		}
	}
}

func TestOrderUsersByIDsSmallResultAvoidsMapAllocation(t *testing.T) {
	users := []*User{{Id: 4}, {Id: 2}, nil, {Id: 7}}
	got := orderUsersByIDs(users, []int64{7, 99, 2, 4})
	gotIDs := make([]int64, 0, len(got))
	for _, user := range got {
		gotIDs = append(gotIDs, user.Id)
	}
	if want := []int64{7, 2, 4}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("ordered user ids = %#v, want %#v", gotIDs, want)
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = orderUsersByIDs(users, []int64{7, 2, 4})
	})
	if allocs > 1 {
		t.Fatalf("orderUsersByIDs small result allocations = %f, want at most 1", allocs)
	}
}

func TestOrderUsersByIDsSkipsMissingWithoutInputSizedCapacity(t *testing.T) {
	ids := make([]int64, viewutil.SmallRowsCapacityThreshold+20)
	ids[1] = 7
	ids[viewutil.SmallRowsCapacityThreshold+10] = 9
	users := []*User{{Id: 7}}

	got := orderUsersByIDs(users, ids)
	if len(got) != 1 || got[0].Id != 7 {
		t.Fatalf("ordered users = %#v, want user 7 only", got)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("ordered users cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestUserCacheRelationIndexUsesLinearPathForSmallInput(t *testing.T) {
	users := []*User{{Id: 4}, nil, {Id: 2}, {Id: 4}}
	index := newUserCacheRelationIndex(users)
	if index.byID != nil {
		t.Fatal("small user cache relation index should not allocate map")
	}
	if want := []int64{4, 2}; !reflect.DeepEqual(index.IDs(), want) {
		t.Fatalf("user cache relation ids = %#v, want %#v", index.IDs(), want)
	}
	if got := index.Get(2); got == nil || got.Id != 2 {
		t.Fatalf("user cache relation get(2) = %#v", got)
	}
	if got := index.Get(99); got != nil {
		t.Fatalf("user cache relation get(99) = %#v, want nil", got)
	}
}

func TestUserCacheRelationIndexUpgradesLargeInputToMap(t *testing.T) {
	users := make([]*User, 0, smallUserCacheRelationLookupLimit+1)
	for id := int64(1); id <= smallUserCacheRelationLookupLimit+1; id++ {
		users = append(users, &User{Id: id})
	}
	index := newUserCacheRelationIndex(users)
	if cap(index.users) != smallUserCacheRelationLookupLimit+1 {
		t.Fatalf("user cache relation users cap = %d, want %d", cap(index.users), smallUserCacheRelationLookupLimit+1)
	}
	if cap(index.ids) != smallUserCacheRelationLookupLimit+1 {
		t.Fatalf("user cache relation ids cap = %d, want %d", cap(index.ids), smallUserCacheRelationLookupLimit+1)
	}
	if index.byID == nil {
		t.Fatal("large user cache relation index should allocate map")
	}
	if got := index.Get(1); got == nil || got.Id != 1 {
		t.Fatalf("user cache relation get(first) = %#v", got)
	}
	lastID := int64(smallUserCacheRelationLookupLimit + 1)
	if got := index.Get(lastID); got == nil || got.Id != lastID {
		t.Fatalf("user cache relation get(last) = %#v", got)
	}
}

func TestUserCacheRelationIndexUsesCapacityHintForLargeInput(t *testing.T) {
	index := userCacheRelationIndex{
		users:   make([]*User, 0, smallUserCacheRelationLookupLimit+1),
		ids:     make([]int64, 0, smallUserCacheRelationLookupLimit+1),
		capHint: smallUserCacheRelationLookupLimit + 1,
	}
	index.Add(&User{Id: 7, Balance: 100})
	if index.byID != nil {
		t.Fatal("large hinted user cache relation index should not allocate map for one unique user")
	}
	index.Add(&User{Id: 7, Balance: 999})
	if len(index.users) != 1 || len(index.ids) != 1 {
		t.Fatalf("user cache relation index duplicate handling mismatch: %#v", index)
	}
	if got := index.Get(7); got == nil || got.Balance != 100 {
		t.Fatalf("user cache relation get(7) = %#v, want first user", got)
	}
}

func TestUserCacheRelationIndexAvoidsInputSizedCapacityForLargeDuplicateInput(t *testing.T) {
	users := make([]*User, smallUserCacheRelationLookupLimit+20)
	users[1] = &User{Id: 7, Balance: 100}
	users[smallUserCacheRelationLookupLimit+10] = &User{Id: 7, Balance: 999}

	index := newUserCacheRelationIndex(users)
	if index.byID != nil {
		t.Fatal("large duplicate user cache relation index should not allocate map")
	}
	if len(index.users) != 1 || len(index.ids) != 1 {
		t.Fatalf("user cache relation rows = users:%d ids:%d, want one unique user", len(index.users), len(index.ids))
	}
	if cap(index.users) != smallUserCacheRelationLookupLimit+1 || cap(index.ids) != smallUserCacheRelationLookupLimit+1 {
		t.Fatalf("user cache relation caps = users:%d ids:%d, want small initial cap", cap(index.users), cap(index.ids))
	}
	if got := index.Get(7); got == nil || got.Balance != 100 {
		t.Fatalf("user cache relation get(7) = %#v, want first user", got)
	}
}

func TestUserCacheRelationIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	users := make([]*User, smallUserCacheRelationLookupLimit+40)
	for index := 0; index <= smallUserCacheRelationLookupLimit; index++ {
		users[index] = &User{Id: int64(index + 1)}
	}
	for index := smallUserCacheRelationLookupLimit + 1; index < len(users); index++ {
		users[index] = &User{Id: 1}
	}

	index := newUserCacheRelationIndex(users)
	if index.byID == nil {
		t.Fatal("user cache relation index should allocate map after actual unique users exceed threshold")
	}
	if maxCap := (smallUserCacheRelationLookupLimit + 1) * 2; cap(index.users) > maxCap || cap(index.ids) > maxCap {
		t.Fatalf("user cache relation caps = users:%d ids:%d, want <= %d", cap(index.users), cap(index.ids), maxCap)
	}
}

func TestUserContextColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userContextColumns))
	for _, column := range userContextColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"avatar",
		"balance",
		"commission",
		"referral_percentage",
		"only_first_purchase",
		"gift_amount",
		"refer_code",
		"referer_id",
		"member_expired_at",
		"enable",
		"is_admin",
		"enable_balance_notify",
		"enable_login_notify",
		"enable_subscribe_notify",
		"enable_trade_notify",
		"rules",
		"created_at",
		"updated_at",
		"deleted_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user context columns must include %q", column)
		}
	}

	forbidden := []string{"password", "algo", "salt"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user context columns must not include private/heavy column %q", column)
		}
	}
}

func TestUserLoginColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userLoginColumns))
	for _, column := range userLoginColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"password",
		"algo",
		"salt",
		"enable",
		"deleted_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user login columns must include %q", column)
		}
	}

	forbidden := []string{
		"avatar",
		"balance",
		"refer_code",
		"referer_id",
		"commission",
		"rules",
		"is_admin",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user login columns must not include non-auth column %q", column)
		}
	}
}

func TestBatchDeleteUserNormalizesIDs(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE user_auth_methods (id integer primary key, user_id integer not null, auth_type text not null default '', auth_identifier text not null default '')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE user_device (id integer primary key, user_id integer not null, identifier text not null default '')`).Error; err != nil {
		t.Fatal(err)
	}
	enable := true
	if err := db.Create([]*User{
		{Id: 41, ReferCode: "u41", Enable: &enable},
		{Id: 42, ReferCode: "u42", Enable: &enable},
		{Id: 43, ReferCode: "u43", Enable: &enable},
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	model := NewModel(db, rds)
	if err := model.BatchDeleteUser(ctx, []int64{41, 0, 41, -1, 42}); err != nil {
		t.Fatal(err)
	}

	var remaining []int64
	if err := db.Model(&User{}).Order("id ASC").Pluck("id", &remaining).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(remaining, []int64{43}) {
		t.Fatalf("remaining user ids = %#v, want [43]", remaining)
	}
}

func TestUserSubscribeAccessColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeAccessColumns))
	for _, column := range userSubscribeAccessColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "enable", "deleted_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user subscribe access columns must include %q", column)
		}
	}

	forbidden := []string{"password", "algo", "salt", "avatar", "balance", "rules", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user subscribe access columns must not include private/heavy column %q", column)
		}
	}
}

func TestUserAdminCheckColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userAdminCheckColumns))
	for _, column := range userAdminCheckColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "enable", "is_admin", "deleted_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user admin check columns must include %q", column)
		}
	}

	forbidden := []string{"password", "algo", "salt", "avatar", "balance", "commission", "rules", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user admin check columns must not include private/heavy column %q", column)
		}
	}
}

func TestUserOrderActivationColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userOrderActivationColumns))
	for _, column := range userOrderActivationColumns {
		columns[column] = struct{}{}
	}

	for _, column := range []string{"id", "referer_id"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user order activation columns must include %q", column)
		}
	}

	forbidden := []string{"password", "algo", "salt", "avatar", "balance", "commission", "rules", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user order activation columns must not include private/heavy column %q", column)
		}
	}
}

func TestUserReferralPolicyColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userReferralPolicyColumns))
	for _, column := range userReferralPolicyColumns {
		columns[column] = struct{}{}
	}

	for _, column := range []string{"id", "referral_percentage", "only_first_purchase"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user referral policy columns must include %q", column)
		}
	}

	forbidden := []string{"password", "algo", "salt", "avatar", "balance", "commission", "rules", "refer_code", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user referral policy columns must not include private/heavy column %q", column)
		}
	}
}

func TestUserEmailLookupColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userEmailLookupColumns))
	for _, column := range userEmailLookupColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "deleted_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user email lookup columns must include %q", column)
		}
	}

	forbidden := []string{
		"password",
		"algo",
		"salt",
		"avatar",
		"balance",
		"refer_code",
		"referer_id",
		"commission",
		"rules",
		"is_admin",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user email lookup columns must not include %q", column)
		}
	}
}

func TestQueryActiveSubscriptionsSkipsEmptyInputMap(t *testing.T) {
	model := NewModel(nil, nil)
	got, err := model.QueryActiveSubscriptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("QueryActiveSubscriptions() = %#v, want nil map", got)
	}
	if count := got[10]; count != 0 {
		t.Fatalf("nil active subscription map lookup = %d, want 0", count)
	}

	got, err = model.QueryActiveSubscriptions(context.Background(), 0, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("QueryActiveSubscriptions(invalid IDs) = %#v, want nil map", got)
	}
}

func TestQueryActiveSubscriptionsNormalizesInputIDs(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	rows := []Subscribe{
		{Id: 1, UserId: 7, SubscribeId: 10, Status: 1, Token: "a", UUID: "a", ExpireTime: now.Add(time.Hour)},
		{Id: 2, UserId: 8, SubscribeId: 10, Status: 1, Token: "b", UUID: "b", ExpireTime: now.Add(time.Hour)},
		{Id: 3, UserId: 9, SubscribeId: 11, Status: 2, Token: "c", UUID: "c", ExpireTime: now.Add(time.Hour)},
		{Id: 4, UserId: 10, SubscribeId: 12, Status: 1, Token: "d", UUID: "d", ExpireTime: now.Add(time.Hour)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).QueryActiveSubscriptions(ctx, 10, 10, 0, 11, -1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[10] != 2 {
		t.Fatalf("QueryActiveSubscriptions normalized counts = %#v, want map[10:2]", got)
	}

	got, err = NewModel(db, nil).QueryActiveSubscriptions(ctx, 11)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("QueryActiveSubscriptions inactive-only result = %#v, want nil map", got)
	}
}

func TestUserReferCodeLookupColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userReferCodeLookupColumns))
	for _, column := range userReferCodeLookupColumns {
		columns[column] = struct{}{}
	}

	if _, ok := columns["id"]; !ok {
		t.Fatal("user refer code lookup columns must include id")
	}

	forbidden := []string{
		"password",
		"algo",
		"salt",
		"avatar",
		"balance",
		"refer_code",
		"referer_id",
		"commission",
		"rules",
		"is_admin",
		"created_at",
		"updated_at",
		"deleted_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user refer code lookup columns must not include %q", column)
		}
	}
}

func TestUserAuthMethodListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userAuthMethodListColumns))
	for _, column := range userAuthMethodListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "user_id", "auth_type", "auth_identifier", "verified"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user auth method list columns must include %q", column)
		}
	}

	forbidden := []string{"created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user auth method list columns must not include %q", column)
		}
	}
}

func TestUserPrimaryAuthMethodListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userPrimaryAuthMethodListColumns))
	for _, column := range userPrimaryAuthMethodListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"user_id",
		"auth_type",
		"auth_identifier",
		"verified",
		"ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY id ASC) AS rn",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("primary auth method list columns must include %q", column)
		}
	}
	forbidden := []string{"created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("primary auth method list columns must not include %q", column)
		}
	}
}

func TestUserAuthMethodContextColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userAuthMethodContextColumns))
	for _, column := range userAuthMethodContextColumns {
		columns[column] = struct{}{}
	}

	required := []string{"user_id", "auth_type", "auth_identifier", "verified"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user auth method context columns must include %q", column)
		}
	}

	forbidden := []string{"id", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user auth method context columns must not include %q", column)
		}
	}
}

func TestUserAuthMethodSummaryColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userAuthMethodSummaryColumns))
	for _, column := range userAuthMethodSummaryColumns {
		columns[column] = struct{}{}
	}

	required := []string{"auth_type", "auth_identifier", "verified"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user auth method summary columns must include %q", column)
		}
	}

	forbidden := []string{"id", "user_id", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user auth method summary columns must not include %q", column)
		}
	}
}

func TestUserAuthMethodQueriesKeepStableEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	methods, err := model.FindUserAuthMethods(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if methods == nil || len(methods) != 0 {
		t.Fatalf("user auth methods = %#v, want non-nil empty slice", methods)
	}

	summaries, err := model.FindUserAuthMethodSummaries(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if summaries == nil || len(summaries) != 0 {
		t.Fatalf("user auth method summaries = %#v, want non-nil empty slice", summaries)
	}
}

func TestFindOneForContextSelectsOnlyContextColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enable := true
	isAdmin := true
	if err := db.Create(&User{
		Id:        11,
		Password:  "private-password",
		Algo:      "default",
		Salt:      "private-salt",
		Avatar:    "avatar",
		Balance:   123,
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		IsAdmin:   &isAdmin,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*AuthMethods{
		{Id: 10, UserId: 11, AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true, CreatedAt: now, UpdatedAt: now},
		{Id: 11, UserId: 11, AuthType: "telegram", AuthIdentifier: "telegram-alpha", Verified: false, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	model := NewModel(db, rds)
	got, err := model.FindOneForContext(ctx, 11)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 11 || got.Avatar != "avatar" || got.Balance != 123 || got.Enable == nil || !*got.Enable || got.IsAdmin == nil || !*got.IsAdmin {
		t.Fatalf("context query dropped required user fields: %#v", got)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Rules != `["large-rule"]` || len(got.UserDevices) != 0 {
		t.Fatalf("context query loaded private/heavy fields: %#v", got)
	}
	if len(got.AuthMethods) != 2 {
		t.Fatalf("context auth methods = %d, want 2", len(got.AuthMethods))
	}
	if got.AuthMethods[0].Id != 0 || got.AuthMethods[0].UserId != 11 || got.AuthMethods[0].AuthType != "email" || got.AuthMethods[0].AuthIdentifier == "" || !got.AuthMethods[0].Verified {
		t.Fatalf("context auth method fields = %#v", got.AuthMethods[0])
	}
	if !got.AuthMethods[0].CreatedAt.IsZero() || !got.AuthMethods[0].UpdatedAt.IsZero() {
		t.Fatalf("context auth method loaded timestamps: %#v", got.AuthMethods[0])
	}
}

func TestFindOneForBasicUpdateSelectsOnlyAdjustmentColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enable := true
	isAdmin := true
	if err := db.Create(&User{
		Id:         15,
		Password:   "private-password",
		Algo:       "default",
		Salt:       "private-salt",
		Avatar:     "avatar",
		Balance:    123,
		Commission: 456,
		GiftAmount: 789,
		Rules:      `["large-rule"]`,
		Enable:     &enable,
		IsAdmin:    &isAdmin,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 15, UserId: 15, AuthType: "email", AuthIdentifier: "basic@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}
	model := NewModel(db, nil)
	got, err := model.FindOneForBasicUpdate(ctx, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 15 || got.Balance != 123 || got.Commission != 456 || got.GiftAmount != 789 || got.Enable == nil || !*got.Enable || got.IsAdmin == nil || !*got.IsAdmin {
		t.Fatalf("basic update query dropped required fields: %#v", got)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Rules != "" {
		t.Fatalf("basic update query loaded private/heavy fields: %#v", got)
	}
	if len(got.AuthMethods) != 0 || len(got.UserDevices) != 0 {
		t.Fatalf("basic update query loaded associations: auth=%d devices=%d", len(got.AuthMethods), len(got.UserDevices))
	}
}

func TestFindOneForSubscribeAccessSelectsOnlyStatusColumnsAndIncludesDeletedUsers(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enable := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&User{
		Id:        16,
		Password:  "private-password",
		Algo:      "default",
		Salt:      "private-salt",
		Avatar:    "avatar",
		Balance:   123,
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 16, UserId: 16, AuthType: "email", AuthIdentifier: "subscribe-access@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&User{}, 16).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForSubscribeAccess(ctx, 16)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 16 || got.Enable == nil || !*got.Enable || !got.DeletedAt.Valid {
		t.Fatalf("subscribe access query dropped status fields: %#v", got)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.Rules != "" || len(got.AuthMethods) != 0 || len(got.UserDevices) != 0 {
		t.Fatalf("subscribe access query loaded private/heavy fields: %#v", got)
	}
}

func TestFindOneForAdminCheckSelectsOnlyAdminStatusColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enable := true
	isAdmin := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&User{
		Id:         17,
		Password:   "private-password",
		Algo:       "default",
		Salt:       "private-salt",
		Avatar:     "avatar",
		Balance:    123,
		Commission: 456,
		Rules:      `["large-rule"]`,
		Enable:     &enable,
		IsAdmin:    &isAdmin,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 17, UserId: 17, AuthType: "email", AuthIdentifier: "admin-check@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForAdminCheck(ctx, 17)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 17 || got.Enable == nil || !*got.Enable || got.IsAdmin == nil || !*got.IsAdmin || got.DeletedAt.Valid {
		t.Fatalf("admin check query dropped status fields: %#v", got)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.Commission != 0 || got.Rules != "" || len(got.AuthMethods) != 0 || len(got.UserDevices) != 0 {
		t.Fatalf("admin check query loaded private/heavy fields: %#v", got)
	}
}

func TestFindOneForOrderActivationSelectsOnlyNotificationAndReferralColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enable := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&User{
		Id:                 18,
		Password:           "private-password",
		Algo:               "default",
		Salt:               "private-salt",
		Avatar:             "avatar",
		Balance:            123,
		Commission:         456,
		ReferCode:          "invite-code",
		RefererId:          99,
		ReferralPercentage: 12,
		Rules:              `["large-rule"]`,
		Enable:             &enable,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*AuthMethods{
		{Id: 18, UserId: 18, AuthType: "email", AuthIdentifier: "order@example.com", Verified: true, CreatedAt: now, UpdatedAt: now},
		{Id: 19, UserId: 18, AuthType: "telegram", AuthIdentifier: "123456", Verified: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForOrderActivation(ctx, 18)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 18 || got.RefererId != 99 {
		t.Fatalf("order activation query dropped required fields: %#v", got)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.Commission != 0 || got.ReferCode != "" || got.Rules != "" || got.ReferralPercentage != 0 {
		t.Fatalf("order activation query loaded private/heavy fields: %#v", got)
	}
	if len(got.AuthMethods) != 1 || got.AuthMethods[0].AuthType != "telegram" || got.AuthMethods[0].AuthIdentifier != "123456" {
		t.Fatalf("order activation auth methods = %#v, want only telegram", got.AuthMethods)
	}
	if got.AuthMethods[0].Id != 0 || got.AuthMethods[0].Verified || !got.AuthMethods[0].CreatedAt.IsZero() || !got.AuthMethods[0].UpdatedAt.IsZero() {
		t.Fatalf("order activation auth loaded non-cache fields: %#v", got.AuthMethods[0])
	}
}

func TestFindOneForReferralPolicySelectsOnlyReferralPolicyColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	onlyFirstPurchase := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&User{
		Id:                  19,
		Password:            "private-password",
		Algo:                "default",
		Salt:                "private-salt",
		Avatar:              "avatar",
		Balance:             123,
		Commission:          456,
		ReferCode:           "invite-code",
		ReferralPercentage:  25,
		OnlyFirstPurchase:   &onlyFirstPurchase,
		Rules:               `["large-rule"]`,
		CreatedAt:           now,
		UpdatedAt:           now,
		EnableBalanceNotify: &onlyFirstPurchase,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 20, UserId: 19, AuthType: "telegram", AuthIdentifier: "654321", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForReferralPolicy(ctx, 19)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 19 || got.ReferralPercentage != 25 || got.OnlyFirstPurchase == nil || !*got.OnlyFirstPurchase {
		t.Fatalf("referral policy query dropped required fields: %#v", got)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.Commission != 0 || got.ReferCode != "" || got.Rules != "" || len(got.AuthMethods) != 0 {
		t.Fatalf("referral policy query loaded private/heavy fields: %#v", got)
	}
}

func TestFindOneForLoginSelectsOnlyLoginColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enable := true
	isAdmin := true
	if err := db.Create(&User{
		Id:        12,
		Password:  "login-password",
		Algo:      "default",
		Salt:      "login-salt",
		Avatar:    "avatar",
		Balance:   123,
		ReferCode: "refer-code",
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		IsAdmin:   &isAdmin,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 12, UserId: 12, AuthType: "email", AuthIdentifier: "login@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	model := NewModel(db, rds)
	got, err := model.FindOneForLogin(ctx, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 12 || got.Password != "login-password" || got.Algo != "default" || got.Salt != "login-salt" || got.Enable == nil || !*got.Enable {
		t.Fatalf("login query dropped required fields: %#v", got)
	}
	if got.Avatar != "" || got.Balance != 0 || got.ReferCode != "" || got.Rules != "" || got.IsAdmin != nil || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("login query loaded non-auth fields: %#v", got)
	}
	if len(got.AuthMethods) != 0 || len(got.UserDevices) != 0 {
		t.Fatalf("login query loaded associations: auth=%d devices=%d", len(got.AuthMethods), len(got.UserDevices))
	}
}

func TestFindOneForLoginByEmailUsesSeparateCacheAndSelectsOnlyLoginColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enable := true
	if err := db.Create(&User{
		Id:        13,
		Password:  "email-login-password",
		Algo:      "argon2id",
		Salt:      "email-salt",
		Avatar:    "avatar",
		Balance:   456,
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 13, UserId: 13, AuthType: "email", AuthIdentifier: "login-cache@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	stale, err := json.Marshal(&User{Id: 99, Password: "stale-full-email-cache"})
	if err != nil {
		t.Fatal(err)
	}
	if err := rds.Set(ctx, cacheUserEmailPrefix+"login-cache@example.com", stale, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, rds)
	got, err := model.FindOneForLoginByEmail(ctx, "login-cache@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 13 || got.Password != "email-login-password" || got.Algo != "argon2id" || got.Salt != "email-salt" {
		t.Fatalf("email login query returned wrong user or dropped auth fields: %#v", got)
	}
	if got.Avatar != "" || got.Balance != 0 || got.Rules != "" || len(got.AuthMethods) != 0 || len(got.UserDevices) != 0 {
		t.Fatalf("email login query loaded non-auth fields: %#v", got)
	}
	if exists := rds.Exists(ctx, cacheUserLoginEmailPrefix+"login-cache@example.com").Val(); exists != 1 {
		t.Fatalf("login email cache was not populated")
	}
}

func TestFindOneByEmailUsesLookupCacheAndSelectsOnlyLookupColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enable := true
	if err := db.Create(&User{
		Id:        14,
		Password:  "private-password",
		Algo:      "default",
		Salt:      "private-salt",
		Avatar:    "avatar",
		Balance:   789,
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 14, UserId: 14, AuthType: "email", AuthIdentifier: "lookup-cache@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	stale, err := json.Marshal(&User{Id: 99, Password: "stale-full-email-cache"})
	if err != nil {
		t.Fatal(err)
	}
	if err := rds.Set(ctx, cacheUserEmailPrefix+"lookup-cache@example.com", stale, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, rds)
	got, err := model.FindOneByEmail(ctx, "lookup-cache@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 14 {
		t.Fatalf("email lookup returned user id %d, want 14", got.Id)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.Rules != "" {
		t.Fatalf("email lookup loaded private/heavy fields: %#v", got)
	}
	if len(got.AuthMethods) != 0 || len(got.UserDevices) != 0 {
		t.Fatalf("email lookup loaded associations: auth=%d devices=%d", len(got.AuthMethods), len(got.UserDevices))
	}
	if exists := rds.Exists(ctx, cacheUserEmailLookupPrefix+"lookup-cache@example.com").Val(); exists != 1 {
		t.Fatalf("email lookup cache was not populated")
	}
}

func TestFindOneByReferCodeSelectsOnlyLookupColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}

	enable := true
	if err := db.Create(&User{
		Id:        15,
		Password:  "private-password",
		Algo:      "default",
		Salt:      "private-salt",
		Avatar:    "avatar",
		Balance:   789,
		ReferCode: "invite-code",
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		CreatedAt: time.Unix(1700000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	got, err := model.FindOneByReferCode(ctx, "invite-code")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 15 {
		t.Fatalf("refer code lookup returned user id %d, want 15", got.Id)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.ReferCode != "" || got.Rules != "" {
		t.Fatalf("refer code lookup loaded private/heavy fields: %#v", got)
	}
	if got.Enable != nil || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("refer code lookup loaded non-required fields: %#v", got)
	}
}

func TestUserCacheColumnsStayFocused(t *testing.T) {
	want := []string{"id"}
	if !reflect.DeepEqual(userCacheColumns, want) {
		t.Fatalf("user cache columns = %#v, want %#v", userCacheColumns, want)
	}
}

func TestFindOneUserForCacheSelectsOnlyCacheKeyFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE user_device (
	id integer PRIMARY KEY,
	ip text NOT NULL,
	user_id integer NOT NULL,
	user_agent text,
	identifier text,
	identifier_bidx text NOT NULL DEFAULT '',
	identifier_display text NOT NULL DEFAULT '',
	ip_display text NOT NULL DEFAULT '',
	user_agent_display text NOT NULL DEFAULT '',
	online boolean NOT NULL DEFAULT false,
	enabled boolean NOT NULL DEFAULT true,
	created_at datetime,
	updated_at datetime
)`).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enable := true
	if err := db.Create(&User{
		Id:        21,
		Password:  "private-password",
		Algo:      "default",
		Salt:      "private-salt",
		Avatar:    "avatar",
		Balance:   100,
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*AuthMethods{
		{Id: 1, UserId: 21, AuthType: "email", AuthIdentifier: "cache@example.com", Verified: true, CreatedAt: now, UpdatedAt: now},
		{Id: 2, UserId: 21, AuthType: "telegram", AuthIdentifier: "1001", Verified: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*Device{
		{Id: 11, UserId: 21, Ip: "192.0.2.1", Identifier: "device-a", UserAgent: "agent-a", Online: true, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customUserModel)
	got, err := model.findOneUserForCache(ctx, 21)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 21 {
		t.Fatalf("cache user id = %d, want 21", got.Id)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.Rules != "" || got.Enable != nil || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("cache user loaded non-cache fields: %#v", got)
	}
	if len(got.AuthMethods) != 2 {
		t.Fatalf("cache auth methods = %d, want 2", len(got.AuthMethods))
	}
	if got.AuthMethods[0].Id != 0 || got.AuthMethods[0].UserId != 21 || got.AuthMethods[0].AuthType != "email" || got.AuthMethods[0].AuthIdentifier != "cache@example.com" || got.AuthMethods[0].Verified {
		t.Fatalf("cache auth method fields = %#v", got.AuthMethods[0])
	}
	if !got.AuthMethods[0].CreatedAt.IsZero() || !got.AuthMethods[0].UpdatedAt.IsZero() {
		t.Fatalf("cache auth method loaded timestamps: %#v", got.AuthMethods[0])
	}
	if len(got.UserDevices) != 1 {
		t.Fatalf("cache devices = %d, want 1", len(got.UserDevices))
	}
	if got.UserDevices[0].Id != 11 || got.UserDevices[0].Identifier != "device-a" {
		t.Fatalf("cache device dropped cache key fields: %#v", got.UserDevices[0])
	}
	if got.UserDevices[0].UserId != 0 || got.UserDevices[0].Ip != "" || got.UserDevices[0].UserAgent != "" || got.UserDevices[0].Online || got.UserDevices[0].Enabled || !got.UserDevices[0].CreatedAt.IsZero() || !got.UserDevices[0].UpdatedAt.IsZero() {
		t.Fatalf("cache device loaded non-cache fields: %#v", got.UserDevices[0])
	}
}

func TestBatchGetCacheKeysIncludesAuthAndDeviceKeys(t *testing.T) {
	keys := (&defaultUserModel{}).batchGetCacheKeys(&User{
		Id: 21,
		AuthMethods: []AuthMethods{
			{UserId: 21, AuthType: "email", AuthIdentifier: "cache@example.com"},
		},
		UserDevices: []Device{
			{Id: 11, Identifier: "device-a"},
		},
	})
	for _, want := range []string{
		cacheUserIdPrefix + "21",
		cacheUserEmailPrefix + "cache@example.com",
		cacheUserEmailLookupPrefix + "cache@example.com",
		cacheUserLoginEmailPrefix + "cache@example.com",
		cacheUserDeviceIdPrefix + "11",
		cacheUserDeviceNumberPrefix + "device-a",
	} {
		if !containsString(keys, want) {
			t.Fatalf("batch cache keys = %#v, want %q", keys, want)
		}
	}
}

func TestBatchGetCacheKeysDeduplicatesRepeatedDeviceKeys(t *testing.T) {
	keys := (&defaultUserModel{}).batchGetCacheKeys(&User{
		Id: 21,
		UserDevices: []Device{
			{Id: 11, Identifier: "device-a"},
			{Id: 11, Identifier: "device-a"},
		},
	}, nil)

	if got := countString(keys, cacheUserDeviceIdPrefix+"11"); got != 1 {
		t.Fatalf("device id cache key count = %d, want 1: %#v", got, keys)
	}
	if got := countString(keys, cacheUserDeviceNumberPrefix+"device-a"); got != 1 {
		t.Fatalf("device identifier cache key count = %d, want 1: %#v", got, keys)
	}
}

func TestBatchGetCacheKeysEmptyInputAvoidsAllocation(t *testing.T) {
	model := &defaultUserModel{}
	got := model.batchGetCacheKeys(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty batch cache keys = %#v, want non-nil empty slice", got)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = model.batchGetCacheKeys(nil)
	})
	if allocs > 0 {
		t.Fatalf("empty batch cache keys allocations = %f, want zero", allocs)
	}
}

func TestCacheKeysForModelsDeduplicatesOverlappingUserKeys(t *testing.T) {
	keys := cacheKeysForModels(
		&User{Id: 21},
		&AuthMethods{UserId: 21, AuthType: "email", AuthIdentifier: "cache@example.com"},
		&Subscribe{Id: 31, UserId: 21, Token: "subscribe-token"},
		nil,
	)

	for _, want := range []string{
		cacheUserIdPrefix + "21",
		cacheUserContextPrefix + "21",
		cacheUserLoginIdPrefix + "21",
		cacheUserSubscribeAccessPrefix + "21",
		cacheUserSubscribeUserPrefix + "21",
		cacheUserEmailPrefix + "cache@example.com",
		cacheUserSubscribeTokenPrefix + "subscribe-token",
	} {
		if !containsString(keys, want) {
			t.Fatalf("cache keys = %#v, want %q", keys, want)
		}
	}
	for _, unique := range []string{
		cacheUserIdPrefix + "21",
		cacheUserContextPrefix + "21",
		cacheUserLoginIdPrefix + "21",
		cacheUserSubscribeAccessPrefix + "21",
	} {
		if got := countString(keys, unique); got != 1 {
			t.Fatalf("cache key %q count = %d, want 1: %#v", unique, got, keys)
		}
	}
}

func TestCacheKeysForModelsSmallInputAvoidsMapAllocation(t *testing.T) {
	model := staticCacheKeys([]string{"user:id:1", "user:ctx:1", "user:id:1", ""})
	allocs := testing.AllocsPerRun(1000, func() {
		_ = cacheKeysForModels(model)
	})
	if allocs > 2 {
		t.Fatalf("cacheKeysForModels small input allocations = %f, want at most 2", allocs)
	}
}

func TestCacheKeysForModelsEmptyInputAvoidsAllocation(t *testing.T) {
	emptyKeys := staticCacheKeys([]string{"", ""})
	got := cacheKeysForModels(nil, staticCacheKeys(nil), emptyKeys)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty model cache keys = %#v, want non-nil empty slice", got)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = cacheKeysForModels(nil, staticCacheKeys(nil), emptyKeys)
	})
	if allocs > 1 {
		t.Fatalf("cacheKeysForModels empty input allocations = %f, want at most 1", allocs)
	}
}

func TestCacheKeysForTypedModelsDeduplicatesAndSkipsNil(t *testing.T) {
	keys := cacheKeysForTypedModels(3*userModelCacheKeyHint,
		&User{Id: 21},
		nil,
		&User{Id: 21},
	)

	for _, want := range []string{
		cacheUserIdPrefix + "21",
		cacheUserContextPrefix + "21",
		cacheUserLoginIdPrefix + "21",
		cacheUserSubscribeAccessPrefix + "21",
		cacheUserAdminCheckPrefix + "21",
	} {
		if !containsString(keys, want) {
			t.Fatalf("typed cache keys = %#v, want %q", keys, want)
		}
		if got := countString(keys, want); got != 1 {
			t.Fatalf("typed cache key %q count = %d, want 1: %#v", want, got, keys)
		}
	}
}

func TestCacheKeysForTypedModelsNilInputAvoidsAllocation(t *testing.T) {
	got := cacheKeysForTypedModels[*User](2*userModelCacheKeyHint, nil, nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("nil typed cache keys = %#v, want non-nil empty slice", got)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = cacheKeysForTypedModels[*User](2*userModelCacheKeyHint, nil, nil)
	})
	if allocs > 0 {
		t.Fatalf("nil typed model cache keys allocations = %f, want zero", allocs)
	}
}

func TestAppendUniqueModelCacheKeyIgnoresCapacityHintForSmallActualInput(t *testing.T) {
	keys := make([]string, 0, smallModelCacheKeySetThreshold+1)
	var seen map[string]struct{}
	keys, seen = appendUniqueModelCacheKey(keys, "user:id:1", seen)
	if seen != nil {
		t.Fatal("small actual model cache key set should not allocate map from capacity hint alone")
	}
	keys, seen = appendUniqueModelCacheKey(keys, "user:id:1", seen)
	if len(keys) != 1 {
		t.Fatalf("cache keys = %#v, want one unique key", keys)
	}
}

func TestAppendUniqueModelCacheKeyUpgradesLargeActualInputToMap(t *testing.T) {
	var keys []string
	var seen map[string]struct{}
	for index := 0; index <= smallModelCacheKeySetThreshold; index++ {
		keys, seen = appendUniqueModelCacheKey(keys, "user:key:"+strconv.Itoa(index), seen)
	}
	if seen == nil {
		t.Fatal("large actual model cache key set should allocate map")
	}
	keys, seen = appendUniqueModelCacheKey(keys, "user:key:0", seen)
	if len(keys) != smallModelCacheKeySetThreshold+1 {
		t.Fatalf("duplicate key after upgrade changed keys = %#v", keys)
	}
}

func TestModelCacheKeyCollectorAvoidsMapForLargeDuplicateInput(t *testing.T) {
	collector := newModelCacheKeyCollector(smallModelCacheKeySetThreshold + 20)
	for index := 0; index < smallModelCacheKeySetThreshold+20; index++ {
		collector.Add("user:id:1")
	}

	if collector.seen != nil {
		t.Fatal("duplicate-heavy model cache key collector should not allocate map")
	}
	if !reflect.DeepEqual(collector.Keys(), []string{"user:id:1"}) {
		t.Fatalf("cache keys = %#v, want single key", collector.Keys())
	}
	if cap(collector.Keys()) != smallModelCacheKeySetThreshold+1 {
		t.Fatalf("cache key cap = %d, want %d", cap(collector.Keys()), smallModelCacheKeySetThreshold+1)
	}
}

func TestModelCacheKeyCollectorBoundsSparseUpgradeCapacity(t *testing.T) {
	collector := newModelCacheKeyCollector(smallModelCacheKeySetThreshold + 20)
	for index := 0; index <= smallModelCacheKeySetThreshold; index++ {
		collector.Add("user:key:" + strconv.Itoa(index))
	}
	for index := 0; index < smallModelCacheKeySetThreshold+20; index++ {
		collector.Add("user:key:0")
	}

	if collector.seen == nil {
		t.Fatal("model cache key collector should allocate map after actual unique keys exceed threshold")
	}
	if len(collector.Keys()) != smallModelCacheKeySetThreshold+1 {
		t.Fatalf("cache key len = %d, want %d", len(collector.Keys()), smallModelCacheKeySetThreshold+1)
	}
	if maxCap := (smallModelCacheKeySetThreshold + 1) * 2; cap(collector.Keys()) > maxCap {
		t.Fatalf("cache key cap = %d, want <= %d", cap(collector.Keys()), maxCap)
	}
}

func TestUniqueCacheKeysKeepsOrderAndSkipsEmpty(t *testing.T) {
	got := uniqueCacheKeys([]string{"user:id:1", "", "user:ctx:1", "user:id:1", "user:login:1"})
	want := []string{"user:id:1", "user:ctx:1", "user:login:1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueCacheKeys() = %#v, want %#v", got, want)
	}
}

type staticCacheKeys []string

func (s staticCacheKeys) GetCacheKeys() []string {
	return []string(s)
}

func TestUniqueCacheKeysSmallInputAvoidsMapAllocation(t *testing.T) {
	keys := []string{"user:id:1", "user:ctx:1", "user:id:1"}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = uniqueCacheKeys(keys)
	})
	if allocs > 1 {
		t.Fatalf("uniqueCacheKeys small input allocations = %f, want at most 1", allocs)
	}
}

func TestUniqueCacheKeysLargeDuplicateAvoidsInputSizedCapacity(t *testing.T) {
	keys := make([]string, smallModelCacheKeySetThreshold+20)
	for index := range keys {
		keys[index] = "user:id:1"
	}

	got := uniqueCacheKeys(keys)
	if want := []string{"user:id:1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueCacheKeys() = %#v, want %#v", got, want)
	}
	if cap(got) != smallModelCacheKeySetThreshold+1 {
		t.Fatalf("unique cache key cap = %d, want small initial cap", cap(got))
	}
}

func TestNormalizeUserCacheIDsKeepsOrderAndSkipsInvalid(t *testing.T) {
	got := normalizeUserCacheIDs([]int64{4, 0, 4, -1, 2, 3, 2})
	want := []int64{4, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeUserCacheIDs() = %#v, want %#v", got, want)
	}
}

func TestNormalizeUserCacheIDsSmallInputAvoidsMapAllocation(t *testing.T) {
	ids := []int64{4, 0, 4, -1, 2, 3, 2}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = normalizeUserCacheIDs(ids)
	})
	if allocs > 1 {
		t.Fatalf("normalizeUserCacheIDs small input allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizeUserCacheIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	ids := make([]int64, smallUserCacheRelationLookupLimit+20)
	ids[3] = 7
	ids[smallUserCacheRelationLookupLimit+10] = 7

	got := normalizeUserCacheIDs(ids)
	if want := []int64{7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeUserCacheIDs() = %#v, want %#v", got, want)
	}
	if cap(got) != smallUserCacheRelationLookupLimit+1 {
		t.Fatalf("normalized user id cap = %d, want small initial cap", cap(got))
	}
}

func TestNormalizeUserCacheIDsBoundsSparseUpgradeCapacity(t *testing.T) {
	ids := make([]int64, smallUserCacheRelationLookupLimit+40)
	for i := 0; i <= smallUserCacheRelationLookupLimit; i++ {
		ids[i] = int64(i + 1)
	}

	got := normalizeUserCacheIDs(ids)
	if len(got) != smallUserCacheRelationLookupLimit+1 {
		t.Fatalf("normalizeUserCacheIDs len = %d, want %d", len(got), smallUserCacheRelationLookupLimit+1)
	}
	if maxCap := (smallUserCacheRelationLookupLimit + 1) * 2; cap(got) > maxCap {
		t.Fatalf("normalizeUserCacheIDs cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestUserAuthMethodCacheColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userAuthMethodCacheColumns))
	for _, column := range userAuthMethodCacheColumns {
		columns[column] = struct{}{}
	}

	required := []string{"user_id", "auth_type", "auth_identifier"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user auth method cache columns must include %q", column)
		}
	}

	forbidden := []string{"id", "verified", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user auth method cache columns must not include %q", column)
		}
	}
}

func TestUserAuthMethodLookupColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userAuthMethodLookupColumns))
	for _, column := range userAuthMethodLookupColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "user_id", "auth_type", "auth_identifier", "verified"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user auth method lookup columns must include %q", column)
		}
	}

	forbidden := []string{"created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user auth method lookup columns must not include %q", column)
		}
	}
}

func TestUserDeviceCachePreloadColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userDeviceCachePreloadColumns))
	for _, column := range userDeviceCachePreloadColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "user_id", "identifier"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user device cache preload columns must include %q", column)
		}
	}

	forbidden := []string{"ip", "user_agent", "online", "enabled", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user device cache preload columns must not include %q", column)
		}
	}
}

func TestFindOneSelectsOnlyCacheKeyFields(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE user_device (
	id integer PRIMARY KEY,
	ip text NOT NULL,
	user_id integer NOT NULL,
	user_agent text,
	identifier text,
	identifier_bidx text NOT NULL DEFAULT '',
	identifier_display text NOT NULL DEFAULT '',
	ip_display text NOT NULL DEFAULT '',
	user_agent_display text NOT NULL DEFAULT '',
	online boolean NOT NULL DEFAULT false,
	enabled boolean NOT NULL DEFAULT true,
	created_at datetime,
	updated_at datetime
)`).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enable := true
	if err := db.Create(&User{
		Id:        31,
		Password:  "private-password",
		Algo:      "default",
		Salt:      "private-salt",
		Avatar:    "avatar",
		Balance:   100,
		Rules:     `["large-rule"]`,
		Enable:    &enable,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*AuthMethods{
		{Id: 1, UserId: 31, AuthType: "email", AuthIdentifier: "findone@example.com", Verified: true, CreatedAt: now, UpdatedAt: now},
		{Id: 2, UserId: 31, AuthType: "telegram", AuthIdentifier: "1001", Verified: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Device{Id: 12, UserId: 31, Ip: "192.0.2.2", Identifier: "device-b", UserAgent: "agent-b", Online: true, Enabled: true, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOne(ctx, 31)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 31 {
		t.Fatalf("user id = %d, want 31", got.Id)
	}
	if got.Password != "" || got.Algo != "" || got.Salt != "" || got.Avatar != "" || got.Balance != 0 || got.Rules != "" || got.Enable != nil || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("FindOne loaded non-cache user fields: %#v", got)
	}
	if len(got.AuthMethods) != 2 {
		t.Fatalf("FindOne auth methods = %d, want 2", len(got.AuthMethods))
	}
	if got.AuthMethods[0].Id != 0 || got.AuthMethods[0].UserId != 31 || got.AuthMethods[0].AuthType != "email" || got.AuthMethods[0].AuthIdentifier != "findone@example.com" || got.AuthMethods[0].Verified {
		t.Fatalf("FindOne auth method fields = %#v", got.AuthMethods[0])
	}
	if !got.AuthMethods[0].CreatedAt.IsZero() || !got.AuthMethods[0].UpdatedAt.IsZero() {
		t.Fatalf("FindOne auth method loaded timestamps: %#v", got.AuthMethods[0])
	}
	if len(got.UserDevices) != 1 {
		t.Fatalf("FindOne devices = %d, want 1", len(got.UserDevices))
	}
	if got.UserDevices[0].Id != 12 || got.UserDevices[0].UserId != 31 || got.UserDevices[0].Identifier != "device-b" {
		t.Fatalf("FindOne device cache fields = %#v", got.UserDevices[0])
	}
	if got.UserDevices[0].Ip != "" || got.UserDevices[0].UserAgent != "" || got.UserDevices[0].Online || got.UserDevices[0].Enabled || !got.UserDevices[0].CreatedAt.IsZero() || !got.UserDevices[0].UpdatedAt.IsZero() {
		t.Fatalf("FindOne device loaded non-cache fields: %#v", got.UserDevices[0])
	}
}

func TestFindUserAuthMethodsSelectsOnlyCacheColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	createdAt := time.Unix(1700000000, 0)
	if err := db.Create([]*AuthMethods{
		{Id: 1, UserId: 7, AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true, CreatedAt: createdAt, UpdatedAt: createdAt},
		{Id: 2, UserId: 7, AuthType: "mobile", AuthIdentifier: "+15551234567", Verified: true, CreatedAt: createdAt, UpdatedAt: createdAt},
		{Id: 3, UserId: 8, AuthType: "email", AuthIdentifier: "other@example.com", Verified: true, CreatedAt: createdAt, UpdatedAt: createdAt},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, err := model.FindUserAuthMethods(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("cache auth methods = %d, want 2", len(list))
	}
	for _, item := range list {
		if item.Id != 0 || item.Verified || !item.CreatedAt.IsZero() || !item.UpdatedAt.IsZero() {
			t.Fatalf("cache query loaded non-cache columns: %#v", item)
		}
		if item.UserId != 7 || item.AuthType == "" || item.AuthIdentifier == "" {
			t.Fatalf("cache query dropped cache key fields: %#v", item)
		}
	}
}

func TestFindUserAuthMethodByOpenIDSelectsOnlyLookupColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	createdAt := time.Unix(1700000000, 0)
	if err := db.Create(&AuthMethods{
		Id:             21,
		UserId:         7,
		AuthType:       "email",
		AuthIdentifier: "lookup@example.com",
		Verified:       true,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	got, err := model.FindUserAuthMethodByOpenID(ctx, "email", "lookup@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 21 || got.UserId != 7 || got.AuthType != "email" || got.AuthIdentifier != "lookup@example.com" || !got.Verified {
		t.Fatalf("lookup query dropped required fields: %#v", got)
	}
	if !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("lookup query loaded timestamps: %#v", got)
	}
}

func TestFindUserAuthMethodSummariesSelectsOnlyResponseColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	createdAt := time.Unix(1700000000, 0)
	if err := db.Create([]*AuthMethods{
		{Id: 3, UserId: 7, AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true, CreatedAt: createdAt, UpdatedAt: createdAt},
		{Id: 4, UserId: 7, AuthType: "telegram", AuthIdentifier: "telegram-user", Verified: false, CreatedAt: createdAt, UpdatedAt: createdAt},
		{Id: 5, UserId: 8, AuthType: "email", AuthIdentifier: "other@example.com", Verified: true, CreatedAt: createdAt, UpdatedAt: createdAt},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, err := model.FindUserAuthMethodSummaries(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("summary auth methods = %d, want 2", len(list))
	}

	gotTypes := []string{list[0].AuthType, list[1].AuthType}
	if want := []string{"email", "telegram"}; !reflect.DeepEqual(gotTypes, want) {
		t.Fatalf("auth method order = %#v, want %#v", gotTypes, want)
	}
	for _, item := range list {
		if item.Id != 0 || item.UserId != 0 || !item.CreatedAt.IsZero() || !item.UpdatedAt.IsZero() {
			t.Fatalf("summary query loaded internal columns: %#v", item)
		}
		if item.AuthType == "" || item.AuthIdentifier == "" {
			t.Fatalf("summary query dropped response fields: %#v", item)
		}
	}
}

func TestAdminNotificationColumnsStayLightweight(t *testing.T) {
	userColumns := make(map[string]struct{}, len(userAdminNotificationColumns))
	for _, column := range userAdminNotificationColumns {
		userColumns[column] = struct{}{}
	}
	if _, ok := userColumns["id"]; !ok {
		t.Fatal("admin notification user columns must include id")
	}
	forbiddenUserColumns := []string{"password", "algo", "salt", "rules", "balance", "created_at", "updated_at"}
	for _, column := range forbiddenUserColumns {
		if _, ok := userColumns[column]; ok {
			t.Fatalf("admin notification user columns must not include %q", column)
		}
	}

	authColumns := make(map[string]struct{}, len(userAdminNotificationAuthColumns))
	for _, column := range userAdminNotificationAuthColumns {
		authColumns[column] = struct{}{}
	}
	for _, column := range []string{"user_id", "auth_type", "auth_identifier"} {
		if _, ok := authColumns[column]; !ok {
			t.Fatalf("admin notification auth columns must include %q", column)
		}
	}
	forbiddenAuthColumns := []string{"id", "verified", "created_at", "updated_at"}
	for _, column := range forbiddenAuthColumns {
		if _, ok := authColumns[column]; ok {
			t.Fatalf("admin notification auth columns must not include %q", column)
		}
	}
}

func TestQueryAdminUsersSelectsOnlyTelegramAuthMethods(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	admin := true
	normal := false
	now := time.Unix(1700000000, 0)
	if err := db.Create([]*User{
		{Id: 1, Password: "admin-secret", Balance: 100, Rules: "private-admin-rules", IsAdmin: &admin, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Password: "admin-no-telegram-secret", Balance: 200, Rules: "private-admin-rules-2", IsAdmin: &admin, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Password: "normal-secret", Balance: 300, Rules: "private-user-rules", IsAdmin: &normal, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*AuthMethods{
		{Id: 1, UserId: 1, AuthType: "email", AuthIdentifier: "admin@example.com", Verified: true, CreatedAt: now, UpdatedAt: now},
		{Id: 2, UserId: 1, AuthType: "telegram", AuthIdentifier: "1001", Verified: true, CreatedAt: now, UpdatedAt: now},
		{Id: 3, UserId: 2, AuthType: "email", AuthIdentifier: "admin2@example.com", Verified: true, CreatedAt: now, UpdatedAt: now},
		{Id: 4, UserId: 3, AuthType: "telegram", AuthIdentifier: "3001", Verified: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, err := model.QueryAdminUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("admin users = %d, want 2", len(list))
	}
	if list[0].Id != 1 || list[1].Id != 2 {
		t.Fatalf("admin IDs = [%d %d], want [1 2]", list[0].Id, list[1].Id)
	}
	for _, item := range list {
		if item.Password != "" || item.Rules != "" || item.Balance != 0 || item.IsAdmin != nil {
			t.Fatalf("admin notification query loaded unused user fields: %#v", item)
		}
	}
	if len(list[0].AuthMethods) != 1 {
		t.Fatalf("admin 1 auth methods = %d, want 1", len(list[0].AuthMethods))
	}
	telegram := list[0].AuthMethods[0]
	if telegram.UserId != 1 || telegram.AuthType != "telegram" || telegram.AuthIdentifier != "1001" {
		t.Fatalf("admin telegram auth method = %#v", telegram)
	}
	if telegram.Id != 0 || telegram.Verified || !telegram.CreatedAt.IsZero() || !telegram.UpdatedAt.IsZero() {
		t.Fatalf("admin telegram query loaded unused auth fields: %#v", telegram)
	}
	if len(list[1].AuthMethods) != 0 {
		t.Fatalf("admin 2 auth methods = %d, want 0", len(list[1].AuthMethods))
	}
}

func TestQueryAdminUsersEmptyResultKeepsStableSlice(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	list, err := NewModel(db, nil).QueryAdminUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("admin users = %#v, want non-nil empty slice", list)
	}
}

func TestInsertUserAuthMethodsClearsCacheWithoutLoadingFullUser(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range userTestCacheKeys(41, "new-auth@example.com") {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.InsertUserAuthMethods(ctx, &AuthMethods{
		UserId:         41,
		AuthType:       "email",
		AuthIdentifier: "new-auth@example.com",
		Verified:       true,
	}); err != nil {
		t.Fatal(err)
	}

	var got AuthMethods
	if err := db.Where("user_id = ? AND auth_type = ?", 41, "email").First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.AuthIdentifier != "new-auth@example.com" || !got.Verified {
		t.Fatalf("inserted auth method = %#v", got)
	}
	for _, key := range userTestCacheKeys(41, "new-auth@example.com") {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserAuthMethodsUpdatesOnlyAuthFieldsAndClearsOldNewEmailCaches(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	createdAt := time.Unix(1700000000, 0)
	if err := db.Create(&AuthMethods{
		Id:             31,
		UserId:         42,
		AuthType:       "email",
		AuthIdentifier: "old-auth@example.com",
		Verified:       true,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range append(userTestCacheKeys(42, "old-auth@example.com"), cacheUserEmailPrefix+"new-auth@example.com", cacheUserEmailLookupPrefix+"new-auth@example.com", cacheUserLoginEmailPrefix+"new-auth@example.com") {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserAuthMethods(ctx, 42, "email", UserAuthMethodFieldsUpdate{
		AuthIdentifier: UpdateValue("new-auth@example.com"),
		Verified:       UpdateValue(false),
	}); err != nil {
		t.Fatal(err)
	}

	var got AuthMethods
	if err := db.Where("id = ?", 31).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.UserId != 42 || got.AuthType != "email" || got.AuthIdentifier != "new-auth@example.com" || got.Verified {
		t.Fatalf("updated auth method = %#v", got)
	}
	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at changed: got %v want %v", got.CreatedAt, createdAt)
	}
	for _, key := range append(userTestCacheKeys(42, "old-auth@example.com"), cacheUserEmailPrefix+"new-auth@example.com", cacheUserEmailLookupPrefix+"new-auth@example.com", cacheUserLoginEmailPrefix+"new-auth@example.com") {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestDeleteUserAuthMethodsClearsCacheWithoutLoadingFullUser(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{
		Id:             32,
		UserId:         43,
		AuthType:       "email",
		AuthIdentifier: "delete-auth@example.com",
		Verified:       true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range userTestCacheKeys(43, "delete-auth@example.com") {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.DeleteUserAuthMethods(ctx, 43, "email"); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&AuthMethods{}).Where("user_id = ? AND auth_type = ?", 43, "email").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("auth method count = %d, want 0", count)
	}
	for _, key := range userTestCacheKeys(43, "delete-auth@example.com") {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserPasswordUpdatesOnlyPasswordFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&User{
		Id:       7,
		Password: "old-password",
		Algo:     "md5",
		Salt:     "salt",
		Balance:  123,
		Rules:    `["keep-rule"]`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 1, UserId: 7, AuthType: "email", AuthIdentifier: "alpha@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range userTestCacheKeys(7, "alpha@example.com") {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserPassword(ctx, 7, "new-password"); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 7).Error; err != nil {
		t.Fatal(err)
	}
	if got.Password != "new-password" || got.Algo != "bcrypt" {
		t.Fatalf("password fields = %q/%q, want new-password/bcrypt", got.Password, got.Algo)
	}
	if got.Salt != "salt" || got.Balance != 123 || got.Rules != `["keep-rule"]` {
		t.Fatalf("non-password fields changed: %#v", got)
	}
	for _, key := range userTestCacheKeys(7, "alpha@example.com") {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserBalanceUpdatesOnlyBalanceAndClearsCoreCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&User{
		Id:       51,
		Password: "keep-password",
		Algo:     "default",
		Salt:     "keep-salt",
		Avatar:   "keep-avatar",
		Balance:  100,
		Rules:    `["keep-rule"]`,
		Enable:   &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range (&User{Id: 51}).GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserBalance(ctx, 51, 250); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 51).Error; err != nil {
		t.Fatal(err)
	}
	if got.Balance != 250 {
		t.Fatalf("balance = %d, want 250", got.Balance)
	}
	if got.Password != "keep-password" || got.Algo != "default" || got.Salt != "keep-salt" || got.Avatar != "keep-avatar" || got.Rules != `["keep-rule"]` {
		t.Fatalf("non-balance fields changed: %#v", got)
	}
	for _, key := range (&User{Id: 51}).GetCacheKeys() {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserGiftAmountUpdatesOnlyGiftAmountAndClearsCoreCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&User{
		Id:         54,
		Password:   "keep-password",
		Balance:    100,
		GiftAmount: 200,
		Rules:      `["keep-rule"]`,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range (&User{Id: 54}).GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserGiftAmount(ctx, 54, 350); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 54).Error; err != nil {
		t.Fatal(err)
	}
	if got.GiftAmount != 350 {
		t.Fatalf("gift amount = %d, want 350", got.GiftAmount)
	}
	if got.Password != "keep-password" || got.Balance != 100 || got.Rules != `["keep-rule"]` {
		t.Fatalf("non-gift fields changed: %#v", got)
	}
	for _, key := range (&User{Id: 54}).GetCacheKeys() {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserCommissionUpdatesOnlyCommissionAndClearsCoreCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&User{
		Id:         52,
		Password:   "keep-password",
		Balance:    100,
		Commission: 200,
		Rules:      `["keep-rule"]`,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range (&User{Id: 52}).GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserCommission(ctx, 52, 350); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 52).Error; err != nil {
		t.Fatal(err)
	}
	if got.Commission != 350 {
		t.Fatalf("commission = %d, want 350", got.Commission)
	}
	if got.Password != "keep-password" || got.Balance != 100 || got.Rules != `["keep-rule"]` {
		t.Fatalf("non-commission fields changed: %#v", got)
	}
	for _, key := range (&User{Id: 52}).GetCacheKeys() {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserMemberExpiredAtUpdatesOnlyMembershipAndClearsCoreCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	oldExpiredAt := time.Unix(1700000000, 0).UTC()
	newExpiredAt := time.Unix(1800000000, 0).UTC()
	if err := db.Create(&User{
		Id:              53,
		Password:        "keep-password",
		Balance:         100,
		MemberExpiredAt: oldExpiredAt,
		Rules:           `["keep-rule"]`,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range (&User{Id: 53}).GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserMemberExpiredAt(ctx, 53, newExpiredAt); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 53).Error; err != nil {
		t.Fatal(err)
	}
	if !got.MemberExpiredAt.Equal(newExpiredAt) {
		t.Fatalf("member_expired_at = %s, want %s", got.MemberExpiredAt, newExpiredAt)
	}
	if got.Password != "keep-password" || got.Balance != 100 || got.Rules != `["keep-rule"]` {
		t.Fatalf("non-membership fields changed: %#v", got)
	}
	for _, key := range (&User{Id: 53}).GetCacheKeys() {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserRulesUpdatesOnlyRulesAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&User{
		Id:       8,
		Password: "keep-password",
		Algo:     "default",
		Balance:  456,
		Rules:    `["old-rule"]`,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 2, UserId: 8, AuthType: "email", AuthIdentifier: "beta@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range userTestCacheKeys(8, "beta@example.com") {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserRules(ctx, 8, `["new-rule"]`); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 8).Error; err != nil {
		t.Fatal(err)
	}
	if got.Rules != `["new-rule"]` {
		t.Fatalf("rules = %q, want new-rule", got.Rules)
	}
	if got.Password != "keep-password" || got.Algo != "default" || got.Balance != 456 {
		t.Fatalf("non-rules fields changed: %#v", got)
	}
	for _, key := range userTestCacheKeys(8, "beta@example.com") {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserNotifySettingsUpdatesOnlyNotifyFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	disabled := false
	if err := db.Create(&User{
		Id:                    9,
		Password:              "keep-password",
		Algo:                  "default",
		Balance:               789,
		EnableBalanceNotify:   &disabled,
		EnableLoginNotify:     &disabled,
		EnableSubscribeNotify: &disabled,
		EnableTradeNotify:     &disabled,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 3, UserId: 9, AuthType: "email", AuthIdentifier: "notify@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range userTestCacheKeys(9, "notify@example.com") {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserNotifySettings(ctx, 9, UserNotifySettingsUpdate{
		EnableBalanceNotify:   UpdateValue(true),
		EnableSubscribeNotify: UpdateValue(true),
	}); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 9).Error; err != nil {
		t.Fatal(err)
	}
	if got.EnableBalanceNotify == nil || !*got.EnableBalanceNotify || got.EnableSubscribeNotify == nil || !*got.EnableSubscribeNotify {
		t.Fatalf("notify fields not updated: %#v", got)
	}
	if got.EnableLoginNotify == nil || *got.EnableLoginNotify != disabled || got.EnableTradeNotify == nil || *got.EnableTradeNotify != disabled {
		t.Fatalf("unspecified notify fields changed: %#v", got)
	}
	if got.Password != "keep-password" || got.Algo != "default" || got.Balance != 789 {
		t.Fatalf("non-notify fields changed: %#v", got)
	}
	for _, key := range userTestCacheKeys(9, "notify@example.com") {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserBasicFieldsUpdatesOnlyRequestedFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	admin := true
	onlyFirstPurchase := true
	if err := db.Create(&User{
		Id:                  10,
		Password:            "keep-password",
		Algo:                "default",
		Salt:                "keep-salt",
		Avatar:              "old-avatar",
		Balance:             100,
		Commission:          200,
		GiftAmount:          300,
		ReferCode:           "old-code",
		RefererId:           4,
		ReferralPercentage:  12,
		OnlyFirstPurchase:   &onlyFirstPurchase,
		Enable:              &enabled,
		IsAdmin:             &admin,
		Rules:               `["keep-rule"]`,
		EnableBalanceNotify: &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&AuthMethods{Id: 4, UserId: 10, AuthType: "email", AuthIdentifier: "basic-update@example.com", Verified: true}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range userTestCacheKeys(10, "basic-update@example.com") {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateUserBasicFields(ctx, 10, UserBasicFieldsUpdate{
		Avatar:             UpdateValue(""),
		Balance:            UpdateValue(int64(101)),
		Commission:         UpdateValue(int64(202)),
		GiftAmount:         UpdateValue(int64(303)),
		ReferCode:          UpdateValue("new-code"),
		RefererID:          UpdateValue(int64(5)),
		ReferralPercentage: UpdateValue(uint8(34)),
		OnlyFirstPurchase:  UpdateValue(false),
		Enable:             UpdateValue(false),
		IsAdmin:            UpdateValue(false),
	}); err != nil {
		t.Fatal(err)
	}

	var got User
	if err := db.First(&got, 10).Error; err != nil {
		t.Fatal(err)
	}
	if got.Avatar != "" || got.Balance != 101 || got.Commission != 202 || got.GiftAmount != 303 || got.ReferCode != "new-code" || got.RefererId != 5 || got.ReferralPercentage != 34 {
		t.Fatalf("basic fields not updated: %#v", got)
	}
	if got.OnlyFirstPurchase == nil || *got.OnlyFirstPurchase || got.Enable == nil || *got.Enable || got.IsAdmin == nil || *got.IsAdmin {
		t.Fatalf("false boolean fields were not updated: %#v", got)
	}
	if got.Password != "keep-password" || got.Algo != "default" || got.Salt != "keep-salt" || got.Rules != `["keep-rule"]` {
		t.Fatalf("private/heavy fields changed: %#v", got)
	}
	if got.EnableBalanceNotify == nil || !*got.EnableBalanceNotify {
		t.Fatalf("unrelated notify field changed: %#v", got)
	}
	for _, key := range userTestCacheKeys(10, "basic-update@example.com") {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateUserReferCodeHonorsTransactionAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&User{Id: 11, ReferCode: "old-code"}).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	model := NewModel(db, rds)

	rollback := errors.New("rollback")
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := model.UpdateUserReferCode(ctx, 11, "rolled-back", tx); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("transaction error = %v, want rollback", err)
	}
	var stored User
	if err := db.First(&stored, 11).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ReferCode != "old-code" {
		t.Fatalf("rolled-back refer code = %q, want old-code", stored.ReferCode)
	}

	for _, key := range (&User{Id: 11}).GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}
	if err := model.UpdateUserReferCode(ctx, 11, "new-code"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&stored, 11).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ReferCode != "new-code" {
		t.Fatalf("refer code = %q, want new-code", stored.ReferCode)
	}
	for _, key := range (&User{Id: 11}).GetCacheKeys() {
		if rds.Exists(ctx, key).Val() != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func userTestCacheKeys(userID int64, email string) []string {
	keys := []string{
		cacheUserIdPrefix + strconv.FormatInt(userID, 10),
		cacheUserContextPrefix + strconv.FormatInt(userID, 10),
		cacheUserLoginIdPrefix + strconv.FormatInt(userID, 10),
		cacheUserSubscribeAccessPrefix + strconv.FormatInt(userID, 10),
		cacheUserAdminCheckPrefix + strconv.FormatInt(userID, 10),
	}
	if email != "" {
		keys = append(keys, cacheUserEmailPrefix+email)
		keys = append(keys, cacheUserEmailLookupPrefix+email)
		keys = append(keys, cacheUserLoginEmailPrefix+email)
	}
	return keys
}

func TestQueryPageListSearchDedupesAndPreservesIDOrder(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create([]*User{
		{Id: 1, Password: "secret-1", ReferCode: "needle-one", Rules: "large-rules-1"},
		{Id: 2, Password: "secret-2", ReferCode: "other", Rules: "large-rules-2"},
		{Id: 3, Password: "secret-3", ReferCode: "needle-three", Rules: "large-rules-3"},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*AuthMethods{
		{Id: 1, UserId: 1, AuthType: "mobile", AuthIdentifier: "needle-one-mobile", Verified: true},
		{Id: 2, UserId: 1, AuthType: "email", AuthIdentifier: "needle-one@example.com", Verified: true},
		{Id: 3, UserId: 2, AuthType: "email", AuthIdentifier: "other@example.com", Verified: true},
		{Id: 4, UserId: 3, AuthType: "email", AuthIdentifier: "needle-three@example.com", Verified: true},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, total, err := model.QueryPageList(ctx, 1, 10, &UserFilterParams{Search: "needle", Order: "DESC"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(list) != 2 || cap(list) != 2 {
		t.Fatalf("user list len/cap = %d/%d, want 2/2", len(list), cap(list))
	}
	gotIDs := make([]int64, 0, len(list))
	for _, item := range list {
		gotIDs = append(gotIDs, item.Id)
		if item.Password != "" || item.Rules != "" {
			t.Fatalf("list query loaded private/heavy fields for user %d: password=%q rules=%q", item.Id, item.Password, item.Rules)
		}
	}
	if want := []int64{3, 1}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("user ids = %#v, want %#v", gotIDs, want)
	}
	if len(list[1].AuthMethods) != 1 {
		t.Fatalf("user 1 auth methods = %d, want primary auth only", len(list[1].AuthMethods))
	}
	if list[1].AuthMethods[0].AuthType != "email" || list[1].AuthMethods[0].AuthIdentifier != "needle-one@example.com" {
		t.Fatalf("user 1 primary auth method = %#v, want first supported auth by id", list[1].AuthMethods[0])
	}
}

func TestQueryPageListFiltersLifecycleStatusAndIncludesDeletedRows(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	disabled := false
	deletedAt := time.Now().Add(-time.Hour)
	if err := db.Create([]*User{
		{Id: 1, Password: "secret-1", Enable: &enabled},
		{Id: 2, Password: "secret-2", Enable: &disabled},
		{
			Id:        3,
			Password:  "secret-3",
			Enable:    &enabled,
			DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true},
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		status      string
		wantID      int64
		wantDeleted bool
	}{
		{status: "enabled", wantID: 1},
		{status: "disabled", wantID: 2},
		{status: "deleted", wantID: 3, wantDeleted: true},
	}
	model := NewModel(db, nil)
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			list, total, queryErr := model.QueryPageList(ctx, 1, 10, &UserFilterParams{
				Status: test.status,
				Order:  "ASC",
			})
			if queryErr != nil {
				t.Fatal(queryErr)
			}
			if total != 1 || len(list) != 1 || list[0].Id != test.wantID {
				t.Fatalf("status %q returned total/list = %d/%#v, want user %d", test.status, total, list, test.wantID)
			}
			if list[0].DeletedAt.Valid != test.wantDeleted {
				t.Fatalf("status %q deleted flag = %v, want %v", test.status, list[0].DeletedAt.Valid, test.wantDeleted)
			}
		})
	}
}

func TestQueryPageListSearchMatchesExactUserID(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*User{
		{Id: 7, Password: "secret-7", ReferCode: "OTHER"},
		{Id: 42, Password: "secret-42", ReferCode: "ALPHA"},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, total, err := model.QueryPageList(ctx, 1, 10, &UserFilterParams{Search: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Id != 42 {
		t.Fatalf("numeric search returned total/list = %d/%#v, want user 42", total, list)
	}
}

func TestQueryPageListEmptyResultKeepsStableSlice(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &AuthMethods{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, total, err := model.QueryPageList(ctx, -1, 0, &UserFilterParams{Search: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("list = %#v, want non-nil empty slice", list)
	}
}

func TestUserSubscribeAdminListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeAdminListColumns))
	for _, column := range userSubscribeAdminListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"user_id",
		"subscribe_id",
		"product_name",
		"start_time",
		"expire_time",
		"traffic",
		"download",
		"upload",
		"status",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("admin user subscribe list columns must include %q", column)
		}
	}

	forbidden := []string{"order_id", "product_snapshot", "finished_at", "token", "uuid", "note"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("admin user subscribe list columns must not include %q", column)
		}
	}
}

func TestUserSubscribeCredentialColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeCredentialColumns))
	for _, column := range userSubscribeCredentialColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "token"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user subscribe credential columns must include %q", column)
		}
	}

	forbidden := []string{"user_id", "order_id", "subscribe_id", "product_snapshot", "uuid", "traffic", "download", "upload", "note", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user subscribe credential columns must not include %q", column)
		}
	}
}

func TestUserSubscribeQueriesKeepStableEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	fullList, err := model.QueryUserSubscribe(ctx, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if fullList == nil || len(fullList) != 0 {
		t.Fatalf("full user subscriptions = %#v, want non-nil empty slice", fullList)
	}

	runtimeList, err := model.QueryUserSubscribeRuntime(ctx, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if runtimeList == nil || len(runtimeList) != 0 {
		t.Fatalf("runtime user subscriptions = %#v, want non-nil empty slice", runtimeList)
	}

	pageList, total, err := model.QueryUserSubscribePage(ctx, 7, 1, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || pageList == nil || len(pageList) != 0 {
		t.Fatalf("paged user subscriptions = total %d list %#v, want total 0 and non-nil empty slice", total, pageList)
	}

	activeList, err := model.FindUsersSubscribeBySubscribeId(ctx, 9)
	if err != nil {
		t.Fatal(err)
	}
	if activeList == nil || len(activeList) != 0 {
		t.Fatalf("active subscriptions = %#v, want non-nil empty slice", activeList)
	}
}

func TestQueryUserSubscribePagePreallocatesPageCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &modelSubscribe.Subscribe{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create([]*modelSubscribe.Subscribe{
		{Id: 10, Name: "basic", Show: &enabled, UserVisible: &enabled, Sell: &enabled},
		{Id: 11, Name: "pro", Show: &enabled, UserVisible: &enabled, Sell: &enabled},
	}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := db.Create([]*Subscribe{
		{Id: 1, UserId: 7, SubscribeId: 10, ProductName: "basic", ProductSnapshot: `{}`, ExpireTime: now.Add(time.Hour), Token: "sub-token-1", UUID: "uuid-1", Status: 1},
		{Id: 2, UserId: 7, SubscribeId: 11, ProductName: "pro", ProductSnapshot: `{}`, ExpireTime: now.Add(2 * time.Hour), Token: "sub-token-2", UUID: "uuid-2", Status: 1},
		{Id: 3, UserId: 7, SubscribeId: 11, ProductName: "pro annual", ProductSnapshot: `{}`, ExpireTime: now.Add(3 * time.Hour), Token: "sub-token-3", UUID: "uuid-3", Status: 1},
	}).Error; err != nil {
		t.Fatal(err)
	}

	list, total, err := NewModel(db, nil).QueryUserSubscribePage(ctx, 7, 1, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 2 || cap(list) != 2 {
		t.Fatalf("user subscribe total/len/cap = %d/%d/%d, want 3/2/2", total, len(list), cap(list))
	}
	if list[0].Id != 3 || list[1].Id != 2 {
		t.Fatalf("user subscribe order = [%d %d], want [3 2]", list[0].Id, list[1].Id)
	}
	if list[0].Subscribe == nil || list[0].Subscribe.Name != "pro" {
		t.Fatalf("user subscribe product summary not hydrated: %#v", list[0].Subscribe)
	}
}

func TestFindOneUserSubscribeAccessSelectsOnlyCredentialColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if err := db.Create(&Subscribe{
		Id:              91,
		UserId:          7,
		OrderId:         12,
		SubscribeId:     13,
		ProductName:     "private snapshot",
		ProductSnapshot: `{"kind":"subscribe","name":"private snapshot"}`,
		ExpireTime:      now.Add(time.Hour),
		Traffic:         1024,
		Download:        256,
		Upload:          128,
		Token:           "credential-token",
		UUID:            "credential-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneUserSubscribeAccess(ctx, 91)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 91 || got.Token != "credential-token" {
		t.Fatalf("credential query dropped required fields: %#v", got)
	}
	if got.UserId != 0 || got.OrderId != 0 || got.SubscribeId != 0 || got.ProductName != "" || got.ProductSnapshot != "" || got.UUID != "" || got.Traffic != 0 || got.Download != 0 || got.Upload != 0 || got.Status != 0 || got.Note != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("credential query loaded non-credential fields: %#v", got)
	}
}

func TestQueryUserSubscribeRuntimeSkipsProductPreload(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &modelSubscribe.Subscribe{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&modelSubscribe.Subscribe{
		Id:                9,
		Name:              "current product",
		Description:       "heavy description",
		UnitPrice:         1000,
		UnitTime:          "month",
		Discount:          `[{"quantity":3,"discount":80}]`,
		Replacement:       100,
		Traffic:           1024,
		SpeedLimit:        100,
		DeviceLimit:       3,
		Nodes:             "1,2",
		NodeTags:          "premium",
		AllowDeduction:    &enabled,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		ShowOriginalPrice: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Subscribe{
		Id:              1,
		UserId:          7,
		SubscribeId:     9,
		ProductName:     "snapshot product",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"snapshot product","nodes":"3","node_tags":"snapshot"}`,
		Status:          1,
		Token:           "token",
		UUID:            "uuid",
		ExpireTime:      time.Now().Add(time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	runtimeList, err := model.QueryUserSubscribeRuntime(ctx, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(runtimeList) != 1 {
		t.Fatalf("runtime subscriptions = %d, want 1", len(runtimeList))
	}
	if runtimeList[0].Subscribe != nil {
		t.Fatalf("runtime query preloaded product: %#v", runtimeList[0].Subscribe)
	}
	if runtimeList[0].ProductSnapshot == "" || runtimeList[0].UUID != "uuid" {
		t.Fatalf("runtime query dropped required fields: %#v", runtimeList[0])
	}

	fullList, err := model.QueryUserSubscribe(ctx, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(fullList) != 1 {
		t.Fatalf("full subscriptions = %d, want 1", len(fullList))
	}
	if fullList[0].Subscribe == nil || fullList[0].Subscribe.Name != "current product" {
		t.Fatalf("full query did not preload product: %#v", fullList[0].Subscribe)
	}
	if fullList[0].Subscribe.UnitPrice != 1000 || fullList[0].Subscribe.Traffic != 1024 || fullList[0].Subscribe.ResetCycle != 2 {
		t.Fatalf("full query dropped summary product fields: %#v", fullList[0].Subscribe)
	}
	if fullList[0].Subscribe.Description != "" || fullList[0].Subscribe.Nodes != "" || fullList[0].Subscribe.NodeTags != "" || fullList[0].Subscribe.Inventory != 0 || !fullList[0].Subscribe.CreatedAt.IsZero() || !fullList[0].Subscribe.UpdatedAt.IsZero() {
		t.Fatalf("full query loaded heavy product fields: %#v", fullList[0].Subscribe)
	}
}

func TestCountUserSubscribeUsesActiveWindowAndProductFilter(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	neverFinished := time.UnixMilli(0)
	recentFinish := now.Add(-time.Hour)
	oldFinish := now.AddDate(0, 0, -8)
	rows := []Subscribe{
		{Id: 31, UserId: 7, SubscribeId: 9, Status: 1, Token: "active-a", UUID: "uuid-a", ExpireTime: now.Add(time.Hour), FinishedAt: &neverFinished},
		{Id: 32, UserId: 7, SubscribeId: 9, Status: 2, Token: "active-b", UUID: "uuid-b", ExpireTime: now.Add(2 * time.Hour), FinishedAt: &neverFinished},
		{Id: 33, UserId: 7, SubscribeId: 10, Status: 1, Token: "active-c", UUID: "uuid-c", ExpireTime: now.Add(3 * time.Hour), FinishedAt: &neverFinished},
		{Id: 34, UserId: 7, SubscribeId: 9, Status: 1, Token: "recent-finished", UUID: "uuid-d", ExpireTime: now.Add(-2 * time.Hour), FinishedAt: &recentFinish},
		{Id: 35, UserId: 7, SubscribeId: 9, Status: 1, Token: "old-finished", UUID: "uuid-e", ExpireTime: now.Add(-48 * time.Hour), FinishedAt: &oldFinish},
		{Id: 36, UserId: 8, SubscribeId: 9, Status: 1, Token: "other-user", UUID: "uuid-f", ExpireTime: now.Add(time.Hour), FinishedAt: &neverFinished},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	total, err := model.CountUserSubscribe(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Fatalf("user subscription count = %d, want 4", total)
	}

	productTotal, err := model.CountUserSubscribeByProduct(ctx, 7, 9)
	if err != nil {
		t.Fatal(err)
	}
	if productTotal != 3 {
		t.Fatalf("product subscription count = %d, want 3", productTotal)
	}

	activeProductTotal, err := model.CountUserSubscribeByProduct(ctx, 7, 9, 1)
	if err != nil {
		t.Fatal(err)
	}
	if activeProductTotal != 2 {
		t.Fatalf("active product subscription count = %d, want 2", activeProductTotal)
	}

	invalidProductTotal, err := model.CountUserSubscribeByProduct(ctx, 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	if invalidProductTotal != 0 {
		t.Fatalf("invalid product count = %d, want 0", invalidProductTotal)
	}
}

func TestFindUsersSubscribeBySubscribeIdSelectsOnlyRuntimeColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	rows := []Subscribe{
		{
			Id:              11,
			UserId:          7,
			OrderId:         101,
			SubscribeId:     9,
			ProductName:     "heavy product name",
			ProductSnapshot: `{"name":"runtime snapshot"}`,
			ExpireTime:      now.Add(time.Hour),
			Traffic:         1024,
			Download:        128,
			Upload:          64,
			Token:           "token-active",
			UUID:            "uuid-active",
			Status:          1,
			Note:            "private note",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			Id:          12,
			UserId:      8,
			SubscribeId: 9,
			Token:       "token-disabled",
			UUID:        "uuid-disabled",
			Status:      0,
			ExpireTime:  now.Add(time.Hour),
		},
	}
	if err := db.Select("*").Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	list, err := NewModel(db, nil).FindUsersSubscribeBySubscribeId(ctx, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("active subscriptions = %d, want 1", len(list))
	}
	got := list[0]
	if got.Id != 11 || got.UserId != 7 || got.SubscribeId != 9 || got.ProductSnapshot == "" || got.Token != "token-active" || got.UUID != "uuid-active" || got.Traffic != 1024 || got.Download != 128 || got.Upload != 64 {
		t.Fatalf("runtime subscription fields missing: %#v", got)
	}
	if got.OrderId != 0 || got.ProductName != "" || got.Note != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("active subscription query loaded non-runtime fields: %#v", got)
	}
}

func TestUserDeviceListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userDeviceListColumns))
	for _, column := range userDeviceListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "ip", "identifier", "user_agent", "online", "enabled", "created_at", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user device list columns must include %q", column)
		}
	}

	forbidden := []string{"user_id"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user device list columns must not include filter-only column %q", column)
		}
	}
}

func TestQueryDeviceListSelectsOnlyListColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create([]*Device{
		{Id: 1, UserId: 7, Ip: "192.0.2.1", Identifier: "device-a", UserAgent: "agent-a", Online: true, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{Id: 2, UserId: 7, Ip: "192.0.2.2", Identifier: "device-b", UserAgent: "agent-b", Online: false, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{Id: 3, UserId: 8, Ip: "192.0.2.3", Identifier: "device-c", UserAgent: "agent-c", Online: true, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, total, err := model.QueryDeviceList(ctx, 7, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if got := len(list); got != 2 {
		t.Fatalf("list len = %d, want 2", got)
	}
	if cap(list) != 2 {
		t.Fatalf("device list cap = %d, want 2", cap(list))
	}
	if list[0].Id != 2 || list[1].Id != 1 {
		t.Fatalf("device order = [%d %d], want [2 1]", list[0].Id, list[1].Id)
	}
	for _, item := range list {
		if item.UserId != 0 {
			t.Fatalf("list query loaded filter-only user_id for device %d: %d", item.Id, item.UserId)
		}
		if item.Ip == "" || item.Identifier == "" || item.UserAgent == "" {
			t.Fatalf("list query dropped response fields: %#v", item)
		}
	}

	pageList, pageTotal, err := model.QueryDevicePageList(ctx, 7, 0, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if pageTotal != 2 || len(pageList) != 1 || cap(pageList) != 1 {
		t.Fatalf("device page total/len/cap = %d/%d/%d, want 2/1/1", pageTotal, len(pageList), cap(pageList))
	}
}

func TestDeviceListQueriesKeepStableEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	list, total, err := model.QueryDeviceList(ctx, 7, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("device list = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}

	pageList, pageTotal, err := model.QueryDevicePageList(ctx, 7, 0, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if pageTotal != 0 || pageList == nil || len(pageList) != 0 {
		t.Fatalf("device page list = total %d list %#v, want total 0 and non-nil empty slice", pageTotal, pageList)
	}
}

func TestUserDeviceCacheColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "identifier"}
	if !reflect.DeepEqual(userDeviceCacheColumns, want) {
		t.Fatalf("user device cache columns = %#v, want %#v", userDeviceCacheColumns, want)
	}
}

func TestUserDeviceStatusUpdateColumnsStayFocused(t *testing.T) {
	want := []string{"id", "user_id", "identifier", "online", "enabled"}
	if !reflect.DeepEqual(userDeviceStatusUpdateColumns, want) {
		t.Fatalf("user device status update columns = %#v, want %#v", userDeviceStatusUpdateColumns, want)
	}
}

func TestFindOneDeviceForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Device{
		Id:         10,
		UserId:     7,
		Ip:         "192.0.2.10",
		Identifier: "device-cache",
		UserAgent:  "private-agent",
		Online:     true,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customUserModel)
	got, err := model.findOneDeviceForCache(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 10 || got.Identifier != "device-cache" {
		t.Fatalf("cache query dropped cache fields: %#v", got)
	}
	if got.UserId != 0 || got.Ip != "" || got.UserAgent != "" || got.Online || got.Enabled || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("cache query loaded non-cache device fields: %#v", got)
	}
}

func TestFindOneDeviceForStatusUpdateSelectsOnlyStatusColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Device{
		Id:         12,
		UserId:     7,
		Ip:         "192.0.2.12",
		Identifier: "device-status",
		UserAgent:  "private-agent",
		Online:     true,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	got, err := model.FindOneDeviceForStatusUpdate(ctx, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 12 || got.UserId != 7 || got.Identifier != "device-status" || !got.Online || !got.Enabled {
		t.Fatalf("status query dropped required fields: %#v", got)
	}
	if got.Ip != "" || got.UserAgent != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("status query loaded non-status fields: %#v", got)
	}
}

func TestFindOneDeviceByIdentifierForStatusUpdateSelectsOnlyStatusColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Device{
		Id:         14,
		UserId:     8,
		Ip:         "192.0.2.14",
		Identifier: "device-status-by-identifier",
		UserAgent:  "private-agent",
		Online:     false,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	got, err := model.FindOneDeviceByIdentifierForStatusUpdate(ctx, "device-status-by-identifier")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 14 || got.UserId != 8 || got.Identifier != "device-status-by-identifier" || got.Online || !got.Enabled {
		t.Fatalf("identifier status query dropped required fields: %#v", got)
	}
	if got.Ip != "" || got.UserAgent != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("identifier status query loaded non-status fields: %#v", got)
	}
}

func TestUpdateDeviceFieldsUpdatesOnlyRequestedFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Device{
		Id:         13,
		UserId:     7,
		Ip:         "192.0.2.13",
		Identifier: "device-update-fields",
		UserAgent:  "stable-agent",
		Online:     true,
		Enabled:    true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	cacheModel := &Device{Id: 13, Identifier: "device-update-fields"}
	for _, key := range cacheModel.GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateDeviceFields(ctx, cacheModel, DeviceFieldsUpdate{
		Online:  UpdateValue(false),
		Enabled: UpdateValue(false),
	}); err != nil {
		t.Fatal(err)
	}

	var got Device
	if err := db.First(&got, 13).Error; err != nil {
		t.Fatal(err)
	}
	if got.Online || got.Enabled {
		t.Fatalf("device status fields not updated: %#v", got)
	}
	if got.UserId != 7 || got.Ip != "192.0.2.13" || got.Identifier != "device-update-fields" || got.UserAgent != "stable-agent" || !got.CreatedAt.Equal(now) {
		t.Fatalf("unrelated device fields changed: %#v", got)
	}
	for _, key := range cacheModel.GetCacheKeys() {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateDeviceFieldsCanRebindWithoutSavingWholeRow(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Device{
		Id:         15,
		UserId:     7,
		Ip:         "192.0.2.15",
		Identifier: "device-rebind-fields",
		UserAgent:  "old-agent",
		Online:     true,
		Enabled:    false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	newUserID := int64(9)
	newIP := "192.0.2.99"
	newAgent := "new-agent"
	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	cacheModel := &Device{Id: 15, Identifier: "device-rebind-fields"}
	for _, key := range cacheModel.GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.UpdateDeviceFields(ctx, cacheModel, DeviceFieldsUpdate{
		UserID:    &newUserID,
		IP:        &newIP,
		UserAgent: &newAgent,
		Enabled:   UpdateValue(true),
	}); err != nil {
		t.Fatal(err)
	}

	var got Device
	if err := db.First(&got, 15).Error; err != nil {
		t.Fatal(err)
	}
	if got.UserId != newUserID || got.Ip != newIP || got.UserAgent != newAgent || !got.Enabled {
		t.Fatalf("device rebind fields not updated: %#v", got)
	}
	if !got.Online || got.Identifier != "device-rebind-fields" || !got.CreatedAt.Equal(now) {
		t.Fatalf("unrelated device fields changed: %#v", got)
	}
	for _, key := range cacheModel.GetCacheKeys() {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestDeleteDeviceByModelDeletesWithProvidedCacheKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Device{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&Device{
		Id:         11,
		UserId:     7,
		Ip:         "192.0.2.11",
		Identifier: "device-delete",
		UserAgent:  "private-agent",
		Online:     true,
		Enabled:    true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	cacheModel := &Device{Id: 11, Identifier: "device-delete"}
	for _, key := range cacheModel.GetCacheKeys() {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	if err := model.DeleteDeviceByModel(ctx, cacheModel); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&Device{}).Where("id = ?", 11).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("device count = %d, want 0", count)
	}
	for _, key := range cacheModel.GetCacheKeys() {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUserSubscribeDetailColumnsIncludeMutableFields(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeDetailColumns))
	for _, column := range userSubscribeDetailColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "user_id", "token", "uuid", "product_snapshot", "note"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user subscribe detail columns must include %q", column)
		}
	}
}

func TestUserSubscribeOrderActionColumnsStayFocused(t *testing.T) {
	want := []string{"id", "user_id", "order_id", "subscribe_id", "product_snapshot", "expire_time", "token", "status"}
	if !reflect.DeepEqual(userSubscribeOrderActionColumns, want) {
		t.Fatalf("user subscribe order action columns = %#v, want %#v", userSubscribeOrderActionColumns, want)
	}
}

func TestUserSubscribeRuntimeColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeRuntimeColumns))
	for _, column := range userSubscribeRuntimeColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"user_id",
		"subscribe_id",
		"product_snapshot",
		"expire_time",
		"traffic",
		"download",
		"upload",
		"token",
		"uuid",
		"status",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user subscribe runtime columns must include %q", column)
		}
	}

	forbidden := []string{"order_id", "product_name", "start_time", "finished_at", "note", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user subscribe runtime columns must not include %q", column)
		}
	}
}

func TestUserSubscribeExpiryColumnsStayFocused(t *testing.T) {
	want := []string{"id", "subscribe_id", "product_snapshot", "expire_time"}
	if !reflect.DeepEqual(userSubscribeExpiryColumns, want) {
		t.Fatalf("user subscribe expiry columns = %#v, want %#v", userSubscribeExpiryColumns, want)
	}
}

func TestUserSubscribeDeductionColumnsStayFocused(t *testing.T) {
	want := []string{"id", "order_id", "subscribe_id", "start_time", "expire_time", "traffic", "download", "upload", "status"}
	if !reflect.DeepEqual(userSubscribeDeductionColumns, want) {
		t.Fatalf("user subscribe deduction columns = %#v, want %#v", userSubscribeDeductionColumns, want)
	}
}

func TestUserSubscribePresetColumnsStayFocused(t *testing.T) {
	want := []string{"id", "user_id", "subscribe_id", "product_snapshot"}
	if !reflect.DeepEqual(userSubscribePresetColumns, want) {
		t.Fatalf("user subscribe preset columns = %#v, want %#v", userSubscribePresetColumns, want)
	}
}

func TestUserSubscribeOrderLookupColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "user_id", "token"}
	if !reflect.DeepEqual(userSubscribeOrderLookupColumns, want) {
		t.Fatalf("user subscribe order lookup columns = %#v, want %#v", userSubscribeOrderLookupColumns, want)
	}
}

func TestUserSubscribeCacheColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "user_id", "token"}
	if !reflect.DeepEqual(userSubscribeCacheColumns, want) {
		t.Fatalf("user subscribe cache columns = %#v, want %#v", userSubscribeCacheColumns, want)
	}
}

func TestUserSubscribeMutationColumnsStayFocused(t *testing.T) {
	want := []string{
		"id",
		"user_id",
		"order_id",
		"subscribe_id",
		"product_name",
		"product_snapshot",
		"start_time",
		"expire_time",
		"finished_at",
		"traffic",
		"download",
		"upload",
		"traffic_reset_at",
		"token",
		"status",
		"reset_cycle",
		"next_traffic_reset_at",
	}
	if !reflect.DeepEqual(userSubscribeMutationColumns, want) {
		t.Fatalf("user subscribe mutation columns = %#v, want %#v", userSubscribeMutationColumns, want)
	}
}

func TestUserSubscribeActivationUpdateColumnsStayFocused(t *testing.T) {
	want := []string{"id", "user_id", "subscribe_id", "product_snapshot", "start_time", "expire_time", "finished_at", "download", "upload", "traffic_reset_at", "token", "status", "reset_cycle", "next_traffic_reset_at"}
	if !reflect.DeepEqual(userSubscribeActivationUpdateColumns, want) {
		t.Fatalf("user subscribe activation update columns = %#v, want %#v", userSubscribeActivationUpdateColumns, want)
	}
}

func TestUserSubscribeProductSummaryColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeProductSummaryColumns))
	for _, column := range userSubscribeProductSummaryColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"unit_price",
		"unit_time",
		"discount",
		"replacement",
		"traffic",
		"speed_limit",
		"device_limit",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
		"show_original_price",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user subscribe product summary columns must include %q", column)
		}
	}

	forbidden := []string{
		"language",
		"description",
		"inventory",
		"quota",
		"nodes",
		"node_tags",
		"show",
		"user_visible",
		"sell",
		"sort",
		"deduction_ratio",
		"renewal_reset",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user subscribe product summary columns must not include %q", column)
		}
	}
}

func TestUserSubscribeAdminListProductColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeAdminListProductColumns))
	for _, column := range userSubscribeAdminListProductColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "speed_limit", "device_limit"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("admin user subscribe list product columns must include %q", column)
		}
	}

	forbidden := []string{
		"language",
		"description",
		"unit_price",
		"unit_time",
		"discount",
		"replacement",
		"inventory",
		"traffic",
		"quota",
		"nodes",
		"node_tags",
		"deduction_ratio",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
		"renewal_reset",
		"show_original_price",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("admin user subscribe list product columns must not include heavy column %q", column)
		}
	}
}

func TestUserSubscribeCacheKeysIncludeRuntimeToken(t *testing.T) {
	keys := (&Subscribe{Id: 1, UserId: 2, Token: "subscribe-token"}).GetCacheKeys()
	for _, want := range []string{
		cacheUserSubscribeTokenPrefix + "subscribe-token",
		cacheUserSubscribeRuntimeTokenPrefix + "subscribe-token",
		cacheUserSubscribeExpiryTokenPrefix + "subscribe-token",
		cacheUserSubscribeUserPrefix + "2",
		cacheUserSubscribeIdPrefix + "1",
	} {
		if !containsString(keys, want) {
			t.Fatalf("subscribe cache keys = %#v, want %q", keys, want)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func countString(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func TestFindOneSubscribeForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	if err := db.Create(&Subscribe{
		Id:              20,
		UserId:          7,
		OrderId:         8,
		SubscribeId:     9,
		ProductName:     "private plan",
		ProductSnapshot: `{"nodes":"private"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "cache-token",
		UUID:            "private-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customUserModel)
	got, err := model.findOneSubscribeForCache(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 20 || got.UserId != 7 || got.Token != "cache-token" {
		t.Fatalf("cache query dropped cache fields: %#v", got)
	}
	assertSubscribeCacheProjection(t, got)
}

func TestFindOneSubscribeByOrderIdSelectsOnlyLookupColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Subscribe{
		Id:              24,
		UserId:          7,
		OrderId:         88,
		SubscribeId:     9,
		ProductName:     "private plan",
		ProductSnapshot: `{"nodes":"private"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "order-lookup-token",
		UUID:            "private-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneSubscribeByOrderId(ctx, 88)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 24 || got.UserId != 7 || got.Token != "order-lookup-token" {
		t.Fatalf("order lookup query dropped required fields: %#v", got)
	}
	assertSubscribeCacheProjection(t, got)
}

func TestFindOneSubscribeForPresetSelectsOnlyPresetColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	if err := db.Create(&Subscribe{
		Id:              23,
		UserId:          9,
		OrderId:         10,
		SubscribeId:     11,
		ProductName:     "preset plan",
		ProductSnapshot: `{"version":1,"kind":"subscribe","nodes":"12"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         8192,
		Download:        512,
		Upload:          1024,
		Token:           "preset-token",
		UUID:            "preset-uuid",
		Status:          1,
		Note:            "preset note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneSubscribeForPreset(ctx, 23)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 23 || got.UserId != 9 || got.SubscribeId != 11 || got.ProductSnapshot == "" {
		t.Fatalf("preset query dropped preset fields: %#v", got)
	}
	if got.OrderId != 0 ||
		got.ProductName != "" ||
		!got.StartTime.IsZero() ||
		!got.ExpireTime.IsZero() ||
		got.FinishedAt != nil ||
		got.Traffic != 0 ||
		got.Download != 0 ||
		got.Upload != 0 ||
		got.Token != "" ||
		got.UUID != "" ||
		got.Status != 0 ||
		got.Note != "" ||
		!got.CreatedAt.IsZero() ||
		!got.UpdatedAt.IsZero() {
		t.Fatalf("preset query loaded non-preset subscribe fields: %#v", got)
	}
}

func TestFindOneSubscribeForMutationSelectsOnlyMutationColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	nextResetAt := now.Add(24 * time.Hour)
	if err := db.Create(&Subscribe{
		Id:                 27,
		UserId:             12,
		OrderId:            33,
		SubscribeId:        44,
		ProductName:        "private plan",
		ProductSnapshot:    `{"version":1,"kind":"subscribe","nodes":"12"}`,
		StartTime:          now,
		ExpireTime:         now.Add(2 * time.Hour),
		FinishedAt:         &finishedAt,
		Traffic:            8192,
		Download:           512,
		Upload:             1024,
		ResetCycle:         2,
		NextTrafficResetAt: &nextResetAt,
		Token:              "mutation-token",
		UUID:               "mutation-uuid",
		Status:             2,
		Note:               "private note",
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneSubscribeForMutation(ctx, 27)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 27 ||
		got.UserId != 12 ||
		got.OrderId != 33 ||
		got.SubscribeId != 44 ||
		got.ProductName != "private plan" ||
		got.ProductSnapshot == "" ||
		!got.StartTime.Equal(now) ||
		!got.ExpireTime.Equal(now.Add(2*time.Hour)) ||
		got.FinishedAt == nil ||
		!got.FinishedAt.Equal(finishedAt) ||
		got.Traffic != 8192 ||
		got.Download != 512 ||
		got.Upload != 1024 ||
		got.ResetCycle != 2 ||
		got.NextTrafficResetAt == nil ||
		!got.NextTrafficResetAt.Equal(nextResetAt) ||
		got.Token != "mutation-token" ||
		got.Status != 2 {
		t.Fatalf("mutation query dropped required fields: %#v", got)
	}
	if got.UUID != "" ||
		got.Note != "" ||
		!got.CreatedAt.IsZero() ||
		!got.UpdatedAt.IsZero() {
		t.Fatalf("mutation query loaded non-mutation subscribe fields: %#v", got)
	}
}

func TestFindOneSubscribeByTokenForActivationUpdateSelectsOnlyActivationColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	nextResetAt := now.Add(24 * time.Hour)
	trafficResetAt := now.Add(-time.Hour)
	if err := db.Create(&Subscribe{
		Id:                 25,
		UserId:             9,
		OrderId:            10,
		SubscribeId:        11,
		ProductName:        "private plan",
		ProductSnapshot:    `{"version":1,"kind":"subscribe","name":"activation","nodes":"1"}`,
		StartTime:          now,
		ExpireTime:         now.Add(2 * time.Hour),
		FinishedAt:         &finishedAt,
		Traffic:            4096,
		Download:           128,
		Upload:             256,
		TrafficResetAt:     trafficResetAt,
		ResetCycle:         2,
		NextTrafficResetAt: &nextResetAt,
		Token:              "activation-token",
		UUID:               "runtime-uuid",
		Status:             1,
		Note:               "private note",
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneSubscribeByTokenForActivationUpdate(ctx, "activation-token")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 25 ||
		got.UserId != 9 ||
		got.SubscribeId != 11 ||
		got.ProductSnapshot == "" ||
		!got.StartTime.Equal(now) ||
		!got.ExpireTime.Equal(now.Add(2*time.Hour)) ||
		got.FinishedAt == nil ||
		got.Download != 128 ||
		got.Upload != 256 ||
		!got.TrafficResetAt.Equal(trafficResetAt) ||
		got.ResetCycle != 2 ||
		got.NextTrafficResetAt == nil ||
		!got.NextTrafficResetAt.Equal(nextResetAt) ||
		got.Token != "activation-token" ||
		got.Status != 1 {
		t.Fatalf("activation update query dropped required fields: %#v", got)
	}
	if got.OrderId != 0 ||
		got.ProductName != "" ||
		got.Traffic != 0 ||
		got.UUID != "" ||
		got.Note != "" ||
		!got.CreatedAt.IsZero() ||
		!got.UpdatedAt.IsZero() {
		t.Fatalf("activation update query loaded non-activation fields: %#v", got)
	}
}

func TestFindOneSubscribeByTokenForRuntimeSelectsOnlyRuntimeColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	if err := db.Create(&Subscribe{
		Id:              22,
		UserId:          9,
		OrderId:         10,
		SubscribeId:     11,
		ProductName:     "private plan",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"runtime","nodes":"1"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "runtime-token",
		UUID:            "runtime-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneSubscribeByTokenForRuntime(ctx, "runtime-token")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 22 ||
		got.UserId != 9 ||
		got.SubscribeId != 11 ||
		got.ProductSnapshot == "" ||
		!got.ExpireTime.Equal(now.Add(2*time.Hour)) ||
		got.Traffic != 4096 ||
		got.Download != 128 ||
		got.Upload != 256 ||
		got.Token != "runtime-token" ||
		got.UUID != "runtime-uuid" ||
		got.Status != 1 {
		t.Fatalf("runtime query dropped required fields: %#v", got)
	}
	if got.OrderId != 0 ||
		got.ProductName != "" ||
		!got.StartTime.IsZero() ||
		got.FinishedAt != nil ||
		got.Note != "" ||
		!got.CreatedAt.IsZero() ||
		!got.UpdatedAt.IsZero() {
		t.Fatalf("runtime query loaded non-runtime fields: %#v", got)
	}
}

func TestFindOneSubscribeByTokenForExpirySelectsOnlyExpiryColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	if err := db.Create(&Subscribe{
		Id:              23,
		UserId:          9,
		OrderId:         10,
		SubscribeId:     11,
		ProductName:     "private plan",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"expiry","nodes":"1"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "expiry-token",
		UUID:            "private-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneSubscribeByTokenForExpiry(ctx, "expiry-token")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 23 ||
		got.SubscribeId != 11 ||
		got.ProductSnapshot == "" ||
		!got.ExpireTime.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("expiry query dropped required fields: %#v", got)
	}
	if got.UserId != 0 ||
		got.OrderId != 0 ||
		got.ProductName != "" ||
		!got.StartTime.IsZero() ||
		got.FinishedAt != nil ||
		got.Traffic != 0 ||
		got.Download != 0 ||
		got.Upload != 0 ||
		got.Token != "" ||
		got.UUID != "" ||
		got.Status != 0 ||
		got.Note != "" ||
		!got.CreatedAt.IsZero() ||
		!got.UpdatedAt.IsZero() {
		t.Fatalf("expiry query loaded non-expiry fields: %#v", got)
	}
}

func TestFindOneSubscribeByTokenForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Subscribe{
		Id:              21,
		UserId:          8,
		OrderId:         9,
		SubscribeId:     10,
		ProductName:     "private plan",
		ProductSnapshot: `{"nodes":"private"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		Traffic:         8192,
		Download:        256,
		Upload:          512,
		Token:           "cache-token-by-token",
		UUID:            "private-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customUserModel)
	got, err := model.findOneSubscribeByTokenForCache(ctx, "cache-token-by-token")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 21 || got.UserId != 8 || got.Token != "cache-token-by-token" {
		t.Fatalf("cache query dropped cache fields: %#v", got)
	}
	assertSubscribeCacheProjection(t, got)
}

func TestUpdateSubscribeFieldsUpdatesOnlyRequestedFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	if err := db.Create(&Subscribe{
		Id:              28,
		UserId:          9,
		OrderId:         10,
		SubscribeId:     11,
		ProductName:     "stable plan",
		ProductSnapshot: `{"name":"stable"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "old-mutation-token",
		UUID:            "old-mutation-uuid",
		Status:          2,
		Note:            "old note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	cacheKeys := []string{
		cacheUserSubscribeTokenPrefix + "old-mutation-token",
		cacheUserSubscribeRuntimeTokenPrefix + "old-mutation-token",
		cacheUserSubscribeExpiryTokenPrefix + "old-mutation-token",
		cacheUserSubscribeTokenPrefix + "new-mutation-token",
		cacheUserSubscribeRuntimeTokenPrefix + "new-mutation-token",
		cacheUserSubscribeExpiryTokenPrefix + "new-mutation-token",
		cacheUserSubscribeUserPrefix + "9",
		cacheUserSubscribeIdPrefix + "28",
	}
	for _, key := range cacheKeys {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	newToken := "new-mutation-token"
	newUUID := "new-mutation-uuid"
	newSubscribeID := int64(12)
	newProductName := "new stable plan"
	newProductSnapshot := `{"name":"new-stable"}`
	newExpire := now.Add(72 * time.Hour)
	traffic := int64(8192)
	download := int64(0)
	upload := int64(0)
	status := uint8(1)
	note := "new note"
	model := NewModel(db, rds)
	if err := model.UpdateSubscribeFields(ctx, &Subscribe{
		Id:              28,
		UserId:          9,
		SubscribeId:     11,
		ProductSnapshot: `{"name":"stable"}`,
		Token:           "old-mutation-token",
	}, UserSubscribeFieldsUpdate{
		SubscribeID:     &newSubscribeID,
		ProductName:     &newProductName,
		ProductSnapshot: &newProductSnapshot,
		ExpireTime:      &newExpire,
		Traffic:         &traffic,
		Token:           &newToken,
		UUID:            &newUUID,
		Download:        &download,
		Upload:          &upload,
		ClearFinishedAt: true,
		Status:          &status,
		Note:            &note,
	}); err != nil {
		t.Fatal(err)
	}

	var got Subscribe
	if err := db.First(&got, 28).Error; err != nil {
		t.Fatal(err)
	}
	if got.Token != newToken || got.UUID != newUUID || got.Download != 0 || got.Upload != 0 || got.Status != 1 || got.Note != note {
		t.Fatalf("subscribe fields not updated: %#v", got)
	}
	if got.SubscribeId != newSubscribeID || got.ProductName != newProductName || got.ProductSnapshot != newProductSnapshot || !got.ExpireTime.Equal(newExpire) || got.Traffic != traffic {
		t.Fatalf("subscribe plan fields not updated: %#v", got)
	}
	if got.OrderId != 10 ||
		!got.StartTime.Equal(now) ||
		got.FinishedAt != nil ||
		!got.CreatedAt.Equal(now) {
		t.Fatalf("unrelated subscribe fields changed: %#v", got)
	}
	for _, key := range cacheKeys {
		if rds.Exists(ctx, key).Val() != 0 {
			t.Fatalf("cache key %q was not cleared", key)
		}
	}
}

func TestUpdateSubscribeActivationFieldsUpdatesOnlyRequestedFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	finishedAt := now.Add(time.Hour)
	if err := db.Create(&Subscribe{
		Id:              26,
		UserId:          9,
		OrderId:         10,
		SubscribeId:     11,
		ProductName:     "old plan",
		ProductSnapshot: `{"name":"old"}`,
		StartTime:       now,
		ExpireTime:      now.Add(2 * time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "activation-update-token",
		UUID:            "stable-uuid",
		Status:          2,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	cacheKeys := []string{
		cacheUserSubscribeTokenPrefix + "activation-update-token",
		cacheUserSubscribeRuntimeTokenPrefix + "activation-update-token",
		cacheUserSubscribeExpiryTokenPrefix + "activation-update-token",
		cacheUserSubscribeUserPrefix + "9",
		cacheUserSubscribeIdPrefix + "26",
	}
	for _, key := range cacheKeys {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	newExpire := now.Add(48 * time.Hour)
	model := NewModel(db, rds)
	if err := model.UpdateSubscribeActivationFields(ctx, &Subscribe{Id: 26, UserId: 9, Token: "activation-update-token"}, UserSubscribeActivationUpdate{
		ExpireTime:      UpdateValue(newExpire),
		ClearFinishedAt: true,
		Download:        UpdateValue(int64(0)),
		Upload:          UpdateValue(int64(0)),
		Status:          UpdateValue(uint8(1)),
		ProductName:     UpdateValue("new plan"),
		ProductSnapshot: UpdateValue(`{"name":"new"}`),
	}); err != nil {
		t.Fatal(err)
	}

	var got Subscribe
	if err := db.First(&got, 26).Error; err != nil {
		t.Fatal(err)
	}
	if !got.ExpireTime.Equal(newExpire) || got.FinishedAt != nil || got.Download != 0 || got.Upload != 0 || got.Status != 1 || got.ProductName != "new plan" || got.ProductSnapshot != `{"name":"new"}` {
		t.Fatalf("activation fields not updated: %#v", got)
	}
	if got.OrderId != 10 || got.SubscribeId != 11 || !got.StartTime.Equal(now) || got.Traffic != 4096 || got.Token != "activation-update-token" || got.UUID != "stable-uuid" || got.Note != "private note" || !got.CreatedAt.Equal(now) {
		t.Fatalf("unrelated subscribe fields changed: %#v", got)
	}
	for _, key := range cacheKeys {
		if rds.Exists(ctx, key).Val() != 0 {
			t.Fatalf("cache key %q was not cleared", key)
		}
	}
}

func assertSubscribeCacheProjection(t *testing.T, got *Subscribe) {
	t.Helper()
	if got.OrderId != 0 ||
		got.SubscribeId != 0 ||
		got.ProductName != "" ||
		got.ProductSnapshot != "" ||
		!got.StartTime.IsZero() ||
		!got.ExpireTime.IsZero() ||
		got.FinishedAt != nil ||
		got.Traffic != 0 ||
		got.Download != 0 ||
		got.Upload != 0 ||
		got.UUID != "" ||
		got.Status != 0 ||
		got.Note != "" ||
		!got.CreatedAt.IsZero() ||
		!got.UpdatedAt.IsZero() {
		t.Fatalf("cache query loaded non-cache subscribe fields: %#v", got)
	}
}

func TestUserSubscribeAdminDetailColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeAdminDetailColumns))
	for _, column := range userSubscribeAdminDetailColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"user_id",
		"order_id",
		"subscribe_id",
		"product_name",
		"product_snapshot",
		"start_time",
		"expire_time",
		"finished_at",
		"traffic",
		"download",
		"upload",
		"traffic_reset_at",
		"reset_cycle",
		"next_traffic_reset_at",
		"token",
		"status",
		"note",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("admin subscribe detail columns must include %q", column)
		}
	}

	forbidden := []string{"uuid"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("admin subscribe detail columns must not include private/unused column %q", column)
		}
	}
}

func TestUserSubscribeAdminDetailUserColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeAdminDetailUserColumns))
	for _, column := range userSubscribeAdminDetailUserColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"avatar",
		"balance",
		"commission",
		"gift_amount",
		"created_at",
		"updated_at",
		"deleted_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("admin subscribe detail user columns must include %q", column)
		}
	}

	forbidden := []string{"password", "algo", "salt", "rules"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("admin subscribe detail user columns must not include private/heavy column %q", column)
		}
	}
}

func TestFindOneSubscribeDetailsByIdSelectsOnlyDetailColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}, &Subscribe{}, &modelSubscribe.Subscribe{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	enable := true
	if err := db.Create(&User{
		Id:         7,
		Password:   "private-password",
		Algo:       "bcrypt",
		Salt:       "private-salt",
		Avatar:     "avatar.png",
		Balance:    100,
		Commission: 200,
		GiftAmount: 300,
		Enable:     &enable,
		Rules:      `["private-rule"]`,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&modelSubscribe.Subscribe{
		Id:       11,
		Name:     "Plan A",
		Discount: `[{"months":1,"discount":90}]`,
		Traffic:  1024,
		Nodes:    "1,2",
		NodeTags: "stable",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Subscribe{
		Id:              99,
		UserId:          7,
		OrderId:         8,
		SubscribeId:     11,
		ProductName:     "Snapshot Plan",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"Snapshot Plan"}`,
		StartTime:       now,
		ExpireTime:      now.Add(time.Hour),
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "visible-token",
		UUID:            "private-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	got, err := model.FindOneSubscribeDetailsById(ctx, 99)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 99 || got.UserId != 7 || got.Token != "visible-token" || got.ProductSnapshot == "" {
		t.Fatalf("detail query dropped response fields: %#v", got)
	}
	if got.Note != "private note" || got.UUID != "" {
		t.Fatalf("detail query note/UUID projection mismatch: note=%q uuid=%q", got.Note, got.UUID)
	}
	if got.User == nil {
		t.Fatal("detail query did not preload user")
	}
	if got.User.Balance != 100 || got.User.Commission != 200 || got.User.GiftAmount != 300 {
		t.Fatalf("detail query dropped user response fields: %#v", got.User)
	}
	if got.User.Password != "" || got.User.Algo != "" || got.User.Salt != "" || got.User.Rules != "" {
		t.Fatalf("detail query loaded private/heavy user fields: %#v", got.User)
	}
	if got.Subscribe == nil {
		t.Fatal("detail query did not preload product")
	}
	if got.Subscribe.Name != "Plan A" || got.Subscribe.Discount == "" || got.Subscribe.Nodes == "" || got.Subscribe.NodeTags == "" {
		t.Fatalf("detail query dropped product response fields: %#v", got.Subscribe)
	}
}

func TestUserSubscribeActionProductColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(userSubscribeActionProductColumns))
	for _, column := range userSubscribeActionProductColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"unit_price",
		"unit_time",
		"discount",
		"replacement",
		"nodes",
		"node_tags",
		"deduction_ratio",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("user subscribe action product columns must include %q", column)
		}
	}

	forbidden := []string{"inventory", "show", "user_visible", "sell", "sort", "language", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("user subscribe action product columns must not include %q", column)
		}
	}
}

func TestUserSubscribeDeductionProductColumnsStayFocused(t *testing.T) {
	want := []string{"id", "unit_price", "unit_time", "deduction_ratio", "allow_deduction", "reset_cycle"}
	if !reflect.DeepEqual(userSubscribeDeductionProductColumns, want) {
		t.Fatalf("user subscribe deduction product columns = %#v, want %#v", userSubscribeDeductionProductColumns, want)
	}
}

func TestFindOneUserSubscribeForOrderActionSelectsOnlyActionColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &modelSubscribe.Subscribe{}); err != nil {
		t.Fatal(err)
	}

	allowRenewal := true
	allowResetTraffic := true
	if err := db.Create(&modelSubscribe.Subscribe{
		Id:                11,
		Name:              "Plan A",
		Description:       "description",
		UnitPrice:         1200,
		UnitTime:          "month",
		Discount:          `[{"months":1,"discount":90}]`,
		Replacement:       200,
		Traffic:           999,
		SpeedLimit:        100,
		DeviceLimit:       3,
		Quota:             2,
		Nodes:             "1,2",
		NodeTags:          "stable",
		AllowRenewal:      &allowRenewal,
		AllowResetTraffic: &allowResetTraffic,
		ResetCycle:        2,
	}).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Subscribe{
		Id:              99,
		UserId:          7,
		OrderId:         8,
		SubscribeId:     11,
		ProductName:     "Snapshot Plan",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"Snapshot Plan"}`,
		StartTime:       now,
		ExpireTime:      now.Add(time.Hour),
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "order-token",
		UUID:            "private-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	got, err := model.FindOneUserSubscribeForOrderAction(ctx, 99)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 99 || got.UserId != 7 || got.OrderId != 8 || got.SubscribeId != 11 || got.ProductSnapshot == "" || got.Token != "order-token" || got.Status != 1 {
		t.Fatalf("order action query dropped required subscription fields: %#v", got)
	}
	if !got.ExpireTime.Equal(now.Add(time.Hour)) {
		t.Fatalf("order action query dropped expire_time: %#v", got.ExpireTime)
	}
	if got.ProductName != "" || !got.StartTime.IsZero() || got.Traffic != 0 || got.Download != 0 || got.Upload != 0 || got.UUID != "" || got.Note != "" {
		t.Fatalf("order action query loaded unused/private subscription fields: %#v", got)
	}
	if got.Subscribe == nil {
		t.Fatal("order action query did not preload product")
	}
	if got.Subscribe.Name != "Plan A" || got.Subscribe.UnitPrice != 1200 || got.Subscribe.Discount == "" || got.Subscribe.Replacement != 200 || got.Subscribe.Nodes != "1,2" || got.Subscribe.NodeTags != "stable" {
		t.Fatalf("order action query dropped product snapshot fields: %#v", got.Subscribe)
	}
}

func TestFindOneUserSubscribeForDeductionSelectsOnlyDeductionColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Subscribe{}, &modelSubscribe.Subscribe{}); err != nil {
		t.Fatal(err)
	}

	allowDeduction := false
	if err := db.Create(&modelSubscribe.Subscribe{
		Id:             11,
		Name:           "Plan A",
		Description:    "long description",
		UnitPrice:      1200,
		UnitTime:       "month",
		Discount:       `[{"months":1,"discount":90}]`,
		Replacement:    200,
		Traffic:        999,
		SpeedLimit:     100,
		DeviceLimit:    3,
		Quota:          2,
		Nodes:          "1,2",
		NodeTags:       "stable",
		DeductionRatio: 80,
		AllowDeduction: &allowDeduction,
		ResetCycle:     2,
	}).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Subscribe{
		Id:              99,
		UserId:          7,
		OrderId:         8,
		SubscribeId:     11,
		ProductName:     "Snapshot Plan",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"Snapshot Plan"}`,
		StartTime:       now,
		ExpireTime:      now.Add(time.Hour),
		Traffic:         4096,
		Download:        128,
		Upload:          256,
		Token:           "private-token",
		UUID:            "private-uuid",
		Status:          1,
		Note:            "private note",
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	got, err := model.FindOneUserSubscribeForDeduction(ctx, 99)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 99 || got.OrderId != 8 || got.SubscribeId != 11 || !got.StartTime.Equal(now) || !got.ExpireTime.Equal(now.Add(time.Hour)) {
		t.Fatalf("deduction query dropped required subscription fields: %#v", got)
	}
	if got.Traffic != 4096 || got.Download != 128 || got.Upload != 256 || got.Status != 1 {
		t.Fatalf("deduction query dropped required traffic fields: %#v", got)
	}
	if got.UserId != 0 || got.ProductName != "" || got.ProductSnapshot != "" || got.Token != "" || got.UUID != "" || got.Note != "" {
		t.Fatalf("deduction query loaded unused/private subscription fields: %#v", got)
	}
	if got.Subscribe == nil {
		t.Fatal("deduction query did not preload product")
	}
	if got.Subscribe.UnitPrice != 1200 || got.Subscribe.UnitTime != "month" || got.Subscribe.DeductionRatio != 80 || got.Subscribe.ResetCycle != 2 {
		t.Fatalf("deduction query dropped product formula fields: %#v", got.Subscribe)
	}
	if got.Subscribe.AllowDeduction == nil || *got.Subscribe.AllowDeduction {
		t.Fatalf("deduction query did not load allow_deduction=false: %#v", got.Subscribe)
	}
	if got.Subscribe.Name != "" || got.Subscribe.Description != "" || got.Subscribe.Discount != "" || got.Subscribe.Replacement != 0 || got.Subscribe.Nodes != "" || got.Subscribe.NodeTags != "" {
		t.Fatalf("deduction query loaded unused product fields: %#v", got.Subscribe)
	}
}

func TestSubscribeDetailsToSubscribeCopiesRuntimeFields(t *testing.T) {
	finishedAt := time.Unix(300, 0)
	source := &SubscribeDetails{
		Id:              1,
		UserId:          2,
		OrderId:         3,
		SubscribeId:     4,
		ProductName:     "plan",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"plan"}`,
		StartTime:       time.Unix(100, 0),
		ExpireTime:      time.Unix(200, 0),
		FinishedAt:      &finishedAt,
		Traffic:         5,
		Download:        6,
		Upload:          7,
		Token:           "token",
		UUID:            "uuid",
		Status:          1,
		Note:            "note",
		CreatedAt:       time.Unix(400, 0),
		UpdatedAt:       time.Unix(500, 0),
	}

	got := source.ToSubscribe()
	want := &Subscribe{
		Id:              source.Id,
		UserId:          source.UserId,
		OrderId:         source.OrderId,
		SubscribeId:     source.SubscribeId,
		ProductName:     source.ProductName,
		ProductSnapshot: source.ProductSnapshot,
		StartTime:       source.StartTime,
		ExpireTime:      source.ExpireTime,
		FinishedAt:      source.FinishedAt,
		Traffic:         source.Traffic,
		Download:        source.Download,
		Upload:          source.Upload,
		Token:           source.Token,
		UUID:            source.UUID,
		Status:          source.Status,
		Note:            source.Note,
		CreatedAt:       source.CreatedAt,
		UpdatedAt:       source.UpdatedAt,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToSubscribe() = %#v, want %#v", got, want)
	}
}
