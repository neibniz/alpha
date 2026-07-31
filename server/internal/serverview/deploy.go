package serverview

import (
	modelNode "github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func DeployRows(rows []*modelNode.Server) []types.DeployServer {
	collector := viewutil.NewRowsCollector[types.DeployServer](len(rows))
	for _, item := range rows {
		if item == nil {
			continue
		}
		collector.Add(types.DeployServer{
			Id:      item.Id,
			Name:    item.Name,
			Address: item.Address,
		})
	}
	return collector.Rows()
}
