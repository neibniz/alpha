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

type GetMessageLogListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetMessageLogListLogic Get message log list
func NewGetMessageLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageLogListLogic {
	return &GetMessageLogListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMessageLogListLogic) GetMessageLogList(req *types.GetMessageLogListRequest) (resp *types.GetMessageLogListResponse, err error) {

	data, total, err := l.svcCtx.LogModel.FilterMessageLogs(l.ctx, &modelLog.FilterParams{
		Page:   req.Page,
		Size:   req.Size,
		Type:   req.Type,
		Search: req.Search,
	})

	if err != nil {
		l.Errorf("[GetMessageLogList] failed to filter system log: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to filter system log: %v", err.Error())
	}

	return &types.GetMessageLogListResponse{
		Total: total,
		List:  logview.MessageRows(data, l.Logger, "[GetMessageLogList]"),
	}, nil
}
