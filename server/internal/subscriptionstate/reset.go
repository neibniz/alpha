package subscriptionstate

import "time"

var trafficResetLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

const (
	ResetCycleNone    uint8 = 0
	ResetCycleFirst   uint8 = 1
	ResetCycleMonthly uint8 = 2
	ResetCycleYearly  uint8 = 3
)

func TrafficResetLocation() *time.Location {
	return trafficResetLocation
}

func NormalizeResetCycle(cycle int64) uint8 {
	if cycle < int64(ResetCycleFirst) || cycle > int64(ResetCycleYearly) {
		return ResetCycleNone
	}
	return uint8(cycle)
}

// NextTrafficResetAt returns the first reset boundary strictly after the given
// time. Monthly and yearly cycles stay anchored to the subscription start date.
func NextTrafficResetAt(cycle uint8, anchor, after, expireAt time.Time, location *time.Location) *time.Time {
	if cycle < ResetCycleFirst || cycle > ResetCycleYearly {
		return nil
	}
	if location == nil {
		location = time.UTC
	}
	if anchor.IsZero() {
		anchor = after
	}

	anchor = anchor.In(location)
	after = after.In(location)
	var next time.Time
	switch cycle {
	case ResetCycleFirst:
		next = time.Date(after.Year(), after.Month(), 1, 0, 0, 0, 0, location)
		if !next.After(after) {
			next = time.Date(after.Year(), after.Month()+1, 1, 0, 0, 0, 0, location)
		}
	case ResetCycleMonthly:
		next = anchoredCalendarDate(after.Year(), after.Month(), anchor.Day(), location)
		if !next.After(after) {
			next = anchoredCalendarDate(after.Year(), after.Month()+1, anchor.Day(), location)
		}
	case ResetCycleYearly:
		next = anchoredCalendarDate(after.Year(), anchor.Month(), anchor.Day(), location)
		if !next.After(after) {
			next = anchoredCalendarDate(after.Year()+1, anchor.Month(), anchor.Day(), location)
		}
	}

	if !IsUnlimitedExpiry(expireAt) && !next.Before(expireAt) {
		return nil
	}
	return &next
}

func anchoredCalendarDate(year int, month time.Month, day int, location *time.Location) time.Time {
	if month < time.January || month > time.December {
		base := time.Date(year, month, 1, 0, 0, 0, 0, location)
		year, month = base.Year(), base.Month()
	}
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, location).Day()
	if day > lastDay {
		day = lastDay
	}
	if day < 1 {
		day = 1
	}
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}
