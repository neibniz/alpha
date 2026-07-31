package user

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	batchTrafficResetSize            = 500
	maxBatchTrafficResetSelection    = 1000
	resetAllTrafficConfirmation      = "RESET ALL TRAFFIC"
	batchTrafficResetScopeAll        = "all"
	batchTrafficResetScopeUsers      = "users"
	batchTrafficResetScopeSubscribes = "subscriptions"
)

var batchTrafficResetColumns = []string{
	"id",
	"user_id",
	"subscribe_id",
	"product_snapshot",
	"expire_time",
	"token",
	"status",
}

type BatchResetUserSubscribeTrafficLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchResetUserSubscribeTrafficLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchResetUserSubscribeTrafficLogic {
	return &BatchResetUserSubscribeTrafficLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchResetUserSubscribeTrafficLogic) Reset(req *types.BatchResetUserSubscribeTrafficRequest) (*types.BatchResetUserSubscribeTrafficResponse, error) {
	if err := validateBatchTrafficResetRequest(req); err != nil {
		return nil, err
	}
	req.UserIds = normalizePositiveIDs(req.UserIds)
	req.UserSubscribeIds = normalizePositiveIDs(req.UserSubscribeIds)
	if req.Scope == batchTrafficResetScopeUsers && len(req.UserIds) == 0 {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "at least one user is required")
	}
	if req.Scope == batchTrafficResetScopeSubscribes && len(req.UserSubscribeIds) == 0 {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "at least one subscription is required")
	}

	resetAt := time.Now().UTC()
	batchID := uuid.NewString()
	operatorID := int64(0)
	if current, ok := l.ctx.Value(constant.CtxKeyUser).(*modelUser.User); ok && current != nil {
		operatorID = current.Id
	}

	var maxTargetID int64
	maxQuery := applyBatchTrafficResetScope(l.svcCtx.DB.WithContext(l.ctx).Model(&modelUser.Subscribe{}), req)
	if err := maxQuery.Select("COALESCE(MAX(id), 0)").Scan(&maxTargetID).Error; err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query reset boundary: %v", err)
	}

	resp := &types.BatchResetUserSubscribeTrafficResponse{
		BatchId: batchID,
		ResetAt: resetAt.UnixMilli(),
	}
	for maxTargetID > 0 {
		rows, reactivated, err := l.resetNextBatch(req, maxTargetID, resetAt, batchID, operatorID)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		resp.ResetCount += int64(len(rows))
		resp.Reactivated += reactivated
		l.clearResetCaches(rows)
	}
	return resp, nil
}

func (l *BatchResetUserSubscribeTrafficLogic) resetNextBatch(req *types.BatchResetUserSubscribeTrafficRequest, maxTargetID int64, resetAt time.Time, batchID string, operatorID int64) ([]*modelUser.Subscribe, int64, error) {
	rows := make([]*modelUser.Subscribe, 0, batchTrafficResetSize)
	reactivated := int64(0)
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		query := applyBatchTrafficResetScope(tx.Model(&modelUser.Subscribe{}), req).
			Select(batchTrafficResetColumns).
			Where("id <= ? AND traffic_reset_at < ?", maxTargetID, resetAt).
			Order("id ASC").
			Limit(batchTrafficResetSize).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		if err := query.Find(&rows).Error; err != nil || len(rows) == 0 {
			return err
		}

		ids := make([]int64, 0, len(rows))
		reactivateIDs := make([]int64, 0, len(rows))
		logs := make([]log.SystemLog, 0, len(rows))
		for _, row := range rows {
			if row == nil {
				continue
			}
			ids = append(ids, row.Id)
			if row.Status == subscriptionstate.StatusExhausted && !subscriptionstate.IsExpired(row.ExpireTime, resetAt) {
				reactivateIDs = append(reactivateIDs, row.Id)
			}
			entry := &log.ResetSubscribe{
				Type:           log.ResetSubscribeTypeAdvance,
				UserId:         row.UserId,
				BatchId:        batchID,
				Scope:          req.Scope,
				OperatorUserId: operatorID,
				Timestamp:      resetAt.UnixMilli(),
			}
			content, _ := entry.Marshal()
			logs = append(logs, log.SystemLog{
				Type:     log.TypeResetSubscribe.Uint8(),
				Date:     resetAt.Format(time.DateOnly),
				ObjectID: row.Id,
				EventAt:  resetAt.UnixMilli(),
				Success:  true,
				Content:  string(content),
			})
		}
		if len(ids) == 0 {
			return nil
		}
		updated := tx.Model(&modelUser.Subscribe{}).
			Where("id IN ? AND traffic_reset_at < ?", ids, resetAt).
			Updates(map[string]any{
				"download":         int64(0),
				"upload":           int64(0),
				"traffic_reset_at": resetAt,
			})
		if updated.Error != nil {
			return updated.Error
		}
		if len(reactivateIDs) > 0 {
			if err := tx.Model(&modelUser.Subscribe{}).
				Where("id IN ? AND status = ?", reactivateIDs, subscriptionstate.StatusExhausted).
				Updates(map[string]any{"status": subscriptionstate.StatusActive, "finished_at": nil}).Error; err != nil {
				return err
			}
			reactivated = int64(len(reactivateIDs))
		}
		if len(logs) > 0 {
			return tx.CreateInBatches(&logs, batchTrafficResetSize).Error
		}
		return nil
	})
	if err != nil {
		return nil, 0, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "reset subscription traffic batch: %v", err)
	}
	return rows, reactivated, nil
}

func applyBatchTrafficResetScope(query *gorm.DB, req *types.BatchResetUserSubscribeTrafficRequest) *gorm.DB {
	switch req.Scope {
	case batchTrafficResetScopeUsers:
		return query.Where("user_id IN ?", req.UserIds)
	case batchTrafficResetScopeSubscribes:
		return query.Where("id IN ?", req.UserSubscribeIds)
	default:
		return query
	}
}

func validateBatchTrafficResetRequest(req *types.BatchResetUserSubscribeTrafficRequest) error {
	if req == nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "reset request is required")
	}
	if len(req.UserIds) > maxBatchTrafficResetSelection || len(req.UserSubscribeIds) > maxBatchTrafficResetSelection {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "too many reset targets")
	}
	switch req.Scope {
	case batchTrafficResetScopeAll:
		if strings.TrimSpace(req.Confirmation) != resetAllTrafficConfirmation {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "confirmation text does not match")
		}
		if len(req.UserIds) > 0 || len(req.UserSubscribeIds) > 0 {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "all scope cannot include selected ids")
		}
	case batchTrafficResetScopeUsers:
		if len(req.UserSubscribeIds) > 0 {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "user scope cannot include subscription ids")
		}
	case batchTrafficResetScopeSubscribes:
		if len(req.UserIds) > 0 {
			return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "subscription scope cannot include user ids")
		}
	default:
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid reset scope")
	}
	return nil
}

func normalizePositiveIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func (l *BatchResetUserSubscribeTrafficLogic) clearResetCaches(rows []*modelUser.Subscribe) {
	if len(rows) == 0 {
		return
	}
	if err := l.svcCtx.UserModel.ClearSubscribeCache(l.ctx, rows...); err != nil {
		l.Errorw("[BatchResetTraffic] clear subscription cache failed", logger.Field("error", err.Error()))
	}
	productIDs := make(map[int64]struct{}, len(rows))
	snapshots := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		productIDs[row.SubscribeId] = struct{}{}
		if row.ProductSnapshot != "" {
			snapshots[row.ProductSnapshot] = struct{}{}
		}
	}
	for id := range productIDs {
		if err := l.svcCtx.SubscribeModel.ClearCache(l.ctx, id); err != nil {
			l.Errorw("[BatchResetTraffic] clear product cache failed", logger.Field("error", err.Error()), logger.Field("subscribe_id", id))
		}
	}
	for snapshot := range snapshots {
		if err := subscribecache.ClearBySnapshot(l.ctx, l.svcCtx.DB, l.svcCtx.Redis, snapshot); err != nil {
			l.Errorw("[BatchResetTraffic] clear server user cache failed", logger.Field("error", err.Error()))
		}
	}
}
