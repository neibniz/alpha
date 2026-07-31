package subscribeview

import (
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func GroupFromModel(item *modelSubscribe.Group) types.SubscribeGroup {
	if item == nil {
		return types.SubscribeGroup{}
	}
	return types.SubscribeGroup{
		Id:          item.Id,
		Name:        item.Name,
		Description: item.Description,
		CreatedAt:   timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:   timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func GroupsFromModels(items []*modelSubscribe.Group) []types.SubscribeGroup {
	if len(items) == 0 {
		return []types.SubscribeGroup{}
	}
	collector := viewutil.NewRowsCollector[types.SubscribeGroup](len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		collector.Add(GroupFromModel(item))
	}
	return collector.Rows()
}
