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

type FilterSubscribeLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterSubscribeLogLogic Filter subscribe log
func NewFilterSubscribeLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterSubscribeLogLogic {
	return &FilterSubscribeLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterSubscribeLogLogic) FilterSubscribeLog(req *types.FilterSubscribeLogRequest) (resp *types.FilterSubscribeLogResponse, err error) {
	reveal, err := prepareSensitiveLogReveal(l.ctx, l.svcCtx, req.RevealSensitive, req.RevealReason, "subscribe_access")
	if err != nil {
		return nil, err
	}
	data, total, err := l.svcCtx.LogModel.FilterSubscribeAccessLog(l.ctx, &log.SubscribeAccessFilterParams{
		Page:            req.Page,
		Size:            req.Size,
		Date:            req.Date,
		UserId:          req.UserId,
		UserSubscribeId: req.UserSubscribeId,
		RevealSensitive: req.RevealSensitive,
	})
	if err != nil {
		l.Errorf("[FilterSubscribeLog] failed to filter subscribe access log: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to filter subscribe access log")
	}
	if err := reveal.audit(len(data)); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "failed to audit sensitive log reveal")
	}

	return &types.FilterSubscribeLogResponse{
		Total: total,
		List:  logview.SubscribeRows(data, logview.FromIPLocation(l.svcCtx.GeoIP)),
	}, nil
}
