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

type FilterCommissionLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterCommissionLogLogic Filter commission log
func NewFilterCommissionLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterCommissionLogLogic {
	return &FilterCommissionLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterCommissionLogLogic) FilterCommissionLog(req *types.FilterCommissionLogRequest) (resp *types.FilterCommissionLogResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterCommissionLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		Data:     req.Date,
		Type:     log.TypeCommission.Uint8(),
		ObjectID: req.UserId,
	})
	if err != nil {
		l.Errorw("Query User Commission Log failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query User Commission Log failed")
	}
	return &types.FilterCommissionLogResponse{
		Total: total,
		List:  logview.CommissionRows(data),
	}, nil
}
