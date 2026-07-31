package payment

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetPaymentMethodLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetPaymentMethodLogic Get Payment Method
func NewGetPaymentMethodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPaymentMethodLogic {
	return &GetPaymentMethodLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPaymentMethodLogic) GetPaymentMethod(req *types.GetPaymentMethodRequest) (*types.PaymentConfig, error) {
	data, err := l.svcCtx.PaymentModel.FindOneForAdmin(l.ctx, req.Id, true)
	if err != nil {
		l.Errorw("find payment method error", logger.Field("id", req.Id), logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method error: %s", err.Error())
	}

	resp := paymentConfigFromModel(data)
	return &resp, nil
}
