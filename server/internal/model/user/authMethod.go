package user

import (
	"context"

	"gorm.io/gorm"
)

func (m *defaultUserModel) FindUserAuthMethods(ctx context.Context, userId int64) ([]*AuthMethods, error) {
	data := make([]*AuthMethods, 0)
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&AuthMethods{}).
			Select(m.authMethodDisplayColumns(userAuthMethodCacheColumns)).
			Where("user_id = ?", userId).
			Find(v).Error
	})
	return data, err
}

func (m *defaultUserModel) FindUserAuthMethodSummaries(ctx context.Context, userId int64) ([]*AuthMethods, error) {
	data := make([]*AuthMethods, 0)
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&AuthMethods{}).
			Select(m.authMethodDisplayColumns(userAuthMethodSummaryColumns)).
			Where("user_id = ?", userId).
			Order("id ASC").
			Find(v).Error
	})
	return data, err
}

// RevealUserAuthMethods is the explicit, uncached path for authorized PII access.
// Callers must enforce administrator authorization and write an audit record.
func (m *defaultUserModel) RevealUserAuthMethods(ctx context.Context, userId int64) ([]*AuthMethods, error) {
	data := make([]*AuthMethods, 0)
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&AuthMethods{}).
			Select(userAuthMethodLookupColumns).
			Where("user_id = ?", userId).
			Order("id ASC").
			Find(v).Error
	})
	if err != nil {
		return nil, err
	}
	for _, method := range data {
		if err := m.openAuthMethod(method); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (m *defaultUserModel) FindUserAuthMethodByOpenID(ctx context.Context, method, openID string) (*AuthMethods, error) {
	var data AuthMethods
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return m.identityLookup(conn.Model(&AuthMethods{}).Select(userAuthMethodLookupColumns), method, openID).
			First(v).Error
	})
	if err == nil {
		err = m.openAuthMethod(&data)
	}
	return &data, err
}

func (m *defaultUserModel) FindUserAuthMethodByPlatform(ctx context.Context, userId int64, platform string) (*AuthMethods, error) {
	var data AuthMethods
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&AuthMethods{}).
			Select(userAuthMethodLookupColumns).
			Where("user_id = ? AND auth_type = ?", userId, platform).
			First(v).Error
	})
	if err == nil {
		err = m.openAuthMethod(&data)
	}
	return &data, err
}

func (m *defaultUserModel) InsertUserAuthMethods(ctx context.Context, data *AuthMethods, tx ...*gorm.DB) error {
	protected, err := m.protectAuthMethod(data)
	if err != nil {
		return err
	}
	keys, err := m.userMutationCacheKeys(ctx, data.UserId, data)
	if err != nil {
		return err
	}

	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&AuthMethods{}).Create(protected).Error
	}, keys...)
	if err == nil && protected != data {
		data.Id = protected.Id
		data.CreatedAt = protected.CreatedAt
		data.UpdatedAt = protected.UpdatedAt
	}
	return err
}

func (m *defaultUserModel) UpdateUserAuthMethods(ctx context.Context, userId int64, authType string, update UserAuthMethodFieldsUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	cacheModel := update.cacheModel(userId, authType)
	if update.AuthIdentifier != nil && m.privacyCodec.Enabled() {
		protected, err := m.protectAuthMethod(cacheModel)
		if err != nil {
			return err
		}
		updates["auth_identifier"] = protected.AuthIdentifier
		updates["auth_identifier_bidx"] = protected.AuthIdentifierBlindIndex
		updates["auth_identifier_display"] = protected.AuthIdentifierDisplay
	}
	if len(updates) == 0 {
		return nil
	}

	keys, err := m.userMutationCacheKeys(ctx, userId, cacheModel)
	if err != nil {
		return err
	}

	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&AuthMethods{}).
			Where("user_id = ? AND auth_type = ?", userId, authType).
			Updates(updates).Error
	}, keys...)
}

func (m *defaultUserModel) DeleteUserAuthMethods(ctx context.Context, userId int64, platform string, tx ...*gorm.DB) error {
	keys, err := m.userMutationCacheKeys(ctx, userId)
	if err != nil {
		return err
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&AuthMethods{}).Where("user_id = ? AND auth_type = ?", userId, platform).Delete(&AuthMethods{}).Error
	}, keys...)
}

func (m *defaultUserModel) FindUserAuthMethodByUserId(ctx context.Context, method string, userId int64) (*AuthMethods, error) {
	var data AuthMethods
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&AuthMethods{}).
			Select(userAuthMethodLookupColumns).
			Where("auth_type = ? AND user_id = ?", method, userId).
			First(v).Error
	})
	if err == nil {
		err = m.openAuthMethod(&data)
	}
	return &data, err
}

func (m *defaultUserModel) FindUserAuthMethodsByType(ctx context.Context, authType string, userIds []int64, reveal bool) ([]*AuthMethods, error) {
	data := make([]*AuthMethods, 0)
	if len(userIds) == 0 {
		return data, nil
	}
	columns := m.authMethodDisplayColumns(userAuthMethodLookupColumns)
	if reveal {
		columns = userAuthMethodLookupColumns
	}
	err := m.QueryNoCacheCtx(ctx, &data, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&AuthMethods{}).
			Select(columns).
			Where("auth_type = ? AND user_id IN ?", authType, userIds).
			Order("id ASC").
			Find(v).Error
	})
	if err != nil || !reveal {
		return data, err
	}
	for _, item := range data {
		if err := m.openAuthMethod(item); err != nil {
			return nil, err
		}
	}
	return data, nil
}
