package portal

import (
	"context"

	"github.com/neibniz/alpha/server/internal/paymentview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetAvailablePaymentMethodsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAvailablePaymentMethodsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAvailablePaymentMethodsLogic {
	return &GetAvailablePaymentMethodsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAvailablePaymentMethodsLogic) GetAvailablePaymentMethods() (resp *types.GetAvailablePaymentMethodsResponse, err error) {
	data, err := l.svcCtx.PaymentModel.FindAvailableMethods(l.ctx)
	if err != nil {
		l.Errorw("[GetAvailablePaymentMethods] database error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetAvailablePaymentMethods: %v", err.Error())
	}
	return &types.GetAvailablePaymentMethodsResponse{
		List: paymentview.MethodRows(data),
	}, nil
}
