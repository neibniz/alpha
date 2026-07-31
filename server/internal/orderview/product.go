package orderview

import (
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/types"
)

func SubscribeProductFromModel(item *modelSubscribe.Subscribe) types.OrderSubscribeProduct {
	return SubscribeProductFromSubscribe(subscribeview.SubscribeFromModelLenient(item, subscribeview.SubscribeOptions{}))
}

func SubscribeProductFromSnapshot(snapshot ordersnapshot.Snapshot) types.OrderSubscribeProduct {
	return SubscribeProductFromSubscribe(ordersnapshot.SubscribeType(snapshot))
}

func SubscribeProductFromSubscribe(item types.Subscribe) types.OrderSubscribeProduct {
	return types.OrderSubscribeProduct{
		Id:                item.Id,
		Name:              item.Name,
		UnitPrice:         item.UnitPrice,
		UnitTime:          item.UnitTime,
		Traffic:           item.Traffic,
		SpeedLimit:        item.SpeedLimit,
		DeviceLimit:       item.DeviceLimit,
		ShowOriginalPrice: item.ShowOriginalPrice,
	}
}
