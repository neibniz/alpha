package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger.Logger
}

func NewGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserListLogic {
	return &GetUserListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logger.WithContext(ctx),
	}
}
func (l *GetUserListLogic) GetUserList(req *types.GetUserListRequest) (*types.GetUserListResponse, error) {
	list, total, err := l.svcCtx.UserModel.QueryPageList(l.ctx, req.Page, req.Size, &user.UserFilterParams{
		UserId:          req.UserId,
		Search:          req.Search,
		Status:          req.Status,
		Unscoped:        req.Unscoped,
		SubscribeId:     req.SubscribeId,
		UserSubscribeId: req.UserSubscribeId,
		Order:           "DESC",
	})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetUserListLogic failed: %v", err.Error())
	}

	return &types.GetUserListResponse{
		Total: total,
		List:  userListRowsFromModels(list, time.Now()),
	}, nil
}
