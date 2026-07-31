package logview

import (
	"encoding/json"
	"strings"

	"github.com/neibniz/alpha/server/internal/jsonview"
	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/timex"
)

type Location struct {
	CountryShort string
	Region       string
	Location     string
}

type LocationResolver func(ip, countryShort, region string) Location

const smallLocationCacheThreshold = 8
const smallMessageContentCacheThreshold = 8

func BalanceRows(rows []modelLog.BalanceLogRow) []types.BalanceLog {
	list := make([]types.BalanceLog, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.BalanceLog{
			UserId:    row.UserId,
			Amount:    row.Amount,
			Type:      row.Type,
			OrderNo:   row.OrderNo,
			Balance:   row.Balance,
			Timestamp: row.Timestamp,
		})
	}
	return list
}

func CommissionRows(rows []modelLog.CommissionLogRow) []types.CommissionLog {
	list := make([]types.CommissionLog, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.CommissionLog{
			UserId:    row.UserId,
			Type:      row.Type,
			Amount:    row.Amount,
			OrderNo:   row.OrderNo,
			Timestamp: row.Timestamp,
		})
	}
	return list
}

func GiftRows(rows []modelLog.GiftLogRow) []types.GiftLog {
	list := make([]types.GiftLog, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.GiftLog{
			Type:        row.Type,
			UserId:      row.UserId,
			OrderNo:     row.OrderNo,
			SubscribeId: row.SubscribeId,
			Amount:      row.Amount,
			Balance:     row.Balance,
			Remark:      row.Remark,
			Timestamp:   row.Timestamp,
		})
	}
	return list
}

func ResetSubscribeRows(rows []modelLog.ResetSubscribeLogRow) []types.ResetSubscribeLog {
	list := make([]types.ResetSubscribeLog, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.ResetSubscribeLog{
			Type:            row.Type,
			UserId:          row.UserId,
			UserSubscribeId: row.UserSubscribeId,
			OrderNo:         row.OrderNo,
			Timestamp:       row.Timestamp,
		})
	}
	return list
}

func ResetSubscribeTrafficRows(rows []modelLog.ResetSubscribeLogRow) []types.ResetSubscribeTrafficLog {
	list := make([]types.ResetSubscribeTrafficLog, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.ResetSubscribeTrafficLog{
			Id:              row.Id,
			Type:            row.Type,
			OrderNo:         row.OrderNo,
			Timestamp:       row.Timestamp,
			UserSubscribeId: row.UserSubscribeId,
		})
	}
	return list
}

func LoginRows(rows []modelLog.LoginLogRow, resolve LocationResolver) []types.LoginLog {
	list := make([]types.LoginLog, 0, len(rows))
	locations := newLocationResolverCacheWithHint(resolve, len(rows))
	for _, row := range rows {
		location := locations.Resolve(row.LoginIP, row.CountryShort, row.Region, row.Location)
		loginIP := row.LoginIP
		userAgent := row.UserAgent
		if !row.Revealed {
			loginIP = privacy.MaskIP(loginIP)
			userAgent = privacy.MaskUserAgent(userAgent)
		}
		list = append(list, types.LoginLog{
			UserId:       row.UserId,
			Method:       row.Method,
			LoginIP:      loginIP,
			CountryShort: location.CountryShort,
			Region:       location.Region,
			Location:     location.Location,
			UserAgent:    userAgent,
			Success:      row.Success,
			Timestamp:    timex.UnixMilliOrZero(row.CreatedAt),
			Revealed:     row.Revealed,
		})
	}
	return list
}

func UserLoginRows(rows []modelLog.LoginLogRow, resolve LocationResolver) []types.UserLoginLog {
	list := make([]types.UserLoginLog, 0, len(rows))
	locations := newLocationResolverCacheWithHint(resolve, len(rows))
	for _, row := range rows {
		location := locations.Resolve(row.LoginIP, row.CountryShort, row.Region, row.Location)
		list = append(list, types.UserLoginLog{
			Id:           row.Id,
			UserId:       row.UserId,
			LoginIP:      privacy.MaskIP(row.LoginIP),
			CountryShort: location.CountryShort,
			Region:       location.Region,
			Location:     location.Location,
			UserAgent:    privacy.MaskUserAgent(row.UserAgent),
			Success:      row.Success,
			Timestamp:    timex.UnixMilliOrZero(row.CreatedAt),
		})
	}
	return list
}

func RegisterRows(rows []modelLog.RegisterLogRow, resolve LocationResolver) []types.RegisterLog {
	list := make([]types.RegisterLog, 0, len(rows))
	locations := newLocationResolverCacheWithHint(resolve, len(rows))
	for _, row := range rows {
		location := locations.Resolve(row.RegisterIP, row.CountryShort, row.Region, row.Location)
		identifier := row.Identifier
		registerIP := row.RegisterIP
		userAgent := row.UserAgent
		if !row.Revealed {
			identifier = privacy.MaskIdentifier(identifier)
			registerIP = privacy.MaskIP(registerIP)
			userAgent = privacy.MaskUserAgent(userAgent)
		}
		list = append(list, types.RegisterLog{
			UserId:       row.UserId,
			AuthMethod:   row.AuthMethod,
			Identifier:   identifier,
			RegisterIP:   registerIP,
			CountryShort: location.CountryShort,
			Region:       location.Region,
			Location:     location.Location,
			UserAgent:    userAgent,
			Timestamp:    row.Timestamp,
			Revealed:     row.Revealed,
		})
	}
	return list
}

func SubscribeRows(rows []*modelLog.SubscribeAccessLog, resolve LocationResolver) []types.SubscribeLog {
	list := viewutil.NewRowsCollector[types.SubscribeLog](len(rows))
	locations := newLocationResolverCacheWithHint(resolve, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		location := locations.Resolve(row.ClientIP, row.CountryShort, row.Region, row.Location)
		clientIP := row.ClientIP
		userAgent := row.UserAgent
		if !row.Revealed {
			clientIP = privacy.MaskIP(clientIP)
			userAgent = privacy.MaskUserAgent(userAgent)
		}
		list.Add(types.SubscribeLog{
			UserId:          row.UserId,
			Token:           privacy.MaskTokenPrefix(row.TokenPrefix),
			UserAgent:       userAgent,
			ClientIP:        clientIP,
			CountryShort:    location.CountryShort,
			Region:          location.Region,
			Location:        location.Location,
			UserSubscribeId: row.UserSubscribeId,
			Timestamp:       timex.UnixMilliOrZero(row.CreatedAt),
			Revealed:        row.Revealed,
		})
	}
	return list.Rows()
}

func UserSubscribeRows(rows []*modelLog.SubscribeAccessLog, resolve LocationResolver) []types.UserSubscribeLog {
	list := viewutil.NewRowsCollector[types.UserSubscribeLog](len(rows))
	locations := newLocationResolverCacheWithHint(resolve, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		location := locations.Resolve(row.ClientIP, row.CountryShort, row.Region, row.Location)
		list.Add(types.UserSubscribeLog{
			Id:              row.Id,
			UserId:          row.UserId,
			UserSubscribeId: row.UserSubscribeId,
			Token:           privacy.MaskTokenPrefix(row.TokenPrefix),
			IP:              privacy.MaskIP(row.ClientIP),
			CountryShort:    location.CountryShort,
			Region:          location.Region,
			Location:        location.Location,
			UserAgent:       privacy.MaskUserAgent(row.UserAgent),
			Timestamp:       timex.UnixMilliOrZero(row.CreatedAt),
		})
	}
	return list.Rows()
}

func MessageRows(rows []modelLog.MessageLogRow, logr logger.Logger, scope string) []types.AdminMessageLog {
	list := make([]types.AdminMessageLog, 0, len(rows))
	contents := newMessageContentCache(len(rows))
	for _, row := range rows {
		to := row.To
		content := json.RawMessage(nil)
		if row.Revealed {
			content = rawMessageContent(row.Content, logr, scope, row.Id)
		} else {
			to = privacy.MaskIdentifier(to)
			content = contents.Content(row.Content, logr, scope, row.Id)
		}
		list = append(list, types.AdminMessageLog{
			Id:        row.Id,
			Type:      row.Type,
			Platform:  row.Platform,
			To:        to,
			Subject:   row.Subject,
			Content:   content,
			Status:    row.Status,
			CreatedAt: timex.UnixMilliOrZero(row.CreatedAt),
			Revealed:  row.Revealed,
		})
	}
	return list
}

func rawMessageContent(content string, logr logger.Logger, scope string, id int64) json.RawMessage {
	raw, ok := jsonview.RawValue(content)
	if ok {
		return raw
	}
	if logr != nil && strings.TrimSpace(content) != "" {
		logr.Errorf("%s invalid revealed message content json, id=%d", scope, id)
	}
	return jsonview.RawValueOrNull("")
}

func MessageContent(content string, logr logger.Logger, scope string, id int64) json.RawMessage {
	raw, ok := jsonview.RawValue(content)
	if !ok {
		if logr != nil {
			if strings.TrimSpace(content) != "" {
				logr.Errorf("%s invalid message content json, id=%d", scope, id)
			}
		}
		raw = jsonview.RawValueOrNull("")
	}
	return json.RawMessage(privacy.MaskMessageContentJSON(raw))
}

type messageContentCacheEntry struct {
	raw    string
	masked json.RawMessage
}

type messageContentCache struct {
	rows    []messageContentCacheEntry
	index   map[string]int
	capHint int
}

func newMessageContentCache(capHint int) *messageContentCache {
	if capHint <= 1 {
		return nil
	}
	return &messageContentCache{capHint: capHint}
}

func (c *messageContentCache) Content(content string, logr logger.Logger, scope string, id int64) json.RawMessage {
	raw, ok := jsonview.RawValue(content)
	if !ok {
		return MessageContent(content, logr, scope, id)
	}
	key := string(raw)
	if c == nil {
		return json.RawMessage(privacy.MaskMessageContentJSON(raw))
	}
	if c.index != nil {
		if pos, ok := c.index[key]; ok {
			return c.rows[pos].masked
		}
		return c.add(key, raw)
	}
	for _, row := range c.rows {
		if row.raw == key {
			return row.masked
		}
	}
	return c.add(key, raw)
}

func (c *messageContentCache) add(key string, raw json.RawMessage) json.RawMessage {
	masked := json.RawMessage(privacy.MaskMessageContentJSON(raw))
	c.ensureRows()
	c.rows = append(c.rows, messageContentCacheEntry{
		raw:    key,
		masked: masked,
	})
	if c.index != nil {
		c.index[key] = len(c.rows) - 1
	} else if len(c.rows) > smallMessageContentCacheThreshold {
		nextCap := logViewCacheCap(len(c.rows), c.capHint)
		c.index = make(map[string]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]messageContentCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.raw] = i
		}
	}
	return masked
}

func (c *messageContentCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallMessageContentCacheThreshold
	} else if capHint > smallMessageContentCacheThreshold+1 {
		capHint = smallMessageContentCacheThreshold + 1
	}
	c.rows = make([]messageContentCacheEntry, 0, capHint)
}

func resolveLocation(resolve LocationResolver, ip, countryShort, region, location string) Location {
	cache := newLocationResolverCache(resolve)
	return cache.Resolve(ip, countryShort, region, location)
}

type locationCacheKey struct {
	ip           string
	countryShort string
	region       string
}

type locationCacheEntry struct {
	key      locationCacheKey
	location Location
}

type locationResolverCache struct {
	resolve LocationResolver
	rows    []locationCacheEntry
	index   map[locationCacheKey]int
	capHint int
}

func newLocationResolverCache(resolve LocationResolver) locationResolverCache {
	return newLocationResolverCacheWithHint(resolve, 0)
}

func newLocationResolverCacheWithHint(resolve LocationResolver, capHint int) locationResolverCache {
	if capHint <= 0 {
		return locationResolverCache{resolve: resolve}
	}
	return locationResolverCache{resolve: resolve, capHint: capHint}
}

func (c *locationResolverCache) Resolve(ip, countryShort, region, location string) Location {
	countryShort = strings.TrimSpace(countryShort)
	region = strings.TrimSpace(region)
	location = strings.TrimSpace(location)
	if location != "" {
		return Location{
			CountryShort: countryShort,
			Region:       region,
			Location:     location,
		}
	}
	if c == nil || c.resolve == nil {
		return Location{
			CountryShort: countryShort,
			Region:       region,
		}
	}
	key := locationCacheKey{
		ip:           ip,
		countryShort: countryShort,
		region:       region,
	}
	if cached, ok := c.Get(key); ok {
		return cached
	}
	resolved := c.resolve(ip, countryShort, region)
	c.Set(key, resolved)
	return resolved
}

func (c *locationResolverCache) Get(key locationCacheKey) (Location, bool) {
	if c == nil {
		return Location{}, false
	}
	if c.index != nil {
		pos, ok := c.index[key]
		if !ok {
			return Location{}, false
		}
		return c.rows[pos].location, true
	}
	for _, row := range c.rows {
		if row.key == key {
			return row.location, true
		}
	}
	return Location{}, false
}

func (c *locationResolverCache) Set(key locationCacheKey, location Location) {
	if c == nil {
		return
	}
	if c.index != nil {
		if pos, ok := c.index[key]; ok {
			c.rows[pos].location = location
			return
		}
		c.index[key] = len(c.rows)
		c.ensureRows()
		c.rows = append(c.rows, locationCacheEntry{key: key, location: location})
		return
	}
	for i, row := range c.rows {
		if row.key == key {
			c.rows[i].location = location
			return
		}
	}
	c.ensureRows()
	c.rows = append(c.rows, locationCacheEntry{key: key, location: location})
	if len(c.rows) > smallLocationCacheThreshold {
		nextCap := logViewCacheCap(len(c.rows), c.capHint)
		c.index = make(map[locationCacheKey]int, nextCap)
		if nextCap > cap(c.rows) {
			rows := make([]locationCacheEntry, len(c.rows), nextCap)
			copy(rows, c.rows)
			c.rows = rows
		}
		for i, row := range c.rows {
			c.index[row.key] = i
		}
	}
}

func (c *locationResolverCache) ensureRows() {
	if c.rows != nil {
		return
	}
	capHint := c.capHint
	if capHint <= 0 {
		capHint = smallLocationCacheThreshold
	} else if capHint > smallLocationCacheThreshold+1 {
		capHint = smallLocationCacheThreshold + 1
	}
	c.rows = make([]locationCacheEntry, 0, capHint)
}

func logViewCacheCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}
