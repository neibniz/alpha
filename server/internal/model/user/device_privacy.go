package user

import (
	"context"
	"fmt"
	"strings"

	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (m *defaultUserModel) protectDevice(data *Device) (*Device, error) {
	if data == nil || !m.privacyCodec.Enabled() {
		return data, nil
	}
	protected := *data
	identifier := strings.TrimSpace(data.Identifier)
	var err error
	protected.Identifier, err = m.privacyCodec.Seal(privacycodec.DeviceScope(data.UserId, "identifier"), identifier)
	if err != nil {
		return nil, err
	}
	protected.Ip, err = m.privacyCodec.Seal(privacycodec.DeviceScope(data.UserId, "ip"), strings.TrimSpace(data.Ip))
	if err != nil {
		return nil, err
	}
	protected.UserAgent, err = m.privacyCodec.Seal(privacycodec.DeviceScope(data.UserId, "user_agent"), strings.TrimSpace(data.UserAgent))
	if err != nil {
		return nil, err
	}
	protected.IdentifierBlindIndex = m.privacyCodec.BlindIndex("device", identifier)
	protected.IdentifierDisplay = privacycodec.MaskIdentifier(identifier)
	protected.IPDisplay = privacycodec.MaskIP(data.Ip)
	protected.UserAgentDisplay = privacycodec.MaskUserAgent(data.UserAgent)
	data.IdentifierBlindIndex = protected.IdentifierBlindIndex
	data.IdentifierDisplay = protected.IdentifierDisplay
	data.IPDisplay = protected.IPDisplay
	data.UserAgentDisplay = protected.UserAgentDisplay
	return &protected, nil
}

func (m *defaultUserModel) openDevice(data *Device) error {
	if data == nil || !m.privacyCodec.Enabled() {
		return nil
	}
	var err error
	data.Identifier, err = m.privacyCodec.Open(privacycodec.DeviceScope(data.UserId, "identifier"), data.Identifier)
	if err != nil {
		return err
	}
	data.Ip, err = m.privacyCodec.Open(privacycodec.DeviceScope(data.UserId, "ip"), data.Ip)
	if err != nil {
		return err
	}
	data.UserAgent, err = m.privacyCodec.Open(privacycodec.DeviceScope(data.UserId, "user_agent"), data.UserAgent)
	return err
}

func (m *defaultUserModel) deviceIdentifierLookup(query *gorm.DB, identifier string) *gorm.DB {
	identifier = strings.TrimSpace(identifier)
	if !m.privacyCodec.Enabled() {
		return query.Where("identifier = ?", identifier)
	}
	return query.Where(
		"identifier_bidx = ? OR (identifier_bidx = '' AND identifier = ?)",
		m.privacyCodec.BlindIndex("device", identifier),
		identifier,
	)
}

func (m *defaultUserModel) deviceCacheToken(identifier string) string {
	return m.privacyCodec.CacheToken("device", identifier)
}

func (m *defaultUserModel) deviceCacheColumns(columns []string) []string {
	if !m.privacyCodec.Enabled() {
		return columns
	}
	return []string{"id", "user_id", "identifier_bidx"}
}

func (m *defaultUserModel) deviceListColumns(columns []string) []string {
	if !m.privacyCodec.Enabled() {
		return columns
	}
	result := append([]string(nil), columns...)
	return append(result, "user_id", "identifier_bidx", "identifier_display", "ip_display", "user_agent_display")
}

func (m *defaultUserModel) deviceStatusColumns(columns []string) []string {
	if !m.privacyCodec.Enabled() {
		return columns
	}
	result := append([]string(nil), columns...)
	return append(result, "ip", "user_agent", "identifier_bidx")
}

func (m *defaultUserModel) MigrateDevicePrivacy(ctx context.Context, batchSize int) (int64, error) {
	if !m.privacyCodec.Enabled() {
		return 0, nil
	}
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = 200
	}
	var migrated int64
	for {
		var count int
		err := m.Transaction(ctx, func(tx *gorm.DB) error {
			var rows []Device
			query := tx.Where("identifier_bidx = ''")
			if tx.Dialector.Name() == "postgres" {
				query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
			}
			if err := query.Order("id ASC").Limit(batchSize).Find(&rows).Error; err != nil {
				return err
			}
			count = len(rows)
			for index := range rows {
				protected, err := m.protectDevice(&rows[index])
				if err != nil {
					return fmt.Errorf("protect device %d: %w", rows[index].Id, err)
				}
				if err := tx.Model(&Device{}).Where("id = ?", rows[index].Id).Updates(map[string]any{
					"identifier":         protected.Identifier,
					"identifier_bidx":    protected.IdentifierBlindIndex,
					"identifier_display": protected.IdentifierDisplay,
					"ip":                 protected.Ip,
					"ip_display":         protected.IPDisplay,
					"user_agent":         protected.UserAgent,
					"user_agent_display": protected.UserAgentDisplay,
				}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return migrated, err
		}
		migrated += int64(count)
		if count < batchSize {
			return migrated, nil
		}
	}
}
