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

type UpdateServerLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdateServerLogic Update Server
func NewUpdateServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateServerLogic {
	return &UpdateServerLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateServerLogic) UpdateServer(req *types.UpdateServerRequest) error {
	data, err := l.svcCtx.NodeModel.FindOneServer(l.ctx, req.Id)
	if err != nil {
		l.Errorf("[UpdateServer] FindOneServer Error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find server error: %v", err.Error())
	}
	data.Name = req.Name
	data.Country = req.Country
	data.City = req.City
	// only update address when it's  different
	if req.Address != data.Address {
		// update address
		data.Address = req.Address
		if data.City == "" && data.Country == "" {
			fillServerLocationFromGeoIP(l.svcCtx, req.Address, data)
		}
	}

	err = l.svcCtx.NodeModel.UpdateServer(l.ctx, data)
	if err != nil {
		l.Errorf("[UpdateServer] UpdateServer Error: %v", err.Error())
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update server error: %v", err.Error())
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventServer)
	}

	return l.svcCtx.NodeModel.ClearNodeCache(l.ctx, &node.FilterNodeParams{
		Page:     1,
		Size:     1000,
		ServerId: []int64{req.Id},
		Search:   "",
	})
}
