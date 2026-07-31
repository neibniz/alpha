package server

import "testing"

func TestNormalizeRealtimeServerIDsDeduplicatesAndLimits(t *testing.T) {
	input := make([]int64, 0, maxRealtimeServerSubscriptions+20)
	input = append(input, 0, -1, 7, 7)
	for id := int64(1); id <= maxRealtimeServerSubscriptions+20; id++ {
		input = append(input, id)
	}

	ids := normalizeRealtimeServerIDs(input)
	if len(ids) != maxRealtimeServerSubscriptions {
		t.Fatalf("len(ids) = %d, want %d", len(ids), maxRealtimeServerSubscriptions)
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			t.Fatalf("invalid id %d", id)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = struct{}{}
	}
}

func TestParseServerIDsNormalizesInput(t *testing.T) {
	ids := parseServerIDs(" 3,2,3,invalid,-1,1 ")
	want := []int64{3, 2, 1}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids = %v, want %v", ids, want)
		}
	}
}
