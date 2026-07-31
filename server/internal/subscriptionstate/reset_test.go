package subscriptionstate

import (
	"testing"
	"time"
)

func TestNextTrafficResetAtMonthlyClampsEndOfMonth(t *testing.T) {
	location := time.FixedZone("UTC+8", 8*60*60)
	anchor := time.Date(2026, time.January, 31, 12, 0, 0, 0, location)
	after := time.Date(2026, time.February, 1, 0, 0, 0, 0, location)

	got := NextTrafficResetAt(ResetCycleMonthly, anchor, after, time.Time{}, location)
	want := time.Date(2026, time.February, 28, 0, 0, 0, 0, location)
	if got == nil || !got.Equal(want) {
		t.Fatalf("next reset = %v, want %v", got, want)
	}
}

func TestNextTrafficResetAtYearlyClampsLeapDay(t *testing.T) {
	location := time.UTC
	anchor := time.Date(2024, time.February, 29, 12, 0, 0, 0, location)
	after := time.Date(2026, time.January, 1, 0, 0, 0, 0, location)

	got := NextTrafficResetAt(ResetCycleYearly, anchor, after, time.Time{}, location)
	want := time.Date(2026, time.February, 28, 0, 0, 0, 0, location)
	if got == nil || !got.Equal(want) {
		t.Fatalf("next reset = %v, want %v", got, want)
	}
}

func TestNextTrafficResetAtSkipsBoundaryAtExpiry(t *testing.T) {
	location := time.UTC
	anchor := time.Date(2026, time.January, 15, 12, 0, 0, 0, location)
	expireAt := time.Date(2026, time.February, 15, 0, 0, 0, 0, location)

	if got := NextTrafficResetAt(ResetCycleMonthly, anchor, anchor, expireAt, location); got != nil {
		t.Fatalf("next reset = %v, want nil at expiry boundary", got)
	}
}

func TestNextTrafficResetAtFirstDayIsStrictlyAfter(t *testing.T) {
	location := time.UTC
	after := time.Date(2026, time.July, 1, 0, 0, 0, 0, location)

	got := NextTrafficResetAt(ResetCycleFirst, after, after, time.Time{}, location)
	want := time.Date(2026, time.August, 1, 0, 0, 0, 0, location)
	if got == nil || !got.Equal(want) {
		t.Fatalf("next reset = %v, want %v", got, want)
	}
}
