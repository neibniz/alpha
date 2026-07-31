package subscriptionstate

import "time"

const (
	ReasonProductUnavailable = "product_unavailable"
	ReasonRenewalDisabled    = "renewal_disabled"
	ReasonResetDisabled      = "reset_disabled"
)

type Capability struct {
	Allowed bool
	Reason  string
}

func RenewalCapability(status uint8, expireAt time.Time, productAvailable, renewalEnabled bool, now time.Time) Capability {
	if !productAvailable {
		return Capability{Reason: ReasonProductUnavailable}
	}
	if !renewalEnabled {
		return Capability{Reason: ReasonRenewalDisabled}
	}

	effective, reason := EffectiveStatus(status, expireAt, 0, 0, 0, now)
	switch effective {
	case StatusActive, StatusExhausted, StatusExpired:
		return Capability{Allowed: true}
	default:
		return Capability{Reason: reason}
	}
}

func ResetTrafficCapability(status uint8, expireAt time.Time, productAvailable, resetEnabled bool, now time.Time) Capability {
	if !productAvailable {
		return Capability{Reason: ReasonProductUnavailable}
	}
	if !resetEnabled {
		return Capability{Reason: ReasonResetDisabled}
	}

	effective, reason := EffectiveStatus(status, expireAt, 0, 0, 0, now)
	switch effective {
	case StatusActive, StatusExhausted:
		return Capability{Allowed: true}
	default:
		return Capability{Reason: reason}
	}
}
