package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetUserAuthMethodLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user auth method
func NewGetUserAuthMethodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserAuthMethodLogic {
	return &GetUserAuthMethodLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserAuthMethodLogic) GetUserAuthMethod(req *types.GetUserAuthMethodRequest) (resp *types.GetUserAuthMethodResponse, err error) {
	methods, err := l.svcCtx.UserModel.FindUserAuthMethodSummaries(l.ctx, req.UserId)
	if err != nil {
		l.Errorw("[GetUserAuthMethodLogic] Get User Auth Method Error:", logger.Field("err", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Get User Auth Method Error")
	}

	return &types.GetUserAuthMethodResponse{
		AuthMethods: userAuthMethodsFromModels(methods, false),
	}, nil
}
