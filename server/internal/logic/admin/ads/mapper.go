package ads

import (
	modelAds "github.com/neibniz/alpha/server/internal/model/ads"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func adsFromModel(item *modelAds.Ads, includeContent bool) types.Ads {
	if item == nil {
		return types.Ads{}
	}
	resp := types.Ads{
		Id:          int(item.Id),
		Title:       item.Title,
		Type:        item.Type,
		Description: item.Description,
		TargetURL:   item.TargetURL,
		StartTime:   timex.UnixMilliOrZero(item.StartTime),
		EndTime:     timex.UnixMilliOrZero(item.EndTime),
		Status:      item.Status,
		CreatedAt:   timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:   timex.UnixMilliOrZero(item.UpdatedAt),
	}
	if includeContent {
		resp.Content = item.Content
	}
	return resp
}

func adsListItemFromModel(item *modelAds.Ads) types.AdsListItem {
	if item == nil {
		return types.AdsListItem{}
	}
	return types.AdsListItem{
		Id:          int(item.Id),
		Title:       item.Title,
		Type:        item.Type,
		Description: item.Description,
		TargetURL:   item.TargetURL,
		StartTime:   timex.UnixMilliOrZero(item.StartTime),
		EndTime:     timex.UnixMilliOrZero(item.EndTime),
		Status:      item.Status,
		CreatedAt:   timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:   timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func adsRows(rows []*modelAds.Ads, includeContent bool) []types.AdsListItem {
	if len(rows) == 0 {
		return []types.AdsListItem{}
	}
	collector := viewutil.NewRowsCollector[types.AdsListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(adsListItemFromModel(row))
	}
	return collector.Rows()
}
