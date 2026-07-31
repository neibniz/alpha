package log

import (
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestTrafficDailyResponseColumnsStayLean(t *testing.T) {
	tests := []struct {
		name      string
		columns   []string
		required  []string
		forbidden []string
	}{
		{
			name:     "server",
			columns:  serverTrafficLogDailySelectColumns(),
			required: []string{"date", "server_id", "download", "upload", "total"},
			forbidden: []string{
				"id",
				"created_at",
				"updated_at",
			},
		},
		{
			name:     "user subscribe",
			columns:  userSubscribeTrafficLogDailySelectColumns(),
			required: []string{"date", "user_id", "subscribe_id", "download", "upload", "total"},
			forbidden: []string{
				"id",
				"created_at",
				"updated_at",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make(map[string]struct{}, len(tt.columns))
			for _, column := range tt.columns {
				got[column] = struct{}{}
			}
			for _, column := range tt.required {
				if _, ok := got[column]; !ok {
					t.Fatalf("columns must include %q: %#v", column, tt.columns)
				}
			}
			for _, column := range tt.forbidden {
				if _, ok := got[column]; ok {
					t.Fatalf("columns must not include %q: %#v", column, tt.columns)
				}
			}
		})
	}
}

func TestTrafficDailyMappers(t *testing.T) {
	server := serverTrafficLogFromDaily(traffic.DailyServerTraffic{
		ServerId: 7,
		Upload:   10,
		Download: 20,
		Total:    30,
		Date:     "2026-07-07",
	}, true)
	if server.ServerId != 7 || server.Upload != 10 || server.Download != 20 || server.Total != 30 || !server.Details {
		t.Fatalf("server traffic mapping mismatch: %+v", server)
	}

	user := userSubscribeTrafficLogFromDaily(traffic.DailyUserTraffic{
		UserId:      8,
		SubscribeId: 9,
		Upload:      11,
		Download:    21,
		Total:       32,
		Date:        "2026-07-07",
	}, false)
	if user.UserId != 8 || user.SubscribeId != 9 || user.Upload != 11 || user.Total != 32 || user.Details {
		t.Fatalf("user traffic mapping mismatch: %+v", user)
	}
}

func TestSplitTrafficPage(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		size       int
		todayTotal int64
		want       trafficPageWindow
	}{
		{
			name:       "inside today page",
			page:       1,
			size:       10,
			todayTotal: 25,
			want: trafficPageWindow{
				todayLimit: 10,
			},
		},
		{
			name:       "crosses from today into history",
			page:       3,
			size:       10,
			todayTotal: 25,
			want: trafficPageWindow{
				todayOffset:  20,
				todayLimit:   5,
				historyLimit: 5,
			},
		},
		{
			name:       "starts in history",
			page:       4,
			size:       10,
			todayTotal: 25,
			want: trafficPageWindow{
				historyOffset: 5,
				historyLimit:  10,
			},
		},
		{
			name:       "invalid size returns empty window",
			page:       1,
			size:       0,
			todayTotal: 25,
			want:       trafficPageWindow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitTrafficPage(tt.page, tt.size, tt.todayTotal); got != tt.want {
				t.Fatalf("splitTrafficPage(%d, %d, %d) = %+v, want %+v", tt.page, tt.size, tt.todayTotal, got, tt.want)
			}
		})
	}
}

func TestNewTrafficPageRowsUsesSharedRowsCap(t *testing.T) {
	rows, offset, ok := newTrafficPageRows[types.ServerTrafficLog](3, 1, 10)
	if rows == nil || len(rows) != 0 || cap(rows) != 3 || offset != 0 || !ok {
		t.Fatalf("small traffic rows = len:%d cap:%d nil:%v, want len 0 cap 3 non-nil", len(rows), cap(rows), rows == nil)
	}

	rows, offset, ok = newTrafficPageRows[types.ServerTrafficLog](23, 3, 10)
	if len(rows) != 0 || cap(rows) != 3 || offset != 20 || !ok {
		t.Fatalf("last traffic page = len:%d cap:%d offset:%d ok:%v, want 0/3/20/true", len(rows), cap(rows), offset, ok)
	}

	empty, offset, ok := newTrafficPageRows[types.UserSubscribeTrafficLog](20, 3, 10)
	if empty == nil || len(empty) != 0 || cap(empty) != 0 || offset != 0 || ok {
		t.Fatalf("empty traffic rows = len:%d cap:%d nil:%v, want non-nil empty", len(empty), cap(empty), empty == nil)
	}
}

func TestTrafficDetailsAvailable(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.Local)
	got, err := trafficDetailsAvailable("2026-07-06", now, true, 7)
	if err != nil || !got {
		t.Fatalf("recent date should keep details, got %v err %v", got, err)
	}

	got, err = trafficDetailsAvailable("2026-06-01", now, true, 7)
	if err != nil || got {
		t.Fatalf("old date should not keep details, got %v err %v", got, err)
	}

	got, err = trafficDetailsAvailable("not-a-date", now, true, 7)
	if err == nil || !got {
		t.Fatalf("parse error should preserve previous permissive behavior, got %v err %v", got, err)
	}

	got, err = trafficDetailsAvailable("2026-06-01", now, false, 7)
	if err != nil || !got {
		t.Fatalf("disabled auto clear should keep details, got %v err %v", got, err)
	}
}

func TestTrafficDetailsAvailabilityCacheAppliesServerAndUserLists(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.Local)
	cache := newTrafficDetailsAvailabilityCache(now, true, 7)
	servers := []types.ServerTrafficLog{
		{Date: "2026-07-06"},
		{Date: "2026-07-06"},
		{Date: "2026-06-01"},
	}

	if err := applyServerTrafficDetailsAvailability(servers, cache); err != nil {
		t.Fatalf("apply server details: %v", err)
	}
	if !servers[0].Details || !servers[1].Details || servers[2].Details {
		t.Fatalf("server details markers = %#v", servers)
	}
	if len(cache.rows) != 2 {
		t.Fatalf("details availability cache size = %d, want 2", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small details availability cache should not allocate map")
	}

	users := []types.UserSubscribeTrafficLog{
		{Date: "2026-07-06"},
		{Date: "not-a-date"},
	}
	err := applyUserSubscribeTrafficDetailsAvailability(users, cache)
	if err == nil {
		t.Fatal("invalid date should return first parse error")
	}
	if !users[0].Details || !users[1].Details {
		t.Fatalf("user details should preserve recent and parse-error compatibility: %#v", users)
	}
	if len(cache.rows) != 3 {
		t.Fatalf("details availability cache size after user list = %d, want 3", len(cache.rows))
	}
	if cache.index != nil {
		t.Fatal("small details availability cache should stay on linear path after user list")
	}
}

func TestTrafficDetailsAvailabilityCacheAddUpdatesExistingRows(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.Local)
	cache := newTrafficDetailsAvailabilityCache(now, true, 7)

	cache.add("2026-07-06", trafficDetailsAvailabilityResult{available: true})
	cache.add("2026-07-06", trafficDetailsAvailabilityResult{available: false})
	if len(cache.rows) != 1 {
		t.Fatalf("linear cache rows len = %d, want 1", len(cache.rows))
	}
	got, ok := cache.lookup("2026-07-06")
	if !ok || got.available {
		t.Fatalf("linear cache should update existing row, got=%#v ok=%v", got, ok)
	}

	indexed := newTrafficDetailsAvailabilityCacheWithHint(now, true, 7, smallTrafficDetailsAvailabilityCacheThreshold+1)
	indexed.add("2026-07-01", trafficDetailsAvailabilityResult{available: true})
	indexed.add("2026-07-01", trafficDetailsAvailabilityResult{available: false})
	if indexed.index != nil {
		t.Fatal("hinted cache should stay linear for duplicate dates")
	}
	if len(indexed.rows) != 1 {
		t.Fatalf("indexed cache rows len = %d, want 1", len(indexed.rows))
	}
	got, ok = indexed.lookup("2026-07-01")
	if !ok || got.available {
		t.Fatalf("indexed cache should update existing row, got=%#v ok=%v", got, ok)
	}
}

func TestTrafficDetailsAvailabilityCacheUpgradesLargeInputToMap(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.Local)
	cache := newTrafficDetailsAvailabilityCache(now, true, 30)

	for i := 0; i <= smallTrafficDetailsAvailabilityCacheThreshold; i++ {
		date := time.Date(2026, 7, i+1, 0, 0, 0, 0, time.Local).Format(time.DateOnly)
		if _, err := cache.available(date); err != nil {
			t.Fatalf("available(%q) error = %v", date, err)
		}
	}

	if cache.index == nil {
		t.Fatal("large details availability cache should allocate map")
	}
	if len(cache.rows) != smallTrafficDetailsAvailabilityCacheThreshold+1 {
		t.Fatalf("details availability cache rows = %d, want %d", len(cache.rows), smallTrafficDetailsAvailabilityCacheThreshold+1)
	}
	if _, ok := cache.index["2026-07-01"]; !ok {
		t.Fatalf("upgraded details availability cache missing first date: %#v", cache.index)
	}
}

func TestTrafficDetailsAvailabilityCacheUsesCapacityHintLazily(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.Local)
	cache := newTrafficDetailsAvailabilityCacheWithHint(now, true, 30, smallTrafficDetailsAvailabilityCacheThreshold+1)
	if cache.rows != nil || cache.index != nil {
		t.Fatalf("hinted details availability cache should not allocate before use: %#v", cache)
	}

	available, err := cache.available("2026-07-19")
	if err != nil || !available {
		t.Fatalf("available recent date = %v err %v", available, err)
	}
	if cap(cache.rows) != smallTrafficDetailsAvailabilityCacheThreshold+1 {
		t.Fatalf("details availability cache cap = %d, want %d", cap(cache.rows), smallTrafficDetailsAvailabilityCacheThreshold+1)
	}
	if cache.index != nil {
		t.Fatal("large hinted details availability cache should not allocate map for one unique date")
	}
	if len(cache.rows) != 1 {
		t.Fatalf("details availability cache rows = %d, want 1", len(cache.rows))
	}

	available, err = cache.available("2026-07-19")
	if err != nil || !available || len(cache.rows) != 1 {
		t.Fatalf("cached availability should reuse row, available=%v err=%v rows=%#v", available, err, cache.rows)
	}
}

func TestTrafficDetailsAvailabilityCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.Local)
	cache := newTrafficDetailsAvailabilityCacheWithHint(now, true, 30, smallTrafficDetailsAvailabilityCacheThreshold+20)
	for i := 0; i <= smallTrafficDetailsAvailabilityCacheThreshold; i++ {
		date := time.Date(2026, 7, i+1, 0, 0, 0, 0, time.Local).Format(time.DateOnly)
		if _, err := cache.available(date); err != nil {
			t.Fatalf("available(%q) error = %v", date, err)
		}
	}
	if cache.index == nil {
		t.Fatal("details availability cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("details availability cache cap = %d, want <= %d", cap(cache.rows), maxCap)
	}
}
