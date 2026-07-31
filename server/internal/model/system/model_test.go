package system

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

func TestSystemConfigColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(systemConfigColumns))
	for _, column := range systemConfigColumns {
		columns[column] = struct{}{}
	}

	for _, column := range []string{"key", "value", "type"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("system config columns must include %q", column)
		}
	}

	for _, column := range []string{"id", "category", "desc", "created_at", "updated_at"} {
		if _, ok := columns[column]; ok {
			t.Fatalf("system config columns must not include %q", column)
		}
	}
}

func TestSystemLookupColumnsStayLightweight(t *testing.T) {
	if want := []string{"id", "key", "value"}; !reflect.DeepEqual(systemKeyLookupColumns, want) {
		t.Fatalf("system key lookup columns = %#v, want %#v", systemKeyLookupColumns, want)
	}
	if want := []string{"id", "key"}; !reflect.DeepEqual(systemCacheColumns, want) {
		t.Fatalf("system cache columns = %#v, want %#v", systemCacheColumns, want)
	}
	if want := []string{"value"}; !reflect.DeepEqual(systemValueColumns, want) {
		t.Fatalf("system value columns = %#v, want %#v", systemValueColumns, want)
	}
}

func TestSystemCacheKeysIncludeKeyLookup(t *testing.T) {
	keys := (&defaultSystemModel{}).getCacheKeys(&System{Id: 7, Key: "WebAD"})
	want := map[string]struct{}{
		cacheSystemIdPrefix + "7":      {},
		cacheSystemKeyPrefix + "WebAD": {},
	}
	if len(keys) != len(want) {
		t.Fatalf("cache keys = %#v, want %d keys", keys, len(want))
	}
	for _, key := range keys {
		if _, ok := want[key]; !ok {
			t.Fatalf("unexpected cache key %q in %#v", key, keys)
		}
	}
}

func TestSystemCacheKeysAvoidMapForSmallFixedSet(t *testing.T) {
	model := &defaultSystemModel{}
	if got := model.getCacheKeys(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil cache keys = %#v, want non-nil empty slice", got)
	}
	if got := model.getCacheKeys(&System{Id: 9}); !reflect.DeepEqual(got, []string{cacheSystemIdPrefix + "9"}) {
		t.Fatalf("id-only cache keys = %#v", got)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = model.getCacheKeys(&System{Id: 9})
	})
	if allocs > 3 {
		t.Fatalf("id-only cache keys allocations = %f, want at most 3", allocs)
	}
}

func TestGetSiteConfigSelectsOnlyReflectionColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&System{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&System{
		Id:        1,
		Category:  "site",
		Key:       "SiteName",
		Value:     "Alpha",
		Type:      "string",
		Desc:      "large admin-only description",
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

	model := NewModel(db, rds)
	list, err := model.GetSiteConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}
	got := list[0]
	if got.Key != "SiteName" || got.Value != "Alpha" || got.Type != "string" {
		t.Fatalf("reflection fields not loaded: %#v", got)
	}
	if got.Id != 0 || got.Category != "" || got.Desc != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("non-reflection fields should not be loaded: %#v", got)
	}
}

func TestFindOneByKeySelectsOnlyLookupColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&System{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&System{
		Id:        2,
		Category:  "site",
		Key:       "WebAD",
		Value:     "true",
		Type:      "bool",
		Desc:      "large description",
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

	got, err := NewModel(db, rds).FindOneByKey(ctx, "WebAD")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 2 || got.Key != "WebAD" || got.Value != "true" {
		t.Fatalf("lookup fields not loaded: %#v", got)
	}
	if got.Category != "" || got.Type != "" || got.Desc != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("non-lookup fields should not be loaded: %#v", got)
	}
}

func TestFindSystemForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&System{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&System{
		Id:        4,
		Category:  "site",
		Key:       "SiteName",
		Value:     "Alpha",
		Type:      "string",
		Desc:      "large description",
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).(*customSystemModel).findOneForCache(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 4 || got.Key != "SiteName" {
		t.Fatalf("cache lookup returned %#v, want id/key", got)
	}
	if got.Category != "" || got.Value != "" || got.Type != "" || got.Desc != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("cache lookup loaded non-key fields: %#v", got)
	}
}

func TestFindNodeMultiplierConfigSelectsOnlyValue(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&System{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&System{
		Id:        3,
		Category:  "server",
		Key:       "NodeMultiplierConfig",
		Value:     "[]",
		Type:      "json",
		Desc:      "large description",
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindNodeMultiplierConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "[]" {
		t.Fatalf("value = %q, want []", got.Value)
	}
	if got.Id != 0 || got.Category != "" || got.Key != "" || got.Type != "" || got.Desc != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("non-value fields should not be loaded: %#v", got)
	}
}
