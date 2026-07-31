package log

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

type FilterResetSubscribeLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterResetSubscribeLogLogic Filter reset subscribe log
func NewFilterResetSubscribeLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterResetSubscribeLogLogic {
	return &FilterResetSubscribeLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterResetSubscribeLogLogic) FilterResetSubscribeLog(req *types.FilterResetSubscribeLogRequest) (resp *types.FilterResetSubscribeLogResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterResetSubscribeLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		ObjectID: req.UserSubscribeId,
		Data:     req.Date,
		Search:   req.Search,
	})

	if err != nil {
		l.Errorf("[FilterResetSubscribeLog] failed to filter system log: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to filter system log: %v", err.Error())
	}

	return &types.FilterResetSubscribeLogResponse{
		List:  logview.ResetSubscribeRows(data),
		Total: total,
	}, nil
}
