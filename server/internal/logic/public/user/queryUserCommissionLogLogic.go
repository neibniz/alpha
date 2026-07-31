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

type QueryUserCommissionLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Query User Commission Log
func NewQueryUserCommissionLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserCommissionLogLogic {
	return &QueryUserCommissionLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserCommissionLogLogic) QueryUserCommissionLog(req *types.QueryUserCommissionLogListRequest) (resp *types.QueryUserCommissionLogListResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	data, total, err := l.svcCtx.LogModel.FilterCommissionLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		Type:     log.TypeCommission.Uint8(),
		ObjectID: u.Id,
	})
	if err != nil {
		l.Errorw("Query User Commission Log failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query User Commission Log failed: %v", err)
	}

	return &types.QueryUserCommissionLogListResponse{
		List:  logview.CommissionRows(data),
		Total: total,
	}, nil
}
