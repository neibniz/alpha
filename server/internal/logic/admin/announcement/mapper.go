package announcement

import (
	modelAnnouncement "github.com/neibniz/alpha/server/internal/model/announcement"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func announcementFromModel(item *modelAnnouncement.Announcement, includeContent bool) types.Announcement {
	if item == nil {
		return types.Announcement{}
	}
	resp := types.Announcement{
		Id:        item.Id,
		Title:     item.Title,
		Show:      announcementBoolValue(item.Show),
		Pinned:    announcementBoolValue(item.Pinned),
		Popup:     announcementBoolValue(item.Popup),
		CreatedAt: timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
	if includeContent {
		resp.Content = item.Content
	}
	return resp
}

func announcementListItemFromModel(item *modelAnnouncement.Announcement) types.AnnouncementListItem {
	if item == nil {
		return types.AnnouncementListItem{}
	}
	return types.AnnouncementListItem{
		Id:        item.Id,
		Title:     item.Title,
		Show:      announcementBoolValue(item.Show),
		Pinned:    announcementBoolValue(item.Pinned),
		Popup:     announcementBoolValue(item.Popup),
		CreatedAt: timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func announcementRows(rows []*modelAnnouncement.Announcement, includeContent bool) []types.AnnouncementListItem {
	if len(rows) == 0 {
		return []types.AnnouncementListItem{}
	}
	collector := viewutil.NewRowsCollector[types.AnnouncementListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(announcementListItemFromModel(row))
	}
	return collector.Rows()
}

func announcementBoolValue(value *bool) bool {
	return value != nil && *value
}
