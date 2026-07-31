package timex

import (
	"fmt"
	"time"
)

// ReprOfDuration returns the string representation of given duration in ms.
func ReprOfDuration(duration time.Duration) string {
	return fmt.Sprintf("%.1fms", float32(duration)/float32(time.Millisecond))
}

func UnixMilliOrZero(t time.Time) int64 {
	if t.IsZero() || !t.After(time.UnixMilli(0)) {
		return 0
	}
	return t.UnixMilli()
}

func UnixMilliPointerOrZero(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return UnixMilliOrZero(*t)
}

func UnixOrZero(t time.Time) int64 {
	if t.IsZero() || !t.After(time.Unix(0, 0)) {
		return 0
	}
	return t.Unix()
}
