package subscribe

import (
	subscribeModel "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/types"
)

func publicSubscribeFromModel(item *subscribeModel.Subscribe) types.PublicSubscribe {
	return subscribeview.PublicSubscribeFromModel(item)
}

func publicSubscribeRowsFromModels(rows []*subscribeModel.Subscribe) []types.PublicSubscribe {
	return subscribeview.PublicSubscribeRowsLenient(rows)
}
