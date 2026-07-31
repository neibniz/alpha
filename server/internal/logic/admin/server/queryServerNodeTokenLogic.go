package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/nodeauth"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryServerNodeTokenLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryServerNodeTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryServerNodeTokenLogic {
	return &QueryServerNodeTokenLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryServerNodeTokenLogic) QueryServerNodeToken(req *types.QueryServerNodeTokenRequest) (*types.QueryServerNodeTokenResponse, error) {
	if req == nil || req.Id <= 0 {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "server id is required")
	}
	exists, err := l.svcCtx.NodeModel.ServerExists(l.ctx, req.Id)
	if err != nil {
		l.Errorw("[QueryServerNodeToken] query server error", logger.Field("error", err.Error()))
		return nil, errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "query server")
	}
	if !exists {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.NodeNotExist), "server does not exist")
	}
	token, err := nodeauth.DeriveToken(l.svcCtx.ConfigSnapshot().Node.NodeSecret, req.Id)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.SecretIsEmpty), "derive node authentication token")
	}
	return &types.QueryServerNodeTokenResponse{Token: token}, nil
}
