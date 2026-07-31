package nodeMultiplier

import (
	"testing"
	"time"
)

func TestNewNodeMultiplierManager(t *testing.T) {
	periods := []TimePeriod{
		{
			StartTime:  "23:00.000",
			EndTime:    "1:59.000",
			Multiplier: 1.2,
		},
		{
			StartTime:  "12:00.000",
			EndTime:    "13:59.000",
			Multiplier: 0.5,
		},
	}
	m := NewNodeMultiplierManager(periods)
	if len(m.Periods) != 2 {
		t.Fatalf("expected 2 valid periods, got %d", len(m.Periods))
	}

	if got := m.GetMultiplier(time.Date(0, 1, 1, 0, 10, 0, 0, time.UTC)); got != 1.2 {
		t.Fatalf("00:10 multiplier: expected 1.2, got %v", got)
	}
	if got := m.GetMultiplier(time.Date(0, 1, 1, 12, 0, 0, 0, time.UTC)); got != 0.5 {
		t.Fatalf("12:00 multiplier: expected 0.5, got %v", got)
	}
	if got := m.GetMultiplier(time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)); got != 1 {
		t.Fatalf("14:00 multiplier: expected 1, got %v", got)
	}
	if got := m.GetMultiplierScale(time.Date(0, 1, 1, 0, 10, 0, 0, time.UTC)); got != 1200 {
		t.Fatalf("00:10 multiplier scale: expected 1200, got %v", got)
	}
}

func TestNewNodeMultiplierManagerSkipsInvalidPeriods(t *testing.T) {
	m := NewNodeMultiplierManager([]TimePeriod{
		{StartTime: "bad", EndTime: "01:00.000", Multiplier: 1.2},
		{StartTime: "02:00.000", EndTime: "03:00.000", Multiplier: 0},
		{StartTime: "04:00.000", EndTime: "05:00.000", Multiplier: 0.8},
	})

	if len(m.Periods) != 1 {
		t.Fatalf("expected 1 valid period, got %d", len(m.Periods))
	}
	if got := m.GetMultiplier(time.Date(0, 1, 1, 4, 30, 0, 0, time.UTC)); got != 0.8 {
		t.Fatalf("04:30 multiplier: expected 0.8, got %v", got)
	}
}

func TestApplyTrafficMultiplierUsesIntegerMath(t *testing.T) {
	tests := []struct {
		name            string
		bytes           int64
		multiplierScale int64
		want            int64
	}{
		{name: "default", bytes: 1024, multiplierScale: MultiplierScale, want: 1024},
		{name: "increase", bytes: 1024, multiplierScale: 1200, want: 1228},
		{name: "decrease", bytes: 1024, multiplierScale: 500, want: 512},
		{name: "zero bytes", bytes: 0, multiplierScale: 1200, want: 0},
		{name: "invalid multiplier keeps bytes", bytes: 1024, multiplierScale: 0, want: 1024},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyTrafficMultiplier(tt.bytes, tt.multiplierScale); got != tt.want {
				t.Fatalf("ApplyTrafficMultiplier(%d, %d): expected %d, got %d", tt.bytes, tt.multiplierScale, tt.want, got)
			}
		})
	}
}
