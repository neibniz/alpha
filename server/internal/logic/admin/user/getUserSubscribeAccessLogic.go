package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserSubscribeAccessLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subcribe access
func NewGetUserSubscribeAccessLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeAccessLogic {
	return &GetUserSubscribeAccessLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeAccessLogic) GetUserSubscribeAccess(req *types.GetUserSubscribeAccessRequest) (resp *types.UserSubscribeAccess, err error) {
	sub, err := l.svcCtx.UserModel.FindOneUserSubscribeAccess(l.ctx, req.UserSubscribeId)
	if err != nil {
		l.Errorw("[GetUserSubscribeAccessLogic] FindOneUserSubscribeAccess error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "FindOneUserSubscribeAccess error: %v", err.Error())
	}
	short, _ := tool.FixedUniqueString(sub.Token, 8, "")
	return &types.UserSubscribeAccess{
		Token: sub.Token,
		Short: short,
	}, nil
}
