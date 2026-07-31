package trafficview

import (
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/analytics"
)

func TestResolveSeriesWindowUsesBoundedPointCounts(t *testing.T) {
	now := time.Date(2026, time.July, 9, 12, 34, 56, 0, time.FixedZone("test", -6*60*60))
	tests := []struct {
		period   string
		interval time.Duration
		points   int
	}{
		{period: "5m", interval: 5 * time.Second, points: 60},
		{period: "15m", interval: 5 * time.Second, points: 180},
		{period: "1h", interval: time.Minute, points: 60},
		{period: "24h", interval: 5 * time.Minute, points: 288},
		{period: "7d", interval: time.Hour, points: 168},
		{period: "30d", interval: 6 * time.Hour, points: 120},
	}
	for _, test := range tests {
		t.Run(test.period, func(t *testing.T) {
			window, err := ResolveSeriesWindow(test.period, now)
			if err != nil {
				t.Fatal(err)
			}
			if window.Interval != test.interval {
				t.Fatalf("interval = %v, want %v", window.Interval, test.interval)
			}
			if got := int(window.End.Sub(window.Start) / window.Interval); got != test.points {
				t.Fatalf("points = %d, want %d", got, test.points)
			}
			if window.Start.Location() != time.UTC || window.End.Location() != time.UTC {
				t.Fatalf("window must be UTC: %#v", window)
			}
		})
	}
}

func TestResolveSeriesWindowDefaultsTo24HoursAndRejectsUnknownPeriod(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	got, err := ResolveSeriesWindow("", now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Interval != 5*time.Minute || got.End.Sub(got.Start) != 24*time.Hour {
		t.Fatalf("default window = %#v", got)
	}
	if _, err := ResolveSeriesWindow("2h", now); err == nil {
		t.Fatal("expected unsupported period error")
	}
}

func TestBuildSeriesResponseFillsMissingBuckets(t *testing.T) {
	start := time.Date(2026, time.July, 9, 18, 0, 0, 0, time.UTC)
	window := SeriesWindow{Start: start, End: start.Add(3 * time.Minute), Interval: time.Minute}
	response := BuildSeriesResponse(window, analytics.TrafficSeries{
		Download: 25,
		Upload:   7,
		List: []analytics.TrafficSeriesPoint{
			{Timestamp: start, Download: 10, Upload: 2},
			{Timestamp: start.Add(2 * time.Minute), Download: 15, Upload: 5},
		},
	})
	if len(response.List) != 3 {
		t.Fatalf("list length = %d, want 3", len(response.List))
	}
	if response.List[1].Timestamp != start.Add(time.Minute).UnixMilli() || response.List[1].Download != 0 || response.List[1].Upload != 0 {
		t.Fatalf("missing bucket was not zero-filled: %#v", response.List[1])
	}
	if response.Download != 25 || response.Upload != 7 || response.IntervalSeconds != 60 {
		t.Fatalf("response summary = %#v", response)
	}
}
