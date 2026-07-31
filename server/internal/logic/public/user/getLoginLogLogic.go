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

type GetLoginLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Get Login Log
func NewGetLoginLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLoginLogLogic {
	return &GetLoginLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLoginLogLogic) GetLoginLog(req *types.GetLoginLogRequest) (resp *types.GetLoginLogResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	data, total, err := l.svcCtx.LogModel.FilterLoginLogs(l.ctx, &log.FilterParams{
		Page:     req.Page,
		Size:     req.Size,
		ObjectID: u.Id,
	})
	if err != nil {
		l.Errorw("find login log failed:", logger.Field("error", err.Error()), logger.Field("user_id", u.Id))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find login log failed: %v", err.Error())
	}

	return &types.GetLoginLogResponse{
		Total: total,
		List:  logview.UserLoginRows(data, logview.FromIPLocation(l.svcCtx.GeoIP)),
	}, nil
}
