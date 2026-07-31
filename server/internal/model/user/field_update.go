package user

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type UserNotifySettingsUpdate struct {
	EnableBalanceNotify   *bool
	EnableLoginNotify     *bool
	EnableSubscribeNotify *bool
	EnableTradeNotify     *bool
}

type UserBasicFieldsUpdate struct {
	Avatar             *string
	Balance            *int64
	Commission         *int64
	ReferralPercentage *uint8
	OnlyFirstPurchase  *bool
	GiftAmount         *int64
	ReferCode          *string
	RefererID          *int64
	Enable             *bool
	IsAdmin            *bool
	Password           *string
}

type UserAuthMethodFieldsUpdate struct {
	AuthIdentifier *string
	Verified       *bool
}

type UserSubscribeActivationUpdate struct {
	ExpireTime              *time.Time
	FinishedAt              *time.Time
	ClearFinishedAt         bool
	Download                *int64
	Upload                  *int64
	TrafficResetAt          *time.Time
	ResetCycle              *uint8
	NextTrafficResetAt      *time.Time
	ClearNextTrafficResetAt bool
	Status                  *uint8
	ProductName             *string
	ProductSnapshot         *string
}

type UserSubscribeFieldsUpdate struct {
	SubscribeID             *int64
	ProductName             *string
	ProductSnapshot         *string
	ExpireTime              *time.Time
	FinishedAt              *time.Time
	ClearFinishedAt         bool
	Traffic                 *int64
	Token                   *string
	UUID                    *string
	Download                *int64
	Upload                  *int64
	TrafficResetAt          *time.Time
	ResetCycle              *uint8
	NextTrafficResetAt      *time.Time
	ClearNextTrafficResetAt bool
	Status                  *uint8
	Note                    *string
}

type DeviceFieldsUpdate struct {
	IP        *string
	UserAgent *string
	Online    *bool
	Enabled   *bool
	UserID    *int64
}

type SubscribeTrafficDelta struct {
	Download int64
	Upload   int64
}

func UpdateValue[T any](value T) *T {
	return &value
}

func (u UserNotifySettingsUpdate) updates() map[string]any {
	updates := make(map[string]any, 4)
	if u.EnableBalanceNotify != nil {
		updates["enable_balance_notify"] = *u.EnableBalanceNotify
	}
	if u.EnableLoginNotify != nil {
		updates["enable_login_notify"] = *u.EnableLoginNotify
	}
	if u.EnableSubscribeNotify != nil {
		updates["enable_subscribe_notify"] = *u.EnableSubscribeNotify
	}
	if u.EnableTradeNotify != nil {
		updates["enable_trade_notify"] = *u.EnableTradeNotify
	}
	return updates
}

func (u UserBasicFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 12)
	if u.Avatar != nil {
		updates["avatar"] = *u.Avatar
	}
	if u.Balance != nil {
		updates["balance"] = *u.Balance
	}
	if u.Commission != nil {
		updates["commission"] = *u.Commission
	}
	if u.ReferralPercentage != nil {
		updates["referral_percentage"] = *u.ReferralPercentage
	}
	if u.OnlyFirstPurchase != nil {
		updates["only_first_purchase"] = *u.OnlyFirstPurchase
	}
	if u.GiftAmount != nil {
		updates["gift_amount"] = *u.GiftAmount
	}
	if u.ReferCode != nil {
		updates["refer_code"] = *u.ReferCode
	}
	if u.RefererID != nil {
		updates["referer_id"] = *u.RefererID
	}
	if u.Enable != nil {
		updates["enable"] = *u.Enable
	}
	if u.IsAdmin != nil {
		updates["is_admin"] = *u.IsAdmin
	}
	if u.Password != nil {
		updates["password"] = *u.Password
		updates["algo"] = "bcrypt"
	}
	return updates
}

func (u UserSubscribeFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 11)
	if u.SubscribeID != nil {
		updates["subscribe_id"] = *u.SubscribeID
	}
	if u.ProductName != nil {
		updates["product_name"] = *u.ProductName
	}
	if u.ProductSnapshot != nil {
		updates["product_snapshot"] = *u.ProductSnapshot
	}
	if u.ExpireTime != nil {
		updates["expire_time"] = *u.ExpireTime
	}
	if u.ClearFinishedAt {
		updates["finished_at"] = nil
	} else if u.FinishedAt != nil {
		updates["finished_at"] = *u.FinishedAt
	}
	if u.Traffic != nil {
		updates["traffic"] = *u.Traffic
	}
	if u.Token != nil {
		updates["token"] = *u.Token
	}
	if u.UUID != nil {
		updates["uuid"] = *u.UUID
	}
	if u.Download != nil {
		updates["download"] = *u.Download
	}
	if u.Upload != nil {
		updates["upload"] = *u.Upload
	}
	if u.TrafficResetAt != nil {
		updates["traffic_reset_at"] = *u.TrafficResetAt
	}
	if u.ResetCycle != nil {
		updates["reset_cycle"] = *u.ResetCycle
	}
	if u.ClearNextTrafficResetAt {
		updates["next_traffic_reset_at"] = nil
	} else if u.NextTrafficResetAt != nil {
		updates["next_traffic_reset_at"] = *u.NextTrafficResetAt
	}
	if u.Status != nil {
		updates["status"] = *u.Status
	}
	if u.Note != nil {
		updates["note"] = *u.Note
	}
	return updates
}

func (u UserAuthMethodFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 2)
	if u.AuthIdentifier != nil {
		updates["auth_identifier"] = *u.AuthIdentifier
	}
	if u.Verified != nil {
		updates["verified"] = *u.Verified
	}
	return updates
}

func (u UserSubscribeActivationUpdate) updates() map[string]any {
	updates := make(map[string]any, 8)
	if u.ExpireTime != nil {
		updates["expire_time"] = *u.ExpireTime
	}
	if u.ClearFinishedAt {
		updates["finished_at"] = nil
	} else if u.FinishedAt != nil {
		updates["finished_at"] = *u.FinishedAt
	}
	if u.Download != nil {
		updates["download"] = *u.Download
	}
	if u.Upload != nil {
		updates["upload"] = *u.Upload
	}
	if u.TrafficResetAt != nil {
		updates["traffic_reset_at"] = *u.TrafficResetAt
	}
	if u.ResetCycle != nil {
		updates["reset_cycle"] = *u.ResetCycle
	}
	if u.ClearNextTrafficResetAt {
		updates["next_traffic_reset_at"] = nil
	} else if u.NextTrafficResetAt != nil {
		updates["next_traffic_reset_at"] = *u.NextTrafficResetAt
	}
	if u.Status != nil {
		updates["status"] = *u.Status
	}
	if u.ProductName != nil {
		updates["product_name"] = *u.ProductName
	}
	if u.ProductSnapshot != nil {
		updates["product_snapshot"] = *u.ProductSnapshot
	}
	return updates
}

func (u DeviceFieldsUpdate) updates() map[string]any {
	updates := make(map[string]any, 5)
	if u.IP != nil {
		updates["ip"] = *u.IP
	}
	if u.UserAgent != nil {
		updates["user_agent"] = *u.UserAgent
	}
	if u.Online != nil {
		updates["online"] = *u.Online
	}
	if u.Enabled != nil {
		updates["enabled"] = *u.Enabled
	}
	if u.UserID != nil {
		updates["user_id"] = *u.UserID
	}
	return updates
}

func (u UserAuthMethodFieldsUpdate) cacheModel(userId int64, authType string) *AuthMethods {
	if u.AuthIdentifier == nil {
		return nil
	}
	return &AuthMethods{
		UserId:         userId,
		AuthType:       authType,
		AuthIdentifier: *u.AuthIdentifier,
	}
}

func (u SubscribeTrafficDelta) Updates() map[string]any {
	return map[string]any{
		"download": gorm.Expr("download + ?", u.Download),
		"upload":   gorm.Expr("upload + ?", u.Upload),
	}
}

func (m *defaultUserModel) UpdateUserPassword(ctx context.Context, userId int64, password string) error {
	keys, err := m.userMutationCacheKeys(ctx, userId)
	if err != nil {
		return err
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Updates(map[string]any{
				"password": password,
				"algo":     "bcrypt",
			}).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserBalance(ctx context.Context, userId int64, balance int64, tx ...*gorm.DB) error {
	keys := (&User{Id: userId}).GetCacheKeys()
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Update("balance", balance).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserGiftAmount(ctx context.Context, userId int64, giftAmount int64, tx ...*gorm.DB) error {
	keys := (&User{Id: userId}).GetCacheKeys()
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Update("gift_amount", giftAmount).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserCommission(ctx context.Context, userId int64, commission int64, tx ...*gorm.DB) error {
	keys := (&User{Id: userId}).GetCacheKeys()
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Update("commission", commission).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserReferCode(ctx context.Context, userId int64, referCode string, tx ...*gorm.DB) error {
	keys := (&User{Id: userId}).GetCacheKeys()
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Update("refer_code", referCode).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserMemberExpiredAt(ctx context.Context, userId int64, expiredAt time.Time, tx ...*gorm.DB) error {
	keys := (&User{Id: userId}).GetCacheKeys()
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Update("member_expired_at", expiredAt).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserRules(ctx context.Context, userId int64, rules string) error {
	keys, err := m.userMutationCacheKeys(ctx, userId)
	if err != nil {
		return err
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Update("rules", rules).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserNotifySettings(ctx context.Context, userId int64, update UserNotifySettingsUpdate) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	keys, err := m.userMutationCacheKeys(ctx, userId)
	if err != nil {
		return err
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Updates(updates).Error
	}, keys...)
}

func (m *defaultUserModel) UpdateUserBasicFields(ctx context.Context, userId int64, update UserBasicFieldsUpdate, tx ...*gorm.DB) error {
	updates := update.updates()
	if len(updates) == 0 {
		return nil
	}
	keys, err := m.userMutationCacheKeys(ctx, userId)
	if err != nil {
		return err
	}
	return m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&User{}).
			Where("id = ?", userId).
			Updates(updates).Error
	}, keys...)
}

func (m *defaultUserModel) userMutationCacheKeys(ctx context.Context, userId int64, extraAuths ...*AuthMethods) ([]string, error) {
	auths, err := m.FindUserAuthMethods(ctx, userId)
	if err != nil {
		return nil, err
	}
	collector := newModelCacheKeyCollector(userModelCacheKeyHint + (len(auths)+len(extraAuths))*authMethodModelCacheKeyHint)
	addModelCacheKeys(&collector, &User{Id: userId})
	for _, auth := range auths {
		addModelCacheKeys(&collector, auth)
	}
	for _, auth := range extraAuths {
		addModelCacheKeys(&collector, auth)
	}
	if collector.Len() == 0 {
		return emptyModelCacheKeys, nil
	}
	return collector.Keys(), nil
}

func uniqueCacheKeys(keys []string) []string {
	collector := newModelCacheKeyCollector(len(keys))
	for _, key := range keys {
		collector.Add(key)
	}
	if collector.Len() == 0 {
		return emptyModelCacheKeys
	}
	return collector.Keys()
}
