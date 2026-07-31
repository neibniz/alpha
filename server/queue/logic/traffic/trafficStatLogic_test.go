package traffic

import (
	"reflect"
	"testing"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelTraffic "github.com/neibniz/alpha/server/internal/model/traffic"
)

func TestBuildDailyUserTrafficRows(t *testing.T) {
	got := buildDailyUserTrafficRows("2026-07-06", []modelLog.UserTraffic{
		{UserId: 7, SubscribeId: 11, Download: 100, Upload: 20, Total: 120},
		{UserId: 8, SubscribeId: 12, Download: 300, Upload: 40, Total: 340},
	})
	want := []modelTraffic.DailyUserTraffic{
		{Date: "2026-07-06", UserId: 7, SubscribeId: 11, Download: 100, Upload: 20, Total: 120},
		{Date: "2026-07-06", UserId: 8, SubscribeId: 12, Download: 300, Upload: 40, Total: 340},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("daily user rows = %#v, want %#v", got, want)
	}
	if got := buildDailyUserTrafficRows("2026-07-06", nil); got != nil {
		t.Fatalf("empty daily user rows = %#v, want nil", got)
	}
}

func TestBuildDailyServerTrafficRows(t *testing.T) {
	got := buildDailyServerTrafficRows("2026-07-06", []modelLog.ServerTraffic{
		{ServerId: 101, Download: 100, Upload: 20, Total: 120},
		{ServerId: 102, Download: 300, Upload: 40, Total: 340},
	})
	want := []modelTraffic.DailyServerTraffic{
		{Date: "2026-07-06", ServerId: 101, Download: 100, Upload: 20, Total: 120},
		{Date: "2026-07-06", ServerId: 102, Download: 300, Upload: 40, Total: 340},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("daily server rows = %#v, want %#v", got, want)
	}
	if got := buildDailyServerTrafficRows("2026-07-06", nil); got != nil {
		t.Fatalf("empty daily server rows = %#v, want nil", got)
	}
}
