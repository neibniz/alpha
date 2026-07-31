package quotapolicy

import (
	"errors"
	"math"
)

const (
	MaxExtensionDays  uint64 = 36_500
	MaxGiftPercentage uint64 = 100
)

func Validate(days uint64, giftType uint8, giftValue uint64) error {
	if days > MaxExtensionDays {
		return errors.New("quota extension exceeds 36500 days")
	}
	if giftValue == 0 {
		return nil
	}
	switch giftType {
	case 1:
		if giftValue > math.MaxInt64 {
			return errors.New("quota gift amount exceeds int64")
		}
	case 2:
		if giftValue > MaxGiftPercentage {
			return errors.New("quota gift percentage exceeds 100")
		}
	default:
		return errors.New("invalid quota gift type")
	}
	return nil
}

func ExtensionDays(days uint64) (int, error) {
	if err := Validate(days, 0, 0); err != nil {
		return 0, err
	}
	return int(days), nil // #nosec G115 -- days is bounded to 36500 above.
}

func GiftAmount(unitPrice int64, giftType uint8, giftValue uint64) (int64, error) {
	if err := Validate(0, giftType, giftValue); err != nil {
		return 0, err
	}
	if giftValue == 0 {
		return 0, nil
	}
	if giftType == 1 {
		return int64(giftValue), nil // #nosec G115 -- Validate bounds giftValue to MaxInt64.
	}
	if unitPrice <= 0 {
		return 0, nil
	}
	percentage := int64(giftValue) // #nosec G115 -- percentage is bounded to 100.
	return (unitPrice/100)*percentage + ((unitPrice%100)*percentage)/100, nil
}

func AddGiftBalance(current, amount int64) (int64, error) {
	if current < 0 || amount < 0 || amount > math.MaxInt64-current {
		return 0, errors.New("quota gift balance overflow")
	}
	return current + amount, nil
}
