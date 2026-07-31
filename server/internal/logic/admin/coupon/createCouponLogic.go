package coupon

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/random"
	"github.com/neibniz/alpha/server/pkg/snowflake"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CreateCouponLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Create coupon
func NewCreateCouponLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateCouponLogic {
	return &CreateCouponLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateCouponLogic) CreateCoupon(req *types.CreateCouponRequest) error {
	if err := normalizeCreateCoupon(req); err != nil {
		return errors.Wrap(err, "validate coupon")
	}
	if req.Code == "" {
		sid := snowflake.GetID()
		req.Code = random.KeyNew(4, 2) + "-" + random.StrToDashedString(random.EncodeBase36(sid))
	}
	couponInfo := couponFromCreateRequest(req)
	err := l.svcCtx.CouponModel.Insert(l.ctx, couponInfo)
	if err != nil {
		l.Errorw("[CreateCoupon] Database Error", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create coupon error: %v", err.Error())
	}
	return nil
}
