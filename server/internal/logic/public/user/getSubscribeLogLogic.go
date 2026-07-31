package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/logview"
	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetSubscribeLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetSubscribeLogLogic Get Subscribe Log
func NewGetSubscribeLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSubscribeLogLogic {
	return &GetSubscribeLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSubscribeLogLogic) GetSubscribeLog(req *types.GetSubscribeLogRequest) (resp *types.GetSubscribeLogResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	subscribeLogs, total, err := l.svcCtx.LogModel.FilterSubscribeAccessLog(l.ctx, &log.SubscribeAccessFilterParams{
		Page:   req.Page,
		Size:   req.Size,
		UserId: u.Id,
	})
	if err != nil {
		l.Errorw("[GetUserSubscribeLogs] Get User Subscribe Logs Error:", logger.Field("err", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Get User Subscribe Logs Error")
	}

	return &types.GetSubscribeLogResponse{
		List:  logview.UserSubscribeRows(subscribeLogs, logview.FromIPLocation(l.svcCtx.GeoIP)),
		Total: total,
	}, err
}
