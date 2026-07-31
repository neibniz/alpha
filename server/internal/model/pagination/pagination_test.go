package pagination

import (
	"math"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		size     int
		wantPage int
		wantSize int
	}{
		{name: "keeps valid values", page: 2, size: 50, wantPage: 2, wantSize: 50},
		{name: "defaults invalid values", page: 0, size: 0, wantPage: 1, wantSize: DefaultPageSize},
		{name: "clamps size", page: 1, size: MaxPageSize + 1, wantPage: 1, wantSize: MaxPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, size := Normalize(tt.page, tt.size)
			if page != tt.wantPage || size != tt.wantSize {
				t.Fatalf("Normalize(%d, %d) = (%d, %d), want (%d, %d)", tt.page, tt.size, page, size, tt.wantPage, tt.wantSize)
			}
		})
	}
}

func TestNormalizeWithLimit(t *testing.T) {
	page, size := NormalizeWithLimit(-1, 200, 10, 60)
	if page != 1 || size != 60 {
		t.Fatalf("NormalizeWithLimit(-1, 200, 10, 60) = (%d, %d), want (1, 60)", page, size)
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		name string
		page int
		size int
		want int
	}{
		{name: "regular page", page: 3, size: 20, want: 40},
		{name: "invalid page", page: 0, size: 20, want: 0},
		{name: "invalid size", page: 2, size: 0, want: 0},
		{name: "saturates overflow", page: math.MaxInt, size: 2, want: math.MaxInt},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Offset(tt.page, tt.size); got != tt.want {
				t.Fatalf("Offset(%d, %d) = %d, want %d", tt.page, tt.size, got, tt.want)
			}
		})
	}
}

func TestPageWindow(t *testing.T) {
	tests := []struct {
		name         string
		total        int64
		page         int
		size         int
		wantOffset   int
		wantCapacity int
		wantOK       bool
	}{
		{name: "first full page", total: 45, page: 1, size: 20, wantOffset: 0, wantCapacity: 20, wantOK: true},
		{name: "last partial page", total: 45, page: 3, size: 20, wantOffset: 40, wantCapacity: 5, wantOK: true},
		{name: "exact last page", total: 40, page: 2, size: 20, wantOffset: 20, wantCapacity: 20, wantOK: true},
		{name: "page out of range", total: 40, page: 3, size: 20, wantOK: false},
		{name: "empty total", total: 0, page: 1, size: 20, wantOK: false},
		{name: "invalid page", total: 1, page: 0, size: 20, wantOK: false},
		{name: "invalid size", total: 1, page: 1, size: 0, wantOK: false},
		{name: "overflowing page", total: math.MaxInt64, page: math.MaxInt, size: 2, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, capacity, ok := PageWindow(tt.total, tt.page, tt.size)
			if offset != tt.wantOffset || capacity != tt.wantCapacity || ok != tt.wantOK {
				t.Fatalf("PageWindow(%d, %d, %d) = (%d, %d, %v), want (%d, %d, %v)", tt.total, tt.page, tt.size, offset, capacity, ok, tt.wantOffset, tt.wantCapacity, tt.wantOK)
			}
		})
	}
}

func TestRowsCap(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		size  int
		want  int
	}{
		{name: "empty total", total: 0, size: 20, want: 0},
		{name: "invalid size", total: 10, size: 0, want: 0},
		{name: "partial page", total: 3, size: 20, want: 3},
		{name: "full page", total: 20, size: 20, want: 20},
		{name: "more than one page", total: 120, size: 20, want: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RowsCap(tt.total, tt.size); got != tt.want {
				t.Fatalf("RowsCap(%d, %d) = %d, want %d", tt.total, tt.size, got, tt.want)
			}
		})
	}
}
