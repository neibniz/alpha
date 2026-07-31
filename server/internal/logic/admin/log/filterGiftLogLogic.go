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

type FilterGiftLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Filter gift log
func NewFilterGiftLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FilterGiftLogLogic {
	return &FilterGiftLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FilterGiftLogLogic) FilterGiftLog(req *types.FilterGiftLogRequest) (resp *types.FilterGiftLogResponse, err error) {
	data, total, err := l.svcCtx.LogModel.FilterGiftLogs(l.ctx, &modelLog.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		ObjectID: req.UserId,
		Data:     req.Date,
		Search:   req.Search,
	})

	if err != nil {
		l.Errorf("[FilterGiftLog] failed to filter system log: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "failed to filter system log: %v", err.Error())
	}

	return &types.FilterGiftLogResponse{
		Total: total,
		List:  logview.GiftRows(data),
	}, nil
}
