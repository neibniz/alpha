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

type QueryUserBalanceLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQueryUserBalanceLogLogic Query User Balance Log
func NewQueryUserBalanceLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserBalanceLogLogic {
	return &QueryUserBalanceLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserBalanceLogLogic) QueryUserBalanceLog(req *types.QueryUserBalanceLogListRequest) (resp *types.QueryUserBalanceLogListResponse, err error) {
	if req == nil {
		req = &types.QueryUserBalanceLogListRequest{}
	}
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}

	data, total, err := l.svcCtx.LogModel.FilterBalanceLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		Type:     log.TypeBalance.Uint8(),
		ObjectID: u.Id,
	})
	if err != nil {
		l.Errorw("[QueryUserBalanceLog] Query User Balance Log Error:", logger.Field("err", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query User Balance Log Error")
	}

	return &types.QueryUserBalanceLogListResponse{
		Total: total,
		List:  logview.BalanceRows(data),
	}, nil
}
