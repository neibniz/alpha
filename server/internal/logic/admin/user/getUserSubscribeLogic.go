package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subcribe
func NewGetUserSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeLogic {
	return &GetUserSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeLogic) GetUserSubscribe(req *types.GetUserSubscribeListRequest) (resp *types.GetUserSubscribeListResponse, err error) {
	data, total, err := l.svcCtx.UserModel.QueryUserSubscribePage(l.ctx, req.UserId, req.Page, req.Size, 0, 1, 2, 3, 4, 5)
	if err != nil {
		l.Errorw("[GetUserSubscribeLogs] Get User Subscribe Error:", logger.Field("err", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Get User Subscribe Error")
	}

	resp = &types.GetUserSubscribeListResponse{
		List:  userSubscribeListRowsFromDetails(data),
		Total: total,
	}
	return
}
