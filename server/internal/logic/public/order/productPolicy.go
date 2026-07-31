package order

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ensureNewPurchaseAvailable(product *subscribe.Subscribe, now time.Time) error {
	switch productpolicy.NewPurchaseAvailability(product, now) {
	case productpolicy.AvailabilityAvailable:
		return nil
	case productpolicy.AvailabilityOutOfStock:
		return errors.Wrap(xerr.NewErrCode(xerr.SubscribeOutOfStock), "product is out of stock")
	case productpolicy.AvailabilityComingSoon:
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "product sale has not started")
	case productpolicy.AvailabilityEnded:
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "product sale has ended")
	default:
		return errors.Wrap(xerr.NewErrCode(xerr.ERROR), "product is not available for purchase")
	}
}

func lockCurrentSubscribeForOrder(ctx context.Context, db *gorm.DB, subscribeID int64) (*subscribe.Subscribe, error) {
	var product subscribe.Subscribe
	if err := db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", subscribeID).
		First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func ensureProductSnapshotCurrent(product *subscribe.Subscribe, expected string) error {
	if product == nil || expected == "" || ordersnapshot.Marshal(ordersnapshot.FromSubscribe(product)) != expected {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "product changed; refresh and retry")
	}
	return nil
}

func ensureProductQuantity(product *subscribe.Subscribe, quantity int64) error {
	err := productpolicy.ValidateQuantity(product, quantity, MaxQuantity)
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, productpolicy.ErrQuantityBelowMinimum):
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "quantity is below product minimum")
	default:
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "quantity exceeds product maximum")
	}
}
