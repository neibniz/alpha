package common

import (
	"encoding/json"
	"testing"
	"time"

	modelAds "github.com/neibniz/alpha/server/internal/model/ads"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestAdsListItemFromModelMapsCommonListShapeWithoutContent(t *testing.T) {
	start := time.UnixMilli(1700000000123)
	end := time.UnixMilli(1700003600456)
	item := &modelAds.Ads{
		Id:          14,
		Title:       "Public banner",
		Type:        "home",
		Content:     "private content",
		Description: "desc",
		TargetURL:   "https://example.com",
		StartTime:   start,
		EndTime:     end,
		Status:      1,
		CreatedAt:   start,
		UpdatedAt:   end,
	}

	got := adsListItemFromModel(item)
	if got.Id != int(item.Id) || got.Title != item.Title || got.Type != item.Type || got.Description != item.Description || got.TargetURL != item.TargetURL {
		t.Fatalf("common ads fields not mapped: %#v", got)
	}
	if got.StartTime != start.UnixMilli() || got.EndTime != end.UnixMilli() || got.CreatedAt != start.UnixMilli() || got.UpdatedAt != end.UnixMilli() {
		t.Fatalf("timestamps not mapped: %#v", got)
	}

	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["content"]; ok {
		t.Fatalf("common ads list item should omit content: %s", payload)
	}
}

func TestAdsRowsSkipsNilAndKeepsEmptySlice(t *testing.T) {
	got := adsRows([]*modelAds.Ads{
		nil,
		{Id: 7, Title: "Public banner", Content: "full"},
	})
	if len(got) != 1 {
		t.Fatalf("ads rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Title != "Public banner" {
		t.Fatalf("ads row mismatch: %#v", got[0])
	}

	empty := adsRows(nil)
	if empty == nil {
		t.Fatal("empty ads rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.AdsListItem `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty ads rows marshal = %s, want list []", payload)
	}
}

func TestAdsRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelAds.Ads, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelAds.Ads{Id: 1, Title: "first"}
	rows[len(rows)-1] = &modelAds.Ads{Id: 2, Title: "second"}

	got := adsRows(rows)
	if len(got) != 2 {
		t.Fatalf("ads rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy ads rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("ads row order changed: %#v", got)
	}
}

func TestAdsRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelAds.Ads, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelAds.Ads{Id: int64(index + 1), Title: "ads"}
	}

	got := adsRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("ads rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("ads rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("ads row order changed: %#v", got)
	}
}

func TestAdsListItemFromModelNormalizesZeroTime(t *testing.T) {
	got := adsListItemFromModel(&modelAds.Ads{Id: 15})
	if got.StartTime != 0 || got.EndTime != 0 || got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero timestamps should be encoded as 0: %#v", got)
	}
}
