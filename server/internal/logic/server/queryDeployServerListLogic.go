package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/nodeauth"
	"github.com/neibniz/alpha/server/internal/serverview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryDeployServerListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryDeployServerListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryDeployServerListLogic {
	return &QueryDeployServerListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryDeployServerListLogic) QueryDeployServerList() (*types.QueryDeployServerListResponse, error) {
	servers, err := l.svcCtx.NodeModel.QueryDeployServerList(l.ctx)
	if err != nil {
		l.Errorf("[QueryDeployServerList] query server list error: %v", err.Error())
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query server list error: %v", err)
	}

	rows := serverview.DeployRows(servers)
	for index := range rows {
		token, tokenErr := nodeauth.DeriveToken(l.svcCtx.ConfigSnapshot().Node.NodeSecret, rows[index].Id)
		if tokenErr != nil {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.SecretIsEmpty), "derive node deployment token")
		}
		rows[index].Token = token
	}
	return &types.QueryDeployServerListResponse{List: rows}, nil
}
