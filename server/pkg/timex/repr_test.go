package timex

import (
	"testing"
	"time"
)

func TestUnixMilliOrZero(t *testing.T) {
	if got := UnixMilliOrZero(time.Time{}); got != 0 {
		t.Fatalf("zero time should return 0, got %d", got)
	}

	if got := UnixMilliOrZero(time.UnixMilli(-1)); got != 0 {
		t.Fatalf("pre-epoch time should return 0, got %d", got)
	}

	want := int64(1716192000000)
	if got := UnixMilliOrZero(time.UnixMilli(want)); got != want {
		t.Fatalf("valid time should return unix milliseconds, got %d want %d", got, want)
	}
}

func TestUnixOrZero(t *testing.T) {
	if got := UnixOrZero(time.Time{}); got != 0 {
		t.Fatalf("zero time should return 0, got %d", got)
	}

	if got := UnixOrZero(time.Unix(-1, 0)); got != 0 {
		t.Fatalf("pre-epoch time should return 0, got %d", got)
	}

	want := int64(1716192000)
	if got := UnixOrZero(time.Unix(want, 0)); got != want {
		t.Fatalf("valid time should return unix seconds, got %d want %d", got, want)
	}
}
