package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/userview"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetDeviceListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Device List
func NewGetDeviceListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceListLogic {
	return &GetDeviceListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDeviceListLogic) GetDeviceList(req *types.GetDeviceListRequest) (resp *types.GetDeviceListResponse, err error) {
	if req == nil {
		req = &types.GetDeviceListRequest{}
	}
	userInfo, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok || userInfo == nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is unavailable")
	}
	list, count, err := l.svcCtx.UserModel.QueryDeviceList(l.ctx, userInfo.Id, req.Page, req.Size)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "GetDeviceList failed: %v", err.Error())
	}
	resp = &types.GetDeviceListResponse{
		Total: count,
		List:  userview.DeviceRows(list, userview.FromIPLocation(l.svcCtx.GeoIP)),
	}
	return
}
