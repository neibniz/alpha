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

type FilterRegisterLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Filter register log
func NewFilterRegisterLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterRegisterLogLogic {
	return &FilterRegisterLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterRegisterLogLogic) FilterRegisterLog(req *types.FilterRegisterLogRequest) (resp *types.FilterRegisterLogResponse, err error) {
	reveal, err := prepareSensitiveLogReveal(l.ctx, l.svcCtx, req.RevealSensitive, req.RevealReason, "register")
	if err != nil {
		return nil, err
	}
	data, total, err := l.svcCtx.LogModel.FilterRegisterLogs(l.ctx, &log.FilterParams{
		Page:            req.Page,
		Size:            req.Size,
		ObjectID:        req.UserId,
		Data:            req.Date,
		Search:          req.Search,
		RevealSensitive: req.RevealSensitive,
	})

	if err != nil {
		l.Errorf("[FilterRegisterLog] failed to filter system log: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to filter system log: %v", err.Error())
	}
	if err := reveal.audit(len(data)); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "failed to audit sensitive log reveal")
	}

	return &types.FilterRegisterLogResponse{
		List:  logview.RegisterRows(data, logview.FromIPLocation(l.svcCtx.GeoIP)),
		Total: total,
	}, nil
}
