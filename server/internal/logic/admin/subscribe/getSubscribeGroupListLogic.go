package subscribe

import (
	"context"

	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetSubscribeGroupListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get subscribe group list
func NewGetSubscribeGroupListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscribeGroupListLogic {
	return &GetSubscribeGroupListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscribeGroupListLogic) GetSubscribeGroupList() (resp *types.GetSubscribeGroupListResponse, err error) {
	list, err := l.svcCtx.SubscribeModel.QueryGroupList(l.ctx)
	if err != nil {
		l.Logger.Error("[GetSubscribeGroupListLogic] get subscribe group list failed: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get subscribe group list failed: %v", err.Error())
	}
	groupList := subscribeview.GroupsFromModels(list)
	return &types.GetSubscribeGroupListResponse{
		Total: int64(len(groupList)),
		List:  groupList,
	}, nil
}
