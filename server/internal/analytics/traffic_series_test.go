package analytics

import (
	"testing"
	"time"
)

func TestTrafficSeriesSourceUsesResolutionAppropriateRollup(t *testing.T) {
	tests := []struct {
		interval time.Duration
		table    string
		column   string
	}{
		{interval: 5 * time.Second, table: "traffic_events", column: "event_time"},
		{interval: time.Minute, table: "traffic_minute", column: "bucket_time"},
		{interval: 5 * time.Minute, table: "traffic_minute", column: "bucket_time"},
		{interval: time.Hour, table: "traffic_hourly", column: "bucket_time"},
		{interval: 6 * time.Hour, table: "traffic_hourly", column: "bucket_time"},
	}
	for _, test := range tests {
		table, column := trafficSeriesSource(test.interval)
		if table != test.table || column != test.column {
			t.Fatalf("source for %v = %s.%s, want %s.%s", test.interval, table, column, test.table, test.column)
		}
	}
}
