package trafficview

import (
	"testing"
	"time"

	modelTraffic "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestRowsSkipNilAndMapTrafficLogShape(t *testing.T) {
	rows := []*modelTraffic.TrafficLog{
		nil,
		{
			Id:          1,
			ServerId:    2,
			UserId:      3,
			SubscribeId: 4,
			Download:    5,
			Upload:      6,
			Timestamp:   time.Unix(7, 8000000),
		},
	}

	got := Rows(rows)
	if len(got) != 1 {
		t.Fatalf("expected nil row to be skipped, got %d rows", len(got))
	}
	if got[0].Id != 1 || got[0].ServerId != 2 || got[0].UserId != 3 || got[0].SubscribeId != 4 {
		t.Fatalf("ids not mapped: %#v", got[0])
	}
	if got[0].Download != 5 || got[0].Upload != 6 || got[0].Timestamp != 7008 {
		t.Fatalf("traffic values not mapped: %#v", got[0])
	}
}

func TestRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelTraffic.TrafficLog, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelTraffic.TrafficLog{Id: 1, Timestamp: time.Unix(1, 0)}
	rows[len(rows)-1] = &modelTraffic.TrafficLog{Id: 2, Timestamp: time.Unix(2, 0)}

	got := Rows(rows)
	if len(got) != 2 {
		t.Fatalf("traffic rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy traffic rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("traffic rows order/ids not preserved: %#v", got)
	}
}

func TestRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelTraffic.TrafficLog, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelTraffic.TrafficLog{Id: int64(index + 1), Timestamp: time.Unix(int64(index+1), 0)}
	}

	got := Rows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("traffic rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("traffic rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("traffic rows order/ids not preserved: %#v", got)
	}
}

func TestDetailRowsSkipNilAndMapTrafficLogShape(t *testing.T) {
	rows := []*modelTraffic.TrafficLog{
		nil,
		{
			Id:          8,
			ServerId:    9,
			UserId:      10,
			SubscribeId: 11,
			Download:    12,
			Upload:      13,
			Timestamp:   time.Unix(14, 15000000),
		},
	}

	got := DetailRows(rows)
	if len(got) != 1 {
		t.Fatalf("expected nil row to be skipped, got %d rows", len(got))
	}
	if got[0].Id != 8 || got[0].ServerId != 9 || got[0].UserId != 10 || got[0].SubscribeId != 11 {
		t.Fatalf("ids not mapped: %#v", got[0])
	}
	if got[0].Download != 12 || got[0].Upload != 13 || got[0].Timestamp != 14015 {
		t.Fatalf("traffic detail values not mapped: %#v", got[0])
	}
}

func TestTrafficRowsKeepEmptyListShape(t *testing.T) {
	if got := Rows(nil); got == nil || len(got) != 0 {
		t.Fatalf("empty traffic rows = %#v, want non-nil empty slice", got)
	}
	if got := DetailRows(nil); got == nil || len(got) != 0 {
		t.Fatalf("empty traffic detail rows = %#v, want non-nil empty slice", got)
	}
}

func TestTrafficLogDetailsReusesTrafficLogShape(t *testing.T) {
	var details []types.TrafficLogDetails = Rows([]*modelTraffic.TrafficLog{{
		Id:        1,
		ServerId:  2,
		Timestamp: time.Unix(3, 0),
	}})
	if len(details) != 1 || details[0].Id != 1 || details[0].ServerId != 2 || details[0].Timestamp != 3000 {
		t.Fatalf("traffic detail alias rows = %#v", details)
	}
}
