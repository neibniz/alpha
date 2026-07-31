package userview

import (
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/timex"
)

type Location struct {
	CountryShort string
	Region       string
	Location     string
}

type IPLookup func(ip string) Location

const smallDeviceLocationCacheThreshold = 8

func FromIPLocation(ipLoc *svc.IPLocation) IPLookup {
	return func(ip string) Location {
		info := ipLoc.Lookup(ip)
		return Location{
			CountryShort: info.CountryShort,
			Region:       info.Region,
			Location:     info.Location,
		}
	}
}

func DeviceRows(rows []*modelUser.Device, lookup IPLookup) []types.UserDevice {
	collector := viewutil.NewRowsCollector[types.UserDevice](len(rows))
	locations := newDeviceLocationCache(lookup, len(rows))
	for _, item := range rows {
		if item == nil {
			continue
		}
		location := locations.Lookup(item.Ip)
		collector.Add(types.UserDevice{
			Id:           item.Id,
			Ip:           privacy.MaskIP(item.Ip),
			CountryShort: location.CountryShort,
			Region:       location.Region,
			Location:     location.Location,
			Identifier:   privacy.MaskIdentifier(item.Identifier),
			UserAgent:    privacy.MaskUserAgent(item.UserAgent),
			Online:       item.Online,
			Enabled:      item.Enabled,
			CreatedAt:    timex.UnixMilliOrZero(item.CreatedAt),
			UpdatedAt:    timex.UnixMilliOrZero(item.UpdatedAt),
		})
	}
	return collector.Rows()
}

type deviceLocationCacheEntry struct {
	ip       string
	location Location
}

type deviceLocationCache struct {
	lookup  IPLookup
	rows    []deviceLocationCacheEntry
	index   map[string]int
	capHint int
}

func newDeviceLocationCache(lookup IPLookup, capHint int) deviceLocationCache {
	return deviceLocationCache{lookup: lookup, capHint: capHint}
}

func (c *deviceLocationCache) Lookup(ip string) Location {
	if c == nil || c.lookup == nil {
		return Location{}
	}
	if location, ok := c.Get(ip); ok {
		return location
	}
	location := c.lookup(ip)
	c.Set(ip, location)
	return location
}

func (c *deviceLocationCache) Get(ip string) (Location, bool) {
	if c == nil {
		return Location{}, false
	}
	if c.index != nil {
		pos, ok := c.index[ip]
		if !ok {
			return Location{}, false
		}
		return c.rows[pos].location, true
	}
	for _, row := range c.rows {
		if row.ip == ip {
			return row.location, true
		}
	}
	return Location{}, false
}

func (c *deviceLocationCache) Set(ip string, location Location) {
	if c == nil {
		return
	}
	if c.index != nil {
		if pos, ok := c.index[ip]; ok {
			c.rows[pos].location = location
			return
		}
		c.index[ip] = len(c.rows)
		c.rows = append(c.rows, deviceLocationCacheEntry{ip: ip, location: location})
		return
	}
	for i, row := range c.rows {
		if row.ip == ip {
			c.rows[i].location = location
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, deviceLocationCacheEntry{ip: ip, location: location})
	if len(c.rows) > smallDeviceLocationCacheThreshold {
		c.upgradeLookup()
	}
}

func (c *deviceLocationCache) ensureRows() {
	if c.rows != nil {
		return
	}
	c.rows = make([]deviceLocationCacheEntry, 0, deviceLocationCacheInitialCap(c.capHint))
}

func (c *deviceLocationCache) upgradeLookup() {
	if c.index != nil {
		return
	}
	nextCap := deviceLocationCacheCap(len(c.rows), c.capHint)
	if nextCap > cap(c.rows) {
		rows := make([]deviceLocationCacheEntry, len(c.rows), nextCap)
		copy(rows, c.rows)
		c.rows = rows
	}
	c.index = make(map[string]int, nextCap)
	for i, row := range c.rows {
		c.index[row.ip] = i
	}
}

func deviceLocationCacheCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func deviceLocationCacheInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallDeviceLocationCacheThreshold+1 {
		return smallDeviceLocationCacheThreshold + 1
	}
	return capHint
}
