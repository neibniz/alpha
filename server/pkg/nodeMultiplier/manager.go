package nodeMultiplier

import (
	"math"
	"time"
)

const MultiplierScale int64 = 1000

type TimePeriod struct {
	StartTime  string  `json:"start_time"`
	EndTime    string  `json:"end_time"`
	Multiplier float32 `json:"multiplier"`
}

type Manager struct {
	Periods []TimePeriod
	periods []parsedPeriod
}

type parsedPeriod struct {
	start           time.Time
	end             time.Time
	multiplierScale int64
}

func NewNodeMultiplierManager(periods []TimePeriod) *Manager {
	validPeriods := make([]TimePeriod, 0, len(periods))
	parsedPeriods := make([]parsedPeriod, 0, len(periods))
	for _, period := range periods {
		parsed, ok := parsePeriod(period)
		if !ok {
			continue
		}
		validPeriods = append(validPeriods, period)
		parsedPeriods = append(parsedPeriods, parsed)
	}
	return &Manager{
		Periods: validPeriods,
		periods: parsedPeriods,
	}
}

func (m *Manager) GetMultiplier(current time.Time) float32 {
	return float32(m.GetMultiplierScale(current)) / float32(MultiplierScale)
}

func (m *Manager) GetMultiplierScale(current time.Time) int64 {
	currentTime := normalizeTime(current)
	for _, period := range m.periods {
		if isInTimePeriod(currentTime, period.start, period.end) {
			return period.multiplierScale
		}
	}
	return MultiplierScale // Default multiplier is 1 (no change)
}

func ApplyTrafficMultiplier(bytes int64, multiplierScale int64) int64 {
	if bytes <= 0 {
		return 0
	}
	if multiplierScale <= 0 {
		return bytes
	}
	whole := bytes / MultiplierScale
	rem := bytes % MultiplierScale
	if whole > math.MaxInt64/multiplierScale {
		return math.MaxInt64
	}
	result := whole * multiplierScale
	if rem == 0 {
		return result
	}
	if rem > math.MaxInt64/multiplierScale {
		return math.MaxInt64
	}
	extra := (rem * multiplierScale) / MultiplierScale
	if result > math.MaxInt64-extra {
		return math.MaxInt64
	}
	return result + extra
}

func (m *Manager) isInTimePeriod(current time.Time, start, end string) bool {
	startTime, err := parseClock(start)
	if err != nil {
		return false
	}
	endTime, err := parseClock(end)
	if err != nil {
		return false
	}

	return isInTimePeriod(normalizeTime(current), startTime, endTime)
}

func parsePeriod(period TimePeriod) (parsedPeriod, bool) {
	multiplierScale := multiplierToScale(period.Multiplier)
	if multiplierScale <= 0 {
		return parsedPeriod{}, false
	}
	start, err := parseClock(period.StartTime)
	if err != nil {
		return parsedPeriod{}, false
	}
	end, err := parseClock(period.EndTime)
	if err != nil {
		return parsedPeriod{}, false
	}
	return parsedPeriod{
		start:           start,
		end:             end,
		multiplierScale: multiplierScale,
	}, true
}

func multiplierToScale(multiplier float32) int64 {
	if multiplier <= 0 {
		return 0
	}
	return int64(math.Round(float64(multiplier) * float64(MultiplierScale)))
}

func parseClock(value string) (time.Time, error) {
	parsed, err := time.Parse("15:04.000", value)
	if err != nil {
		return time.Time{}, err
	}
	return normalizeTime(parsed), nil
}

func normalizeTime(value time.Time) time.Time {
	return time.Date(0, 1, 1, value.Hour(), value.Minute(), 0, 0, time.UTC)
}

func isInTimePeriod(current, start, end time.Time) bool {
	if start.Equal(end) {
		return true
	}
	if start.Before(end) {
		return !current.Before(start) && !current.After(end)
	}
	// Handle ranges that cross midnight
	return !current.Before(start) || !current.After(end)
}
