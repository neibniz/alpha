package auth

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthCacheColumnsStayMinimal(t *testing.T) {
	if want := []string{"id", "method"}; !reflect.DeepEqual(authCacheColumns, want) {
		t.Fatalf("auth cache columns = %#v, want %#v", authCacheColumns, want)
	}
}

func TestAuthRuntimeColumnsStayFocused(t *testing.T) {
	if want := []string{"id", "method", "config", "enabled"}; !reflect.DeepEqual(authRuntimeColumns, want) {
		t.Fatalf("auth runtime columns = %#v, want %#v", authRuntimeColumns, want)
	}
}

func TestFindAuthForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Auth{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&Auth{
		Id:        7,
		Method:    "email",
		Config:    `{"secret":"private"}`,
		Enabled:   &enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).(*customAuthModel).findOneForCache(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 7 || got.Method != "email" {
		t.Fatalf("cache lookup returned %#v, want id/method", got)
	}
	if got.Config != "" || got.Enabled != nil || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("cache lookup loaded non-key fields: %#v", got)
	}
}

func TestFindOneByMethodSelectsOnlyRuntimeColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Auth{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&Auth{
		Id:        8,
		Method:    "google",
		Config:    `{"client_secret":"private"}`,
		Enabled:   &enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneByMethod(ctx, "google")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 8 || got.Method != "google" || got.Config == "" || got.Enabled == nil || !*got.Enabled {
		t.Fatalf("runtime auth lookup dropped required fields: %#v", got)
	}
	if !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("runtime auth lookup loaded timestamps: %#v", got)
	}
}

func TestFindAllAuthSelectsOnlySummaryColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Auth{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	disabled := false
	now := time.Unix(1700000000, 0)
	if err := db.Create([]*Auth{
		{
			Id:        2,
			Method:    "google",
			Config:    `{"client_secret":"private"}`,
			Enabled:   &enabled,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Id:        1,
			Method:    "email",
			Config:    `{"password":"private"}`,
			Enabled:   &disabled,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("FindAll returned %d rows, want 2", len(got))
	}
	if got[0].Id != 1 || got[0].Method != "email" || got[0].Enabled == nil || *got[0].Enabled {
		t.Fatalf("FindAll first row = %#v, want email summary", got[0])
	}
	if got[1].Id != 2 || got[1].Method != "google" || got[1].Enabled == nil || !*got[1].Enabled {
		t.Fatalf("FindAll second row = %#v, want google summary", got[1])
	}
	for _, item := range got {
		if item.Config != "" || !item.CreatedAt.IsZero() || !item.UpdatedAt.IsZero() {
			t.Fatalf("FindAll loaded non-summary fields: %#v", item)
		}
	}
}

func TestFindAllForGlobalConfigExcludesRemovedMethodsAndSecrets(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Auth{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	now := time.Unix(1700000000, 0)
	if err := db.Create([]*Auth{
		{
			Id:        1,
			Method:    "google",
			Config:    `{"client_secret":"private"}`,
			Enabled:   &enabled,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Id:        2,
			Method:    "device",
			Config:    `{"show_ads":true,"enable_security":true}`,
			Enabled:   &enabled,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindAllForGlobalConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("FindAllForGlobalConfig returned %d rows, want 1", len(got))
	}
	if got[0].Method != "google" || got[0].Config != "" {
		t.Fatalf("global config query loaded unexpected data: %#v", got[0])
	}
	for _, item := range got {
		if item.Id == 0 || item.Enabled == nil {
			t.Fatalf("global config query dropped summary fields: %#v", item)
		}
		if !item.CreatedAt.IsZero() || !item.UpdatedAt.IsZero() {
			t.Fatalf("global config query loaded timestamps: %#v", item)
		}
	}
}

func TestGetAuthListByPageSelectsOnlySummaryColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Auth{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&Auth{
		Id:        3,
		Method:    "telegram",
		Config:    `{"bot_token":"private"}`,
		Enabled:   &enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).GetAuthListByPage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("GetAuthListByPage returned %d rows, want 1", len(got))
	}
	if got[0].Id != 3 || got[0].Method != "telegram" || got[0].Enabled == nil || !*got[0].Enabled {
		t.Fatalf("GetAuthListByPage dropped summary fields: %#v", got[0])
	}
	if got[0].Config != "" || !got[0].CreatedAt.IsZero() || !got[0].UpdatedAt.IsZero() {
		t.Fatalf("GetAuthListByPage loaded non-summary fields: %#v", got[0])
	}
}

func TestAuthListMethodsReturnNonNilEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Auth{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	authList, err := model.GetAuthListByPage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if authList == nil || len(authList) != 0 {
		t.Fatalf("auth list = %#v, want non-nil empty slice", authList)
	}

	summaries, err := model.FindAllSummaries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summaries == nil || len(summaries) != 0 {
		t.Fatalf("auth summaries = %#v, want non-nil empty slice", summaries)
	}

	globalConfig, err := model.FindAllForGlobalConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if globalConfig == nil || len(globalConfig) != 0 {
		t.Fatalf("auth global config = %#v, want non-nil empty slice", globalConfig)
	}

	all, err := model.FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if all == nil || len(all) != 0 {
		t.Fatalf("all auth methods = %#v, want non-nil empty slice", all)
	}
}

func TestUpdateAuthFieldsUpdatesOnlyRequestedFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Auth{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&Auth{
		Id:      9,
		Method:  "email",
		Config:  `{"smtp":"keep"}`,
		Enabled: &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range []string{cacheAuthIdPrefix + "9", cacheAuthMethodPrefix + "email"} {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	disabled := false
	if err := NewModel(db, rds).UpdateFields(ctx, &Auth{Id: 9, Method: "email"}, AuthFieldsUpdate{
		Enabled: &disabled,
	}); err != nil {
		t.Fatal(err)
	}

	var got Auth
	if err := db.First(&got, 9).Error; err != nil {
		t.Fatal(err)
	}
	if got.Enabled == nil || *got.Enabled {
		t.Fatalf("enabled=false was not persisted: %#v", got)
	}
	if got.Config != `{"smtp":"keep"}` {
		t.Fatalf("unrequested config changed: %#v", got)
	}
	for _, key := range []string{cacheAuthIdPrefix + "9", cacheAuthMethodPrefix + "email"} {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}
