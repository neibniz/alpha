package quotapolicy

import (
	"math"
	"testing"
)

func TestValidateRejectsOverflowingQuotaValues(t *testing.T) {
	for _, testCase := range []struct {
		days      uint64
		giftType  uint8
		giftValue uint64
	}{
		{days: MaxExtensionDays + 1},
		{giftType: 1, giftValue: uint64(math.MaxInt64) + 1},
		{giftType: 2, giftValue: MaxGiftPercentage + 1},
		{giftType: 9, giftValue: 1},
	} {
		if err := Validate(testCase.days, testCase.giftType, testCase.giftValue); err == nil {
			t.Fatalf("Validate(%d, %d, %d) accepted invalid values", testCase.days, testCase.giftType, testCase.giftValue)
		}
	}
}

func TestGiftAmountAndBalanceAreOverflowSafe(t *testing.T) {
	amount, err := GiftAmount(999, 2, 10)
	if err != nil || amount != 99 {
		t.Fatalf("GiftAmount() = %d, %v; want 99, nil", amount, err)
	}
	if _, err := AddGiftBalance(math.MaxInt64, 1); err == nil {
		t.Fatal("overflowing balance was accepted")
	}
}

func TestExtensionDaysUsesValidatedIntRange(t *testing.T) {
	if got, err := ExtensionDays(MaxExtensionDays); err != nil || got != int(MaxExtensionDays) {
		t.Fatalf("ExtensionDays(max) = %d, %v", got, err)
	}
	if _, err := ExtensionDays(MaxExtensionDays + 1); err == nil {
		t.Fatal("ExtensionDays should reject values above the policy limit")
	}
}
