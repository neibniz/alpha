package analytics

import (
	"math"
	"testing"
)

func TestAnalyticsIntegerConversionsRejectOverflowAndNegativeValues(t *testing.T) {
	if _, err := analyticsUint64("download", -1); err == nil {
		t.Fatal("negative signed analytics value should be rejected")
	}
	if got, err := analyticsUint64("download", math.MaxInt64); err != nil || got != math.MaxInt64 {
		t.Fatalf("analyticsUint64(max) = %d, %v", got, err)
	}
	if _, err := analyticsInt64("download", uint64(math.MaxInt64)+1); err == nil {
		t.Fatal("oversized unsigned analytics value should be rejected")
	}
	if got, err := analyticsInt64("download", math.MaxInt64); err != nil || got != math.MaxInt64 {
		t.Fatalf("analyticsInt64(max) = %d, %v", got, err)
	}
}
