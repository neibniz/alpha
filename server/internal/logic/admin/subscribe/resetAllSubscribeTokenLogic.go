package subscribe

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"gorm.io/gorm"
)

const (
	resetSubscribeTokenBatchSize      = 500
	resetSubscribeTokenCacheScanCount = 500
)

var resetSubscribeTokenCachePatterns = []string{
	"cache:user:subscribe:token:*",
	"cache:user:subscribe:user:*",
	"cache:user:subscribe:id:*",
}

type subscribeTokenResetID struct {
	Id int64 `gorm:"column:id"`
}

type subscribeTokenResetRow struct {
	Id    int64  `gorm:"column:id"`
	Token string `gorm:"column:token"`
	UUID  string `gorm:"column:uuid"`
}

type ResetAllSubscribeTokenLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Reset all subscribe tokens
func NewResetAllSubscribeTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetAllSubscribeTokenLogic {
	return &ResetAllSubscribeTokenLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetAllSubscribeTokenLogic) ResetAllSubscribeToken() (resp *types.ResetAllSubscribeTokenResponse, err error) {
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
CREATE TEMP TABLE tmp_user_subscribe_token_reset (
    id bigint PRIMARY KEY,
    token varchar(255) NOT NULL,
    uuid varchar(255) NOT NULL
) ON COMMIT DROP
`).Error; err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "create temporary token reset table failed: %v", err.Error())
		}

		var lastID int64
		for {
			ids := make([]subscribeTokenResetID, 0, resetSubscribeTokenBatchSize)
			if err := tx.Model(&user.Subscribe{}).
				Select("id").
				Where("status IN ? AND id > ?", []int64{1, 2}, lastID).
				Order("id ASC").
				Limit(resetSubscribeTokenBatchSize).
				Find(&ids).Error; err != nil {
				return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "fetch subscribe ids failed: %v", err.Error())
			}
			if len(ids) == 0 {
				break
			}

			rows := make([]subscribeTokenResetRow, 0, len(ids))
			for _, item := range ids {
				uuid := uuidx.NewUUID().String()
				token, err := uuidx.NewSubscribeToken()
				if err != nil {
					return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate subscription token")
				}
				rows = append(rows, subscribeTokenResetRow{
					Id:    item.Id,
					Token: token,
					UUID:  uuid,
				})
			}
			if err := tx.Table("tmp_user_subscribe_token_reset").CreateInBatches(rows, resetSubscribeTokenBatchSize).Error; err != nil {
				return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "stage subscribe token reset rows failed: %v", err.Error())
			}
			lastID = ids[len(ids)-1].Id
		}

		if err := tx.Exec(`
UPDATE "user_subscribe" AS us
SET "token" = tmp."token",
    "uuid" = tmp."uuid",
    "updated_at" = ?
FROM tmp_user_subscribe_token_reset AS tmp
WHERE us."id" = tmp."id"
`, time.Now()).Error; err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "batch update subscribe tokens failed: %v", err.Error())
		}

		return nil
	})
	if err != nil {
		logger.Errorf("[ResetAllSubscribeToken] Failed to reset subscribe tokens: %v", err.Error())
		return &types.ResetAllSubscribeTokenResponse{Success: false}, err
	}

	if err = l.clearAllSubscribeCaches(); err != nil {
		logger.Errorf("[ResetAllSubscribeToken] Failed to clear subscribe caches: %v", err.Error())
		return &types.ResetAllSubscribeTokenResponse{Success: false}, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "clear subscribe caches failed: %v", err.Error())
	}

	return &types.ResetAllSubscribeTokenResponse{
		Success: true,
	}, nil
}

func (l *ResetAllSubscribeTokenLogic) clearAllSubscribeCaches() error {
	for _, pattern := range resetSubscribeTokenCachePatterns {
		var cursor uint64
		for {
			keys, nextCursor, err := l.svcCtx.Redis.Scan(l.ctx, cursor, pattern, resetSubscribeTokenCacheScanCount).Result()
			if err != nil {
				return err
			}
			if len(keys) > 0 {
				if err := l.svcCtx.Redis.Del(l.ctx, keys...).Err(); err != nil {
					return err
				}
			}
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}
	return nil
}
