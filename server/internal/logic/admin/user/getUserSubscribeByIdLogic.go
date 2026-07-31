package user

import (
	"context"
	"time"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/subscribeprofile"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserSubscribeByIdLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subcribe by id
func NewGetUserSubscribeByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeByIdLogic {
	return &GetUserSubscribeByIdLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeByIdLogic) GetUserSubscribeById(req *types.GetUserSubscribeByIdRequest) (resp *types.UserSubscribeDetail, err error) {
	sub, err := l.svcCtx.UserModel.FindOneSubscribeDetailsById(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[GetUserSubscribeByIdLogic] FindOneSubscribeDetailsById error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneSubscribeDetailsById error: %v", err.Error())
	}
	subscribeDetails := userSubscribeDetailFromDetails(sub, time.Now())
	presets, err := subscribeprofile.QueryPresets(l.ctx, l.svcCtx, sub.UserId, sub.Id)
	if err != nil {
		l.Errorw("[GetUserSubscribeByIdLogic] QueryPresets error", logger.Field("error", err.Error()))
		return nil, err
	}
	subscribeDetails.Presets = presets
	if sub.OrderId > 0 {
		var orderRow modelOrder.Order
		if err = l.svcCtx.DB.WithContext(l.ctx).
			Model(&modelOrder.Order{}).
			Select("order_no").
			Where("id = ?", sub.OrderId).
			Scan(&orderRow).Error; err != nil {
			l.Errorw("[GetUserSubscribeByIdLogic] query order number error", logger.Field("error", err.Error()), logger.Field("order_id", sub.OrderId))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query order number error: %v", err)
		}
		subscribeDetails.OrderNo = orderRow.OrderNo
	}
	return &subscribeDetails, nil
}
