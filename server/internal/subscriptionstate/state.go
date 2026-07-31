package subscriptionstate

import (
	"math"
	"time"
)

const (
	StatusPending   uint8 = 0
	StatusActive    uint8 = 1
	StatusExhausted uint8 = 2
	StatusExpired   uint8 = 3
	StatusDeducted  uint8 = 4
	StatusStopped   uint8 = 5

	ReasonPending   = "pending"
	ReasonActive    = "active"
	ReasonExhausted = "traffic_exhausted"
	ReasonExpired   = "expired"
	ReasonDeducted  = "deducted"
	ReasonStopped   = "stopped"
	ReasonUnknown   = "unknown"
)

// IsUnlimitedExpiry supports both Go's zero time and the legacy Unix epoch sentinel.
func IsUnlimitedExpiry(expireAt time.Time) bool {
	return expireAt.IsZero() || expireAt.UnixMilli() == 0
}

func IsExpired(expireAt, now time.Time) bool {
	return !IsUnlimitedExpiry(expireAt) && !expireAt.After(now)
}

// TrafficExhausted avoids adding two untrusted counters, which could overflow int64.
func TrafficExhausted(limit, download, upload int64) bool {
	if limit <= 0 {
		return false
	}
	if download < 0 || upload < 0 {
		return true
	}
	if download >= limit {
		return true
	}
	return upload >= limit-download
}

func UsedTraffic(download, upload int64) int64 {
	if download < 0 {
		download = 0
	}
	if upload < 0 {
		upload = 0
	}
	if download > math.MaxInt64-upload {
		return math.MaxInt64
	}
	return download + upload
}

// RemainingTraffic returns -1 for an unlimited quota.
func RemainingTraffic(limit, download, upload int64) int64 {
	if limit <= 0 {
		return -1
	}
	used := UsedTraffic(download, upload)
	if used >= limit {
		return 0
	}
	return limit - used
}

func EffectiveStatus(status uint8, expireAt time.Time, traffic, download, upload int64, now time.Time) (uint8, string) {
	if (status == StatusActive || status == StatusExhausted) && IsExpired(expireAt, now) {
		return StatusExpired, ReasonExpired
	}
	if status == StatusActive {
		if TrafficExhausted(traffic, download, upload) {
			return StatusExhausted, ReasonExhausted
		}
	}
	switch status {
	case StatusPending:
		return status, ReasonPending
	case StatusActive:
		return status, ReasonActive
	case StatusExhausted:
		return status, ReasonExhausted
	case StatusExpired:
		return status, ReasonExpired
	case StatusDeducted:
		return status, ReasonDeducted
	case StatusStopped:
		return status, ReasonStopped
	default:
		return status, ReasonUnknown
	}
}

func Available(status uint8, expireAt time.Time, traffic, download, upload int64, now time.Time) bool {
	effective, _ := EffectiveStatus(status, expireAt, traffic, download, upload, now)
	return effective == StatusActive
}
