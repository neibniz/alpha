package cachekeys

import (
	"reflect"
	"strconv"
	"testing"
)

func TestCacheKeyBuildersUsePrefixConcatenation(t *testing.T) {
	if got, want := Int("user:", 42), "user:42"; got != want {
		t.Fatalf("Int() = %q, want %q", got, want)
	}
	if got, want := String("email:", "a@example.test"), "email:a@example.test"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestMergeDeduplicatesGroups(t *testing.T) {
	got := Merge([]string{"a", "", "b", "a"}, []string{"b", "c"}, nil, []string{"a", "d"})
	want := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Merge() = %#v, want %#v", got, want)
	}
}

func TestMergeLargeDeduplicatesGroups(t *testing.T) {
	got := Merge(
		[]string{"a", "b", "c", "d", "e"},
		[]string{"c", "f", "g", "h", "i", "a", ""},
	)
	want := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Merge() = %#v, want %#v", got, want)
	}
}

func TestMergeEmptyKeepsSliceShape(t *testing.T) {
	got := Merge(nil, []string{""})
	if got == nil || len(got) != 0 {
		t.Fatalf("Merge(empty) = %#v, want non-nil empty slice", got)
	}
}

func TestMergeSmallInputAvoidsMapAllocation(t *testing.T) {
	first := []string{"a", "b"}
	second := []string{"a", "c", ""}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = Merge(first, second)
	})
	if allocs > 1 {
		t.Fatalf("Merge small input allocations = %f, want at most 1", allocs)
	}
}

func TestMergeCollectorDoesNotAllocateMapFromCapacityHintOnly(t *testing.T) {
	collector := newMergeCollector(100)
	collector.Add("a")
	collector.Add("a")

	if collector.seen != nil {
		t.Fatal("merge collector should not allocate map until actual unique keys exceed threshold")
	}
	if got := collector.Keys(); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("collector keys = %#v, want only first unique key", got)
	}
}

func TestMergeCollectorUpgradesLargeActualInputToMap(t *testing.T) {
	collector := newMergeCollector(smallMergeSetThreshold + 40)
	for index := 0; index <= smallMergeSetThreshold; index++ {
		collector.Add(String("key:", strconv.Itoa(index)))
	}
	collector.Add("key:0")

	if collector.seen == nil {
		t.Fatal("merge collector should allocate map after actual unique keys exceed threshold")
	}
	if got := len(collector.Keys()); got != smallMergeSetThreshold+1 {
		t.Fatalf("collector key count = %d, want %d", got, smallMergeSetThreshold+1)
	}
	if maxCap := len(collector.Keys()) * 2; cap(collector.Keys()) > maxCap {
		t.Fatalf("collector key cap = %d, want <= %d", cap(collector.Keys()), maxCap)
	}
}
