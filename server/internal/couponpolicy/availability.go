package couponpolicy

import (
	"time"

	"github.com/neibniz/alpha/server/internal/model/coupon"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

const unixMillisecondsThreshold int64 = 100_000_000_000

func Time(value int64) time.Time {
	if value >= unixMillisecondsThreshold {
		return time.UnixMilli(value)
	}
	return time.Unix(value, 0)
}

func NormalizeTimestamp(value int64) int64 {
	if value <= 0 || value >= unixMillisecondsThreshold {
		return value
	}
	return value * int64(time.Second/time.Millisecond)
}

func Validate(info *coupon.Coupon, subscribeID int64, now time.Time) error {
	if info == nil || info.Enable == nil || !*info.Enable {
		return errors.Wrap(xerr.NewErrCode(xerr.CouponNotApplicable), "coupon is disabled")
	}
	if info.StartTime <= 0 || now.Before(Time(info.StartTime)) {
		return errors.Wrap(xerr.NewErrCode(xerr.CouponNotApplicable), "coupon has not started")
	}
	if info.ExpireTime <= 0 || !now.Before(Time(info.ExpireTime)) {
		return errors.Wrap(xerr.NewErrCode(xerr.CouponExpired), "coupon expired")
	}
	if info.Count > 0 && info.UsedCount >= info.Count {
		return errors.Wrap(xerr.NewErrCode(xerr.CouponInsufficientUsage), "coupon usage exhausted")
	}
	if !validDiscount(info.Type, info.Discount) {
		return errors.Wrap(xerr.NewErrCode(xerr.CouponNotApplicable), "coupon discount is invalid")
	}
	hasScope, matchesSubscribe := tool.Int64CSVContains(info.Subscribe, subscribeID)
	if hasScope && !matchesSubscribe {
		return errors.Wrap(xerr.NewErrCode(xerr.CouponNotApplicable), "coupon does not match subscription")
	}
	return nil
}

func validDiscount(couponType uint8, discount int64) bool {
	switch couponType {
	case 1:
		return discount >= 1 && discount <= 100
	case 2:
		return discount > 0
	default:
		return false
	}
}
