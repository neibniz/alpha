package subscribe

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/user"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type DeleteSubscribeLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Delete subscribe
func NewDeleteSubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSubscribeLogic {
	return &DeleteSubscribeLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSubscribeLogic) DeleteSubscribe(req *types.DeleteSubscribeRequest) error {
	// Check if the subscribe exists
	var activeSubscribeID int64
	err := l.svcCtx.UserModel.Transaction(l.ctx, func(db *gorm.DB) error {
		return scanActiveSubscribeUsage(db, req.Id, &activeSubscribeID)
	})
	if err != nil {
		l.Logger.Error("[DeleteSubscribeLogic] check subscribe failed: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "check subscribe failed: %v", err.Error())
	}
	if activeSubscribeID > 0 {
		return errors.Wrapf(xerr.NewErrCode(xerr.SubscribeIsUsedError), "subscribe is used")
	}

	err = l.svcCtx.SubscribeModel.Delete(l.ctx, req.Id)
	if err != nil {
		l.Logger.Error("[DeleteSubscribeLogic] delete subscribe failed: ", logger.Field("error", err.Error()))
		return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseDeletedError), "delete subscribe failed: %v", err.Error())
	}
	return nil
}

func scanActiveSubscribeUsage(db *gorm.DB, subscribeID int64, dest *int64) error {
	return buildActiveSubscribeUsageQuery(db, subscribeID).Scan(dest).Error
}

func buildActiveSubscribeUsageQuery(db *gorm.DB, subscribeID int64) *gorm.DB {
	return db.Model(&user.Subscribe{}).
		Select("subscribe_id").
		Where("subscribe_id = ? AND status = ?", subscribeID, 1).
		Limit(1)
}
