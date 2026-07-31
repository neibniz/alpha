package subscribecache

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func ClearBySnapshot(ctx context.Context, db *gorm.DB, rds *redis.Client, raw string) error {
	nodeIDs, nodeTags, ok := ordersnapshot.SubscribeTopology(raw)
	if !ok {
		return nil
	}
	return ClearByNodeSelectors(ctx, db, rds, nodeIDs, nodeTags)
}

func ClearBySubscribe(ctx context.Context, db *gorm.DB, rds *redis.Client, sub *subscribe.Subscribe) error {
	if db == nil || rds == nil || sub == nil {
		return nil
	}
	keys, err := serverUserCacheKeys(ctx, db, sub)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return rds.Del(ctx, keys...).Err()
}

func ClearByNodeSelectors(ctx context.Context, db *gorm.DB, rds *redis.Client, nodeIDs []int64, nodeTags []string) error {
	if db == nil || rds == nil {
		return nil
	}
	keys, err := subscribe.ServerUserCacheKeysForNodeSelectors(ctx, db, nodeIDs, nodeTags)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return rds.Del(ctx, keys...).Err()
}

func serverUserCacheKeys(ctx context.Context, db *gorm.DB, sub *subscribe.Subscribe) ([]string, error) {
	return subscribe.ServerUserCacheKeysForSubscribe(ctx, db, sub)
}
