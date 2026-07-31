package announcement

import (
	"encoding/json"
	"testing"
	"time"

	modelAnnouncement "github.com/neibniz/alpha/server/internal/model/announcement"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestAnnouncementFromModelCanOmitContentForAdminList(t *testing.T) {
	show := true
	pinned := false
	popup := true
	createdAt := time.UnixMilli(1700000000123)
	updatedAt := time.UnixMilli(1700000100456)
	item := &modelAnnouncement.Announcement{
		Id:        7,
		Title:     "Maintenance",
		Content:   "large markdown content",
		Show:      &show,
		Pinned:    &pinned,
		Popup:     &popup,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	got := announcementFromModel(item, false)
	if got.Id != item.Id || got.Title != item.Title || got.Content != "" {
		t.Fatalf("admin list shape not mapped correctly: %#v", got)
	}
	if got.Show != show || got.Pinned != pinned || got.Popup != popup {
		t.Fatalf("booleans not mapped: %#v", got)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamps = %d/%d, want %d/%d", got.CreatedAt, got.UpdatedAt, createdAt.UnixMilli(), updatedAt.UnixMilli())
	}
}

func TestAnnouncementRowsSkipsNilAndKeepsEmptySlice(t *testing.T) {
	got := announcementRows([]*modelAnnouncement.Announcement{
		nil,
		{Id: 7, Title: "Maintenance", Content: "full"},
	}, false)
	if len(got) != 1 {
		t.Fatalf("announcement rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Title != "Maintenance" {
		t.Fatalf("announcement row mismatch: %#v", got[0])
	}

	empty := announcementRows(nil, false)
	if empty == nil {
		t.Fatal("empty announcement rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.AnnouncementListItem `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty announcement rows marshal = %s, want list []", payload)
	}
}

func TestAnnouncementRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelAnnouncement.Announcement, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelAnnouncement.Announcement{Id: 1, Title: "first"}
	rows[len(rows)-1] = &modelAnnouncement.Announcement{Id: 2, Title: "second"}

	got := announcementRows(rows, false)
	if len(got) != 2 {
		t.Fatalf("announcement rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy announcement rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("announcement row order changed: %#v", got)
	}
}

func TestAnnouncementRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelAnnouncement.Announcement, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelAnnouncement.Announcement{Id: int64(index + 1), Title: "announcement"}
	}

	got := announcementRows(rows, true)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("announcement rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("announcement rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("announcement row order changed: %#v", got)
	}
}

func TestAnnouncementFromModelIncludesContentForAdminDetail(t *testing.T) {
	item := &modelAnnouncement.Announcement{Id: 8, Title: "Detail", Content: "full content"}

	got := announcementFromModel(item, true)
	if got.Content != item.Content {
		t.Fatalf("content = %q, want %q", got.Content, item.Content)
	}
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero timestamps should be encoded as 0: %#v", got)
	}
	if got.Show || got.Pinned || got.Popup {
		t.Fatalf("nil booleans should default false in API response: %#v", got)
	}
}
