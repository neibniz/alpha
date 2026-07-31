package document

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDocumentListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(documentListColumns))
	for _, column := range documentListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "title", "tags", "show", "created_at", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("document list columns must include %q", column)
		}
	}

	if _, ok := columns["content"]; ok {
		t.Fatal("document list columns must not include text content")
	}
}

func TestDocumentDetailColumnsIncludeContent(t *testing.T) {
	columns := make(map[string]struct{}, len(documentDetailColumns))
	for _, column := range documentDetailColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "title", "content", "tags", "show", "created_at", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("document detail columns must include %q", column)
		}
	}
}

func TestQueryDocumentDetailUsesExplicitColumnsWithoutGroupPreload(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}

	show := true
	now := time.Unix(1700000000, 0)
	if err := db.Create(&Document{
		Id:        7,
		Title:     "detail",
		Content:   "full content",
		Tags:      "alpha,beta",
		Show:      &show,
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).QueryDocumentDetail(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 7 || got.Title != "detail" || got.Content != "full content" || got.Tags != "alpha,beta" {
		t.Fatalf("detail query dropped fields: %#v", got)
	}
	if got.Show == nil || !*got.Show || !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
		t.Fatalf("detail query dropped metadata: %#v", got)
	}
}

func TestQueryDocumentDetailReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}

	_, err = NewModel(db, nil).QueryDocumentDetail(ctx, 404)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("QueryDocumentDetail error = %v, want ErrRecordNotFound", err)
	}
}

func TestQueryDocumentListDoesNotLoadContent(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}

	show := true
	if err := db.Create(&Document{
		Id:      9,
		Title:   "list",
		Content: "large content should not be loaded",
		Tags:    "alpha",
		Show:    &show,
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).QueryDocumentList(ctx, 1, 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("total/list = %d/%d, want 1/1", total, len(list))
	}
	if list[0].Content != "" {
		t.Fatalf("document list loaded content: %#v", list[0])
	}
}

func TestDocumentListsPreallocatePageCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}

	show := true
	if err := db.Create([]*Document{
		{Id: 1, Title: "first", Tags: "alpha", Show: &show},
		{Id: 2, Title: "second", Tags: "beta", Show: &show},
		{Id: 3, Title: "third", Tags: "alpha", Show: &show},
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).QueryDocumentList(ctx, 1, 2, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 2 || cap(list) != 2 {
		t.Fatalf("document total/len/cap = %d/%d/%d, want 3/2/2", total, len(list), cap(list))
	}

	total, publicList, err := NewModel(db, nil).GetDocumentListByAll(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(publicList) != 2 || cap(publicList) != 2 {
		t.Fatalf("public document total/len/cap = %d/%d/%d, want 3/2/2", total, len(publicList), cap(publicList))
	}
}

func TestDocumentListQueriesKeepStableEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customDocumentModel)
	total, list, err := model.QueryDocumentList(ctx, 1, 10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("document list = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}

	publicTotal, publicList, err := model.GetDocumentListByAll(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if publicTotal != 0 || publicList == nil || len(publicList) != 0 {
		t.Fatalf("public document list = total %d list %#v, want total 0 and non-nil empty slice", publicTotal, publicList)
	}

	cacheRows, err := model.queryDocumentCacheRows(ctx, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if cacheRows == nil || len(cacheRows) != 0 {
		t.Fatalf("document cache rows = %#v, want non-nil empty slice", cacheRows)
	}
}

func TestUpdateDocumentClearsIDCacheAndPersistsZeroValues(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}
	show := true
	if err := db.Create(&Document{
		Id:      7,
		Title:   "existing",
		Content: "content",
		Tags:    "alpha",
		Show:    &show,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheDocumentIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	hidden := false
	if err := NewModel(db, rds).Update(ctx, &Document{Id: 7, Show: &hidden}); err != nil {
		t.Fatal(err)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("document id cache key still exists")
	}

	var got Document
	if err := db.First(&got, 7).Error; err != nil {
		t.Fatal(err)
	}
	if got.Title != "" || got.Content != "" || got.Tags != "" || got.Show == nil || *got.Show {
		t.Fatalf("zero fields were not persisted: %#v", got)
	}
}

func TestDeleteDocumentClearsIDCacheWithoutPreload(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}
	show := true
	if err := db.Create(&Document{Id: 7, Title: "delete", Show: &show}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheDocumentIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("alpha:test_document_delete_query_count", func(tx *gorm.DB) {
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
	if err := db.Model(&Document{}).Where("id = ?", 7).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted document count = %d, want 0", count)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("document id cache key still exists")
	}
	if err := model.Delete(ctx, 0); err != nil {
		t.Fatal(err)
	}
}

func TestBatchDeleteNormalizesIDsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Document{}); err != nil {
		t.Fatal(err)
	}
	show := true
	if err := db.Create([]*Document{
		{Id: 7, Title: "first", Show: &show},
		{Id: 8, Title: "second", Show: &show},
		{Id: 9, Title: "third", Show: &show},
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, id := range []string{"7", "8", "9"} {
		if err := rds.Set(ctx, cacheDocumentIdPrefix+id, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	if err := NewModel(db, rds).BatchDelete(ctx, []int64{7, 0, 7, -1, 8}); err != nil {
		t.Fatal(err)
	}

	var remaining []int64
	if err := db.Model(&Document{}).Order("id ASC").Pluck("id", &remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0] != 9 {
		t.Fatalf("remaining document ids = %#v, want [9]", remaining)
	}
	if exists := rds.Exists(ctx, cacheDocumentIdPrefix+"7", cacheDocumentIdPrefix+"8").Val(); exists != 0 {
		t.Fatalf("deleted document cache keys still exist")
	}
	if exists := rds.Exists(ctx, cacheDocumentIdPrefix+"9").Val(); exists != 1 {
		t.Fatalf("untouched document cache key was removed")
	}
}

func TestNormalizeDocumentIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	ids := make([]int64, smallDocumentSetThreshold+20)
	ids[2] = 7
	ids[smallDocumentSetThreshold+10] = 7

	got := normalizeDocumentIDs(ids)
	if want := []int64{7}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("normalizeDocumentIDs() = %#v, want %#v", got, want)
	}
	if cap(got) != smallDocumentSetThreshold+1 {
		t.Fatalf("normalized document id cap = %d, want small initial cap", cap(got))
	}
}

func TestNormalizeDocumentIDsBoundsSparseUpgradeCapacity(t *testing.T) {
	ids := make([]int64, 0, smallDocumentSetThreshold+40)
	for id := int64(1); id <= smallDocumentSetThreshold+1; id++ {
		ids = append(ids, id)
	}
	for len(ids) < smallDocumentSetThreshold+40 {
		ids = append(ids, 1)
	}

	got := normalizeDocumentIDs(ids)
	if len(got) != smallDocumentSetThreshold+1 {
		t.Fatalf("normalizeDocumentIDs() len = %d, want %d", len(got), smallDocumentSetThreshold+1)
	}
	if maxCap := len(got) * 2; cap(got) > maxCap {
		t.Fatalf("normalized document id cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestDocumentCacheKeyCollectorSkipsEmptyAndDuplicatesWithoutInputSizedCapacity(t *testing.T) {
	collector := newDocumentCacheKeyCollector(smallDocumentSetThreshold + 20)
	for i := 0; i < smallDocumentSetThreshold+20; i++ {
		collector.Add("")
		collector.Add(cacheDocumentIdPrefix + "7")
	}

	keys := collector.Values()
	if want := []string{cacheDocumentIdPrefix + "7"}; len(keys) != len(want) || keys[0] != want[0] {
		t.Fatalf("document cache keys = %#v, want %#v", keys, want)
	}
	if cap(keys) != smallDocumentSetThreshold+1 {
		t.Fatalf("document cache key cap = %d, want small initial cap", cap(keys))
	}
}

func TestDocumentCacheKeyCollectorBoundsSparseUpgradeCapacity(t *testing.T) {
	collector := newDocumentCacheKeyCollector(smallDocumentSetThreshold + 40)
	for i := 0; i <= smallDocumentSetThreshold; i++ {
		collector.Add(cacheDocumentIdPrefix + string(rune('a'+i)))
	}
	for i := 0; i < smallDocumentSetThreshold+40; i++ {
		collector.Add(cacheDocumentIdPrefix + "a")
	}

	keys := collector.Values()
	if collector.seen == nil {
		t.Fatal("document cache key collector should allocate map after actual unique keys exceed threshold")
	}
	if maxCap := len(keys) * 2; cap(keys) > maxCap {
		t.Fatalf("document cache key cap = %d, want <= %d", cap(keys), maxCap)
	}
}

func TestBatchDocumentCacheKeysDeduplicates(t *testing.T) {
	keys := (&defaultDocumentModel{}).batchGetCacheKeys(
		&Document{Id: 7},
		&Document{Id: 7},
		nil,
	)
	want := []string{cacheDocumentIdPrefix + "7"}
	if len(keys) != len(want) || keys[0] != want[0] {
		t.Fatalf("document cache keys = %#v, want %#v", keys, want)
	}
}
