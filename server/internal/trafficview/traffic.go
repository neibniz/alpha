package trafficview

import (
	modelTraffic "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func Rows(rows []*modelTraffic.TrafficLog) []types.TrafficLog {
	collector := viewutil.NewRowsCollector[types.TrafficLog](len(rows))
	for _, item := range rows {
		if item == nil {
			continue
		}
		collector.Add(types.TrafficLog{
			Id:          item.Id,
			ServerId:    item.ServerId,
			UserId:      item.UserId,
			SubscribeId: item.SubscribeId,
			Download:    item.Download,
			Upload:      item.Upload,
			Timestamp:   timex.UnixMilliOrZero(item.Timestamp),
		})
	}
	return collector.Rows()
}

func DetailRows(rows []*modelTraffic.TrafficLog) []types.TrafficLogDetails {
	return Rows(rows)
}
