package user

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/pagination"
	"github.com/neibniz/alpha/server/pkg/constant"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type QueryUserAffiliateListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Query User Affiliate List
func NewQueryUserAffiliateListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryUserAffiliateListLogic {
	return &QueryUserAffiliateListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryUserAffiliateListLogic) QueryUserAffiliateList(req *types.QueryUserAffiliateListRequest) (resp *types.QueryUserAffiliateListResponse, err error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		logger.Error("current user is not found in context")
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "Invalid Access")
	}
	var total int64
	req.Page, req.Size = pagination.Normalize(req.Page, req.Size)
	countQuery := l.svcCtx.DB.WithContext(l.ctx).Model(&user.User{}).Where("referer_id = ?", u.Id)
	err = countQuery.Count(&total).Error
	data := make([]affiliateUserListRow, 0)
	if err == nil {
		offset, capacity, ok := pagination.PageWindow(total, req.Page, req.Size)
		data = make([]affiliateUserListRow, 0, capacity)
		if ok {
			err = l.svcCtx.DB.WithContext(l.ctx).
				Table(`"user" AS u`).
				Select(affiliateUserListSelectColumns).
				Joins(affiliateIdentifierJoinSQL).
				Where("u.referer_id = ?", u.Id).
				Order("u.id DESC").
				Limit(req.Size).
				Offset(offset).
				Scan(&data).Error
		}
	}
	if err != nil {
		l.Errorw("Query User Affiliate List failed: %v", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Query User Affiliate List failed: %v", err.Error())
	}

	return &types.QueryUserAffiliateListResponse{
		Total: total,
		List:  affiliateUserRowsToTypes(data),
	}, nil
}

type affiliateUserListRow struct {
	Identifier   string
	Avatar       string
	RegisteredAt time.Time
	Enable       bool
}

var affiliateUserListSelectColumns = []string{
	"COALESCE(am.auth_identifier, '') AS identifier",
	"u.avatar",
	"u.created_at AS registered_at",
	"COALESCE(u.enable, false) AS enable",
}

const affiliateIdentifierPrioritySQL = `
CASE auth_type
	WHEN 'email' THEN 0
	WHEN 'mobile' THEN 1
	ELSE 2
END`

const affiliateIdentifierJoinSQL = `LEFT JOIN LATERAL (
	SELECT auth_identifier
	FROM user_auth_methods
	WHERE user_id = u.id
	ORDER BY ` + affiliateIdentifierPrioritySQL + `, id ASC
	LIMIT 1
) AS am ON true`

func maskAffiliateIdentifier(identifier string) string {
	hideTextLength := len(identifier) / 3
	if hideTextLength > 0 {
		return identifier[0:hideTextLength] + "***" + identifier[hideTextLength*2:]
	}
	return identifier
}
