package server

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CreateServerLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateServerLogic Create Server
func NewCreateServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateServerLogic {
	return &CreateServerLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateServerLogic) CreateServer(req *types.CreateServerRequest) error {
	data := node.Server{
		Name:    req.Name,
		Country: req.Country,
		City:    req.City,
		Address: req.Address,
		Sort:    req.Sort,
	}
	if data.City == "" && data.Country == "" {
		fillServerLocationFromGeoIP(l.svcCtx, req.Address, &data)
	}
	err := l.svcCtx.NodeModel.InsertServer(l.ctx, &data)
	if err != nil {
		l.Errorf("[CreateServer] Insert Server error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert server error: %v", err)
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventServer)
	}
	return nil
}
