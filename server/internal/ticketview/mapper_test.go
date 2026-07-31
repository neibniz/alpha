package ticketview

import (
	"encoding/json"
	"testing"
	"time"

	ticketmodel "github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestFromDetailsMapsTicketAndFollows(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	item := &ticketmodel.Details{
		Id:          7,
		Title:       "Need help",
		Description: "full detail",
		Images:      []string{"data:image/png;base64,root"},
		UserId:      9,
		Status:      ticketmodel.Waiting,
		CreatedAt:   now.Add(-2 * time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
		Follows: []ticketmodel.Follow{
			{Id: 1, TicketId: 7, From: "user", Type: 1, Content: "hello", CreatedAt: now.Add(-90 * time.Minute)},
			{Id: 2, TicketId: 7, From: "admin", Type: 1, Content: "reply", Images: []string{"data:image/png;base64,reply"}, CreatedAt: now.Add(-80 * time.Minute)},
		},
	}

	got := FromDetails(item)
	if got.Id != item.Id || got.Title != item.Title || got.Description != item.Description || got.UserId != item.UserId || got.Status != item.Status {
		t.Fatalf("ticket fields not mapped: %#v", got)
	}
	if got.CreatedAt != item.CreatedAt.UnixMilli() || got.UpdatedAt != item.UpdatedAt.UnixMilli() {
		t.Fatalf("ticket times not mapped: %#v", got)
	}
	if len(got.Follows) != 2 {
		t.Fatalf("follows = %d, want 2", len(got.Follows))
	}
	if len(got.Images) != 1 || got.Images[0] != "data:image/png;base64,root" {
		t.Fatalf("ticket images not mapped: %#v", got.Images)
	}
	if got.Follows[0].Id != 1 || got.Follows[0].TicketId != 7 || got.Follows[0].From != "user" || got.Follows[0].Content != "hello" {
		t.Fatalf("follow fields not mapped: %#v", got.Follows[0])
	}
	if got.Follows[0].CreatedAt != item.Follows[0].CreatedAt.UnixMilli() {
		t.Fatalf("follow time not mapped: %#v", got.Follows[0])
	}
	if len(got.Follows[1].Images) != 1 || got.Follows[1].Images[0] != "data:image/png;base64,reply" {
		t.Fatalf("follow images not mapped: %#v", got.Follows[1].Images)
	}
}

func TestFromFollowPromotesLegacyImageContentToGallery(t *testing.T) {
	legacy := "data:image/png;base64,legacy"
	got := FromFollow(ticketmodel.Follow{Id: 1, Type: 2, Content: legacy})
	if got.Content != "" || len(got.Images) != 1 || got.Images[0] != legacy {
		t.Fatalf("legacy image mapping = %#v", got)
	}
}

func TestTicketListItemKeepsAdminListShapeLean(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	item := &ticketmodel.Ticket{
		Id:          8,
		Title:       "List item",
		Description: "large body",
		UserId:      10,
		Status:      ticketmodel.Pending,
		CreatedAt:   now,
		UpdatedAt:   now.Add(time.Hour),
	}

	got := TicketListItem(item)
	if got.Id != item.Id || got.Title != item.Title || got.UserId != item.UserId || got.Status != item.Status {
		t.Fatalf("admin ticket list fields not mapped: %#v", got)
	}
	if got.UpdatedAt != item.UpdatedAt.UnixMilli() {
		t.Fatalf("admin ticket list updated time not mapped: %#v", got)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal admin ticket list item: %v", err)
	}
	if string(payload) != `{"id":8,"title":"List item","user_id":10,"status":1,"updated_at":1700003600000}` {
		t.Fatalf("admin ticket list JSON = %s, want lean list shape", payload)
	}
}

func TestUserTicketListItemIncludesOnlyDescriptionPreviewShape(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	item := &ticketmodel.Ticket{
		Id:          8,
		Title:       "List item",
		Description: "description preview",
		UserId:      10,
		Status:      ticketmodel.Pending,
		CreatedAt:   now,
		UpdatedAt:   now.Add(time.Hour),
	}

	got := UserTicketListItem(item)
	if got.Id != item.Id || got.Title != item.Title || got.Description != item.Description || got.Status != item.Status {
		t.Fatalf("user ticket list fields not mapped: %#v", got)
	}
	if got.UpdatedAt != item.UpdatedAt.UnixMilli() {
		t.Fatalf("user ticket list updated time not mapped: %#v", got)
	}
	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal user ticket list item: %v", err)
	}
	if string(payload) != `{"id":8,"title":"List item","description":"description preview","status":1,"updated_at":1700003600000}` {
		t.Fatalf("user ticket list JSON = %s, want preview list shape", payload)
	}
}

func TestTicketRowsSkipsNilAndUsesEndpointShapes(t *testing.T) {
	now := time.UnixMilli(1700000000000)
	rows := []*ticketmodel.Ticket{
		{
			Id:          11,
			Title:       "Admin list item",
			Description: "description preview",
			UserId:      10,
			Status:      ticketmodel.Pending,
			CreatedAt:   now,
			UpdatedAt:   now.Add(time.Hour),
		},
		nil,
	}

	adminRows := TicketListRows(rows)
	if len(adminRows) != 1 {
		t.Fatalf("admin ticket rows = %d, want 1: %#v", len(adminRows), adminRows)
	}
	if adminRows[0].UpdatedAt != now.Add(time.Hour).UnixMilli() {
		t.Fatalf("admin ticket times not mapped: %#v", adminRows[0])
	}

	userRows := UserTicketListRows(rows)
	if len(userRows) != 1 || userRows[0].Description != "description preview" {
		t.Fatalf("user ticket rows should include description preview: %#v", userRows)
	}
}

func TestTicketRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*ticketmodel.Ticket, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &ticketmodel.Ticket{Id: 1, Title: "first"}
	rows[len(rows)-1] = &ticketmodel.Ticket{Id: 2, Title: "second"}

	got := TicketListRows(rows)
	if len(got) != 2 {
		t.Fatalf("ticket rows = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy ticket rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("ticket row order changed: %#v", got)
	}
}

func TestTicketRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*ticketmodel.Ticket, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &ticketmodel.Ticket{Id: int64(index + 1), Title: "ticket"}
	}

	got := UserTicketListRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("ticket rows = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("ticket rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("ticket row order changed: %#v", got)
	}
}

func TestTicketRowsKeepsEmptyListShape(t *testing.T) {
	adminRows := TicketListRows(nil)
	if adminRows == nil || len(adminRows) != 0 {
		t.Fatalf("empty admin ticket rows should return non-nil empty slice: %#v", adminRows)
	}
	userRows := UserTicketListRows(nil)
	if userRows == nil || len(userRows) != 0 {
		t.Fatalf("empty user ticket rows should return non-nil empty slice: %#v", userRows)
	}
}

func TestFromDetailsKeepsZeroTimesAtZero(t *testing.T) {
	got := FromDetails(&ticketmodel.Details{
		Follows: []ticketmodel.Follow{{}},
	})
	if got.CreatedAt != 0 || got.UpdatedAt != 0 {
		t.Fatalf("zero ticket times should stay zero: %#v", got)
	}
	if len(got.Follows) != 1 || got.Follows[0].CreatedAt != 0 {
		t.Fatalf("zero follow times should stay zero: %#v", got.Follows)
	}
}

func TestFromDetailsKeepsEmptyFollowArrayShape(t *testing.T) {
	got := FromDetails(&ticketmodel.Details{Id: 12, Title: "No replies"})
	if got.Follows == nil || len(got.Follows) != 0 {
		t.Fatalf("empty follows should be non-nil empty slice: %#v", got.Follows)
	}

	payload, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal ticket detail: %v", err)
	}
	if string(payload) != `{"id":12,"title":"No replies","images":[],"user_id":0,"follow":[],"status":0,"created_at":0,"updated_at":0}` {
		t.Fatalf("ticket detail JSON = %s, want stable empty follow array", payload)
	}
}
