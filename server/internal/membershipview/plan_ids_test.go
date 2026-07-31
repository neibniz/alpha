package membershipview

import (
	"reflect"
	"testing"
)

func TestPlanIDSetSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	set := NewPlanIDSet(smallPlanIndexThreshold + 20)
	for i := 0; i < smallPlanIndexThreshold+20; i++ {
		set.Add(0)
		set.Add(7)
	}

	got := set.Values()
	if want := []int64{7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("plan ids = %#v, want %#v", got, want)
	}
	if cap(got) != smallPlanIndexThreshold+1 {
		t.Fatalf("plan id cap = %d, want small initial cap", cap(got))
	}
}

func TestPlanIDSetBoundsSparseUpgradeCapacity(t *testing.T) {
	set := NewPlanIDSet(smallPlanIndexThreshold + 20)
	for id := int64(1); id <= smallPlanIndexThreshold+1; id++ {
		set.Add(id)
	}
	for id := int64(0); id < smallPlanIndexThreshold+20; id++ {
		set.Add(1)
	}

	got := set.Values()
	if len(got) != smallPlanIndexThreshold+1 {
		t.Fatalf("plan ids len = %d, want %d", len(got), smallPlanIndexThreshold+1)
	}
	if maxCap := len(got) * 2; cap(got) > maxCap {
		t.Fatalf("plan id cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestNewPlanRowsUsesUniquePlanCapacity(t *testing.T) {
	rows := NewPlanRows([]int64{7, 8, 9})
	if rows == nil || len(rows) != 0 || cap(rows) != 3 {
		t.Fatalf("membership plan rows len/cap = %d/%d, want 0/3", len(rows), cap(rows))
	}
	empty := NewPlanRows(nil)
	if empty == nil || len(empty) != 0 || cap(empty) != 0 {
		t.Fatalf("empty membership plan rows len/cap = %d/%d, want 0/0 non-nil", len(empty), cap(empty))
	}
}
