package coupon

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdateCouponLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Update coupon
func NewUpdateCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCouponLogic {
	return &UpdateCouponLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCouponLogic) UpdateCoupon(req *types.UpdateCouponRequest) error {
	if req == nil || req.Id <= 0 {
		return errors.Wrap(invalidCoupon("coupon id is required"), "validate coupon")
	}
	current, err := l.svcCtx.CouponModel.FindOne(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[UpdateCoupon] Query Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query coupon error: %v", err)
	}
	if req.Code == "" {
		req.Code = current.Code
	}
	if err = normalizeUpdateCoupon(req, current.UsedCount); err != nil {
		return errors.Wrap(err, "validate coupon")
	}
	couponInfo := couponFromUpdateRequest(req)
	err = l.svcCtx.CouponModel.Update(l.ctx, couponInfo)
	if err != nil {
		l.Errorw("[UpdateCoupon] Database Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update coupon error: %v", err.Error())
	}
	return nil
}
