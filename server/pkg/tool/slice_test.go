package tool

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestStringMergeAndRemoveDuplicatesNormalizesCSV(t *testing.T) {
	got := StringMergeAndRemoveDuplicates(" edge, premium,,edge ", "", "trial,premium")
	want := []string{"edge", "premium", "trial"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringMergeAndRemoveDuplicates() = %#v, want %#v", got, want)
	}
}

func TestStringMergeAndRemoveDuplicatesEmpty(t *testing.T) {
	got := StringMergeAndRemoveDuplicates("", " , ")
	if len(got) != 0 {
		t.Fatalf("StringMergeAndRemoveDuplicates() = %#v, want empty", got)
	}
}

func TestStringMergeAndRemoveDuplicatesSmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = StringMergeAndRemoveDuplicates(" edge, premium ", "edge")
	})
	if allocs > 1 {
		t.Fatalf("StringMergeAndRemoveDuplicates small input allocations = %f, want at most 1", allocs)
	}
}

func TestStringMergeAndRemoveDuplicatesEmptyAllocFree(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = StringMergeAndRemoveDuplicates("", " , ")
	})
	if allocs != 0 {
		t.Fatalf("StringMergeAndRemoveDuplicates empty allocations = %f, want 0", allocs)
	}
}

func TestStringMergeAndRemoveDuplicatesLargeDuplicateAvoidsMapAllocationAndFullCap(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+20)
	for index := range values {
		values[index] = " edge "
	}
	merged := strings.Join(values, ",")
	got := StringMergeAndRemoveDuplicates(merged)
	want := []string{"edge"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringMergeAndRemoveDuplicates large duplicate = %#v, want %#v", got, want)
	}
	if cap(got) != smallNormalizeSetThreshold+1 {
		t.Fatalf("StringMergeAndRemoveDuplicates large duplicate cap = %d, want %d", cap(got), smallNormalizeSetThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = StringMergeAndRemoveDuplicates(merged)
	})
	if allocs > 1 {
		t.Fatalf("StringMergeAndRemoveDuplicates large duplicate allocations = %f, want at most 1", allocs)
	}
}

func TestStringMergeAndRemoveDuplicatesLargeUniqueUsesCapacityHintAfterUpgrade(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+2)
	for index := range values {
		values[index] = "tag-" + strconv.Itoa(index+1)
	}
	merged := strings.Join(values, ",")
	got := StringMergeAndRemoveDuplicates(merged)
	if !reflect.DeepEqual(got, values) {
		t.Fatalf("StringMergeAndRemoveDuplicates large unique = %#v, want %#v", got, values)
	}
	if cap(got) != len(values) {
		t.Fatalf("StringMergeAndRemoveDuplicates large unique cap = %d, want %d", cap(got), len(values))
	}
}

func TestStringMergeAndRemoveDuplicatesBoundsSparseUpgradeCapacity(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+40)
	want := make([]string, 0, smallNormalizeSetThreshold+1)
	for index := 0; index <= smallNormalizeSetThreshold; index++ {
		value := "tag-" + strconv.Itoa(index+1)
		values[index] = value
		want = append(want, value)
	}
	for index := smallNormalizeSetThreshold + 1; index < len(values); index++ {
		values[index] = "tag-1"
	}

	got := StringMergeAndRemoveDuplicates(strings.Join(values, ","))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringMergeAndRemoveDuplicates sparse = %#v, want %#v", got, want)
	}
	if maxCap := len(want) * 2; cap(got) > maxCap {
		t.Fatalf("StringMergeAndRemoveDuplicates sparse cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestStringSliceToStringNormalizesItems(t *testing.T) {
	got := StringSliceToString([]string{" edge ", "", "premium", "edge", " trial "})
	if want := "edge,premium,trial"; got != want {
		t.Fatalf("StringSliceToString() = %q, want %q", got, want)
	}
}

func TestNormalizeStringSliceTrimsAndDeduplicates(t *testing.T) {
	got := NormalizeStringSlice([]string{" edge ", "", "premium", "edge", " trial "})
	want := []string{"edge", "premium", "trial"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeStringSlice() = %#v, want %#v", got, want)
	}
}

func TestNormalizeStringSliceSmallInputAvoidsMapAllocation(t *testing.T) {
	values := []string{" edge ", "premium", "edge"}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizeStringSlice(values)
	})
	if allocs > 1 {
		t.Fatalf("NormalizeStringSlice small input allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizeStringSliceEmptyKeepsArrayShape(t *testing.T) {
	got := NormalizeStringSlice([]string{"", "  "})
	if got == nil || len(got) != 0 {
		t.Fatalf("NormalizeStringSlice(empty) = %#v, want non-nil empty slice", got)
	}

	got = NormalizeStringSlice(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("NormalizeStringSlice(nil) = %#v, want non-nil empty slice", got)
	}
}

func TestNormalizeStringSliceBlankSmallInputAllocFree(t *testing.T) {
	values := []string{"", "  "}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizeStringSlice(values)
	})
	if allocs != 0 {
		t.Fatalf("NormalizeStringSlice blank small input allocations = %f, want 0", allocs)
	}
}

func TestNormalizeStringSliceLargeDuplicateAvoidsMapAllocationAndFullCap(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+20)
	for index := range values {
		values[index] = " edge "
	}
	got := NormalizeStringSlice(values)
	want := []string{"edge"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeStringSlice large duplicate = %#v, want %#v", got, want)
	}
	if cap(got) != smallNormalizeSetThreshold+1 {
		t.Fatalf("NormalizeStringSlice large duplicate cap = %d, want %d", cap(got), smallNormalizeSetThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizeStringSlice(values)
	})
	if allocs > 1 {
		t.Fatalf("NormalizeStringSlice large duplicate allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizeStringSliceLargeBlankAllocFree(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+20)
	for index := range values {
		values[index] = "  "
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizeStringSlice(values)
	})
	if allocs != 0 {
		t.Fatalf("NormalizeStringSlice large blank allocations = %f, want 0", allocs)
	}
	got := NormalizeStringSlice(values)
	if got == nil || len(got) != 0 {
		t.Fatalf("NormalizeStringSlice large blank = %#v, want non-nil empty slice", got)
	}
}

func TestNormalizeStringSliceLargeUniqueUsesCapacityHintAfterUpgrade(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+2)
	for index := range values {
		values[index] = "tag-" + strconv.Itoa(index+1)
	}
	got := NormalizeStringSlice(values)
	if !reflect.DeepEqual(got, values) {
		t.Fatalf("NormalizeStringSlice large unique = %#v, want %#v", got, values)
	}
	if cap(got) != len(values) {
		t.Fatalf("NormalizeStringSlice large unique cap = %d, want %d", cap(got), len(values))
	}
}

func TestNormalizeStringSliceBoundsSparseUpgradeCapacity(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+40)
	want := make([]string, 0, smallNormalizeSetThreshold+1)
	for index := 0; index <= smallNormalizeSetThreshold; index++ {
		value := "tag-" + strconv.Itoa(index+1)
		values[index] = " " + value + " "
		want = append(want, value)
	}
	for index := smallNormalizeSetThreshold + 1; index < len(values); index++ {
		values[index] = " tag-1 "
	}

	got := NormalizeStringSlice(values)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeStringSlice sparse = %#v, want %#v", got, want)
	}
	if maxCap := len(want) * 2; cap(got) > maxCap {
		t.Fatalf("NormalizeStringSlice sparse cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestStringToInt64SliceScansCSV(t *testing.T) {
	got := StringToInt64Slice(" 3, ,2,bad,3 ")
	want := []int64{3, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringToInt64Slice() = %#v, want %#v", got, want)
	}
}

func TestStringToPositiveUniqueInt64SliceNormalizesIDs(t *testing.T) {
	got := StringToPositiveUniqueInt64Slice(" 3,0,2,bad,3,-1,4 ")
	want := []int64{3, 2, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringToPositiveUniqueInt64Slice() = %#v, want %#v", got, want)
	}
}

func TestStringToPositiveUniqueInt64SliceSmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = StringToPositiveUniqueInt64Slice("3,2,3")
	})
	if allocs > 1 {
		t.Fatalf("StringToPositiveUniqueInt64Slice small input allocations = %f, want at most 1", allocs)
	}
}

func TestStringToPositiveUniqueInt64SliceEmptyAllocFree(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = StringToPositiveUniqueInt64Slice(" , ")
	})
	if allocs != 0 {
		t.Fatalf("StringToPositiveUniqueInt64Slice empty allocations = %f, want 0", allocs)
	}
}

func TestStringToPositiveUniqueInt64SliceLargeDuplicateAvoidsMapAllocationAndFullCap(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+20)
	for index := range values {
		values[index] = "7"
	}
	raw := strings.Join(values, ",")
	got := StringToPositiveUniqueInt64Slice(raw)
	want := []int64{7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringToPositiveUniqueInt64Slice large duplicate = %#v, want %#v", got, want)
	}
	if cap(got) != smallNormalizeSetThreshold+1 {
		t.Fatalf("StringToPositiveUniqueInt64Slice large duplicate cap = %d, want %d", cap(got), smallNormalizeSetThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = StringToPositiveUniqueInt64Slice(raw)
	})
	if allocs > 1 {
		t.Fatalf("StringToPositiveUniqueInt64Slice large duplicate allocations = %f, want at most 1", allocs)
	}
}

func TestStringToPositiveUniqueInt64SliceLargeInvalidAllocFree(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+20)
	raw := strings.Join(values, ",")
	allocs := testing.AllocsPerRun(1000, func() {
		_ = StringToPositiveUniqueInt64Slice(raw)
	})
	if allocs != 0 {
		t.Fatalf("StringToPositiveUniqueInt64Slice large invalid allocations = %f, want 0", allocs)
	}
	got := StringToPositiveUniqueInt64Slice(raw)
	if got == nil || len(got) != 0 {
		t.Fatalf("StringToPositiveUniqueInt64Slice large invalid = %#v, want non-nil empty slice", got)
	}
}

func TestStringToPositiveUniqueInt64SliceLargeUniqueUsesCapacityHintAfterUpgrade(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+2)
	want := make([]int64, 0, len(values))
	for index := range values {
		value := int64(index + 1)
		values[index] = strconv.FormatInt(value, 10)
		want = append(want, value)
	}
	got := StringToPositiveUniqueInt64Slice(strings.Join(values, ","))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringToPositiveUniqueInt64Slice large unique = %#v, want %#v", got, want)
	}
	if cap(got) != len(values) {
		t.Fatalf("StringToPositiveUniqueInt64Slice large unique cap = %d, want %d", cap(got), len(values))
	}
}

func TestStringToPositiveUniqueInt64SliceBoundsSparseUpgradeCapacity(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+40)
	want := make([]int64, 0, smallNormalizeSetThreshold+1)
	for index := 0; index <= smallNormalizeSetThreshold; index++ {
		value := int64(index + 1)
		values[index] = strconv.FormatInt(value, 10)
		want = append(want, value)
	}
	for index := smallNormalizeSetThreshold + 1; index < len(values); index++ {
		values[index] = "1"
	}

	got := StringToPositiveUniqueInt64Slice(strings.Join(values, ","))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StringToPositiveUniqueInt64Slice sparse = %#v, want %#v", got, want)
	}
	if maxCap := len(want) * 2; cap(got) > maxCap {
		t.Fatalf("StringToPositiveUniqueInt64Slice sparse cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestNormalizePositiveInt64SlicePreservesFirstOccurrence(t *testing.T) {
	got := NormalizePositiveInt64Slice([]int64{3, 0, 2, 3, -1, 4, 2})
	want := []int64{3, 2, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizePositiveInt64Slice() = %#v, want %#v", got, want)
	}
}

func TestNormalizePositiveInt64SliceSmallInputAvoidsMapAllocation(t *testing.T) {
	values := []int64{3, 0, 2, 3, -1}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizePositiveInt64Slice(values)
	})
	if allocs > 1 {
		t.Fatalf("NormalizePositiveInt64Slice small input allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizePositiveInt64SliceInvalidSmallInputAllocFree(t *testing.T) {
	values := []int64{0, -1}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizePositiveInt64Slice(values)
	})
	if allocs != 0 {
		t.Fatalf("NormalizePositiveInt64Slice invalid small input allocations = %f, want 0", allocs)
	}
}

func TestNormalizePositiveInt64SliceLargeDuplicateAvoidsMapAllocationAndFullCap(t *testing.T) {
	values := make([]int64, smallNormalizeSetThreshold+20)
	for index := range values {
		values[index] = 7
	}
	got := NormalizePositiveInt64Slice(values)
	want := []int64{7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizePositiveInt64Slice large duplicate = %#v, want %#v", got, want)
	}
	if cap(got) != smallNormalizeSetThreshold+1 {
		t.Fatalf("NormalizePositiveInt64Slice large duplicate cap = %d, want %d", cap(got), smallNormalizeSetThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizePositiveInt64Slice(values)
	})
	if allocs > 1 {
		t.Fatalf("NormalizePositiveInt64Slice large duplicate allocations = %f, want at most 1", allocs)
	}
}

func TestNormalizePositiveInt64SliceLargeInvalidAllocFree(t *testing.T) {
	values := make([]int64, smallNormalizeSetThreshold+20)
	allocs := testing.AllocsPerRun(1000, func() {
		_ = NormalizePositiveInt64Slice(values)
	})
	if allocs != 0 {
		t.Fatalf("NormalizePositiveInt64Slice large invalid allocations = %f, want 0", allocs)
	}
	got := NormalizePositiveInt64Slice(values)
	if got == nil || len(got) != 0 {
		t.Fatalf("NormalizePositiveInt64Slice large invalid = %#v, want non-nil empty slice", got)
	}
}

func TestNormalizePositiveInt64SliceBoundsSparseUpgradeCapacity(t *testing.T) {
	values := make([]int64, smallNormalizeSetThreshold+40)
	want := make([]int64, 0, smallNormalizeSetThreshold+1)
	for index := 0; index <= smallNormalizeSetThreshold; index++ {
		value := int64(index + 1)
		values[index] = value
		want = append(want, value)
	}
	for index := smallNormalizeSetThreshold + 1; index < len(values); index++ {
		values[index] = 1
	}

	got := NormalizePositiveInt64Slice(values)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizePositiveInt64Slice sparse = %#v, want %#v", got, want)
	}
	if maxCap := len(want) * 2; cap(got) > maxCap {
		t.Fatalf("NormalizePositiveInt64Slice sparse cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestSliceIntersectInt64KeepsUniqueValuesFromFirstSlice(t *testing.T) {
	got := SliceIntersectInt64([]int64{1, 2, 2, 3, 4}, []int64{2, 3, 4, 5})
	want := []int64{3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SliceIntersectInt64() = %#v, want %#v", got, want)
	}
}

func TestSliceDifferenceInt64PreservesExistingSemantics(t *testing.T) {
	got := SliceDifferenceInt64([]int64{1, 2, 2, 3, 4}, []int64{2, 3, 5})
	want := []int64{1, 2, 2, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SliceDifferenceInt64() = %#v, want %#v", got, want)
	}
}

func TestSliceDifferenceInt64KeepsLeftDuplicates(t *testing.T) {
	got := SliceDifferenceInt64([]int64{1, 2, 2, 3}, []int64{2, 3})
	want := []int64{1, 2, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SliceDifferenceInt64() = %#v, want %#v", got, want)
	}
}

func TestSliceDifferenceInt64SmallInputAllRemovedKeepsArrayShape(t *testing.T) {
	got := SliceDifferenceInt64([]int64{1, 2, 3}, []int64{1, 2, 3})
	if got == nil || len(got) != 0 {
		t.Fatalf("SliceDifferenceInt64(all removed) = %#v, want non-nil empty slice", got)
	}
}

func TestSliceIntersectInt64LargeLeftSmallRightUsesCandidateCounts(t *testing.T) {
	got := SliceIntersectInt64([]int64{1, 2, 2, 3, 4, 5, 6, 7, 8, 9}, []int64{2, 3, 9, 10, 3})
	want := []int64{3, 9, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SliceIntersectInt64(candidate counts) = %#v, want %#v", got, want)
	}
}

func TestSliceDifferenceInt64LargeLeftSmallRightUsesCandidateCounts(t *testing.T) {
	got := SliceDifferenceInt64([]int64{1, 2, 2, 3, 4, 5, 6, 7, 8, 9}, []int64{2, 3, 9})
	want := []int64{1, 2, 2, 4, 5, 6, 7, 8}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SliceDifferenceInt64(candidate counts) = %#v, want %#v", got, want)
	}
}

func TestInt64CandidateCounterUsesLinearPathForSmallInput(t *testing.T) {
	counter := newInt64CandidateCounter([]int64{2, 3, 2})
	counter.CountSlice([]int64{1, 2, 2, 3})

	if counter.index != nil {
		t.Fatal("small int64 candidate counter should not allocate map")
	}
	if got := counter.Count(2); got != 2 {
		t.Fatalf("counter.Count(2) = %d, want 2", got)
	}
	if got := counter.Count(3); got != 1 {
		t.Fatalf("counter.Count(3) = %d, want 1", got)
	}
	if got := counter.Count(1); got != 0 {
		t.Fatalf("counter.Count(1) = %d, want 0", got)
	}
}

func TestInt64CandidateCounterUpgradesLargeInputToMap(t *testing.T) {
	var candidates []int64
	for i := int64(1); i <= smallNormalizeSetThreshold+1; i++ {
		candidates = append(candidates, i)
	}
	counter := newInt64CandidateCounter(candidates)
	counter.CountSlice([]int64{1, smallNormalizeSetThreshold + 1, smallNormalizeSetThreshold + 1})

	if counter.index == nil {
		t.Fatal("large int64 candidate counter should allocate map")
	}
	if cap(counter.values) != smallNormalizeSetThreshold+1 || cap(counter.counts) != smallNormalizeSetThreshold+1 {
		t.Fatalf("counter values/counts cap = %d/%d, want %d", cap(counter.values), cap(counter.counts), smallNormalizeSetThreshold+1)
	}
	if got := counter.Count(1); got != 1 {
		t.Fatalf("counter.Count(1) = %d, want 1", got)
	}
	if got := counter.Count(smallNormalizeSetThreshold + 1); got != 2 {
		t.Fatalf("counter.Count(last) = %d, want 2", got)
	}
}

func TestInt64CandidateCounterAvoidsMapForLargeDuplicateInput(t *testing.T) {
	candidates := make([]int64, 0, smallNormalizeSetThreshold+20)
	for i := 0; i < smallNormalizeSetThreshold+20; i++ {
		candidates = append(candidates, 7)
	}

	counter := newInt64CandidateCounter(candidates)
	counter.CountSlice([]int64{7, 7, 9})

	if counter.index != nil {
		t.Fatal("duplicate-heavy int64 candidate counter should not allocate map")
	}
	if len(counter.values) != 1 || len(counter.counts) != 1 {
		t.Fatalf("counter values/counts len = %d/%d, want 1/1", len(counter.values), len(counter.counts))
	}
	if cap(counter.values) != smallNormalizeSetThreshold+1 || cap(counter.counts) != smallNormalizeSetThreshold+1 {
		t.Fatalf("counter values/counts cap = %d/%d, want %d", cap(counter.values), cap(counter.counts), smallNormalizeSetThreshold+1)
	}
	if got := counter.Count(7); got != 2 {
		t.Fatalf("counter.Count(7) = %d, want 2", got)
	}
}

func TestInt64CandidateCounterBoundsSparseUpgradeCapacity(t *testing.T) {
	candidates := make([]int64, 0, smallNormalizeSetThreshold+40)
	for value := int64(1); value <= smallNormalizeSetThreshold+1; value++ {
		candidates = append(candidates, value)
	}
	for len(candidates) < smallNormalizeSetThreshold+40 {
		candidates = append(candidates, 1)
	}

	counter := newInt64CandidateCounter(candidates)
	counter.CountSlice([]int64{1, smallNormalizeSetThreshold + 1})

	if counter.index == nil {
		t.Fatal("sparse int64 candidate counter should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(counter.values) * 2; cap(counter.values) > maxCap || cap(counter.counts) > maxCap {
		t.Fatalf("counter values/counts cap = %d/%d, want <= %d", cap(counter.values), cap(counter.counts), maxCap)
	}
	if got := counter.Count(1); got != 1 {
		t.Fatalf("counter.Count(1) = %d, want 1", got)
	}
	if got := counter.Count(smallNormalizeSetThreshold + 1); got != 1 {
		t.Fatalf("counter.Count(last) = %d, want 1", got)
	}
}

func TestSliceIntersectInt64SmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = SliceIntersectInt64([]int64{1, 2, 2, 3}, []int64{2, 3})
	})
	if allocs > 1 {
		t.Fatalf("SliceIntersectInt64 small input allocations = %f, want at most 1", allocs)
	}
}

func TestSliceDifferenceInt64SmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = SliceDifferenceInt64([]int64{1, 2, 2, 3}, []int64{2, 3})
	})
	if allocs > 1 {
		t.Fatalf("SliceDifferenceInt64 small input allocations = %f, want at most 1", allocs)
	}
}

func TestSliceSetHelpersKeepEmptyArrayShape(t *testing.T) {
	if got := SliceIntersectInt64(nil, []int64{1}); got == nil || len(got) != 0 {
		t.Fatalf("SliceIntersectInt64(empty) = %#v, want non-nil empty slice", got)
	}
	if got := SliceDifferenceInt64(nil, []int64{1}); got == nil || len(got) != 0 {
		t.Fatalf("SliceDifferenceInt64(empty) = %#v, want non-nil empty slice", got)
	}
}

func TestInt64CSVContainsReportsScopedMatches(t *testing.T) {
	hasValues, contains := Int64CSVContains(" 3, ,2,bad,3 ", 2)
	if !hasValues || !contains {
		t.Fatalf("Int64CSVContains(match) = %v/%v, want true/true", hasValues, contains)
	}

	hasValues, contains = Int64CSVContains(" 3, ,2,bad,3 ", 4)
	if !hasValues || contains {
		t.Fatalf("Int64CSVContains(miss) = %v/%v, want true/false", hasValues, contains)
	}

	hasValues, contains = Int64CSVContains(" ,bad, ", 4)
	if hasValues || contains {
		t.Fatalf("Int64CSVContains(empty) = %v/%v, want false/false", hasValues, contains)
	}

	hasValues, contains = Int64CSVContains("0,-1", 4)
	if !hasValues || contains {
		t.Fatalf("Int64CSVContains(non-positive) = %v/%v, want true/false", hasValues, contains)
	}
}

func TestInt64SliceToStringPreallocPath(t *testing.T) {
	got := Int64SliceToString([]int64{3, 2, 1})
	if want := "3,2,1"; got != want {
		t.Fatalf("Int64SliceToString() = %q, want %q", got, want)
	}
}

func TestRemoveDuplicateElementsSkipsEmptyStrings(t *testing.T) {
	got := RemoveDuplicateElements("alpha", "", "beta", "alpha", "")
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveDuplicateElements(string) = %#v, want %#v", got, want)
	}
}

func TestRemoveDuplicateElementsKeepsNonStringZeroValues(t *testing.T) {
	got := RemoveDuplicateElements(0, 1, 0, 2, 1)
	want := []int{0, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveDuplicateElements(int) = %#v, want %#v", got, want)
	}
}

func TestRemoveDuplicateElementsSmallInputAvoidsMapAllocation(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = RemoveDuplicateElements("alpha", "beta", "alpha")
	})
	if allocs > 1 {
		t.Fatalf("RemoveDuplicateElements small input allocations = %f, want at most 1", allocs)
	}
}

func TestRemoveDuplicateElementsEmptyKeepsArrayShape(t *testing.T) {
	got := RemoveDuplicateElements[string]()
	if got == nil || len(got) != 0 {
		t.Fatalf("RemoveDuplicateElements(empty) = %#v, want non-nil empty slice", got)
	}
}

func TestRemoveDuplicateElementsLargeDuplicateAvoidsMapAllocationAndFullCap(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+20)
	for index := range values {
		values[index] = "alpha"
	}
	got := RemoveDuplicateElements(values...)
	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveDuplicateElements large duplicate = %#v, want %#v", got, want)
	}
	if cap(got) != smallNormalizeSetThreshold+1 {
		t.Fatalf("RemoveDuplicateElements large duplicate cap = %d, want %d", cap(got), smallNormalizeSetThreshold+1)
	}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = RemoveDuplicateElements(values...)
	})
	if allocs > 1 {
		t.Fatalf("RemoveDuplicateElements large duplicate allocations = %f, want at most 1", allocs)
	}
}

func TestRemoveDuplicateElementsLargeEmptyStringAllocFree(t *testing.T) {
	values := make([]string, smallNormalizeSetThreshold+20)
	allocs := testing.AllocsPerRun(1000, func() {
		_ = RemoveDuplicateElements(values...)
	})
	if allocs != 0 {
		t.Fatalf("RemoveDuplicateElements large empty-string allocations = %f, want 0", allocs)
	}
	got := RemoveDuplicateElements(values...)
	if got == nil || len(got) != 0 {
		t.Fatalf("RemoveDuplicateElements large empty-string = %#v, want non-nil empty slice", got)
	}
}

func TestRemoveDuplicateElementsLargeUniqueUsesCapacityHintAfterUpgrade(t *testing.T) {
	values := make([]int, smallNormalizeSetThreshold+2)
	for index := range values {
		values[index] = index + 1
	}
	got := RemoveDuplicateElements(values...)
	if !reflect.DeepEqual(got, values) {
		t.Fatalf("RemoveDuplicateElements large unique = %#v, want %#v", got, values)
	}
	if cap(got) != len(values) {
		t.Fatalf("RemoveDuplicateElements large unique cap = %d, want %d", cap(got), len(values))
	}
}

func TestRemoveDuplicateElementsBoundsSparseUpgradeCapacity(t *testing.T) {
	values := make([]int, smallNormalizeSetThreshold+40)
	want := make([]int, 0, smallNormalizeSetThreshold+1)
	for index := 0; index <= smallNormalizeSetThreshold; index++ {
		value := index + 1
		values[index] = value
		want = append(want, value)
	}
	for index := smallNormalizeSetThreshold + 1; index < len(values); index++ {
		values[index] = 1
	}

	got := RemoveDuplicateElements(values...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveDuplicateElements sparse = %#v, want %#v", got, want)
	}
	if maxCap := len(want) * 2; cap(got) > maxCap {
		t.Fatalf("RemoveDuplicateElements sparse cap = %d, want at most %d", cap(got), maxCap)
	}
}

func TestRemoveStringElementPreservesOrderAndDuplicates(t *testing.T) {
	got := RemoveStringElement([]string{"alpha", "beta", "alpha", "gamma"}, "beta")
	want := []string{"alpha", "alpha", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveStringElement() = %#v, want %#v", got, want)
	}
}

func TestRemoveStringElementKeepsEmptyArrayShape(t *testing.T) {
	got := RemoveStringElement([]string{"alpha"}, "alpha")
	if got == nil || len(got) != 0 {
		t.Fatalf("RemoveStringElement(all removed) = %#v, want non-nil empty slice", got)
	}
	got = RemoveStringElement(nil, "alpha")
	if got == nil || len(got) != 0 {
		t.Fatalf("RemoveStringElement(empty source) = %#v, want non-nil empty slice", got)
	}
}

func TestRemoveStringElementLargeRemovalSetUsesMap(t *testing.T) {
	elements := []string{"keep", "drop-1", "drop-9", "keep"}
	var removals []string
	for index := 1; index <= smallNormalizeSetThreshold+1; index++ {
		removals = append(removals, "drop-"+strconv.Itoa(index))
	}

	got := RemoveStringElement(elements, removals...)
	want := []string{"keep", "keep"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RemoveStringElement(large removals) = %#v, want %#v", got, want)
	}
}

func TestRemoveStringElementSmallRemovalSetAvoidsExtraMapAllocation(t *testing.T) {
	values := []string{"alpha", "beta", "gamma"}
	allocs := testing.AllocsPerRun(1000, func() {
		_ = RemoveStringElement(values, "beta")
	})
	if allocs > 1 {
		t.Fatalf("RemoveStringElement small removal allocations = %f, want at most 1", allocs)
	}
}
