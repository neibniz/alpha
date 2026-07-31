package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/logview"
	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserSubscribeLogsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subcribe logs
func NewGetUserSubscribeLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeLogsLogic {
	return &GetUserSubscribeLogsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeLogsLogic) GetUserSubscribeLogs(req *types.GetUserSubscribeLogsRequest) (resp *types.GetUserSubscribeLogsResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterSubscribeAccessLog(l.ctx, &log.SubscribeAccessFilterParams{
		Page:            req.Page,
		Size:            req.Size,
		UserId:          req.UserId,
		UserSubscribeId: req.SubscribeId,
	})

	if err != nil {
		l.Errorw("[GetUserSubscribeLogs] Get User Subscribe Logs Error:", logger.Field("err", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Get User Subscribe Logs Error")
	}

	return &types.GetUserSubscribeLogsResponse{
		List:  logview.UserSubscribeRows(data, logview.FromIPLocation(l.svcCtx.GeoIP)),
		Total: total,
	}, err
}
