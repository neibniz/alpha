package log

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/gorm"
)

var _ Model = (*customSystemLogModel)(nil)

type (
	Model interface {
		systemLogModel
		customSystemLogLogicModel
	}
	systemLogModel interface {
		Insert(ctx context.Context, data *SystemLog) error
		InsertSubscribeAccessLog(ctx context.Context, data *SubscribeAccessLog) error
		FindOne(ctx context.Context, id int64) (*SystemLog, error)
		Update(ctx context.Context, data *SystemLog) error
		Delete(ctx context.Context, id int64) error
	}
	customSystemLogModel struct {
		*defaultLogModel
	}
	Hooks struct {
		LookupIPLocation           func(context.Context, string) IPLocation
		SystemLogInserted          func(context.Context, *SystemLog)
		SubscribeAccessLogInserted func(context.Context, *SubscribeAccessLog)
	}
	IPLocation struct {
		CountryShort string
		Region       string
		Location     string
	}
	defaultLogModel struct {
		*gorm.DB
		hooks Hooks
	}
)

func newSystemLogModel(db *gorm.DB, hooks Hooks) *defaultLogModel {
	return &defaultLogModel{
		DB:    db,
		hooks: hooks,
	}
}

func (m *defaultLogModel) Insert(ctx context.Context, data *SystemLog) error {
	m.enrichSystemLog(ctx, data)
	if err := m.WithContext(ctx).Create(data).Error; err != nil {
		return err
	}
	if m.hooks.SystemLogInserted != nil {
		m.hooks.SystemLogInserted(ctx, data)
	}
	return nil
}

func (m *defaultLogModel) InsertSubscribeAccessLog(ctx context.Context, data *SubscribeAccessLog) error {
	m.enrichSubscribeAccessLog(ctx, data)
	if err := m.WithContext(ctx).Create(data).Error; err != nil {
		return err
	}
	if m.hooks.SubscribeAccessLogInserted != nil {
		m.hooks.SubscribeAccessLogInserted(ctx, data)
	}
	return nil
}

func (m *defaultLogModel) FindOne(ctx context.Context, id int64) (*SystemLog, error) {
	var log SystemLog
	err := m.WithContext(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (m *defaultLogModel) Update(ctx context.Context, data *SystemLog) error {
	return m.WithContext(ctx).Where("id = ?", data.Id).Save(data).Error
}

func (m *defaultLogModel) Delete(ctx context.Context, id int64) error {
	return m.WithContext(ctx).Where("id = ?", id).Delete(&SystemLog{}).Error
}

func (m *defaultLogModel) enrichSystemLog(ctx context.Context, data *SystemLog) {
	if data == nil {
		return
	}
	enrichSystemLogFields(data)
	if data.IP != "" && m.hooks.LookupIPLocation != nil {
		location := m.hooks.LookupIPLocation(ctx, data.IP)
		data.CountryShort = location.CountryShort
		data.Region = location.Region
		data.Location = location.Location
		if data.Location == "" {
			data.Location = formatIPLocation(location.CountryShort, location.Region)
		}
	}
	data.IP = privacy.MaskIP(data.IP)
	data.UserAgent = privacy.MaskUserAgent(data.UserAgent)
	data.Identifier = privacy.MaskIdentifier(data.Identifier)
	data.Content = string(privacy.MaskJSON([]byte(data.Content)))
}

func (m *defaultLogModel) enrichSubscribeAccessLog(ctx context.Context, data *SubscribeAccessLog) {
	if data == nil {
		return
	}
	if data.ClientIP != "" && m.hooks.LookupIPLocation != nil {
		location := m.hooks.LookupIPLocation(ctx, data.ClientIP)
		data.CountryShort = location.CountryShort
		data.Region = location.Region
		data.Location = location.Location
		if data.Location == "" {
			data.Location = formatIPLocation(location.CountryShort, location.Region)
		}
	}
	data.ClientIP = privacy.MaskIP(data.ClientIP)
	data.UserAgent = privacy.MaskUserAgent(data.UserAgent)
}

func formatIPLocation(countryShort, region string) string {
	countryShort = strings.TrimSpace(countryShort)
	region = strings.TrimSpace(region)
	if countryShort == "" {
		return region
	}
	if region == "" {
		return countryShort
	}
	return countryShort + " · " + region
}

func enrichSystemLogFields(data *SystemLog) {
	switch Type(data.Type) {
	case TypeLogin:
		var content Login
		if content.Unmarshal([]byte(data.Content)) == nil {
			setSystemLogNetworkFields(data, content.LoginIP, content.UserAgent, content.Timestamp)
			if data.Method == "" {
				data.Method = content.Method
			}
			data.Success = content.Success
		}
	case TypeRegister:
		var content Register
		if content.Unmarshal([]byte(data.Content)) == nil {
			setSystemLogNetworkFields(data, content.RegisterIP, content.UserAgent, content.Timestamp)
			if data.Method == "" {
				data.Method = content.AuthMethod
			}
			if data.Identifier == "" {
				data.Identifier = privacy.MaskIdentifier(content.Identifier)
			}
		}
	case TypeLogout:
		var content Logout
		if content.Unmarshal([]byte(data.Content)) == nil {
			setSystemLogNetworkFields(data, content.LogoutIP, content.UserAgent, content.Timestamp)
		}
	case TypeSubscribe:
		var content Subscribe
		if content.Unmarshal([]byte(data.Content)) == nil {
			setSystemLogNetworkFields(data, content.ClientIP, content.UserAgent, 0)
		}
	}
}

func setSystemLogNetworkFields(data *SystemLog, ip, userAgent string, eventAt int64) {
	if data.IP == "" {
		data.IP = ip
	}
	if data.UserAgent == "" {
		data.UserAgent = userAgent
	}
	if data.EventAt == 0 {
		data.EventAt = eventAt
	}
}
