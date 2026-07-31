package coupon

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetCouponLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get coupon
func NewGetCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCouponLogic {
	return &GetCouponLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCouponLogic) GetCoupon(req *types.GetCouponRequest) (resp *types.Coupon, err error) {
	data, err := l.svcCtx.CouponModel.FindOne(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[GetCoupon] Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get coupon error: %v", err.Error())
	}
	response := couponResponseFromModel(data)
	return &response, nil
}
