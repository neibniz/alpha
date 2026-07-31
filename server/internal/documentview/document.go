package documentview

import (
	modelDocument "github.com/neibniz/alpha/server/internal/model/document"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/tool"
)

const smallDocumentTagsCacheThreshold = 8

func AdminListRows(rows []*modelDocument.Document) []types.DocumentListItem {
	if len(rows) == 0 {
		return []types.DocumentListItem{}
	}
	collector := viewutil.NewRowsCollector[types.DocumentListItem](len(rows))
	tags := newDocumentTagsCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(adminListWithTagsCache(row, tags))
	}
	return collector.Rows()
}

func PublicSummaryRows(rows []*modelDocument.Document) []types.PublicDocumentListItem {
	if len(rows) == 0 {
		return []types.PublicDocumentListItem{}
	}
	collector := viewutil.NewRowsCollector[types.PublicDocumentListItem](len(rows))
	tags := newDocumentTagsCache(len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(publicSummaryWithTagsCache(row, tags))
	}
	return collector.Rows()
}

func AdminList(item *modelDocument.Document) types.DocumentListItem {
	return adminListWithTagsCache(item, nil)
}

func adminListWithTagsCache(item *modelDocument.Document, tags *documentTagsCache) types.DocumentListItem {
	if item == nil {
		return types.DocumentListItem{}
	}
	return types.DocumentListItem{
		Id:        item.Id,
		Title:     item.Title,
		Tags:      tags.Tags(item.Tags),
		Show:      showValue(item.Show),
		CreatedAt: timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func AdminDetail(item *modelDocument.Document) types.Document {
	if item == nil {
		return types.Document{}
	}
	resp := types.Document{
		Id:        item.Id,
		Title:     item.Title,
		Tags:      tool.StringMergeAndRemoveDuplicates(item.Tags),
		Show:      showValue(item.Show),
		CreatedAt: timex.UnixMilliOrZero(item.CreatedAt),
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
	resp.Content = item.Content
	return resp
}

func PublicSummary(item *modelDocument.Document) types.PublicDocumentListItem {
	return publicSummaryWithTagsCache(item, nil)
}

func publicSummaryWithTagsCache(item *modelDocument.Document, tags *documentTagsCache) types.PublicDocumentListItem {
	if item == nil {
		return types.PublicDocumentListItem{}
	}
	return types.PublicDocumentListItem{
		Id:        item.Id,
		Title:     item.Title,
		Tags:      tags.Tags(item.Tags),
		UpdatedAt: timex.UnixMilliOrZero(item.UpdatedAt),
	}
}

func PublicDetail(item *modelDocument.Document) types.Document {
	resp := AdminDetail(item)
	if item == nil {
		return resp
	}
	resp.Show = showValue(item.Show)
	return resp
}

func showValue(value *bool) bool {
	return value != nil && *value
}

type documentTagsCacheEntry struct {
	raw  string
	tags []string
}

type documentTagsCache struct {
	rows    []documentTagsCacheEntry
	index   map[string]int
	capHint int
}

func newDocumentTagsCache(capHint int) *documentTagsCache {
	if capHint <= 1 {
		return nil
	}
	return &documentTagsCache{capHint: capHint}
}

func (c *documentTagsCache) Tags(raw string) []string {
	if c == nil {
		return tool.StringMergeAndRemoveDuplicates(raw)
	}
	if c.index != nil {
		if pos, ok := c.index[raw]; ok {
			return c.rows[pos].tags
		}
		return c.add(raw)
	}
	for _, row := range c.rows {
		if row.raw == raw {
			return row.tags
		}
	}
	return c.add(raw)
}

func (c *documentTagsCache) add(raw string) []string {
	tags := tool.StringMergeAndRemoveDuplicates(raw)
	c.ensureRows()
	c.rows = append(c.rows, documentTagsCacheEntry{raw: raw, tags: tags})
	if c.index != nil {
		c.index[raw] = len(c.rows) - 1
	} else if len(c.rows) > smallDocumentTagsCacheThreshold {
		nextCap := documentTagsCacheCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]documentTagsCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return tags
}

func (c *documentTagsCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallDocumentTagsCacheThreshold
	} else if capHint > smallDocumentTagsCacheThreshold+1 {
		capHint = smallDocumentTagsCacheThreshold + 1
	}
	c.rows = make([]documentTagsCacheEntry, 0, capHint)
}

func documentTagsCacheCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
