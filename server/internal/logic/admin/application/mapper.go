package application

import (
	"github.com/neibniz/alpha/server/internal/clientview"
	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func downloadLinkToModel(link types.DownloadLink) client.DownloadLink {
	return clientview.DownloadLinkToModel(link)
}

func downloadLinkFromModel(link client.DownloadLink) types.DownloadLink {
	return clientview.DownloadLinkFromModel(link)
}

func downloadLinkFromRaw(raw string) types.DownloadLink {
	return clientview.DownloadLinkFromRaw(raw)
}

func subscribeApplicationRows(rows []*client.SubscribeApplication) []types.SubscribeApplication {
	if len(rows) == 0 {
		return []types.SubscribeApplication{}
	}
	collector := viewutil.NewRowsCollector[types.SubscribeApplication](len(rows))
	linkCache := clientview.NewDownloadLinkCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(subscribeApplicationFromModel(row, linkCache.Link(row.DownloadLink)))
	}
	return collector.Rows()
}

func subscribeApplicationListRows(rows []*client.SubscribeApplication) []types.SubscribeApplicationListItem {
	if len(rows) == 0 {
		return []types.SubscribeApplicationListItem{}
	}
	collector := viewutil.NewRowsCollector[types.SubscribeApplicationListItem](len(rows))
	linkCache := clientview.NewDownloadLinkCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(subscribeApplicationListItemFromModel(row, linkCache.Link(row.DownloadLink)))
	}
	return collector.Rows()
}

func subscribeApplicationFromModel(item *client.SubscribeApplication, link types.DownloadLink) types.SubscribeApplication {
	if item == nil {
		return types.SubscribeApplication{}
	}
	return types.SubscribeApplication{
		Id:                item.Id,
		Name:              item.Name,
		Description:       item.Description,
		Icon:              item.Icon,
		Scheme:            item.Scheme,
		UserAgent:         item.UserAgent,
		Enabled:           item.Enabled,
		IsDefault:         item.IsDefault,
		SubscribeTemplate: item.SubscribeTemplate,
		OutputFormat:      item.OutputFormat,
		DownloadLink:      link,
		CreatedAt:         timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:         timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func subscribeApplicationListItemFromModel(item *client.SubscribeApplication, link types.DownloadLink) types.SubscribeApplicationListItem {
	if item == nil {
		return types.SubscribeApplicationListItem{}
	}
	return types.SubscribeApplicationListItem{
		Id:           item.Id,
		Name:         item.Name,
		Description:  item.Description,
		Icon:         item.Icon,
		Scheme:       item.Scheme,
		UserAgent:    item.UserAgent,
		Enabled:      item.Enabled,
		IsDefault:    item.IsDefault,
		OutputFormat: item.OutputFormat,
		DownloadLink: link,
		CreatedAt:    timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:    timex.UnixMilliOrZero(item.UpdatedAt),
	}
}
