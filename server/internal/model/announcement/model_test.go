package announcement

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAnnouncementListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(announcementListColumns))
	for _, column := range announcementListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "title", "show", "pinned", "popup", "created_at", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("announcement list columns must include %q", column)
		}
	}

	if _, ok := columns["content"]; ok {
		t.Fatal("announcement list columns must not include text content")
	}
}

func TestGetAnnouncementListByPagePreallocatesPageCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Announcement{}); err != nil {
		t.Fatal(err)
	}
	show := true
	if err := db.Create([]*Announcement{
		{Id: 1, Title: "first", Show: &show},
		{Id: 2, Title: "second", Show: &show},
		{Id: 3, Title: "third", Show: &show},
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).GetAnnouncementListByPage(ctx, 1, 2, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 2 || cap(list) != 2 {
		t.Fatalf("announcement total/len/cap = %d/%d/%d, want 3/2/2", total, len(list), cap(list))
	}
}

func TestGetAnnouncementListByPageEmptyResultKeepsStableSlice(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Announcement{}); err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).GetAnnouncementListByPage(ctx, 1, 10, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("announcement page = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}
}

func TestUpdateAnnouncementClearsIDCacheAndPersistsZeroValues(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Announcement{}); err != nil {
		t.Fatal(err)
	}
	enabled := true
	if err := db.Create(&Announcement{
		Id:      7,
		Title:   "existing",
		Content: "content",
		Show:    &enabled,
		Pinned:  &enabled,
		Popup:   &enabled,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheAnnouncementIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	disabled := false
	if err := NewModel(db, rds).Update(ctx, &Announcement{Id: 7, Show: &disabled, Pinned: &disabled, Popup: &disabled}); err != nil {
		t.Fatal(err)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("announcement id cache key still exists")
	}

	var got Announcement
	if err := db.First(&got, 7).Error; err != nil {
		t.Fatal(err)
	}
	if got.Title != "" || got.Content != "" || got.Show == nil || *got.Show || got.Pinned == nil || *got.Pinned || got.Popup == nil || *got.Popup {
		t.Fatalf("zero fields were not persisted: %#v", got)
	}
}

func TestDeleteAnnouncementClearsIDCacheWithoutPreload(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Announcement{}); err != nil {
		t.Fatal(err)
	}
	show := true
	if err := db.Create(&Announcement{Id: 7, Title: "delete", Show: &show}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheAnnouncementIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("alpha:test_announcement_delete_query_count", func(tx *gorm.DB) {
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
	if err := db.Model(&Announcement{}).Where("id = ?", 7).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted announcement count = %d, want 0", count)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("announcement id cache key still exists")
	}
	if err := model.Delete(ctx, 0); err != nil {
		t.Fatal(err)
	}
}
