package coupon

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/neibniz/alpha/server/internal/couponpolicy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/xerr"
)

const maxCouponAmount int64 = 2_000_000_000

var couponCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{2,63}$`)

func normalizeCreateCoupon(req *types.CreateCouponRequest) error {
	if req == nil {
		return invalidCoupon("coupon request is required")
	}
	req.UsedCount = 0
	return normalizeCouponFields(
		&req.Name,
		&req.Code,
		&req.Type,
		&req.Discount,
		&req.StartTime,
		&req.ExpireTime,
		req.Count,
		req.UserLimit,
		0,
		true,
	)
}

func normalizeUpdateCoupon(req *types.UpdateCouponRequest, currentUsedCount int64) error {
	if req == nil || req.Id <= 0 {
		return invalidCoupon("coupon id is required")
	}
	if req.Enable == nil {
		return invalidCoupon("coupon enabled state is required")
	}
	req.UsedCount = currentUsedCount
	return normalizeCouponFields(
		&req.Name,
		&req.Code,
		&req.Type,
		&req.Discount,
		&req.StartTime,
		&req.ExpireTime,
		req.Count,
		req.UserLimit,
		currentUsedCount,
		false,
	)
}

func normalizeCouponFields(
	name *string,
	code *string,
	couponType *uint8,
	discount *int64,
	startTime *int64,
	expireTime *int64,
	count int64,
	userLimit int64,
	usedCount int64,
	allowEmptyCode bool,
) error {
	*name = strings.TrimSpace(*name)
	*code = strings.TrimSpace(*code)
	if length := utf8.RuneCountInString(*name); length == 0 || length > 255 {
		return invalidCoupon("coupon name must contain 1 to 255 characters")
	}
	if (!allowEmptyCode || *code != "") && !couponCodePattern.MatchString(*code) {
		return invalidCoupon("coupon code must contain 3 to 64 letters, numbers, underscores, or hyphens")
	}
	if *couponType != 1 && *couponType != 2 {
		return invalidCoupon("coupon type must be percentage or fixed amount")
	}
	if (*couponType == 1 && (*discount < 1 || *discount > 100)) ||
		(*couponType == 2 && (*discount < 1 || *discount > maxCouponAmount)) {
		return invalidCoupon("coupon discount is out of range")
	}
	*startTime = couponpolicy.NormalizeTimestamp(*startTime)
	*expireTime = couponpolicy.NormalizeTimestamp(*expireTime)
	if *startTime <= 0 || *expireTime <= *startTime {
		return invalidCoupon("coupon expiry must be later than its start time")
	}
	if count < 0 || userLimit < 0 || usedCount < 0 {
		return invalidCoupon("coupon usage limits cannot be negative")
	}
	if count > 0 && usedCount > count {
		return invalidCoupon("coupon count cannot be lower than its used count")
	}
	return nil
}

func invalidCoupon(message string) error {
	return xerr.NewErrCodeMsg(xerr.InvalidParams, message)
}
