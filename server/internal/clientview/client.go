package clientview

import (
	"encoding/json"
	"strings"

	modelClient "github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

const smallDownloadLinkCacheThreshold = 8

func DownloadLinkToModel(link types.DownloadLink) modelClient.DownloadLink {
	link = NormalizeDownloadLink(link)
	return modelClient.DownloadLink{
		IOS:     link.IOS,
		Android: link.Android,
		Windows: link.Windows,
		Mac:     link.Mac,
		Linux:   link.Linux,
		Harmony: link.Harmony,
	}
}

func DownloadLinkFromModel(link modelClient.DownloadLink) types.DownloadLink {
	return NormalizeDownloadLink(types.DownloadLink{
		IOS:     link.IOS,
		Android: link.Android,
		Windows: link.Windows,
		Mac:     link.Mac,
		Linux:   link.Linux,
		Harmony: link.Harmony,
	})
}

func DownloadLinkFromRaw(raw string) types.DownloadLink {
	raw = strings.TrimSpace(raw)
	return downloadLinkFromTrimmedRaw(raw)
}

func downloadLinkFromTrimmedRaw(raw string) types.DownloadLink {
	if raw == "" || raw == "{}" || raw == "null" {
		return types.DownloadLink{}
	}
	var link types.DownloadLink
	_ = json.Unmarshal([]byte(raw), &link)
	return NormalizeDownloadLink(link)
}

func NormalizeDownloadLink(link types.DownloadLink) types.DownloadLink {
	return types.DownloadLink{
		IOS:     strings.TrimSpace(link.IOS),
		Android: strings.TrimSpace(link.Android),
		Windows: strings.TrimSpace(link.Windows),
		Mac:     strings.TrimSpace(link.Mac),
		Linux:   strings.TrimSpace(link.Linux),
		Harmony: strings.TrimSpace(link.Harmony),
	}
}

func SubscribeClientListFromModel(data []*modelClient.SubscribeApplication) []types.SubscribeClient {
	collector := viewutil.NewRowsCollector[types.SubscribeClient](len(data))
	cache := NewDownloadLinkCache(len(data))
	for _, item := range data {
		if item == nil {
			continue
		}
		collector.Add(subscribeClientFromModelWithCache(item, cache))
	}
	return collector.Rows()
}

func SubscribeClientFromModel(item *modelClient.SubscribeApplication) types.SubscribeClient {
	return subscribeClientFromModelWithCache(item, nil)
}

func subscribeClientFromModelWithCache(item *modelClient.SubscribeApplication, cache *DownloadLinkCache) types.SubscribeClient {
	if item == nil {
		return types.SubscribeClient{}
	}
	return types.SubscribeClient{
		Id:           item.Id,
		Name:         item.Name,
		Description:  item.Description,
		Icon:         item.Icon,
		Scheme:       item.Scheme,
		Enabled:      item.Enabled,
		IsDefault:    item.IsDefault,
		DownloadLink: cache.Link(item.DownloadLink),
	}
}

type downloadLinkCacheEntry struct {
	raw  string
	link types.DownloadLink
}

type DownloadLinkCache struct {
	rows    []downloadLinkCacheEntry
	index   map[string]int
	capHint int
}

func NewDownloadLinkCache(capHint int) *DownloadLinkCache {
	if capHint <= 1 {
		return nil
	}
	return &DownloadLinkCache{capHint: capHint}
}

func (c *DownloadLinkCache) Link(raw string) types.DownloadLink {
	raw = strings.TrimSpace(raw)
	if c == nil {
		return downloadLinkFromTrimmedRaw(raw)
	}
	if c.index != nil {
		if pos, ok := c.index[raw]; ok {
			return c.rows[pos].link
		}
		return c.add(raw)
	}
	for _, row := range c.rows {
		if row.raw == raw {
			return row.link
		}
	}
	return c.add(raw)
}

func (c *DownloadLinkCache) add(raw string) types.DownloadLink {
	link := downloadLinkFromTrimmedRaw(raw)
	c.ensureRows()
	c.rows = append(c.rows, downloadLinkCacheEntry{raw: raw, link: link})
	if c.index != nil {
		c.index[raw] = len(c.rows) - 1
	} else if len(c.rows) > smallDownloadLinkCacheThreshold {
		nextCap := downloadLinkCacheCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]downloadLinkCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return link
}

func (c *DownloadLinkCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallDownloadLinkCacheThreshold
	} else if capHint > smallDownloadLinkCacheThreshold+1 {
		capHint = smallDownloadLinkCacheThreshold + 1
	}
	c.rows = make([]downloadLinkCacheEntry, 0, capHint)
}

func downloadLinkCacheCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
