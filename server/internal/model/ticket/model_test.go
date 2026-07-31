package ticket

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTicketListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(ticketListBaseColumns))
	for _, column := range ticketListBaseColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "title", "status", "updated_at"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("ticket list columns must include %q", column)
		}
	}

	for _, column := range []string{"description", "user_id", "created_at"} {
		if _, ok := columns[column]; ok {
			t.Fatalf("ticket list base columns must not include %q by default", column)
		}
	}
}

func TestTicketListSelectColumnsCanIncludeOptionalColumnsOnDemand(t *testing.T) {
	selected := ticketListSelectColumns(TicketListOptions{
		IncludeUserID:             true,
		IncludeDescriptionPreview: true,
	})
	columns := make(map[string]struct{}, len(selected))
	for _, column := range selected {
		columns[column] = struct{}{}
	}

	previewColumn := "substr(description, 1, 240) AS description"
	if _, ok := columns[previewColumn]; !ok {
		t.Fatal("ticket list columns must include bounded description preview when requested")
	}
	if _, ok := columns["id"]; !ok {
		t.Fatal("ticket list columns must keep base columns when description preview is requested")
	}
	if _, ok := columns["user_id"]; !ok {
		t.Fatal("ticket list columns must include user_id when requested")
	}

	for _, column := range ticketListBaseColumns {
		if column == "description" || column == "user_id" || column == "created_at" {
			t.Fatalf("ticket list base columns must not be mutated with %q", column)
		}
	}
}

func TestTicketFollowLookupColumnsStayMinimal(t *testing.T) {
	want := []string{"id", "user_id"}
	if !reflect.DeepEqual(ticketFollowLookupColumns, want) {
		t.Fatalf("ticket follow lookup columns = %#v, want %#v", ticketFollowLookupColumns, want)
	}
}

func TestFindOneForFollowSelectsOnlyAccessColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Ticket{
		Id:          21,
		UserId:      9,
		Title:       "loaded only for full detail",
		Description: "large private body",
		Status:      Waiting,
		CreatedAt:   now,
		UpdatedAt:   now.Add(time.Minute),
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneForFollow(ctx, 21)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 21 || got.UserId != 9 {
		t.Fatalf("follow lookup dropped access fields: %#v", got)
	}
	if got.Title != "" || got.Description != "" || got.Status != 0 || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("follow lookup loaded non-access fields: %#v", got)
	}
}

func TestTicketDetailColumnsStayExplicit(t *testing.T) {
	wantTicket := []string{"id", "title", "description", "images", "user_id", "status", "created_at", "updated_at"}
	if !reflect.DeepEqual(ticketDetailColumns, wantTicket) {
		t.Fatalf("ticket detail columns = %#v, want %#v", ticketDetailColumns, wantTicket)
	}

	wantFollow := []string{"id", "ticket_id", "from", "type", "content", "images", "created_at"}
	if !reflect.DeepEqual(ticketFollowDetailColumns, wantFollow) {
		t.Fatalf("ticket follow detail columns = %#v, want %#v", ticketFollowDetailColumns, wantFollow)
	}
}

func TestQueryTicketListDescriptionPreviewLoadedOnDemand(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	longDescription := strings.Repeat("x", ticketDescriptionPreviewLimit+80)
	rows := []*Ticket{
		{Id: 1, UserId: 7, Title: "first ticket", Description: "heavy body one", Status: Pending, CreatedAt: now, UpdatedAt: now},
		{Id: 2, UserId: 7, Title: "second ticket", Description: longDescription, Status: Waiting, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)},
		{Id: 3, UserId: 8, Title: "other user", Description: "other body", Status: Pending, CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute)},
		{Id: 4, UserId: 7, Title: "closed ticket", Description: "closed body", Status: Closed, CreatedAt: now.Add(3 * time.Minute), UpdatedAt: now.Add(3 * time.Minute)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	total, adminList, err := model.QueryTicketList(ctx, 1, 10, 7, nil, "", TicketListOptions{
		IncludeUserID: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("admin total = %d, want 2", total)
	}
	if len(adminList) != 2 || cap(adminList) != 2 {
		t.Fatalf("admin list len/cap = %d/%d, want 2/2", len(adminList), cap(adminList))
	}
	gotIDs := make([]int64, 0, len(adminList))
	for _, item := range adminList {
		gotIDs = append(gotIDs, item.Id)
		if item.Description != "" {
			t.Fatalf("admin list loaded description for ticket %d: %q", item.Id, item.Description)
		}
		if item.UserId != 7 {
			t.Fatalf("admin list did not load requested user_id for ticket %d: %#v", item.Id, item)
		}
		if !item.CreatedAt.IsZero() {
			t.Fatalf("admin list loaded unneeded created_at for ticket %d: %v", item.Id, item.CreatedAt)
		}
	}
	if want := []int64{2, 1}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("admin ticket ids = %#v, want %#v", gotIDs, want)
	}

	total, userList, err := model.QueryTicketList(ctx, 1, 10, 7, nil, "", TicketListOptions{
		IncludeDescriptionPreview: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("user total = %d, want 2", total)
	}
	if len(userList) != 2 || cap(userList) != 2 {
		t.Fatalf("user list len/cap = %d/%d, want 2/2", len(userList), cap(userList))
	}
	if userList[0].Description != strings.Repeat("x", ticketDescriptionPreviewLimit) || userList[1].Description != "heavy body one" {
		t.Fatalf("user descriptions = [%q %q]", userList[0].Description, userList[1].Description)
	}
	if userList[0].Description == longDescription {
		t.Fatal("user ticket list loaded full description instead of preview")
	}
	for _, item := range userList {
		if item.UserId != 0 {
			t.Fatalf("user list loaded redundant user_id for ticket %d: %#v", item.Id, item)
		}
		if !item.CreatedAt.IsZero() {
			t.Fatalf("user list loaded unneeded created_at for ticket %d: %v", item.Id, item.CreatedAt)
		}
	}
}

func TestQueryTicketListEmptyResultKeepsStableSlice(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).QueryTicketList(ctx, 1, 10, 7, nil, "", TicketListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || list == nil || len(list) != 0 {
		t.Fatalf("ticket list = total %d list %#v, want total 0 and non-nil empty slice", total, list)
	}
}

func TestQueryTicketDetailUsesExplicitColumnsAndStableFollowOrder(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}, &Follow{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	if err := db.Create(&Ticket{
		Id:          7,
		UserId:      9,
		Title:       "Need help",
		Description: "full detail body",
		Images:      []string{"data:image/png;base64,initial"},
		Status:      Waiting,
		CreatedAt:   now,
		UpdatedAt:   now.Add(time.Minute),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*Follow{
		{Id: 2, TicketId: 7, From: "admin", Type: 1, Content: "reply", Images: []string{"data:image/png;base64,reply"}, CreatedAt: now.Add(2 * time.Minute)},
		{Id: 1, TicketId: 7, From: "user", Type: 1, Content: "hello", CreatedAt: now.Add(time.Minute)},
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).QueryTicketDetail(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 7 || got.UserId != 9 || got.Title != "Need help" || got.Description != "full detail body" || got.Status != Waiting {
		t.Fatalf("ticket detail fields = %#v", got)
	}
	if len(got.Images) != 1 || got.Images[0] != "data:image/png;base64,initial" {
		t.Fatalf("ticket images = %#v", got.Images)
	}
	if !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("ticket detail times = created %v updated %v", got.CreatedAt, got.UpdatedAt)
	}
	if len(got.Follows) != 2 {
		t.Fatalf("follows = %d, want 2", len(got.Follows))
	}
	if got.Follows[0].Id != 1 || got.Follows[1].Id != 2 {
		t.Fatalf("follow order = [%d %d], want [1 2]", got.Follows[0].Id, got.Follows[1].Id)
	}
	if got.Follows[0].TicketId != 7 || got.Follows[0].From != "user" || got.Follows[0].Content != "hello" {
		t.Fatalf("follow fields = %#v", got.Follows[0])
	}
	if len(got.Follows[1].Images) != 1 || got.Follows[1].Images[0] != "data:image/png;base64,reply" {
		t.Fatalf("follow images = %#v", got.Follows[1].Images)
	}
}

func TestUpdateTicketClearsIDCacheAndPersistsZeroValues(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Ticket{
		Id:          7,
		Title:       "existing",
		Description: "description",
		UserId:      9,
		Status:      Closed,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheTicketIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	if err := NewModel(db, rds).Update(ctx, &Ticket{Id: 7, UserId: 9, Status: 0}); err != nil {
		t.Fatal(err)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("ticket id cache key still exists")
	}

	var got Ticket
	if err := db.First(&got, 7).Error; err != nil {
		t.Fatal(err)
	}
	if got.Title != "" || got.Description != "" || got.UserId != 9 || got.Status != 0 {
		t.Fatalf("zero fields were not persisted: %#v", got)
	}
}

func TestDeleteTicketClearsIDCacheWithoutPreload(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Ticket{Id: 7, Title: "delete", UserId: 9, Status: Pending}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cacheKey := cacheTicketIdPrefix + "7"
	if err := rds.Set(ctx, cacheKey, "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("alpha:test_ticket_delete_query_count", func(tx *gorm.DB) {
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
	if err := db.Model(&Ticket{}).Where("id = ?", 7).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted ticket count = %d, want 0", count)
	}
	if exists := rds.Exists(ctx, cacheKey).Val(); exists != 0 {
		t.Fatalf("ticket id cache key still exists")
	}
	if err := model.Delete(ctx, 0); err != nil {
		t.Fatal(err)
	}
}

func TestInsertTicketFollowAndUpdateStatusIsAtomicAndChecksOwner(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}, &Follow{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Ticket{Id: 7, UserId: 9, Title: "support", Status: Waiting}).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	model := NewModel(db, rds)

	denied := &Follow{TicketId: 7, From: "user", Type: 1, Content: "denied"}
	if err := model.InsertTicketFollowAndUpdateStatus(ctx, denied, 10, Pending); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("owner mismatch error = %v, want record not found", err)
	}
	var followCount int64
	if err := db.Model(&Follow{}).Count(&followCount).Error; err != nil || followCount != 0 {
		t.Fatalf("follow count after rejected transaction = %d, %v", followCount, err)
	}

	accepted := &Follow{TicketId: 7, From: "user", Type: 1, Content: "accepted"}
	if err := model.InsertTicketFollowAndUpdateStatus(ctx, accepted, 9, Pending); err != nil {
		t.Fatal(err)
	}
	var stored Ticket
	if err := db.First(&stored, 7).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != Pending || accepted.Id == 0 {
		t.Fatalf("stored ticket/follow = %#v / %#v", stored, accepted)
	}

	if err := db.Model(&Ticket{}).Where("id = ?", 7).Update("status", Closed).Error; err != nil {
		t.Fatal(err)
	}
	closedReply := &Follow{TicketId: 7, From: "user", Type: 1, Content: "closed"}
	if err := model.InsertTicketFollowAndUpdateStatus(ctx, closedReply, 9, Pending); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("closed reply error = %v, want record not found", err)
	}
	if err := model.InsertTicketFollowAndUpdateStatus(ctx, closedReply, 9, Waiting); !errors.Is(err, ErrInvalidTicketTransition) {
		t.Fatalf("invalid reply transition = %v", err)
	}
}

func TestUpdateTicketStatusEnforcesTransitionsAndOwnership(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Ticket{Id: 7, UserId: 9, Title: "support", Status: Pending}).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	model := NewModel(db, rds)

	if err := model.UpdateTicketStatus(ctx, 7, 10, Closed); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("owner mismatch = %v", err)
	}
	if err := model.UpdateTicketStatus(ctx, 7, 9, Waiting); !errors.Is(err, ErrInvalidTicketTransition) {
		t.Fatalf("direct waiting transition = %v", err)
	}
	if err := model.UpdateTicketStatus(ctx, 7, 9, Closed); err != nil {
		t.Fatal(err)
	}
	if err := model.UpdateTicketStatus(ctx, 7, 9, Processed); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("closed to processed transition = %v", err)
	}
	if err := model.UpdateTicketStatus(ctx, 7, 9, Pending); err != nil {
		t.Fatal(err)
	}
}
