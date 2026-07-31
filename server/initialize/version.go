package initialize

import (
	"context"
	"errors"
	"time"

	internalconfig "github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/user"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/initialize/migrate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/orm"
)

func Migrate(ctx *svc.ServiceContext) {
	mc := orm.Postgres{
		Config: ctx.Config.Postgres,
	}
	now := time.Now()
	databaseChanged := false
	if err := migrate.Migrate(mc.Dsn()).Up(); err != nil {
		if errors.Is(err, migrate.NoChange) {
			logger.Info("[Migrate] database not change")
		} else {
			logger.Errorf("[Migrate] Up error: %v", err.Error())
			panic(err)
		}
	} else {
		databaseChanged = true
		logger.Info("[Migrate] Database change, took " + time.Since(now).String())
	}
	if databaseChanged {
		if err := purgeSystemConfigCache(context.Background(), ctx); err != nil {
			logger.Errorf("[Migrate] Purge system config cache error: %v", err)
			panic(err)
		}
	}
	if err := migrate.SyncIdentitySequences(ctx.DB); err != nil {
		logger.Errorf("[Migrate] SyncIdentitySequences error: %v", err.Error())
		panic(err)
	}
	// if not found admin user
	err := ctx.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&user.User{}).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := migrate.CreateAdminUser(ctx.Config.Administrator.Email, ctx.Config.Administrator.Password, tx); err != nil {
				logger.Errorf("[Migrate] CreateAdminUser error: %v", err.Error())
				return err
			}
			logger.Info("[Migrate] Create admin user success")
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	identityMigrated, err := ctx.UserModel.MigratePrivacy(context.Background(), 200)
	if err != nil {
		logger.Errorf("[Migrate] Encrypt user identities error: %v", err)
		panic(err)
	}
	deviceMigrated, err := ctx.UserModel.MigrateDevicePrivacy(context.Background(), 200)
	if err != nil {
		logger.Errorf("[Migrate] Encrypt user devices error: %v", err)
		panic(err)
	}
	if identityMigrated+deviceMigrated > 0 {
		if err := purgeLegacyPrivacyCache(context.Background(), ctx); err != nil {
			logger.Errorf("[Migrate] Purge legacy privacy cache error: %v", err)
			panic(err)
		}
		logger.Infof("[Migrate] Encrypted %d user identities and %d user devices", identityMigrated, deviceMigrated)
	}
	if _, err := migrateWithdrawalPrivacy(context.Background(), ctx); err != nil {
		logger.Errorf("[Migrate] Encrypt withdrawal privacy error: %v", err)
		panic(err)
	}
}

var systemConfigCacheKeys = []string{
	internalconfig.CurrencyConfigKey,
	internalconfig.SiteConfigKey,
	internalconfig.SubscribeConfigKey,
	internalconfig.RegisterConfigKey,
	internalconfig.VerifyConfigKey,
	internalconfig.EmailSmtpConfigKey,
	internalconfig.NodeConfigKey,
	internalconfig.InviteConfigKey,
	internalconfig.TelegramConfigKey,
	internalconfig.TosConfigKey,
	internalconfig.VerifyCodeConfigKey,
	internalconfig.GlobalConfigKey,
}

func purgeSystemConfigCache(ctx context.Context, svcCtx *svc.ServiceContext) error {
	return svcCtx.Redis.Del(ctx, systemConfigCacheKeys...).Err()
}

func migrateWithdrawalPrivacy(ctx context.Context, svcCtx *svc.ServiceContext) (int64, error) {
	if !svcCtx.Privacy.Enabled() {
		return 0, nil
	}
	var migrated int64
	for {
		var rows []user.Withdrawal
		if err := svcCtx.DB.WithContext(ctx).
			Select("id", "user_id", "content", "reason").
			Where("(content <> '' AND content NOT LIKE 'enc:v1:%') OR (reason <> '' AND reason NOT LIKE 'enc:v1:%')").
			Order("id ASC").
			Limit(200).
			Find(&rows).Error; err != nil {
			return migrated, err
		}
		if len(rows) == 0 {
			return migrated, nil
		}
		for index := range rows {
			row := &rows[index]
			content, err := svcCtx.Privacy.Seal(privacycodec.WithdrawalScope(row.UserId, "content"), row.Content)
			if err != nil {
				return migrated, err
			}
			reason, err := svcCtx.Privacy.Seal(privacycodec.WithdrawalScope(row.UserId, "reason"), row.Reason)
			if err != nil {
				return migrated, err
			}
			if err := svcCtx.DB.WithContext(ctx).Model(&user.Withdrawal{}).Where("id = ?", row.Id).Updates(map[string]any{
				"content": content,
				"reason":  reason,
			}).Error; err != nil {
				return migrated, err
			}
			migrated++
		}
	}
}

func purgeLegacyPrivacyCache(ctx context.Context, svcCtx *svc.ServiceContext) error {
	patterns := []string{
		"cache:user:*",
		"auth:verify:email:*",
	}
	for _, pattern := range patterns {
		var cursor uint64
		for {
			keys, next, err := svcCtx.Redis.Scan(ctx, cursor, pattern, 200).Result()
			if err != nil {
				return err
			}
			if len(keys) > 0 {
				if err := svcCtx.Redis.Del(ctx, keys...).Err(); err != nil {
					return err
				}
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	return nil
}
