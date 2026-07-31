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

type FilterLoginLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterLoginLogLogic Filter login log
func NewFilterLoginLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterLoginLogLogic {
	return &FilterLoginLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterLoginLogLogic) FilterLoginLog(req *types.FilterLoginLogRequest) (resp *types.FilterLoginLogResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterLoginLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		ObjectID: req.UserId,
		Data:     req.Date,
		Search:   req.Search,
	})

	if err != nil {
		l.Errorf("[FilterLoginLog] failed to filter system log: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to filter system log: %v", err.Error())
	}
	return &types.FilterLoginLogResponse{
		Total: total,
		List:  logview.LoginRows(data, logview.FromIPLocation(l.svcCtx.GeoIP)),
	}, nil
}
