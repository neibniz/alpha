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

type GetUserLoginLogsLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user login logs
func NewGetUserLoginLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLoginLogsLogic {
	return &GetUserLoginLogsLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserLoginLogsLogic) GetUserLoginLogs(req *types.GetUserLoginLogsRequest) (resp *types.GetUserLoginLogsResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterLoginLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		ObjectID: req.UserId,
	})
	if err != nil {
		l.Errorw("[GetUserLoginLogs] get user login logs failed", logger.Field("error", err.Error()), logger.Field("request", req))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get user login logs failed: %v", err.Error())
	}

	return &types.GetUserLoginLogsResponse{
		Total: total,
		List:  logview.UserLoginRows(data, logview.FromIPLocation(l.svcCtx.GeoIP)),
	}, nil
}
