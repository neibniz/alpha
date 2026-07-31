package order

import (
	"context"
	"errors"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	"gorm.io/gorm"
)

func couponUsageReached(ctx context.Context, db *gorm.DB, userID int64, coupon string, limit int64) (bool, error) {
	if limit <= 0 || userID <= 0 || coupon == "" {
		return false, nil
	}
	if db == nil {
		return false, errors.New("coupon usage database is nil")
	}

	limitedOrders := db.WithContext(ctx).
		Model(&modelOrder.Order{}).
		Select("1").
		Where("user_id = ? AND coupon = ? AND status IN (1, 2, 5)", userID, coupon).
		Limit(int(limit))
	var count int64
	if err := db.WithContext(ctx).
		Table("(?) AS limited_coupon_orders", limitedOrders).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count >= limit, nil
}
