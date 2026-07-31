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

type GetUserSubscribeResetTrafficLogsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subcribe reset traffic logs
func NewGetUserSubscribeResetTrafficLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeResetTrafficLogsLogic {
	return &GetUserSubscribeResetTrafficLogsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeResetTrafficLogsLogic) GetUserSubscribeResetTrafficLogs(req *types.GetUserSubscribeResetTrafficLogsRequest) (resp *types.GetUserSubscribeResetTrafficLogsResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterResetSubscribeLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		ObjectID: req.UserSubscribeId,
	})
	if err != nil {
		l.Errorf("[ResetSubscribeTrafficLog] failed to filter system log: %v", err)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FilterSystemLog failed, err: %v", err)
	}

	return &types.GetUserSubscribeResetTrafficLogsResponse{
		Total: total,
		List:  logview.ResetSubscribeTrafficRows(data),
	}, nil
}
