package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger.Logger
}

func NewGetUserDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserDetailLogic {
	return &GetUserDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logger.WithContext(ctx),
	}
}

func (l *GetUserDetailLogic) GetUserDetail(req *types.GetDetailRequest) (*types.User, error) {
	resp := types.User{}
	userInfo, err := l.svcCtx.UserModel.FindOneForContext(l.ctx, req.Id)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get user detail error: %v", err.Error())
	}
	resp = userDetailFromModel(userInfo, time.Now())
	return &resp, nil
}
