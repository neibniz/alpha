package ads

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdsListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(adsListColumns))
	for _, column := range adsListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "title", "type", "description", "target_url", "start_time", "end_time", "status", "created_at", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("ads list columns must include %q", column)
		}
	}

	if _, ok := columns["content"]; ok {
		t.Fatal("ads list columns must not include content")
	}
}

func TestAdsListSelectColumnsIncludeContentOnlyOnDemand(t *testing.T) {
	withoutContent := make(map[string]struct{}, len(adsListSelectColumns(false)))
	for _, column := range adsListSelectColumns(false) {
		withoutContent[column] = struct{}{}
	}
	if _, ok := withoutContent["content"]; ok {
		t.Fatal("ads admin detail columns must not include content by default")
	}

	withContent := make(map[string]struct{}, len(adsListSelectColumns(true)))
	for _, column := range adsListSelectColumns(true) {
		withContent[column] = struct{}{}
	}
	if _, ok := withContent["content"]; !ok {
		t.Fatal("ads detail columns must include content when requested")
	}
}

func TestGetAdsListByPagePreallocatesPageCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE ads (
			id integer primary key,
			title text not null default '',
			type text not null default '',
			content text,
			description text,
			target_url text,
			start_time datetime,
			end_time datetime,
			status integer default 0,
			created_at datetime,
			updated_at datetime
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*Ads{
		{Id: 1, Title: "first", Type: "banner", Status: 1},
		{Id: 2, Title: "second", Type: "banner", Status: 1},
		{Id: 3, Title: "third", Type: "modal", Status: 1},
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).GetAdsListByPage(ctx, 1, 2, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 2 || cap(list) != 2 {
		t.Fatalf("ads total/len/cap = %d/%d/%d, want 3/2/2", total, len(list), cap(list))
	}
}

func TestGetAdsListByPageEmptyResultKeepsStableSlice(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE ads (
			id integer primary key,
			title text not null default '',
			type text not null default '',
			content text,
			description text,
			target_url text,
			start_time datetime,
			end_time datetime,
			status integer default 0,
			created_at datetime,
			updated_at datetime
		)
	`).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).GetAdsListByPage(ctx, 1, 10, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("ads list = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}
}

func TestUpdateAdsClearsIDCacheAndPersistsZeroValues(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ads{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Ads{
		Id:          7,
		Title:       "existing",
		Type:        "banner",
		Content:     "content",
		Description: "description",
		TargetURL:   "https://example.com",
		Status:      1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheAdsIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	if err := NewModel(db, rds).Update(ctx, &Ads{Id: 7, Type: "banner", Status: 0}); err != nil {
		t.Fatal(err)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("ads id cache key still exists")
	}

	var got Ads
	if err := db.Select("id", "title", "content", "description", "target_url", "status").First(&got, 7).Error; err != nil {
		t.Fatal(err)
	}
	if got.Title != "" || got.Content != "" || got.Description != "" || got.TargetURL != "" || got.Status != 0 {
		t.Fatalf("zero fields were not persisted: %#v", got)
	}
}

func TestDeleteAdsClearsIDCacheWithoutPreload(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ads{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Ads{Id: 7, Title: "delete", Type: "banner"}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheAdsIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("alpha:test_ads_delete_query_count", func(tx *gorm.DB) {
		queryCount++
	}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, rds)
	if err := model.Delete(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if queryCount != 0 {
		t.Fatalf("Delete queried %d times, want 0", queryCount)
	}
	var count int64
	if err := db.Model(&Ads{}).Where("id = ?", 7).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted ads count = %d, want 0", count)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("ads id cache key still exists")
	}
	if err := model.Delete(ctx, 0); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAdsFieldsUpdatesOnlyRequestedFieldsAndPersistsZeroValues(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ads{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Ads{
		Id:          8,
		Title:       "existing",
		Type:        "banner",
		Content:     "content",
		Description: "description",
		TargetURL:   "https://example.com",
		Status:      1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheAdsIdPrefix + "8"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	emptyTitle := ""
	disabled := 0
	targetURL := ""
	if err := NewModel(db, rds).UpdateFields(ctx, &Ads{Id: 8}, AdsFieldsUpdate{
		Title:     &emptyTitle,
		TargetURL: &targetURL,
		Status:    &disabled,
	}); err != nil {
		t.Fatal(err)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("ads id cache key still exists")
	}

	var got Ads
	if err := db.Select("id", "title", "type", "content", "description", "target_url", "status").First(&got, 8).Error; err != nil {
		t.Fatal(err)
	}
	if got.Title != "" || got.TargetURL != "" || got.Status != 0 {
		t.Fatalf("requested zero fields were not persisted: %#v", got)
	}
	if got.Type != "banner" || got.Content != "content" || got.Description != "description" {
		t.Fatalf("unrequested fields changed: %#v", got)
	}
}
