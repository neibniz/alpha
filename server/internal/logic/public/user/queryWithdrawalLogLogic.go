package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type QueryWithdrawalLogLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewQueryWithdrawalLogLogic Query Withdrawal Log
func NewQueryWithdrawalLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryWithdrawalLogLogic {
	return &QueryWithdrawalLogLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryWithdrawalLogLogic) QueryWithdrawalLog(req *types.QueryWithdrawalLogListRequest) (resp *types.QueryWithdrawalLogListResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}

	req.Page, req.Size = pagination.Normalize(req.Page, req.Size)
	query := l.svcCtx.DB.WithContext(l.ctx).Model(&user.Withdrawal{}).Where("user_id = ?", u.Id)
	var total int64
	if err = query.Count(&total).Error; err != nil {
		l.Errorw("Query withdrawal log count failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query withdrawal log count failed: %v", err.Error())
	}

	offset, capacity, ok := pagination.PageWindow(total, req.Page, req.Size)
	rows := make([]user.Withdrawal, 0, capacity)
	if ok {
		err = query.
			Select(withdrawalLogSelectColumns).
			Order("id DESC").
			Limit(req.Size).
			Offset(offset).
			Find(&rows).Error
	}
	if err != nil {
		l.Errorw("Query withdrawal log failed", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query withdrawal log failed: %v", err.Error())
	}
	privacyCodec := l.svcCtx.Privacy
	if privacyCodec == nil {
		privacyCodec = privacy.DisabledCodec()
	}
	for index := range rows {
		rows[index].Content, err = privacyCodec.Open(privacy.WithdrawalScope(rows[index].UserId, "content"), rows[index].Content)
		if err != nil {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "decrypt withdrawal content")
		}
		rows[index].Reason, err = privacyCodec.Open(privacy.WithdrawalScope(rows[index].UserId, "reason"), rows[index].Reason)
		if err != nil {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "decrypt withdrawal reason")
		}
	}

	return &types.QueryWithdrawalLogListResponse{
		List:  withdrawalRowsToTypes(rows),
		Total: total,
	}, nil
}

var withdrawalLogSelectColumns = []string{
	"id",
	"user_id",
	"amount",
	"content",
	"status",
	"reason",
	"created_at",
	"updated_at",
}
