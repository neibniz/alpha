package order

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type GetOrderDetailLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrderDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderDetailLogic {
	return &GetOrderDetailLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrderDetailLogic) GetOrderDetail(req *types.GetOrderDetailRequest) (*types.OrderDetail, error) {
	item, err := l.svcCtx.OrderModel.FindOneAdminDetails(l.ctx, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.OrderNotExist), "order does not exist")
		}
		l.Errorw("[GetOrderDetail] Database error", logger.Field("error", err.Error()), logger.Field("order_id", req.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find order detail: %v", err)
	}
	resp := orderDetailFromDetails(item)
	return &resp, nil
}
