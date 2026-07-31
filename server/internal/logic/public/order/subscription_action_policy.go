package order

import (
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

func ensureRenewalActionAllowed(status uint8, expireTime time.Time, product *modelSubscribe.Subscribe, now time.Time) error {
	productAvailable := product != nil
	renewalEnabled := productAvailable && product.AllowRenewal != nil && *product.AllowRenewal
	capability := subscriptionstate.RenewalCapability(
		status,
		expireTime,
		productAvailable,
		renewalEnabled,
		now,
	)
	if capability.Allowed {
		return nil
	}
	return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "subscription cannot be renewed: %s", capability.Reason)
}

func ensureResetTrafficActionAllowed(status uint8, expireTime time.Time, product *modelSubscribe.Subscribe, now time.Time) error {
	productAvailable := product != nil
	resetEnabled := productAvailable && product.AllowResetTraffic != nil && *product.AllowResetTraffic
	capability := subscriptionstate.ResetTrafficCapability(
		status,
		expireTime,
		productAvailable,
		resetEnabled,
		now,
	)
	if capability.Allowed {
		return nil
	}
	if capability.Reason == subscriptionstate.ReasonExpired {
		return errors.Wrapf(xerr.NewErrCode(xerr.SubscribeExpired), "subscription cannot reset traffic: %s", capability.Reason)
	}
	return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "subscription cannot reset traffic: %s", capability.Reason)
}
