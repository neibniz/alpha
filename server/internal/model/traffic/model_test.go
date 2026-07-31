package traffic

import (
	"context"
	"reflect"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTrafficLogPageColumnsMatchResponse(t *testing.T) {
	columns := make(map[string]struct{}, len(trafficLogPageColumns))
	for _, column := range trafficLogPageColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"server_id",
		"user_id",
		"subscribe_id",
		"download",
		"upload",
		"timestamp",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("traffic log page columns must include %q", column)
		}
	}
}

func TestDailyUserTrafficListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(dailyUserTrafficListColumns))
	for _, column := range dailyUserTrafficListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "date", "user_id", "subscribe_id", "download", "upload", "total"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("daily user traffic list columns must include %q", column)
		}
	}

	forbidden := []string{"created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("daily user traffic list columns must not include %q", column)
		}
	}
}

func TestDailyServerTrafficListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(dailyServerTrafficListColumns))
	for _, column := range dailyServerTrafficListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "date", "server_id", "download", "upload", "total"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("daily server traffic list columns must include %q", column)
		}
	}

	forbidden := []string{"created_at", "updated_at"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("daily server traffic list columns must not include %q", column)
		}
	}
}

func TestQueryTrafficLogPageListFiltersAndOrders(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TrafficLog{}); err != nil {
		t.Fatal(err)
	}

	base := time.Unix(1700000000, 0)
	rows := []*TrafficLog{
		{Id: 1, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 10, Upload: 20, Timestamp: base},
		{Id: 2, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 30, Upload: 40, Timestamp: base.Add(time.Minute)},
		{Id: 3, ServerId: 2, UserId: 7, SubscribeId: 11, Download: 50, Upload: 60, Timestamp: base.Add(time.Minute)},
		{Id: 4, ServerId: 3, UserId: 7, SubscribeId: 12, Download: 70, Upload: 80, Timestamp: base.Add(2 * time.Minute)},
		{Id: 5, ServerId: 4, UserId: 8, SubscribeId: 11, Download: 90, Upload: 100, Timestamp: base.Add(3 * time.Minute)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db)
	list, total, err := model.QueryTrafficLogPageList(ctx, 7, 11, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if cap(list) != 3 {
		t.Fatalf("traffic log page capacity = %d, want 3", cap(list))
	}

	gotIDs := make([]int64, 0, len(list))
	for _, item := range list {
		gotIDs = append(gotIDs, item.Id)
	}
	if want := []int64{3, 2, 1}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("traffic log ids = %#v, want %#v", gotIDs, want)
	}
	if list[0].Download != 50 || list[0].Upload != 60 || list[0].ServerId != 2 {
		t.Fatalf("first traffic row = %#v", list[0])
	}
}

func TestQueryTrafficByDayUsesHalfOpenRange(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TrafficLog{}); err != nil {
		t.Fatal(err)
	}

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	rows := []*TrafficLog{
		{Id: 1, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 10, Upload: 1, Timestamp: day},
		{Id: 2, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 20, Upload: 2, Timestamp: day.Add(24*time.Hour - time.Nanosecond)},
		{Id: 3, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 30, Upload: 3, Timestamp: day.Add(24 * time.Hour)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db).QueryTrafficByDay(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	if got.Download != 30 || got.Upload != 3 {
		t.Fatalf("daily traffic = (%d, %d), want (30, 3)", got.Download, got.Upload)
	}
}

func TestTopTrafficRankingsPreallocateLimitAndAggregate(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TrafficLog{}); err != nil {
		t.Fatal(err)
	}

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	rows := []*TrafficLog{
		{Id: 1, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 10, Upload: 1, Timestamp: day.Add(time.Hour)},
		{Id: 2, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 20, Upload: 2, Timestamp: day.Add(2 * time.Hour)},
		{Id: 3, ServerId: 2, UserId: 8, SubscribeId: 12, Download: 40, Upload: 4, Timestamp: day.Add(3 * time.Hour)},
		{Id: 4, ServerId: 3, UserId: 9, SubscribeId: 13, Download: 100, Upload: 10, Timestamp: day.AddDate(0, 0, 1)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db)
	servers, err := model.TopServersTrafficByDay(ctx, day, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 2 || cap(servers) != 2 {
		t.Fatalf("server rankings len/cap = %d/%d, want 2/2", len(servers), cap(servers))
	}
	if servers[0].ServerId != 2 || servers[0].Total != 44 || servers[1].ServerId != 1 || servers[1].Total != 33 {
		t.Fatalf("server rankings = %#v", servers)
	}

	users, err := model.TopUsersTrafficByDay(ctx, day, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || cap(users) != 2 {
		t.Fatalf("user rankings len/cap = %d/%d, want 2/2", len(users), cap(users))
	}
	if users[0].UserId != 8 || users[0].SubscribeId != 12 || users[0].Total != 44 || users[1].UserId != 7 || users[1].SubscribeId != 11 || users[1].Total != 33 {
		t.Fatalf("user rankings = %#v", users)
	}

	negativeLimit, err := model.TopServersTrafficByDay(ctx, day, -1)
	if err != nil {
		t.Fatal(err)
	}
	if negativeLimit == nil || len(negativeLimit) != 0 {
		t.Fatalf("negative limit server rankings = %#v, want non-nil empty", negativeLimit)
	}
}

func TestQueryDailyUserTrafficPageSelectsOnlyResponseColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&DailyUserTraffic{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	rows := []*DailyUserTraffic{
		{Id: 1, Date: "2026-07-06", UserId: 7, SubscribeId: 11, Download: 10, Upload: 20, Total: 30, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Date: "2026-07-06", UserId: 8, SubscribeId: 12, Download: 30, Upload: 40, Total: 70, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Date: "2026-07-05", UserId: 9, SubscribeId: 13, Download: 50, Upload: 60, Total: 110, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	list, total, err := NewModel(db).QueryDailyUserTrafficPage(ctx, "2026-07-06", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("daily user traffic total/list = %d/%d, want 2/2", total, len(list))
	}
	if cap(list) != 2 {
		t.Fatalf("daily user traffic capacity = %d, want 2", cap(list))
	}
	gotIDs := []int64{list[0].Id, list[1].Id}
	if want := []int64{2, 1}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("daily user traffic ids = %#v, want %#v", gotIDs, want)
	}
	for _, item := range list {
		if item.Date == "" || item.UserId == 0 || item.SubscribeId == 0 || item.Total == 0 {
			t.Fatalf("daily user traffic dropped response fields: %#v", item)
		}
		if !item.CreatedAt.IsZero() || !item.UpdatedAt.IsZero() {
			t.Fatalf("daily user traffic loaded audit fields: %#v", item)
		}
	}
}

func TestQueryDailyServerTrafficPageSelectsOnlyResponseColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&DailyServerTraffic{}); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1700000000, 0)
	rows := []*DailyServerTraffic{
		{Id: 1, Date: "2026-07-06", ServerId: 101, Download: 10, Upload: 20, Total: 30, CreatedAt: now, UpdatedAt: now},
		{Id: 2, Date: "2026-07-06", ServerId: 102, Download: 30, Upload: 40, Total: 70, CreatedAt: now, UpdatedAt: now},
		{Id: 3, Date: "2026-07-05", ServerId: 103, Download: 50, Upload: 60, Total: 110, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	list, total, err := NewModel(db).QueryDailyServerTrafficPage(ctx, "2026-07-06", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("daily server traffic total/list = %d/%d, want 2/2", total, len(list))
	}
	if cap(list) != 2 {
		t.Fatalf("daily server traffic capacity = %d, want 2", cap(list))
	}
	gotIDs := []int64{list[0].Id, list[1].Id}
	if want := []int64{2, 1}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("daily server traffic ids = %#v, want %#v", gotIDs, want)
	}
	for _, item := range list {
		if item.Date == "" || item.ServerId == 0 || item.Total == 0 {
			t.Fatalf("daily server traffic dropped response fields: %#v", item)
		}
		if !item.CreatedAt.IsZero() || !item.UpdatedAt.IsZero() {
			t.Fatalf("daily server traffic loaded audit fields: %#v", item)
		}
	}
}

func TestTopDailyTrafficNormalizesNegativeLimit(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&DailyUserTraffic{}, &DailyServerTraffic{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&[]DailyUserTraffic{
		{Id: 1, Date: "2026-07-06", UserId: 7, SubscribeId: 11, Download: 10, Upload: 20, Total: 30},
		{Id: 2, Date: "2026-07-06", UserId: 8, SubscribeId: 12, Download: 30, Upload: 40, Total: 70},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]DailyServerTraffic{
		{Id: 1, Date: "2026-07-06", ServerId: 101, Download: 10, Upload: 20, Total: 30},
		{Id: 2, Date: "2026-07-06", ServerId: 102, Download: 30, Upload: 40, Total: 70},
	}).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db)
	users, err := model.TopDailyUserTraffic(ctx, "2026-07-06", -1)
	if err != nil {
		t.Fatal(err)
	}
	if users == nil || len(users) != 0 {
		t.Fatalf("negative limit daily user rankings = %#v, want non-nil empty", users)
	}

	servers, err := model.TopDailyServerTraffic(ctx, "2026-07-06", -1)
	if err != nil {
		t.Fatal(err)
	}
	if servers == nil || len(servers) != 0 {
		t.Fatalf("negative limit daily server rankings = %#v, want non-nil empty", servers)
	}
}

func TestQueryTrafficLogDetailsPageFiltersAndOrders(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TrafficLog{}); err != nil {
		t.Fatal(err)
	}

	day := time.Date(2026, 7, 6, 0, 0, 0, 0, time.Local)
	rows := []*TrafficLog{
		{Id: 1, ServerId: 1, UserId: 7, SubscribeId: 11, Download: 10, Upload: 20, Timestamp: day.Add(9 * time.Hour)},
		{Id: 2, ServerId: 2, UserId: 7, SubscribeId: 11, Download: 30, Upload: 40, Timestamp: day.Add(10 * time.Hour)},
		{Id: 3, ServerId: 2, UserId: 7, SubscribeId: 11, Download: 50, Upload: 60, Timestamp: day.Add(10 * time.Hour)},
		{Id: 4, ServerId: 2, UserId: 8, SubscribeId: 11, Download: 70, Upload: 80, Timestamp: day.Add(11 * time.Hour)},
		{Id: 5, ServerId: 2, UserId: 7, SubscribeId: 12, Download: 90, Upload: 100, Timestamp: day.Add(12 * time.Hour)},
		{Id: 6, ServerId: 2, UserId: 7, SubscribeId: 11, Download: 110, Upload: 120, Timestamp: day.AddDate(0, 0, 1)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	model := NewModel(db)
	list, total, err := model.QueryTrafficLogDetailsPage(ctx, &TrafficLogDetailsFilterParams{
		Page:        1,
		Size:        10,
		ServerId:    2,
		UserId:      7,
		SubscribeId: 11,
		Start:       day,
		End:         day.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if cap(list) != 2 {
		t.Fatalf("traffic detail capacity = %d, want 2", cap(list))
	}

	gotIDs := make([]int64, 0, len(list))
	for _, item := range list {
		gotIDs = append(gotIDs, item.Id)
	}
	if want := []int64{3, 2}; !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("traffic detail ids = %#v, want %#v", gotIDs, want)
	}
	if list[0].Download != 50 || list[0].Upload != 60 || list[0].ServerId != 2 || list[0].UserId != 7 || list[0].SubscribeId != 11 {
		t.Fatalf("first traffic detail row = %#v", list[0])
	}
}

func TestDeleteTrafficLogDoesNotPreloadRow(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TrafficLog{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&TrafficLog{Id: 7, ServerId: 1, UserId: 2, SubscribeId: 3, Download: 10, Upload: 20}).Error; err != nil {
		t.Fatal(err)
	}

	queryCount := 0
	if err := db.Callback().Query().Before("gorm:query").Register("alpha:test_traffic_delete_query_count", func(tx *gorm.DB) {
		queryCount++
	}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db)
	if err := model.Delete(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if queryCount != 0 {
		t.Fatalf("Delete queried %d times, want 0", queryCount)
	}
	var count int64
	if err := db.Model(&TrafficLog{}).Where("id = ?", 7).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted traffic log count = %d, want 0", count)
	}
	if err := model.Delete(ctx, 0); err != nil {
		t.Fatal(err)
	}
}

func TestTrafficLimitRowsCap(t *testing.T) {
	tests := []struct {
		limit int
		want  int
	}{
		{limit: -1, want: 0},
		{limit: 0, want: 0},
		{limit: 5, want: 5},
	}
	for _, tt := range tests {
		if got := trafficLimitRowsCap(tt.limit); got != tt.want {
			t.Fatalf("trafficLimitRowsCap(%d) = %d, want %d", tt.limit, got, tt.want)
		}
	}
}
