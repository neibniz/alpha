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

type FilterBalanceLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterBalanceLogLogic Filter balance log
func NewFilterBalanceLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterBalanceLogLogic {
	return &FilterBalanceLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterBalanceLogLogic) FilterBalanceLog(req *types.FilterBalanceLogRequest) (resp *types.FilterBalanceLogResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterBalanceLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		Type:     log.TypeBalance.Uint8(),
		Data:     req.Date,
		ObjectID: req.UserId,
	})

	if err != nil {
		l.Errorw("[FilterBalanceLog] Query User Balance Log Error:", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query User Balance Log Error")
	}

	return &types.FilterBalanceLogResponse{
		Total: total,
		List:  logview.BalanceRows(data),
	}, nil
}
