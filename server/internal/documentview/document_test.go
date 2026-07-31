package documentview

import (
	"reflect"
	"testing"
	"time"

	modelDocument "github.com/neibniz/alpha/server/internal/model/document"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestAdminListRowsSkipsNilAndMapsAdminShape(t *testing.T) {
	show := true
	createdAt := time.UnixMilli(1716192000000)
	updatedAt := time.UnixMilli(1716195600000)
	rows := []*modelDocument.Document{
		{
			Id:        7,
			Title:     "Admin document",
			Content:   "body should stay out of list shape",
			Tags:      "alpha, docs,,alpha",
			Show:      &show,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		nil,
	}

	got := AdminListRows(rows)
	if len(got) != 1 {
		t.Fatalf("admin document rows = %d, want 1: %#v", len(got), got)
	}
	if got[0].Id != 7 || got[0].Title != "Admin document" || !got[0].Show {
		t.Fatalf("admin document fields not mapped: %#v", got[0])
	}
	if got[0].CreatedAt != createdAt.UnixMilli() || got[0].UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("admin document timestamps not mapped: %#v", got[0])
	}
	if !reflect.DeepEqual(got[0].Tags, []string{"alpha", "docs"}) {
		t.Fatalf("admin document tags = %#v, want normalized unique tags", got[0].Tags)
	}
}

func TestAdminListRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelDocument.Document, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelDocument.Document{Id: 1, Title: "first"}
	rows[len(rows)-1] = &modelDocument.Document{Id: 2, Title: "second"}

	got := AdminListRows(rows)
	if len(got) != 2 {
		t.Fatalf("admin document rows = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy admin document rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("admin document row order changed: %#v", got)
	}
}

func TestPublicSummaryRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelDocument.Document, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelDocument.Document{Id: int64(index + 1), Title: "doc"}
	}

	got := PublicSummaryRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("public document rows = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("public document rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("public document row order changed: %#v", got)
	}
}

func TestDocumentRowsKeepEmptyListShape(t *testing.T) {
	admin := AdminListRows(nil)
	if admin == nil || len(admin) != 0 {
		t.Fatalf("empty admin rows should return non-nil empty slice: %#v", admin)
	}
	public := PublicSummaryRows(nil)
	if public == nil || len(public) != 0 {
		t.Fatalf("empty public rows should return non-nil empty slice: %#v", public)
	}
}

func TestPublicSummaryRowsKeepPublicListShape(t *testing.T) {
	show := true
	updatedAt := time.UnixMilli(1716195600000)
	rows := []*modelDocument.Document{
		{
			Id:        8,
			Title:     "Public document",
			Content:   "private list body",
			Tags:      "help, help, guide",
			Show:      &show,
			CreatedAt: time.UnixMilli(1716192000000),
			UpdatedAt: updatedAt,
		},
		{
			Id:        11,
			Title:     "Public document copy",
			Tags:      "help, help, guide",
			UpdatedAt: updatedAt,
		},
	}

	got := PublicSummaryRows(rows)
	if len(got) != 2 {
		t.Fatalf("public summary rows = %d, want 2: %#v", len(got), got)
	}
	if got[0].Id != 8 || got[0].Title != "Public document" || got[0].UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("public summary fields not mapped: %#v", got[0])
	}
	if !reflect.DeepEqual(got[0].Tags, []string{"help", "guide"}) {
		t.Fatalf("public summary tags = %#v, want normalized unique tags", got[0].Tags)
	}
	if !reflect.DeepEqual(got[1].Tags, got[0].Tags) {
		t.Fatalf("duplicate public summary tags = %#v, want %#v", got[1].Tags, got[0].Tags)
	}
}

func TestDocumentDetailsMapContentShowAndZeroTimes(t *testing.T) {
	show := false
	got := AdminDetail(&modelDocument.Document{
		Id:      9,
		Title:   "Detail",
		Content: "full body",
		Tags:    "ops",
		Show:    &show,
	})
	if got.Id != 9 || got.Title != "Detail" || got.Content != "full body" || got.Show {
		t.Fatalf("admin detail fields not mapped: %#v", got)
	}
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero detail timestamps should stay zero: %#v", got)
	}

	visible := true
	public := PublicDetail(&modelDocument.Document{
		Id:      10,
		Title:   "Public detail",
		Content: "public body",
		Show:    &visible,
	})
	if public.Id != 10 || public.Content != "public body" || !public.Show {
		t.Fatalf("public detail fields not mapped: %#v", public)
	}
}

func TestDocumentTagsCacheReusesRawTags(t *testing.T) {
	cache := newDocumentTagsCache(2)

	first := cache.Tags("alpha, docs,,alpha")
	second := cache.Tags("alpha, docs,,alpha")

	if !reflect.DeepEqual(first, []string{"alpha", "docs"}) || !reflect.DeepEqual(second, first) {
		t.Fatalf("cached document tags mismatch: first=%#v second=%#v", first, second)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate raw tags should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small document tags cache should not allocate map")
	}
}

func TestDocumentTagsCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newDocumentTagsCache(smallDocumentTagsCacheThreshold + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large document tags cache should not preallocate storage from hint alone: %#v", cache)
	}
	first := cache.Tags("alpha, docs,,alpha")
	second := cache.Tags("alpha, docs,,alpha")
	if !reflect.DeepEqual(first, second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate document tags should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newDocumentTagsCache(smallDocumentTagsCacheThreshold + 1)
	for i := 0; i <= smallDocumentTagsCacheThreshold; i++ {
		cache.Tags(string(rune('a' + i)))
	}
	cache.Tags("a")

	if len(cache.rows) != smallDocumentTagsCacheThreshold+1 || len(cache.index) != smallDocumentTagsCacheThreshold+1 {
		t.Fatalf("document tags cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), smallDocumentTagsCacheThreshold+1)
	}
	if !reflect.DeepEqual(cache.rows[0].tags, []string{"a"}) {
		t.Fatalf("document tags cache did not normalize first row: %#v", cache.rows[0].tags)
	}
}

func TestDocumentTagsCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newDocumentTagsCache(smallDocumentTagsCacheThreshold + 20)
	for i := 0; i <= smallDocumentTagsCacheThreshold; i++ {
		cache.Tags(string(rune('a' + i)))
	}
	if cache.index == nil {
		t.Fatal("document tags cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("document tags cache row cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallDocumentTagsCacheThreshold+20; i++ {
		cache.Tags("a")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("document tags cache row cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}
