package userview

import (
	"testing"
	"time"

	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestDeviceRowsSkipsNilAndMasksPrivateFields(t *testing.T) {
	rows := []*modelUser.Device{
		nil,
		{
			Id:         8,
			Ip:         "203.0.113.42",
			Identifier: "device@example.com",
			UserAgent:  "AlphaClient/1.2.3",
			Online:     true,
			Enabled:    true,
			CreatedAt:  time.Unix(10, 2000000),
			UpdatedAt:  time.Unix(20, 3000000),
		},
	}

	got := DeviceRows(rows, func(ip string) Location {
		if ip != "203.0.113.42" {
			t.Fatalf("unexpected lookup ip: %s", ip)
		}
		return Location{CountryShort: "US", Region: "CO", Location: "US CO"}
	})

	if len(got) != 1 {
		t.Fatalf("expected nil row to be skipped, got %d rows", len(got))
	}
	if got[0].Ip == "203.0.113.42" || got[0].Identifier == "device@example.com" || got[0].UserAgent == "AlphaClient/1.2.3" {
		t.Fatalf("private fields were not masked: %#v", got[0])
	}
	if got[0].CountryShort != "US" || got[0].Region != "CO" || got[0].Location != "US CO" {
		t.Fatalf("location not mapped: %#v", got[0])
	}
	if got[0].CreatedAt != 10002 || got[0].UpdatedAt != 20003 {
		t.Fatalf("timestamps not converted to milliseconds: %#v", got[0])
	}
}

func TestDeviceRowsCachesLocationsWithinResponse(t *testing.T) {
	calls := 0
	rows := []*modelUser.Device{
		{Id: 1, Ip: "203.0.113.42"},
		{Id: 2, Ip: "203.0.113.42"},
		{Id: 3, Ip: "203.0.113.43"},
	}

	got := DeviceRows(rows, func(ip string) Location {
		calls++
		return Location{CountryShort: "US", Region: "CO", Location: "resolved:" + ip}
	})

	if len(got) != 3 {
		t.Fatalf("device rows len = %d, want 3", len(got))
	}
	if calls != 2 {
		t.Fatalf("lookup calls = %d, want 2 for two unique IPs", calls)
	}
	if got[0].Location != "resolved:203.0.113.42" || got[1].Location != got[0].Location || got[2].Location != "resolved:203.0.113.43" {
		t.Fatalf("cached locations not reused correctly: %#v", got)
	}
}

func TestDeviceRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelUser.Device, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelUser.Device{Id: 1, Ip: "203.0.113.42"}
	rows[len(rows)-1] = &modelUser.Device{Id: 2, Ip: "203.0.113.43"}

	got := DeviceRows(rows, nil)
	if len(got) != 2 {
		t.Fatalf("device rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy device rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("device row order changed: %#v", got)
	}
}

func TestDeviceRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelUser.Device, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelUser.Device{Id: int64(index + 1), Ip: "203.0.113.42"}
	}

	got := DeviceRows(rows, nil)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("device rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("device rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("device row order changed: %#v", got)
	}
}

func TestDeviceLocationCacheUsesLinearPathForSmallInput(t *testing.T) {
	calls := 0
	cache := newDeviceLocationCache(func(ip string) Location {
		calls++
		return Location{Location: ip}
	}, smallDeviceLocationCacheThreshold+20)

	first := cache.Lookup("203.0.113.42")
	second := cache.Lookup("203.0.113.42")
	if cache.index != nil {
		t.Fatal("small device location cache should not allocate map")
	}
	if calls != 1 {
		t.Fatalf("lookup calls = %d, want 1", calls)
	}
	if first != second || first.Location != "203.0.113.42" {
		t.Fatalf("cached location mismatch: first=%#v second=%#v", first, second)
	}
}

func TestDeviceLocationCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newDeviceLocationCache(func(ip string) Location {
		return Location{Location: ip}
	}, smallDeviceLocationCacheThreshold+20)
	for i := 0; i <= smallDeviceLocationCacheThreshold; i++ {
		cache.Lookup("203.0.113." + string(rune('a'+i)))
	}
	if cache.index == nil {
		t.Fatal("large device location cache should allocate map")
	}
	if len(cache.rows) != smallDeviceLocationCacheThreshold+1 {
		t.Fatalf("device location cache rows len = %d, want %d", len(cache.rows), smallDeviceLocationCacheThreshold+1)
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("device location cache rows cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallDeviceLocationCacheThreshold+20; i++ {
		cache.Lookup("203.0.113.a")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("device location cache rows cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestDeviceLocationCacheAvoidsMapForLargeDuplicateInput(t *testing.T) {
	calls := 0
	cache := newDeviceLocationCache(func(ip string) Location {
		calls++
		return Location{Location: ip}
	}, smallDeviceLocationCacheThreshold+20)

	for i := 0; i < smallDeviceLocationCacheThreshold+20; i++ {
		cache.Lookup("203.0.113.42")
	}

	if cache.index != nil {
		t.Fatal("duplicate-heavy device location cache should not allocate map")
	}
	if len(cache.rows) != 1 {
		t.Fatalf("device location cache rows len = %d, want 1", len(cache.rows))
	}
	if cap(cache.rows) != smallDeviceLocationCacheThreshold+1 {
		t.Fatalf("device location cache rows cap = %d, want %d", cap(cache.rows), smallDeviceLocationCacheThreshold+1)
	}
	if calls != 1 {
		t.Fatalf("lookup calls = %d, want 1", calls)
	}
}

func TestDeviceRowsKeepsEmptyListShape(t *testing.T) {
	got := DeviceRows(nil, nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("empty device rows = %#v, want non-nil empty slice", got)
	}
}
