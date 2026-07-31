package subscribe

import "testing"

func TestSubscribeNodeIDsToStringNormalizesIDs(t *testing.T) {
	got := subscribeNodeIDsToString([]int64{4, 0, 2, 4, -1, 3, 2})
	if want := "4,2,3"; got != want {
		t.Fatalf("subscribeNodeIDsToString() = %q, want %q", got, want)
	}
}

func TestSubscribeNodeIDsToStringEmptyWhenOnlyInvalidIDs(t *testing.T) {
	got := subscribeNodeIDsToString([]int64{0, -1, 0})
	if got != "" {
		t.Fatalf("subscribeNodeIDsToString() = %q, want empty", got)
	}
}
