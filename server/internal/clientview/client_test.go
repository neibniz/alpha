package clientview

import (
	"reflect"
	"testing"

	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestSubscribeClientListFromModelSkipsNilAndNormalizesDownloadLinks(t *testing.T) {
	got := SubscribeClientListFromModel([]*client.SubscribeApplication{
		nil,
		{
			Id:           7,
			Name:         "Clash",
			Description:  "client",
			Icon:         "icon",
			Scheme:       "clash",
			Enabled:      true,
			IsDefault:    true,
			DownloadLink: `{"ios":" https://ios.example ","windows":"  ","linux":"https://linux.example"}`,
		},
	})

	want := []types.SubscribeClient{
		{
			Id:          7,
			Name:        "Clash",
			Description: "client",
			Icon:        "icon",
			Scheme:      "clash",
			Enabled:     true,
			IsDefault:   true,
			DownloadLink: types.DownloadLink{
				IOS:   "https://ios.example",
				Linux: "https://linux.example",
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SubscribeClientListFromModel() = %#v, want %#v", got, want)
	}
}

func TestSubscribeClientListFromModelSkipsNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*client.SubscribeApplication, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &client.SubscribeApplication{Id: 1, Name: "first"}
	rows[len(rows)-1] = &client.SubscribeApplication{Id: 2, Name: "second"}

	got := SubscribeClientListFromModel(rows)
	if len(got) != 2 {
		t.Fatalf("subscribe client rows = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy subscribe client rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("subscribe client order changed: %#v", got)
	}
}

func TestSubscribeClientListFromModelUsesCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*client.SubscribeApplication, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &client.SubscribeApplication{Id: int64(index + 1), Name: "client"}
	}

	got := SubscribeClientListFromModel(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe client rows = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe client rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("subscribe client order changed: %#v", got)
	}
}

func TestDownloadLinkFromRawHandlesEmptyAndInvalidJSON(t *testing.T) {
	for _, raw := range []string{"  ", "{}", " {} ", "null", " null "} {
		if got := DownloadLinkFromRaw(raw); got != (types.DownloadLink{}) {
			t.Fatalf("empty download link %q = %#v, want empty", raw, got)
		}
	}
	if got := DownloadLinkFromRaw("{invalid"); got != (types.DownloadLink{}) {
		t.Fatalf("invalid download link = %#v, want empty", got)
	}
}

func TestDownloadLinkCacheReusesTrimmedRaw(t *testing.T) {
	cache := NewDownloadLinkCache(2)

	raw := `{"ios":" https://ios.example ","linux":"https://linux.example"}`
	first := cache.Link(" " + raw + " ")
	second := cache.Link(raw)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("cached download links differ: %#v vs %#v", first, second)
	}
	if len(cache.rows) != 1 {
		t.Fatalf("duplicate trimmed raw should use one cache row, got %d", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small download link cache should not allocate map")
	}
}

func TestDownloadLinkCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := NewDownloadLinkCache(smallDownloadLinkCacheThreshold + 1)
	if cache == nil || cache.rows != nil || cache.index != nil {
		t.Fatalf("large download link cache should not preallocate storage from hint alone: %#v", cache)
	}
	raw := `{"ios":" https://example.com/a "}`
	first := cache.Link(" " + raw + " ")
	second := cache.Link(raw)
	if !reflect.DeepEqual(first, second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate download links should stay on linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = NewDownloadLinkCache(smallDownloadLinkCacheThreshold + 1)
	for i := 0; i <= smallDownloadLinkCacheThreshold; i++ {
		_ = cache.Link(`{"ios":"https://example.com/` + string(rune('a'+i)) + `"}`)
	}
	_ = cache.Link(`{"ios":"https://example.com/a"}`)

	if len(cache.rows) != smallDownloadLinkCacheThreshold+1 || len(cache.index) != smallDownloadLinkCacheThreshold+1 {
		t.Fatalf("download link cache rows/index = %d/%d, want %d", len(cache.rows), len(cache.index), smallDownloadLinkCacheThreshold+1)
	}
	if cache.rows[0].link.IOS != "https://example.com/a" {
		t.Fatalf("download link cache did not parse first row: %#v", cache.rows[0].link)
	}
}

func TestDownloadLinkCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := NewDownloadLinkCache(smallDownloadLinkCacheThreshold + 20)
	for i := 0; i <= smallDownloadLinkCacheThreshold; i++ {
		_ = cache.Link(`{"ios":"https://example.com/` + string(rune('a'+i)) + `"}`)
	}
	if cache.index == nil {
		t.Fatal("download link cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("download link cache row cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallDownloadLinkCacheThreshold+20; i++ {
		_ = cache.Link(`{"ios":"https://example.com/a"}`)
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("download link cache row cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestDownloadLinkToModelTrimsAndOmitsEmptyFields(t *testing.T) {
	modelLink := DownloadLinkToModel(types.DownloadLink{
		IOS:     " https://example.com/ios ",
		Android: " ",
		Linux:   "\thttps://example.com/linux\n",
	})
	if modelLink.IOS != "https://example.com/ios" || modelLink.Android != "" || modelLink.Linux != "https://example.com/linux" {
		t.Fatalf("download link was not normalized: %+v", modelLink)
	}

	data, err := modelLink.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"ios":"https://example.com/ios","linux":"https://example.com/linux"}`; got != want {
		t.Fatalf("download link json = %q, want %q", got, want)
	}
}
