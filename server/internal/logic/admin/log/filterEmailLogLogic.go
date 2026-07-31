package log

import (
	"context"

	"github.com/neibniz/alpha/server/internal/logview"
	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type FilterEmailLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewFilterEmailLogLogic Filter email log
func NewFilterEmailLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterEmailLogLogic {
	return &FilterEmailLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterEmailLogLogic) FilterEmailLog(req *types.FilterEmailLogRequest) (resp *types.FilterEmailLogResponse, err error) {
	reveal, err := prepareSensitiveLogReveal(l.ctx, l.svcCtx, req.RevealSensitive, req.RevealReason, "email")
	if err != nil {
		return nil, err
	}
	data, total, err := l.svcCtx.LogModel.FilterMessageLogs(l.ctx, &modelLog.FilterParams{
		Page:            req.Page,
		Size:            req.Size,
		Type:            modelLog.TypeEmailMessage.Uint8(),
		Data:            req.Date,
		Search:          req.Search,
		RevealSensitive: req.RevealSensitive,
	})

	if err != nil {
		l.Errorf("[FilterEmailLog] failed to filter system log: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to filter system log: %v", err.Error())
	}
	if err := reveal.audit(len(data)); err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseInsertError), "failed to audit sensitive log reveal")
	}

	return &types.FilterEmailLogResponse{
		Total: total,
		List:  logview.MessageRows(data, l.Logger, "[FilterEmailLog]"),
	}, nil
}
