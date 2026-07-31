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

type GetSubscribeListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get subscribe list
func NewGetSubscribeListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscribeListLogic {
	return &GetSubscribeListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscribeListLogic) GetSubscribeList(req *types.GetSubscribeListRequest) (resp *types.GetSubscribeListResponse, err error) {
	total, list, err := l.svcCtx.SubscribeModel.FilterList(l.ctx, &subscribe.FilterParams{
		Page:       int(req.Page),
		Size:       int(req.Size),
		Language:   req.Language,
		Search:     req.Search,
		Projection: subscribe.FilterListProjectionAdmin,
	})
	if err != nil {
		l.Logger.Error("[GetSubscribeListLogic] get subscribe list failed: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get subscribe list failed: %v", err.Error())
	}
	subscribeIdList, resultList := subscribeItemRowsFromModels(list)

	if len(subscribeIdList) > 0 {
		subscribeMaps, err := l.svcCtx.UserModel.QueryActiveSubscriptions(l.ctx, subscribeIdList...)
		if err != nil {
			l.Logger.Error("[GetSubscribeListLogic] get user subscribe failed: ", logger.Field("error", err.Error()))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get user subscribe failed: %v", err.Error())
		}
		for i, item := range resultList {
			if sub, ok := subscribeMaps[item.Id]; ok {
				resultList[i].Sold = sub
			}
		}
	}

	resp = &types.GetSubscribeListResponse{
		Total: total,
		List:  resultList,
	}
	return
}
