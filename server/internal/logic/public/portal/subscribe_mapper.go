package portal

import (
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/types"
)

func subscribeRowsFromModels(rows []*modelSubscribe.Subscribe) []types.PublicSubscribe {
	return subscribeview.PublicSubscribeRowsLenient(rows)
}
