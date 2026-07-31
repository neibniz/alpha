package announcement

import (
	modelAnnouncement "github.com/neibniz/alpha/server/internal/model/announcement"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func announcementFromModel(item *modelAnnouncement.Announcement) types.Announcement {
	if item == nil {
		return types.Announcement{}
	}
	return types.Announcement{
		Id:        item.Id,
		Title:     item.Title,
		Content:   item.Content,
		Show:      announcementBoolValue(item.Show),
		Pinned:    announcementBoolValue(item.Pinned),
		Popup:     announcementBoolValue(item.Popup),
		CreatedAt: timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func announcementRows(rows []*modelAnnouncement.Announcement) []types.Announcement {
	if len(rows) == 0 {
		return []types.Announcement{}
	}
	collector := viewutil.NewRowsCollector[types.Announcement](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(announcementFromModel(row))
	}
	return collector.Rows()
}

func announcementBoolValue(value *bool) bool {
	return value != nil && *value
}
