package auth

import (
	"context"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type CheckUserLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCheckUserLogic Check user is exist
func NewCheckUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckUserLogic {
	return &CheckUserLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckUserLogic) CheckUser(req *types.CheckUserRequest) (resp *types.CheckUserResponse, err error) {
	return &types.CheckUserResponse{
		Exist: false,
	}, nil
}
