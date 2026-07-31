package subscribe

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QuerySubscribeListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get subscribe list
func NewQuerySubscribeListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuerySubscribeListLogic {
	return &QuerySubscribeListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QuerySubscribeListLogic) QuerySubscribeList(req *types.QuerySubscribeListRequest) (resp *types.QuerySubscribeListResponse, err error) {

	total, data, err := l.svcCtx.SubscribeModel.FilterList(l.ctx, &subscribe.FilterParams{
		Page:            1,
		Size:            9999,
		MaxSize:         9999,
		Language:        req.Language,
		UserVisible:     true,
		DefaultLanguage: true,
		Projection:      subscribe.FilterListProjectionPublic,
		SkipCount:       true,
	})
	if err != nil {
		l.Errorw("[QuerySubscribeListLogic] Database Error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "QuerySubscribeList error: %v", err.Error())
	}

	resp = &types.QuerySubscribeListResponse{
		Total: total,
		List:  publicSubscribeRowsFromModels(data),
	}
	return
}
