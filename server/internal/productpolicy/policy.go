package productpolicy

import (
	"errors"
	"strings"
	"time"

	"github.com/neibniz/alpha/server/internal/model/subscribe"
)

const (
	TemplateStandard = "standard"
	TemplatePopular  = "popular"
	TemplatePremium  = "premium"
	TemplateCompact  = "compact"

	AvailabilityAvailable  = "available"
	AvailabilityNotForSale = "not_for_sale"
	AvailabilityComingSoon = "coming_soon"
	AvailabilityEnded      = "ended"
	AvailabilityOutOfStock = "out_of_stock"
)

var (
	ErrQuantityBelowMinimum = errors.New("quantity is below product minimum")
	ErrQuantityAboveMaximum = errors.New("quantity exceeds product maximum")
	ErrInvalidSaleWindow    = errors.New("sale end time must be after sale start time")
)

func NormalizeTemplate(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case TemplatePopular:
		return TemplatePopular
	case TemplatePremium:
		return TemplatePremium
	case TemplateCompact:
		return TemplateCompact
	default:
		return TemplateStandard
	}
}

func NormalizeMinQuantity(value int64) int64 {
	if value < 1 {
		return 1
	}
	return value
}

func EffectiveMaxQuantity(productMax, systemMax int64) int64 {
	if systemMax <= 0 {
		return productMax
	}
	if productMax <= 0 || productMax > systemMax {
		return systemMax
	}
	return productMax
}

func ValidateQuantity(product *subscribe.Subscribe, quantity, systemMax int64) error {
	minimum := int64(1)
	maximum := systemMax
	if product != nil {
		minimum = NormalizeMinQuantity(product.MinQuantity)
		maximum = EffectiveMaxQuantity(product.MaxQuantity, systemMax)
	}
	if quantity < minimum {
		return ErrQuantityBelowMinimum
	}
	if maximum > 0 && quantity > maximum {
		return ErrQuantityAboveMaximum
	}
	return nil
}

func ValidateSaleWindow(startsAt, endsAt *time.Time) error {
	if startsAt != nil && endsAt != nil && !endsAt.After(*startsAt) {
		return ErrInvalidSaleWindow
	}
	return nil
}

func NewPurchaseAvailability(product *subscribe.Subscribe, now time.Time) string {
	if product == nil || product.Sell == nil || !*product.Sell {
		return AvailabilityNotForSale
	}
	if product.SaleStartsAt != nil && now.Before(*product.SaleStartsAt) {
		return AvailabilityComingSoon
	}
	if product.SaleEndsAt != nil && !now.Before(*product.SaleEndsAt) {
		return AvailabilityEnded
	}
	if product.Inventory == 0 {
		return AvailabilityOutOfStock
	}
	return AvailabilityAvailable
}

func IsAvailable(product *subscribe.Subscribe, now time.Time) bool {
	return NewPurchaseAvailability(product, now) == AvailabilityAvailable
}
