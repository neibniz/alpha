package subscribe

import (
	subscribeModel "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func subscribeItemFromModel(item *subscribeModel.Subscribe) types.SubscribeItem {
	if item == nil {
		return types.SubscribeItem{}
	}
	return types.SubscribeItem{
		Id:              item.Id,
		Name:            item.Name,
		Language:        item.Language,
		UnitPrice:       item.UnitPrice,
		UnitTime:        item.UnitTime,
		Replacement:     item.Replacement,
		Inventory:       item.Inventory,
		Traffic:         item.Traffic,
		SpeedLimit:      item.SpeedLimit,
		DeviceLimit:     item.DeviceLimit,
		Quota:           item.Quota,
		Show:            subscribeview.BoolValue(item.Show),
		UserVisible:     subscribeview.BoolValue(item.UserVisible),
		Sell:            subscribeview.BoolValue(item.Sell),
		AllowRenewal:    subscribeview.BoolValue(item.AllowRenewal),
		ResetCycle:      item.ResetCycle,
		DisplayTemplate: item.DisplayTemplate,
		Badge:           item.Badge,
		MinQuantity:     item.MinQuantity,
		MaxQuantity:     item.MaxQuantity,
		SaleStartsAt:    timex.UnixMilliPointerOrZero(item.SaleStartsAt),
		SaleEndsAt:      timex.UnixMilliPointerOrZero(item.SaleEndsAt),
		Sort:            item.Sort,
		CreatedAt:       timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:       timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func subscribeItemRowsFromModels(rows []*subscribeModel.Subscribe) ([]int64, []types.SubscribeItem) {
	if len(rows) == 0 {
		return []int64{}, []types.SubscribeItem{}
	}
	ids := viewutil.NewRowsCollector[int64](len(rows))
	items := viewutil.NewRowsCollector[types.SubscribeItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		if row.Id > 0 {
			ids.Add(row.Id)
		}
		items.Add(subscribeItemFromModel(row))
	}
	return ids.Rows(), items.Rows()
}

func subscribeFromModel(item *subscribeModel.Subscribe, includeTopology bool) (types.Subscribe, error) {
	return subscribeview.SubscribeFromModel(item, subscribeview.SubscribeOptions{IncludeTopology: includeTopology})
}
