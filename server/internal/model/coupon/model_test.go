package coupon

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCouponCacheColumnsStayMinimal(t *testing.T) {
	columns := make(map[string]struct{}, len(couponCacheColumns))
	for _, column := range couponCacheColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "code"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("coupon cache columns must include %q", column)
		}
	}

	forbidden := []string{
		"name",
		"count",
		"type",
		"discount",
		"start_time",
		"expire_time",
		"user_limit",
		"subscribe",
		"used_count",
		"enable",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("coupon cache columns must not include %q", column)
		}
	}
}

func TestCouponListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(couponListColumns))
	for _, column := range couponListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"name",
		"code",
		"count",
		"type",
		"discount",
		"start_time",
		"expire_time",
		"user_limit",
		"used_count",
		"enable",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("coupon list columns must include %q", column)
		}
	}

	forbidden := []string{"subscribe", "created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("coupon list columns must not include %q", column)
		}
	}
}

func TestQueryCouponCacheRowsSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)
	enable := true
	if err := db.Create([]*Coupon{
		{
			Id:         1,
			Name:       "first",
			Code:       "FIRST",
			Count:      100,
			Type:       1,
			Discount:   20,
			StartTime:  1000,
			ExpireTime: 2000,
			UserLimit:  3,
			Subscribe:  "1,2",
			UsedCount:  9,
			Enable:     &enable,
		},
		{
			Id:         2,
			Name:       "second",
			Code:       "SECOND",
			Count:      50,
			Type:       2,
			Discount:   500,
			StartTime:  3000,
			ExpireTime: 4000,
			UserLimit:  1,
			Subscribe:  "3",
			UsedCount:  5,
			Enable:     &enable,
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customCouponModel)
	rows, err := model.queryCouponCacheRows(ctx, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("cache rows = %d, want 2", len(rows))
	}
	for _, row := range rows {
		if row.Id == 0 || row.Code == "" {
			t.Fatalf("cache row missing key fields: %#v", row)
		}
		if row.Name != "" || row.Count != 0 || row.Type != 0 || row.Discount != 0 || row.StartTime != 0 || row.ExpireTime != 0 || row.UserLimit != 0 || row.Subscribe != "" || row.UsedCount != 0 || row.Enable != nil {
			t.Fatalf("cache row loaded non-key fields: %#v", row)
		}
	}
}

func TestFindOneByCodeForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)
	enable := true
	if err := db.Create(&Coupon{
		Id:         3,
		Name:       "third",
		Code:       "THIRD",
		Count:      10,
		Type:       1,
		Discount:   15,
		StartTime:  1000,
		ExpireTime: 2000,
		UserLimit:  2,
		Subscribe:  "1",
		UsedCount:  4,
		Enable:     &enable,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customCouponModel)
	got, err := model.findOneByCodeForCache(ctx, "THIRD")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 3 || got.Code != "THIRD" {
		t.Fatalf("cache lookup returned %#v, want id/code", got)
	}
	if got.Name != "" || got.Count != 0 || got.Discount != 0 || got.Subscribe != "" || got.UsedCount != 0 || got.Enable != nil {
		t.Fatalf("cache lookup loaded non-key fields: %#v", got)
	}
}

func TestFindCouponForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)
	enable := true
	if err := db.Create(&Coupon{
		Id:         5,
		Name:       "fifth",
		Code:       "FIFTH",
		Count:      10,
		Type:       1,
		Discount:   15,
		StartTime:  1000,
		ExpireTime: 2000,
		UserLimit:  2,
		Subscribe:  "1",
		UsedCount:  4,
		Enable:     &enable,
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil).(*customCouponModel)
	got, err := model.findOneForCache(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 5 || got.Code != "FIFTH" {
		t.Fatalf("cache lookup returned %#v, want id/code", got)
	}
	if got.Name != "" || got.Count != 0 || got.Discount != 0 || got.Subscribe != "" || got.UsedCount != 0 || got.Enable != nil {
		t.Fatalf("cache lookup loaded non-key fields: %#v", got)
	}
}

func TestQueryCouponListByPagePreallocatesPageCapacity(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)
	enable := true
	if err := db.Create([]*Coupon{
		{Id: 1, Name: "first", Code: "FIRST", Enable: &enable},
		{Id: 2, Name: "second", Code: "SECOND", Enable: &enable},
		{Id: 3, Name: "third", Code: "THIRD", Enable: &enable},
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).QueryCouponListByPage(ctx, 1, 2, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 2 || cap(list) != 2 {
		t.Fatalf("coupon total/len/cap = %d/%d/%d, want 3/2/2", total, len(list), cap(list))
	}
}

func TestQueryCouponListByPageSelectsOnlyListColumns(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)
	enable := true
	now := time.Now().UTC()
	if err := db.Create(&Coupon{
		Id:         9,
		Name:       "list",
		Code:       "LIST",
		Count:      100,
		Type:       1,
		Discount:   20,
		StartTime:  1000,
		ExpireTime: 2000,
		UserLimit:  3,
		Subscribe:  "1,2,3",
		UsedCount:  9,
		Enable:     &enable,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).QueryCouponListByPage(ctx, 1, 10, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("coupon total/len = %d/%d, want 1/1", total, len(list))
	}
	got := list[0]
	if got.Id != 9 || got.Name != "list" || got.Code != "LIST" || got.Count != 100 || got.UsedCount != 9 || got.Enable == nil || !*got.Enable {
		t.Fatalf("coupon list row missing required fields: %#v", got)
	}
	if got.Subscribe != "" || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("coupon list row loaded redundant fields: %#v", got)
	}
}

func TestCouponListQueriesKeepStableEmptySlices(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)

	model := NewModel(db, nil).(*customCouponModel)
	total, list, err := model.QueryCouponListByPage(ctx, 1, 10, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("coupon page = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}

	cacheRows, err := model.queryCouponCacheRows(ctx, []int64{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if cacheRows == nil || len(cacheRows) != 0 {
		t.Fatalf("coupon cache rows = %#v, want non-nil empty slice", cacheRows)
	}
}

func TestUpdateCountUsesLightweightLookupAndIncrements(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)
	enable := true
	if err := db.Create(&Coupon{
		Id:        4,
		Name:      "limited",
		Code:      "LIMITED",
		Count:     2,
		Type:      1,
		Discount:  25,
		UsedCount: 1,
		Enable:    &enable,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	model := NewModel(db, rds)
	if err := model.UpdateCount(ctx, "LIMITED"); err != nil {
		t.Fatal(err)
	}
	var got Coupon
	if err := db.First(&got, 4).Error; err != nil {
		t.Fatal(err)
	}
	if got.UsedCount != 2 {
		t.Fatalf("used count = %d, want 2", got.UsedCount)
	}
}

func TestBatchDeleteNormalizesIDs(t *testing.T) {
	ctx := context.Background()
	db := newCouponTestDB(t)
	enable := true
	if err := db.Create([]*Coupon{
		{Id: 10, Name: "first", Code: "FIRST", Enable: &enable},
		{Id: 11, Name: "second", Code: "SECOND", Enable: &enable},
		{Id: 12, Name: "third", Code: "THIRD", Enable: &enable},
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	model := NewModel(db, rds).(*customCouponModel)
	if err := model.BatchDelete(ctx, []int64{10, 0, 10, -1, 11}); err != nil {
		t.Fatal(err)
	}

	var remaining []int64
	if err := db.Model(&Coupon{}).Order("id ASC").Pluck("id", &remaining).Error; err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0] != 12 {
		t.Fatalf("remaining coupon ids = %#v, want [12]", remaining)
	}
}

func TestNormalizeCouponIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	ids := make([]int64, smallCouponSetThreshold+20)
	ids[2] = 10
	ids[smallCouponSetThreshold+10] = 10

	got := normalizeCouponIDs(ids)
	if want := []int64{10}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("normalizeCouponIDs() = %#v, want %#v", got, want)
	}
	if cap(got) != smallCouponSetThreshold+1 {
		t.Fatalf("normalized coupon id cap = %d, want small initial cap", cap(got))
	}
}

func TestNormalizeCouponIDsBoundsSparseUpgradeCapacity(t *testing.T) {
	ids := make([]int64, 0, smallCouponSetThreshold+40)
	for id := int64(1); id <= smallCouponSetThreshold+1; id++ {
		ids = append(ids, id)
	}
	for len(ids) < smallCouponSetThreshold+40 {
		ids = append(ids, 1)
	}

	got := normalizeCouponIDs(ids)
	if len(got) != smallCouponSetThreshold+1 {
		t.Fatalf("normalizeCouponIDs() len = %d, want %d", len(got), smallCouponSetThreshold+1)
	}
	if maxCap := len(got) * 2; cap(got) > maxCap {
		t.Fatalf("normalized coupon id cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestCouponCacheKeyCollectorSkipsEmptyAndDuplicatesWithoutInputSizedCapacity(t *testing.T) {
	collector := newCouponCacheKeyCollector(smallCouponSetThreshold + 20)
	for i := 0; i < smallCouponSetThreshold+20; i++ {
		collector.Add("")
		collector.Add(cacheCouponIdPrefix + "10")
	}

	keys := collector.Values()
	if want := []string{cacheCouponIdPrefix + "10"}; len(keys) != len(want) || keys[0] != want[0] {
		t.Fatalf("coupon cache keys = %#v, want %#v", keys, want)
	}
	if cap(keys) != smallCouponSetThreshold+1 {
		t.Fatalf("coupon cache key cap = %d, want small initial cap", cap(keys))
	}
}

func TestCouponCacheKeyCollectorBoundsSparseUpgradeCapacity(t *testing.T) {
	collector := newCouponCacheKeyCollector(smallCouponSetThreshold + 40)
	for i := 0; i <= smallCouponSetThreshold; i++ {
		collector.Add(cacheCouponIdPrefix + string(rune('a'+i)))
	}
	for i := 0; i < smallCouponSetThreshold+40; i++ {
		collector.Add(cacheCouponIdPrefix + "a")
	}

	keys := collector.Values()
	if collector.seen == nil {
		t.Fatal("coupon cache key collector should allocate map after actual unique keys exceed threshold")
	}
	if maxCap := len(keys) * 2; cap(keys) > maxCap {
		t.Fatalf("coupon cache key cap = %d, want <= %d", cap(keys), maxCap)
	}
}

func TestBatchCouponCacheKeysDeduplicates(t *testing.T) {
	keys := (&defaultCouponModel{}).batchGetCacheKeys(
		&Coupon{Id: 10, Code: "SAVE"},
		&Coupon{Id: 10, Code: "SAVE"},
		nil,
	)
	want := []string{
		cacheCouponIdPrefix + "10",
		cacheCouponCodePrefix + "SAVE",
	}
	if len(keys) != len(want) {
		t.Fatalf("coupon cache keys = %#v, want %#v", keys, want)
	}
	for index := range want {
		if keys[index] != want[index] {
			t.Fatalf("coupon cache keys = %#v, want %#v", keys, want)
		}
	}
}

func newCouponTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Coupon{}); err != nil {
		t.Fatal(err)
	}
	return db
}
