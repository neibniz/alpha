package user

import (
	"context"
	"fmt"
	"strings"

	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (m *defaultUserModel) identityCacheToken(kind, value string) string {
	return m.privacyCodec.CacheToken(kind, value)
}

func (m *defaultUserModel) identityLookup(query *gorm.DB, authType, value string) *gorm.DB {
	normalized := privacycodec.Normalize(authType, value)
	if !m.privacyCodec.Enabled() {
		return query.Where("auth_type = ? AND auth_identifier = ?", authType, normalized)
	}
	return query.Where(
		"auth_type = ? AND (auth_identifier_bidx = ? OR (auth_identifier_bidx = '' AND auth_identifier = ?))",
		authType,
		m.privacyCodec.BlindIndex(authType, normalized),
		normalized,
	)
}

func (m *defaultUserModel) protectAuthMethod(data *AuthMethods) (*AuthMethods, error) {
	if data == nil || !m.privacyCodec.Enabled() {
		return data, nil
	}
	protected := *data
	ciphertext, blindIndex, display, err := m.privacyCodec.ProtectIdentity(data.UserId, data.AuthType, data.AuthIdentifier)
	if err != nil {
		return nil, err
	}
	protected.AuthIdentifier = ciphertext
	protected.AuthIdentifierBlindIndex = blindIndex
	protected.AuthIdentifierDisplay = display
	data.AuthIdentifierBlindIndex = blindIndex
	data.AuthIdentifierDisplay = display
	return &protected, nil
}

func (m *defaultUserModel) openAuthMethod(data *AuthMethods) error {
	if data == nil || !m.privacyCodec.Enabled() {
		return nil
	}
	plaintext, err := m.privacyCodec.OpenIdentity(data.UserId, data.AuthType, data.AuthIdentifier)
	if err != nil {
		return err
	}
	data.AuthIdentifier = plaintext
	return nil
}

func (m *defaultUserModel) authMethodDisplayColumns(columns []string) []string {
	if !m.privacyCodec.Enabled() {
		return columns
	}
	result := make([]string, 0, len(columns)+1)
	seen := make(map[string]struct{}, len(columns)+1)
	for _, column := range columns {
		projected := column
		if column == "auth_identifier" {
			projected = "auth_identifier_display AS auth_identifier"
		}
		if _, ok := seen[projected]; ok {
			continue
		}
		seen[projected] = struct{}{}
		result = append(result, projected)
	}
	result = append(result, "auth_identifier_bidx", "auth_identifier_display")
	return result
}

func looksLikeIdentity(value string) bool {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "@") || strings.HasPrefix(value, "+") {
		return true
	}
	if len(value) < 6 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (m *defaultUserModel) identitySearchIndexes(value string) []string {
	kinds := []string{"email", "mobile", "telegram", "google", "apple", "github", "facebook", "device"}
	indexes := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		indexes = append(indexes, m.privacyCodec.BlindIndex(kind, value))
	}
	return indexes
}

func (m *defaultUserModel) MigratePrivacy(ctx context.Context, batchSize int) (int64, error) {
	if !m.privacyCodec.Enabled() {
		return 0, nil
	}
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = 200
	}
	var migrated int64
	for {
		var batchCount int
		err := m.Transaction(ctx, func(tx *gorm.DB) error {
			var rows []AuthMethods
			query := tx.Where("auth_identifier_bidx = ''")
			if tx.Dialector.Name() == "postgres" {
				query = query.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
			}
			if err := query.
				Order("id ASC").
				Limit(batchSize).
				Find(&rows).Error; err != nil {
				return err
			}
			batchCount = len(rows)
			for index := range rows {
				row := &rows[index]
				protected, err := m.protectAuthMethod(row)
				if err != nil {
					return fmt.Errorf("protect auth method %d: %w", row.Id, err)
				}
				if err := tx.Model(&AuthMethods{}).Where("id = ?", row.Id).Updates(map[string]any{
					"auth_identifier":         protected.AuthIdentifier,
					"auth_identifier_bidx":    protected.AuthIdentifierBlindIndex,
					"auth_identifier_display": protected.AuthIdentifierDisplay,
				}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return migrated, err
		}
		migrated += int64(batchCount)
		if batchCount < batchSize {
			return migrated, nil
		}
	}
}
