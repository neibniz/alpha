package portal

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func ensurePaymentEnabled(p *payment.Payment) error {
	if p == nil || p.Enable == nil || !*p.Enable {
		return errors.Wrapf(xerr.NewErrCode(xerr.PaymentMethodNotFound), "payment method is disabled")
	}
	return nil
}

func decreaseSubscribeInventory(ctx context.Context, db *gorm.DB, subscribeID int64) error {
	result := db.WithContext(ctx).
		Model(&subscribe.Subscribe{}).
		Where("id = ? AND inventory > 0", subscribeID).
		Update("inventory", gorm.Expr("inventory - ?", 1))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.Wrapf(xerr.NewErrCode(xerr.SubscribeOutOfStock), "subscribe out of stock")
	}
	return nil
}
