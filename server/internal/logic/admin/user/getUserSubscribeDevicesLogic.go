package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/userview"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type GetUserSubscribeDevicesLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get user subcribe devices
func NewGetUserSubscribeDevicesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserSubscribeDevicesLogic {
	return &GetUserSubscribeDevicesLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserSubscribeDevicesLogic) GetUserSubscribeDevices(req *types.GetUserSubscribeDevicesRequest) (resp *types.GetUserSubscribeDevicesResponse, err error) {
	list, total, err := l.svcCtx.UserModel.QueryDevicePageList(l.ctx, req.UserId, req.SubscribeId, req.Page, req.Size)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetUserSubscribeDevices failed: %v", err.Error())
	}
	return &types.GetUserSubscribeDevicesResponse{
		Total: total,
		List:  userview.DeviceRows(list, userview.FromIPLocation(l.svcCtx.GeoIP)),
	}, nil
}
