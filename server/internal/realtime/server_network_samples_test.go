package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRecordAndQueryServerNetworkSamples(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	now := time.UnixMilli(10_000)
	states := []NodeStatus{
		{ServerID: 7, Online: true, NetRxBps: 10, NetTxBps: 20, XrayRxBps: 30, XrayTxBps: 40},
		{ServerID: 8, Online: false, NetRxBps: 99},
	}
	if err := RecordServerNetworkSnapshot(ctx, rdb, now, states); err != nil {
		t.Fatalf("record samples: %v", err)
	}

	// A second update in the same bucket replaces the previous value.
	states[0].NetRxBps = 15
	if err := RecordServerNetworkSnapshot(ctx, rdb, now.Add(time.Second), states); err != nil {
		t.Fatalf("replace sample: %v", err)
	}

	got, err := QueryServerNetworkSamples(ctx, rdb, []int64{7, 8}, 0, 20_000)
	if err != nil {
		t.Fatalf("query samples: %v", err)
	}
	if len(got[7]) != 1 || got[7][0].NetRxBps != 15 {
		t.Fatalf("server 7 samples = %#v, want one replaced sample", got[7])
	}
	if len(got[8]) != 0 {
		t.Fatalf("offline server samples = %#v, want empty", got[8])
	}
}

func TestCompactServerNetworkSamplesAveragesBuckets(t *testing.T) {
	points := []ServerNetworkSample{
		{Timestamp: 0, NetRxBps: 10, NetTxBps: 20, XrayRxBps: 30, XrayTxBps: 40},
		{Timestamp: 5_000, NetRxBps: 30, NetTxBps: 40, XrayRxBps: 50, XrayTxBps: 60},
		{Timestamp: 60_000, NetRxBps: 70, NetTxBps: 80, XrayRxBps: 90, XrayTxBps: 100},
	}

	got := CompactServerNetworkSamples(points, time.Minute)
	if len(got) != 2 {
		t.Fatalf("compacted points = %d, want 2", len(got))
	}
	if got[0].Timestamp != 0 || got[0].NetRxBps != 20 || got[0].XrayTxBps != 50 {
		t.Fatalf("first compacted point = %#v", got[0])
	}
	if got[1].Timestamp != 60_000 || got[1].NetTxBps != 80 {
		t.Fatalf("second compacted point = %#v", got[1])
	}
}

func TestRecordServerNetworkSamplesPrunesExpiredPoints(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	ctx := context.Background()
	now := time.UnixMilli(ServerNetworkSampleTTL.Milliseconds() + 10_000)
	key := serverNetworkSeriesKey(7)
	old := ServerNetworkSample{Timestamp: 1_000, NetRxBps: 1}
	encoded, _ := json.Marshal(old)
	if err := rdb.ZAdd(ctx, key, redis.Z{Score: 1_000, Member: string(encoded)}).Err(); err != nil {
		t.Fatalf("seed old sample: %v", err)
	}

	if err := RecordServerNetworkSnapshot(ctx, rdb, now, []NodeStatus{{ServerID: 7, Online: true}}); err != nil {
		t.Fatalf("record current sample: %v", err)
	}
	if count := rdb.ZCard(ctx, key).Val(); count != 1 {
		t.Fatalf("sample count = %d, want 1 after pruning", count)
	}
}
