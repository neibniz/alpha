package subscribeview

import (
	"testing"
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestGroupFromModelMapsAPIShape(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	item := &modelSubscribe.Group{
		Id:          7,
		Name:        "Business",
		Description: "business users",
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now,
	}

	got := GroupFromModel(item)
	if got.Id != item.Id || got.Name != item.Name || got.Description != item.Description {
		t.Fatalf("group fields not mapped: %#v", got)
	}
	if got.CreatedAt != item.CreatedAt.UnixMilli() || got.UpdatedAt != item.UpdatedAt.UnixMilli() {
		t.Fatalf("group times not mapped: %#v", got)
	}
}

func TestGroupsFromModelsSkipsNilAndPreservesOrder(t *testing.T) {
	got := GroupsFromModels([]*modelSubscribe.Group{
		{Id: 2, Name: "second"},
		nil,
		{Id: 1, Name: "first"},
	})
	if len(got) != 2 {
		t.Fatalf("groups = %d, want 2", len(got))
	}
	if got[0].Id != 2 || got[1].Id != 1 {
		t.Fatalf("group order changed: %#v", got)
	}
}

func TestGroupsFromModelsKeepEmptyListShape(t *testing.T) {
	got := GroupsFromModels(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty groups = %#v, want non-nil empty slice", got)
	}
}

func TestGroupsFromModelsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelSubscribe.Group, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelSubscribe.Group{Id: 1, Name: "first"}
	rows[len(rows)-1] = &modelSubscribe.Group{Id: 2, Name: "second"}

	got := GroupsFromModels(rows)
	if len(got) != 2 {
		t.Fatalf("groups len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy groups cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("group order changed: %#v", got)
	}
}

func TestGroupsFromModelsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelSubscribe.Group, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelSubscribe.Group{Id: int64(index + 1), Name: "group"}
	}

	got := GroupsFromModels(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("groups len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("groups cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("group order changed: %#v", got)
	}
}

func TestGroupFromModelKeepsZeroTimesAtZero(t *testing.T) {
	got := GroupFromModel(&modelSubscribe.Group{})
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero times should stay zero: %#v", got)
	}
}
