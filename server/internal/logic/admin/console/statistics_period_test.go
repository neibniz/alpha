package console

import (
	"testing"
	"time"
)

func TestStatisticsRangeBucketCountMatchesEachBucket(t *testing.T) {
	loc := time.UTC
	testCases := []struct {
		name string
		r    statisticsRange
		want int
	}{
		{
			name: "daily seven buckets",
			r: statisticsRange{
				Start: time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
				End:   time.Date(2026, 7, 8, 0, 0, 0, 0, loc),
				Step:  24 * time.Hour,
			},
			want: 7,
		},
		{
			name: "weekly yearly range",
			r: statisticsRange{
				Start: time.Date(2026, 1, 5, 0, 0, 0, 0, loc),
				End:   time.Date(2026, 1, 26, 0, 0, 0, 0, loc),
				Step:  7 * 24 * time.Hour,
			},
			want: 3,
		},
		{
			name: "partial final bucket",
			r: statisticsRange{
				Start: time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
				End:   time.Date(2026, 7, 3, 12, 0, 0, 0, loc),
				Step:  24 * time.Hour,
			},
			want: 3,
		},
		{
			name: "invalid step",
			r: statisticsRange{
				Start: time.Date(2026, 7, 1, 0, 0, 0, 0, loc),
				End:   time.Date(2026, 7, 8, 0, 0, 0, 0, loc),
			},
			want: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.bucketCount(); got != tc.want {
				t.Fatalf("bucketCount = %d, want %d", got, tc.want)
			}
			buckets := tc.r.buckets()
			if len(buckets) != tc.want {
				t.Fatalf("buckets count = %d, want %d", len(buckets), tc.want)
			}
			if cap(buckets) != tc.want {
				t.Fatalf("buckets capacity = %d, want %d", cap(buckets), tc.want)
			}
			var iterated []time.Time
			tc.r.eachBucket(func(bucket time.Time) {
				iterated = append(iterated, bucket)
			})
			if len(iterated) != len(buckets) {
				t.Fatalf("eachBucket count = %d, want %d", len(iterated), len(buckets))
			}
			for i := range buckets {
				if !iterated[i].Equal(buckets[i]) {
					t.Fatalf("eachBucket[%d] = %v, want %v", i, iterated[i], buckets[i])
				}
			}
		})
	}
}

func TestStatisticsRangeBucketIndex(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	r := statisticsRange{
		Start: start,
		End:   start.AddDate(0, 0, 3),
		Step:  24 * time.Hour,
	}

	testCases := []struct {
		name      string
		bucket    time.Time
		wantIndex int
		wantOK    bool
	}{
		{name: "first bucket", bucket: start, wantIndex: 0, wantOK: true},
		{name: "last bucket", bucket: start.AddDate(0, 0, 2), wantIndex: 2, wantOK: true},
		{name: "before range", bucket: start.AddDate(0, 0, -1), wantOK: false},
		{name: "range end exclusive", bucket: start.AddDate(0, 0, 3), wantOK: false},
		{name: "not aligned", bucket: start.Add(12 * time.Hour), wantOK: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := r.bucketIndex(tc.bucket)
			if ok != tc.wantOK || got != tc.wantIndex {
				t.Fatalf("bucketIndex() = %d, %v; want %d, %v", got, ok, tc.wantIndex, tc.wantOK)
			}
		})
	}
}

func TestStatisticsBucketSeriesPrecomputesDatesAndIndexes(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	series := newStatisticsBucketSeries(statisticsRange{
		Start:      start,
		End:        start.AddDate(0, 0, 3),
		Step:       24 * time.Hour,
		DateFormat: time.DateOnly,
	})

	if series.len() != 3 {
		t.Fatalf("len = %d, want 3", series.len())
	}
	for i, want := range []string{"2026-07-01", "2026-07-02", "2026-07-03"} {
		if got := series.date(i); got != want {
			t.Fatalf("date(%d) = %q, want %q", i, got, want)
		}
	}

	index, ok := series.index(start.AddDate(0, 0, 2))
	if !ok || index != 2 {
		t.Fatalf("index(last) = %d, %v; want 2, true", index, ok)
	}
	if _, ok := series.index(start.Add(12 * time.Hour)); ok {
		t.Fatal("index accepted a non-aligned bucket")
	}

	empty := newStatisticsBucketSeries(statisticsRange{
		Start: start,
		End:   start.AddDate(0, 0, 3),
	})
	if empty.len() != 0 {
		t.Fatalf("empty len = %d, want 0", empty.len())
	}
	if _, ok := empty.index(start); ok {
		t.Fatal("empty index unexpectedly succeeded")
	}
}
