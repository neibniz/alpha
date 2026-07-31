package common

import (
	"context"
	"encoding/json"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type GetStatLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type commonStatCountsRow struct {
	UserCount    int64 `gorm:"column:user_count"`
	NodeCount    int64 `gorm:"column:node_count"`
	CountryCount int64 `gorm:"column:country_count"`
}

// Get Tos
func NewGetStatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetStatLogic {
	return &GetStatLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetStatLogic) GetStat() (resp *types.GetStatResponse, err error) {
	respJson, err := l.svcCtx.Redis.Get(l.ctx, config.CommonStatCacheKey).Result()
	if err == nil {
		cached := new(types.GetStatResponse)
		if err = json.Unmarshal([]byte(respJson), cached); err == nil {
			normalizeStatResponse(cached)
			return cached, nil
		}
	}
	counts, err := queryCommonStatCounts(l.ctx, l.svcCtx.DB)
	if err != nil {
		l.Logger.Error("[GetStatLogic] get stat counts failed: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get stat counts failed: %v", err.Error())
	}
	u := roundedPublicUserCount(counts.UserCount)
	n := counts.NodeCount
	country := counts.CountryCount
	protocol, err := queryEnabledProtocols(l.ctx, l.svcCtx.DB)
	if err != nil {
		l.Logger.Error("[GetStatLogic] get protocol failed: ", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "get protocol failed: %v", err.Error())
	}
	resp = &types.GetStatResponse{
		User:     u,
		Node:     n,
		Country:  country,
		Protocol: protocol,
	}
	normalizeStatResponse(resp)
	val, _ := json.Marshal(*resp)
	_ = l.svcCtx.Redis.Set(l.ctx, config.CommonStatCacheKey, string(val), time.Duration(3600)*time.Second).Err()
	return resp, nil
}

func normalizeStatResponse(resp *types.GetStatResponse) {
	if resp == nil {
		return
	}
	if resp.Protocol == nil {
		resp.Protocol = []string{}
	}
}

func roundedPublicUserCount(u int64) int64 {
	if u > 100 {
		return u - u%100
	}
	if u > 10 {
		return u - u%10
	}
	return 1
}

const statProtocolSelect = "COALESCE(NULLIF(inbound_alias, ''), protocol)"

const commonStatCountsSQL = `
SELECT
	(SELECT COUNT(*) FROM "user" WHERE enable = ? AND deleted_at IS NULL) AS user_count,
	(SELECT COUNT(*) FROM nodes WHERE enabled = ?) AS node_count,
	(SELECT COUNT(DISTINCT country) FROM servers WHERE country <> ?) AS country_count
`

func queryCommonStatCounts(ctx context.Context, db *gorm.DB) (commonStatCountsRow, error) {
	var row commonStatCountsRow
	err := db.WithContext(ctx).
		Raw(commonStatCountsSQL, true, true, "").
		Scan(&row).Error
	return row, err
}

func queryEnabledProtocols(ctx context.Context, db *gorm.DB) ([]string, error) {
	var protocols []string
	if err := db.WithContext(ctx).
		Model(&node.Node{}).
		Where("enabled = ?", true).
		Distinct(statProtocolSelect).
		Pluck(statProtocolSelect, &protocols).Error; err != nil {
		return nil, err
	}
	if len(protocols) == 0 {
		return []string{}, nil
	}

	result := protocols[:0]
	for _, protocol := range protocols {
		if protocol != "" {
			result = append(result, protocol)
		}
	}
	return result, nil
}
