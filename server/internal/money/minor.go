package money

import (
	"errors"
	"math"
)

func MajorToMinor(value float64) (int64, error) {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("money amount must be a finite non-negative number")
	}
	return roundedInt64(value * 100)
}

func ConvertMinor(value int64, rate float64) (int64, error) {
	if value < 0 {
		return 0, errors.New("money amount must not be negative")
	}
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0, errors.New("exchange rate must be a finite positive number")
	}
	return roundedInt64(float64(value) * rate)
}

func roundedInt64(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= float64(math.MaxInt64) {
		return 0, errors.New("money amount exceeds int64")
	}
	return int64(math.Round(value)), nil
}
