package subscribe

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
)

type BatchDeleteSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Batch delete subscribe
func NewBatchDeleteSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteSubscribeLogic {
	return &BatchDeleteSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

var errorIsExistActiveUser = errors.New("subscription ID belongs to an active user subscription")

func (l *BatchDeleteSubscribeLogic) BatchDeleteSubscribe(req *types.BatchDeleteSubscribeRequest) error {
	ids := normalizeBatchDeleteSubscribeIDs(req.Ids)
	if len(ids) == 0 {
		return nil
	}

	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var usedSubscribeID int64
		// Validate whether any subscription ID belongs to an active user subscription.
		if err := tx.Model(&user.Subscribe{}).
			Select("subscribe_id").
			Where("subscribe_id IN ? AND status = ?", ids, 1).
			Limit(1).
			Scan(&usedSubscribeID).Error; err != nil {
			l.Logger.Error("[BatchDeleteSubscribe] Query Subscribe Error: ", logger.Field("error", err.Error()))
			return err
		}
		if usedSubscribeID > 0 {
			return errorIsExistActiveUser
		}
		return l.svcCtx.SubscribeModel.DeleteByIds(l.ctx, ids, tx)
	})
	if err != nil {
		if errors.Is(err, errorIsExistActiveUser) {
			return errors.Wrapf(xerr.NewErrCode(xerr.SubscribeIsUsedError), "subscription ID belongs to an active user subscription")
		}
		l.Logger.Error("[BatchDeleteSubscribe] Transaction Error: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "delete subscribe failed: %v", err.Error())
	}
	return nil
}

func normalizeBatchDeleteSubscribeIDs(ids []int64) []int64 {
	return tool.NormalizePositiveInt64Slice(ids)
}
