package ticketview

import (
	ticketmodel "github.com/neibniz/alpha/server/internal/model/ticket"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

func TicketListItem(item *ticketmodel.Ticket) types.TicketListItem {
	if item == nil {
		return types.TicketListItem{}
	}
	return types.TicketListItem{
		Id:        item.Id,
		Title:     item.Title,
		UserId:    item.UserId,
		Status:    item.Status,
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func UserTicketListItem(item *ticketmodel.Ticket) types.UserTicketListItem {
	if item == nil {
		return types.UserTicketListItem{}
	}
	return types.UserTicketListItem{
		Id:          item.Id,
		Title:       item.Title,
		Description: item.Description,
		Status:      item.Status,
		UpdatedAt:   timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func TicketListRows(rows []*ticketmodel.Ticket) []types.TicketListItem {
	if len(rows) == 0 {
		return []types.TicketListItem{}
	}
	collector := viewutil.NewRowsCollector[types.TicketListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(TicketListItem(row))
	}
	return collector.Rows()
}

func UserTicketListRows(rows []*ticketmodel.Ticket) []types.UserTicketListItem {
	if len(rows) == 0 {
		return []types.UserTicketListItem{}
	}
	collector := viewutil.NewRowsCollector[types.UserTicketListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(UserTicketListItem(row))
	}
	return collector.Rows()
}

func FromDetails(item *ticketmodel.Details) types.Ticket {
	if item == nil {
		return types.Ticket{}
	}
	resp := types.Ticket{
		Id:          item.Id,
		Title:       item.Title,
		Description: item.Description,
		Images:      append([]string{}, item.Images...),
		UserId:      item.UserId,
		Status:      item.Status,
		Follows:     make([]types.Follow, 0, len(item.Follows)),
		CreatedAt:   timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt:   timex.UnixMilliOrZero(item.UpdatedAt),
	}
	for _, follow := range item.Follows {
		resp.Follows = append(resp.Follows, FromFollow(follow))
	}
	return resp
}

func FromFollow(item ticketmodel.Follow) types.Follow {
	content := item.Content
	images := append([]string{}, item.Images...)
	if item.Type == 2 && len(images) == 0 && content != "" {
		images = append(images, content)
		content = ""
	}
	return types.Follow{
		Id:        item.Id,
		TicketId:  item.TicketId,
		From:      item.From,
		Type:      item.Type,
		Content:   content,
		Images:    images,
		CreatedAt: timex.UnixMilliOrZero(item.CreatedAt),
	}
}
