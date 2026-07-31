package analytics

import "testing"

func TestNormalizeTrafficPage(t *testing.T) {
	tests := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
	}{
		{name: "valid", limit: 50, offset: 100, wantLimit: 50, wantOffset: 100},
		{name: "defaults invalid limit", limit: 0, offset: 0, wantLimit: 10, wantOffset: 0},
		{name: "bounds large limit", limit: 201, offset: 0, wantLimit: 10, wantOffset: 0},
		{name: "bounds negative offset", limit: 20, offset: -1, wantLimit: 20, wantOffset: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset := normalizeTrafficPage(tt.limit, tt.offset)
			if limit != tt.wantLimit || offset != tt.wantOffset {
				t.Fatalf("normalizeTrafficPage(%d, %d) = (%d, %d), want (%d, %d)",
					tt.limit, tt.offset, limit, offset, tt.wantLimit, tt.wantOffset)
			}
		})
	}
}

func TestTrafficPageCapacity(t *testing.T) {
	tests := []struct {
		name     string
		total    uint64
		limit    int
		offset   int
		want     int
		wantRows bool
	}{
		{name: "full page", total: 100, limit: 20, offset: 20, want: 20, wantRows: true},
		{name: "last page", total: 25, limit: 20, offset: 20, want: 5, wantRows: true},
		{name: "empty", total: 0, limit: 20, offset: 0},
		{name: "at end", total: 20, limit: 20, offset: 20},
		{name: "past end", total: 20, limit: 20, offset: 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := trafficPageCapacity(tt.total, tt.limit, tt.offset)
			if got != tt.want || ok != tt.wantRows {
				t.Fatalf("trafficPageCapacity(%d, %d, %d) = (%d, %v), want (%d, %v)",
					tt.total, tt.limit, tt.offset, got, ok, tt.want, tt.wantRows)
			}
		})
	}
}
