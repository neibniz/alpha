package xray

import (
	"testing"
	"time"
)

func TestActiveUserTrackerDeltaAndFull(t *testing.T) {
	tracker := NewActiveUserTracker(2 * time.Minute)
	now := time.Unix(100, 0)

	tracker.Observe([]int64{2, 1, 2}, now)
	delta, ok := tracker.Delta(now)
	if !ok {
		t.Fatal("expected delta")
	}
	if delta.Mode != "delta" || delta.Seq != 1 || delta.TTLSeconds != 120 {
		t.Fatalf("unexpected delta metadata: %#v", delta)
	}
	if delta.InstanceID == "" {
		t.Fatal("expected delta instance id")
	}
	if !sameInt64s(delta.Active, []int64{1, 2}) {
		t.Fatalf("active = %#v, want [1 2]", delta.Active)
	}
	tracker.MarkSent(delta)

	tracker.Observe([]int64{2, 3}, now.Add(time.Minute))
	delta, ok = tracker.Delta(now.Add(time.Minute))
	if !ok {
		t.Fatal("expected refresh delta")
	}
	if !sameInt64s(delta.Active, []int64{2, 3}) {
		t.Fatalf("active = %#v, want [2 3]", delta.Active)
	}
	tracker.MarkSent(delta)

	full := tracker.Full(now.Add(time.Minute))
	if full.Mode != "full" || full.Seq != 3 {
		t.Fatalf("unexpected full metadata: %#v", full)
	}
	if full.InstanceID != delta.InstanceID {
		t.Fatalf("full instance id = %q, want %q", full.InstanceID, delta.InstanceID)
	}
	if !sameInt64s(full.Active, []int64{1, 2, 3}) {
		t.Fatalf("full active = %#v, want [1 2 3]", full.Active)
	}
}

func TestActiveUserTrackerExpiresReportedUsers(t *testing.T) {
	tracker := NewActiveUserTracker(time.Minute)
	now := time.Unix(100, 0)

	tracker.Observe([]int64{1}, now)
	delta, ok := tracker.Delta(now)
	if !ok {
		t.Fatal("expected delta")
	}
	tracker.MarkSent(delta)

	if count := tracker.Count(now.Add(2 * time.Minute)); count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
	delta, ok = tracker.Delta(now.Add(2 * time.Minute))
	if !ok {
		t.Fatal("expected inactive delta")
	}
	if !sameInt64s(delta.Inactive, []int64{1}) {
		t.Fatalf("inactive = %#v, want [1]", delta.Inactive)
	}
}

func TestActiveUserTrackerMarkSentFullClearsPendingDeltas(t *testing.T) {
	tracker := NewActiveUserTracker(time.Minute)
	now := time.Unix(100, 0)

	tracker.Observe([]int64{1}, now)
	delta, ok := tracker.Delta(now)
	if !ok {
		t.Fatal("expected initial active delta")
	}
	tracker.MarkSent(delta)

	if count := tracker.Count(now.Add(2 * time.Minute)); count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
	full := tracker.Full(now.Add(2 * time.Minute))
	tracker.MarkSent(full)

	if delta, ok := tracker.Delta(now.Add(2 * time.Minute)); ok {
		t.Fatalf("full mark sent must clear pending deltas, got %#v", delta)
	}
}

func sameInt64s(a []int64, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
