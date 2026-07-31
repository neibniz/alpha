package money

import (
	"math"
	"testing"
)

func TestMajorToMinorRoundsAndRejectsUnsafeValues(t *testing.T) {
	if got, err := MajorToMinor(10.125); err != nil || got != 1013 {
		t.Fatalf("MajorToMinor() = %d, %v", got, err)
	}
	for _, value := range []float64{-1, math.NaN(), math.Inf(1), float64(math.MaxInt64)} {
		if _, err := MajorToMinor(value); err == nil {
			t.Fatalf("MajorToMinor(%v) should fail", value)
		}
	}
}

func TestConvertMinorValidatesRateAndOverflow(t *testing.T) {
	if got, err := ConvertMinor(123, 7.25); err != nil || got != 892 {
		t.Fatalf("ConvertMinor() = %d, %v", got, err)
	}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1), float64(math.MaxInt64)} {
		if _, err := ConvertMinor(123, rate); err == nil {
			t.Fatalf("ConvertMinor(rate=%v) should fail", rate)
		}
	}
}
