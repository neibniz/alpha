package viewutil

import "testing"

func TestRowsCollectorKeepsEmptySliceShape(t *testing.T) {
	collector := NewRowsCollector[int](0)
	got := collector.Rows()
	if got == nil || len(got) != 0 {
		t.Fatalf("empty rows = %#v, want non-nil empty slice", got)
	}
}

func TestRowsCollectorStartsWithSmallCapacity(t *testing.T) {
	collector := NewRowsCollector[int](SmallRowsCapacityThreshold + 20)
	collector.Add(1)
	collector.Add(2)

	got := collector.Rows()
	if len(got) != 2 {
		t.Fatalf("rows len = %d, want 2", len(got))
	}
	if cap(got) != SmallRowsCapacityThreshold+1 {
		t.Fatalf("small rows cap = %d, want %d", cap(got), SmallRowsCapacityThreshold+1)
	}
}

func TestRowsCollectorKeepsThresholdCapacityAtActualThreshold(t *testing.T) {
	collector := NewRowsCollector[int](SmallRowsCapacityThreshold + 20)
	for index := 0; index <= SmallRowsCapacityThreshold; index++ {
		collector.Add(index)
	}

	got := collector.Rows()
	if len(got) != SmallRowsCapacityThreshold+1 {
		t.Fatalf("rows len = %d, want %d", len(got), SmallRowsCapacityThreshold+1)
	}
	if cap(got) != SmallRowsCapacityThreshold+1 {
		t.Fatalf("rows cap = %d, want %d", cap(got), SmallRowsCapacityThreshold+1)
	}
}

func TestRowsCollectorBoundsSparseGrowthCapacity(t *testing.T) {
	collector := NewRowsCollector[int](SmallRowsCapacityThreshold + 200)
	for index := 0; index < SmallRowsCapacityThreshold+2; index++ {
		collector.Add(index)
	}

	got := collector.Rows()
	if len(got) != SmallRowsCapacityThreshold+2 {
		t.Fatalf("rows len = %d, want %d", len(got), SmallRowsCapacityThreshold+2)
	}
	if maxCap := len(got) * 2; cap(got) > maxCap {
		t.Fatalf("rows cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestRowsCollectorUsesCapacityHintForDenseGrowth(t *testing.T) {
	collector := NewRowsCollector[int](SmallRowsCapacityThreshold + 4)
	for index := 0; index < SmallRowsCapacityThreshold+4; index++ {
		collector.Add(index)
	}

	got := collector.Rows()
	if len(got) != SmallRowsCapacityThreshold+4 {
		t.Fatalf("rows len = %d, want %d", len(got), SmallRowsCapacityThreshold+4)
	}
	if cap(got) != SmallRowsCapacityThreshold+4 {
		t.Fatalf("rows cap = %d, want %d", cap(got), SmallRowsCapacityThreshold+4)
	}
}
