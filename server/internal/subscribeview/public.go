package subscribeview

import (
	"time"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func PublicSubscribeFromModel(item *modelSubscribe.Subscribe) types.PublicSubscribe {
	return PublicSubscribeFromSubscribe(SubscribeFromModelLenient(item, SubscribeOptions{}))
}

func PublicSubscribeRowsLenient(rows []*modelSubscribe.Subscribe) []types.PublicSubscribe {
	if len(rows) == 0 {
		return []types.PublicSubscribe{}
	}
	collector := viewutil.NewRowsCollector[types.PublicSubscribe](len(rows))
	discountCache := newDiscountParseCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		subscribe, _ := subscribeFromModelWithCaches(row, SubscribeOptions{}, discountCache, nil)
		collector.Add(PublicSubscribeFromSubscribe(subscribe))
	}
	return collector.Rows()
}

func PublicSubscribeFromSubscribe(item types.Subscribe) types.PublicSubscribe {
	discount := item.Discount
	if discount == nil {
		discount = []types.SubscribeDiscount{}
	}
	availability := productpolicy.NewPurchaseAvailability(&modelSubscribe.Subscribe{
		Sell:         boolPointer(item.Sell),
		Inventory:    item.Inventory,
		SaleStartsAt: unixMilliPointer(item.SaleStartsAt),
		SaleEndsAt:   unixMilliPointer(item.SaleEndsAt),
	}, time.Now())
	return types.PublicSubscribe{
		Id:                item.Id,
		Name:              item.Name,
		Description:       item.Description,
		UnitPrice:         item.UnitPrice,
		UnitTime:          item.UnitTime,
		Discount:          discount,
		Replacement:       item.Replacement,
		Traffic:           item.Traffic,
		SpeedLimit:        item.SpeedLimit,
		DeviceLimit:       item.DeviceLimit,
		UserVisible:       item.UserVisible,
		Sell:              item.Sell,
		ShowOriginalPrice: item.ShowOriginalPrice,
		DisplayTemplate:   productpolicy.NormalizeTemplate(item.DisplayTemplate),
		Badge:             item.Badge,
		MinQuantity:       productpolicy.NormalizeMinQuantity(item.MinQuantity),
		MaxQuantity:       item.MaxQuantity,
		SaleStartsAt:      item.SaleStartsAt,
		SaleEndsAt:        item.SaleEndsAt,
		Available:         availability == productpolicy.AvailabilityAvailable,
		Availability:      availability,
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func unixMilliPointer(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.UnixMilli(value)
	return &t
}
