package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type QueryUserAffiliateLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Query User Balance Log
func NewQueryUserAffiliateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserAffiliateLogic {
	return &QueryUserAffiliateLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserAffiliateLogic) QueryUserAffiliate() (resp *types.QueryUserAffiliateCountResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	stats, err := l.queryAffiliateStats(u.Id)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query User Affiliate failed: %v", err)
	}

	return &types.QueryUserAffiliateCountResponse{
		Registers:       stats.Registers,
		TotalCommission: stats.TotalCommission,
	}, nil
}

func (l *QueryUserAffiliateLogic) queryAffiliateStats(userId int64) (affiliateStatsRow, error) {
	var stats affiliateStatsRow
	err := l.svcCtx.DB.WithContext(l.ctx).
		Raw(affiliateStatsQuerySQL, userId, log.TypeCommission.Uint8(), userId).
		Scan(&stats).Error
	return stats, err
}

type affiliateStatsRow struct {
	Registers       int64
	TotalCommission int64
}

const affiliateCommissionAmountSumSelect = "COALESCE(SUM((content::jsonb->>'amount')::bigint), 0)"

const affiliateStatsQuerySQL = `
SELECT
	(SELECT COUNT(*) FROM "user" WHERE referer_id = ? AND deleted_at IS NULL) AS registers,
	(SELECT ` + affiliateCommissionAmountSumSelect + ` FROM system_logs WHERE type = ? AND object_id = ?) AS total_commission
`
