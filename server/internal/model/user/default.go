package user

import (
	"context"
	"errors"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	privacycodec "github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/pkg/cache"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	cacheUserIdPrefix              = "cache:user:id:"
	cacheUserContextPrefix         = "cache:user:context:"
	cacheUserLoginIdPrefix         = "cache:user:login:id:"
	cacheUserSubscribeAccessPrefix = "cache:user:subscribe-access:"
	cacheUserAdminCheckPrefix      = "cache:user:admin-check:"
	cacheUserEmailPrefix           = "cache:user:email:"
	cacheUserEmailLookupPrefix     = "cache:user:email:lookup:"
	cacheUserLoginEmailPrefix      = "cache:user:login:email:"
)
var _ Model = (*customUserModel)(nil)

var userDeviceCachePreloadColumns = []string{
	"id",
	"user_id",
	"identifier",
}

type (
	Model interface {
		userModel
		customUserLogicModel
	}
	userModel interface {
		Insert(ctx context.Context, data *User, tx ...*gorm.DB) error
		FindOne(ctx context.Context, id int64) (*User, error)
		FindOneForContext(ctx context.Context, id int64) (*User, error)
		FindOneForAdminCheck(ctx context.Context, id int64) (*User, error)
		FindOneForLogin(ctx context.Context, id int64) (*User, error)
		FindOneForLoginByEmail(ctx context.Context, email string) (*User, error)
		Update(ctx context.Context, data *User, tx ...*gorm.DB) error
		Delete(ctx context.Context, id int64, tx ...*gorm.DB) error
		Transaction(ctx context.Context, fn func(db *gorm.DB) error) error
	}

	customUserModel struct {
		*defaultUserModel
	}
	defaultUserModel struct {
		cache.CachedConn
		table        string
		privacyCodec *privacycodec.Codec
	}
)

func newUserModel(db *gorm.DB, c *redis.Client, codec *privacycodec.Codec) *defaultUserModel {
	if codec == nil {
		codec = privacycodec.DisabledCodec()
	}
	return &defaultUserModel{
		CachedConn:   cache.NewConn(db, c),
		table:        "user",
		privacyCodec: codec,
	}
}

func (m *defaultUserModel) batchGetCacheKeys(users ...*User) []string {
	keyCap := len(users) * userModelCacheKeyHint
	for _, user := range users {
		if user != nil {
			keyCap += len(user.UserDevices) * deviceModelCacheKeyHint
		}
	}
	collector := newModelCacheKeyCollector(keyCap)
	for _, user := range users {
		if user == nil {
			continue
		}
		addModelCacheKeys(&collector, user)
		for i := range user.UserDevices {
			addModelCacheKeys(&collector, &user.UserDevices[i])
		}
	}
	if collector.Len() == 0 {
		return emptyModelCacheKeys
	}
	return collector.Keys()
}

func (m *defaultUserModel) getCacheKeys(data *User) []string {
	if data == nil {
		return []string{}
	}
	return data.GetCacheKeys()
}

func (m *defaultUserModel) clearUserCache(ctx context.Context, data ...*User) error {
	return m.ClearUserCache(ctx, data...)
}

func (m *defaultUserModel) FindOneByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	key := cachekeys.String(cacheUserEmailLookupPrefix, m.identityCacheToken("email", email))
	err := m.QueryCtx(ctx, &user, key, func(conn *gorm.DB, v interface{}) error {
		var data AuthMethods
		if err := m.identityLookup(conn.Model(&AuthMethods{}).Select("user_id"), "email", email).
			First(&data).Error; err != nil {
			return err
		}
		return conn.Model(&User{}).
			Unscoped().
			Select(userEmailLookupSelectColumns).
			Where("id = ?", data.UserId).
			First(v).Error
	})
	return &user, err
}

func (m *defaultUserModel) FindOneForLoginByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	key := cachekeys.String(cacheUserLoginEmailPrefix, m.identityCacheToken("email", email))
	err := m.QueryCtx(ctx, &user, key, func(conn *gorm.DB, v interface{}) error {
		var data AuthMethods
		if err := m.identityLookup(conn.Model(&AuthMethods{}).Select("user_id"), "email", email).
			First(&data).Error; err != nil {
			return err
		}
		return conn.Model(&User{}).
			Unscoped().
			Select(userLoginSelectColumns).
			Where("id = ?", data.UserId).
			First(v).Error
	})
	return &user, err
}

func (m *defaultUserModel) Insert(ctx context.Context, data *User, tx ...*gorm.DB) error {
	if data == nil {
		return errors.New("user is nil")
	}
	if m.privacyCodec.Enabled() {
		for index := range data.AuthMethods {
			_, blindIndex, display, err := m.privacyCodec.ProtectIdentity(0, data.AuthMethods[index].AuthType, data.AuthMethods[index].AuthIdentifier)
			if err != nil {
				return err
			}
			data.AuthMethods[index].AuthIdentifierBlindIndex = blindIndex
			data.AuthMethods[index].AuthIdentifierDisplay = display
		}
	}
	authMethods := append([]AuthMethods(nil), data.AuthMethods...)
	err := m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		data.AuthMethods = nil
		if err := conn.Omit("AuthMethods").Create(data).Error; err != nil {
			data.AuthMethods = authMethods
			return err
		}
		for index := range authMethods {
			authMethods[index].UserId = data.Id
			protected, err := m.protectAuthMethod(&authMethods[index])
			if err != nil {
				data.AuthMethods = authMethods
				return err
			}
			if err := conn.Create(protected).Error; err != nil {
				data.AuthMethods = authMethods
				return err
			}
			authMethods[index].Id = protected.Id
			authMethods[index].CreatedAt = protected.CreatedAt
			authMethods[index].UpdatedAt = protected.UpdatedAt
		}
		data.AuthMethods = authMethods
		return nil
	}, m.getCacheKeys(data)...)
	return err
}

func (m *defaultUserModel) FindOne(ctx context.Context, id int64) (*User, error) {
	userIdKey := cachekeys.Int(cacheUserIdPrefix, id)
	var resp User
	err := m.QueryCtx(ctx, &resp, userIdKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Unscoped().
			Select(userCacheSelectColumns).
			Where("id = ?", id).
			Preload("UserDevices", func(db *gorm.DB) *gorm.DB {
				return db.Select(m.deviceCacheColumns(userDeviceCachePreloadColumns)).Order("id ASC")
			}).
			Preload("AuthMethods", func(db *gorm.DB) *gorm.DB {
				return db.Select(m.authMethodDisplayColumns(userAuthMethodCacheColumns)).Order("id ASC")
			}).
			First(v).Error
	})
	return &resp, err
}

func (m *defaultUserModel) findOneUserForCache(ctx context.Context, id int64) (*User, error) {
	users, err := m.queryUsersForCache(ctx, []int64{id}, true)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return users[0], nil
}

func (m *defaultUserModel) queryUsersForCache(ctx context.Context, ids []int64, unscoped bool, tx ...*gorm.DB) ([]*User, error) {
	ids = normalizeUserCacheIDs(ids)
	if len(ids) == 0 {
		return []*User{}, nil
	}
	var users []*User
	err := m.QueryNoCacheCtx(ctx, &users, func(conn *gorm.DB, v interface{}) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		query := conn.Model(&User{}).Select(userCacheSelectColumns).Where("id IN ?", ids)
		if unscoped {
			query = query.Unscoped()
		}
		return query.Find(v).Error
	})
	if err != nil {
		return nil, err
	}
	if err = m.loadUserCacheRelations(ctx, users, tx...); err != nil {
		return nil, err
	}
	return users, nil
}

func normalizeUserCacheIDs(ids []int64) []int64 {
	set := newUserPositiveIDSet(len(ids))
	for _, id := range ids {
		set.Add(id)
	}
	return set.Values()
}

type userPositiveIDSet struct {
	ids     []int64
	seen    map[int64]struct{}
	capHint int
}

func newUserPositiveIDSet(capHint int) userPositiveIDSet {
	return userPositiveIDSet{capHint: capHint}
}

func (s *userPositiveIDSet) Add(id int64) bool {
	if s == nil || id <= 0 {
		return false
	}
	if s.seen != nil {
		if _, ok := s.seen[id]; ok {
			return false
		}
		s.seen[id] = struct{}{}
		s.ids = append(s.ids, id)
		return true
	}
	for _, existing := range s.ids {
		if existing == id {
			return false
		}
	}
	s.ensureIDs()
	s.ids = append(s.ids, id)
	if len(s.ids) > smallUserCacheRelationLookupLimit {
		s.upgradeLookup()
	}
	return true
}

func (s userPositiveIDSet) Values() []int64 {
	if len(s.ids) == 0 {
		return []int64{}
	}
	return s.ids
}

func (s *userPositiveIDSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	s.ids = make([]int64, 0, userCacheRelationInitialCap(s.capHint))
}

func (s *userPositiveIDSet) upgradeLookup() {
	if s.seen != nil {
		return
	}
	nextCap := userCacheRelationLookupCap(len(s.ids), s.capHint)
	if nextCap > cap(s.ids) {
		ids := make([]int64, len(s.ids), nextCap)
		copy(ids, s.ids)
		s.ids = ids
	}
	s.seen = make(map[int64]struct{}, nextCap)
	for _, id := range s.ids {
		s.seen[id] = struct{}{}
	}
}

func (m *defaultUserModel) loadUserCacheRelations(ctx context.Context, users []*User, tx ...*gorm.DB) error {
	if len(users) == 0 {
		return nil
	}
	index := newUserCacheRelationIndex(users)
	ids := index.IDs()
	if len(ids) == 0 {
		return nil
	}

	var auths []AuthMethods
	err := m.QueryNoCacheCtx(ctx, &auths, func(conn *gorm.DB, v interface{}) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&AuthMethods{}).
			Select(m.authMethodDisplayColumns(userAuthMethodCacheColumns)).
			Where("user_id IN ?", ids).
			Order("user_id ASC, id ASC").
			Find(v).Error
	})
	if err != nil {
		return err
	}
	for _, auth := range auths {
		if item := index.Get(auth.UserId); item != nil {
			item.AuthMethods = append(item.AuthMethods, auth)
		}
	}

	type deviceCacheRow struct {
		Id                   int64
		UserId               int64
		Identifier           string
		IdentifierBlindIndex string `gorm:"column:identifier_bidx"`
	}
	var devices []deviceCacheRow
	err = m.QueryNoCacheCtx(ctx, &devices, func(conn *gorm.DB, v interface{}) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Model(&Device{}).
			Select(m.deviceCacheColumns([]string{"id", "user_id", "identifier"})).
			Where("user_id IN ?", ids).
			Order("user_id ASC, id ASC").
			Find(v).Error
	})
	if err != nil {
		return err
	}
	for _, device := range devices {
		if item := index.Get(device.UserId); item != nil {
			item.UserDevices = append(item.UserDevices, Device{
				Id:                   device.Id,
				Identifier:           device.Identifier,
				IdentifierBlindIndex: device.IdentifierBlindIndex,
			})
		}
	}
	return nil
}

type userCacheRelationIndex struct {
	users   []*User
	ids     []int64
	byID    map[int64]int
	capHint int
}

func newUserCacheRelationIndex(users []*User) userCacheRelationIndex {
	index := userCacheRelationIndex{
		capHint: len(users),
	}
	for _, user := range users {
		index.Add(user)
	}
	return index
}

func (i *userCacheRelationIndex) Add(user *User) {
	if user == nil || user.Id <= 0 {
		return
	}
	if i.byID != nil {
		if _, ok := i.byID[user.Id]; ok {
			return
		}
		i.byID[user.Id] = len(i.users)
		i.users = append(i.users, user)
		i.ids = append(i.ids, user.Id)
		return
	}
	for _, existing := range i.users {
		if existing.Id == user.Id {
			return
		}
	}
	i.ensureRows()
	i.users = append(i.users, user)
	i.ids = append(i.ids, user.Id)
	if len(i.users) > smallUserCacheRelationLookupLimit {
		i.upgradeLookup()
	}
}

func (i *userCacheRelationIndex) ensureRows() {
	if i.users != nil {
		return
	}
	initialCap := userCacheRelationInitialCap(i.capHint)
	i.users = make([]*User, 0, initialCap)
	i.ids = make([]int64, 0, initialCap)
}

func (i *userCacheRelationIndex) upgradeLookup() {
	if i.byID != nil {
		return
	}
	nextCap := userCacheRelationLookupCap(len(i.users), i.capHint)
	if nextCap > cap(i.users) {
		users := make([]*User, len(i.users), nextCap)
		copy(users, i.users)
		i.users = users

		ids := make([]int64, len(i.ids), nextCap)
		copy(ids, i.ids)
		i.ids = ids
	}
	i.byID = make(map[int64]int, nextCap)
	for row, existing := range i.users {
		i.byID[existing.Id] = row
	}
}

func (i userCacheRelationIndex) IDs() []int64 {
	return i.ids
}

func (i userCacheRelationIndex) Get(userID int64) *User {
	if userID <= 0 {
		return nil
	}
	if i.byID != nil {
		row, ok := i.byID[userID]
		if !ok {
			return nil
		}
		return i.users[row]
	}
	for _, user := range i.users {
		if user.Id == userID {
			return user
		}
	}
	return nil
}

const smallUserCacheRelationLookupLimit = 8

func userCacheRelationInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallUserCacheRelationLookupLimit+1 {
		return smallUserCacheRelationLookupLimit + 1
	}
	return capHint
}

func userCacheRelationLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (m *defaultUserModel) FindOneForContext(ctx context.Context, id int64) (*User, error) {
	userContextKey := cachekeys.Int(cacheUserContextPrefix, id)
	var resp User
	err := m.QueryCtx(ctx, &resp, userContextKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Unscoped().
			Select(userContextSelectColumns).
			Where("id = ?", id).
			Preload("AuthMethods", func(db *gorm.DB) *gorm.DB {
				return db.Select(m.authMethodDisplayColumns(userAuthMethodContextColumns)).Order("id ASC")
			}).
			First(v).Error
	})
	return &resp, err
}

func (m *defaultUserModel) FindOneForBasicUpdate(ctx context.Context, id int64) (*User, error) {
	var resp User
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Select(userBasicUpdateSelectColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *defaultUserModel) FindOneForSubscribeAccess(ctx context.Context, id int64) (*User, error) {
	userSubscribeAccessKey := cachekeys.Int(cacheUserSubscribeAccessPrefix, id)
	var resp User
	err := m.QueryCtx(ctx, &resp, userSubscribeAccessKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Unscoped().
			Select(userSubscribeAccessSelectColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *defaultUserModel) FindOneForAdminCheck(ctx context.Context, id int64) (*User, error) {
	userAdminCheckKey := cachekeys.Int(cacheUserAdminCheckPrefix, id)
	var resp User
	err := m.QueryCtx(ctx, &resp, userAdminCheckKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Unscoped().
			Select(userAdminCheckSelectColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *defaultUserModel) FindOneForOrderActivation(ctx context.Context, id int64) (*User, error) {
	var resp User
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Select(userOrderActivationSelectColumns).
			Where("id = ?", id).
			Preload("AuthMethods", func(db *gorm.DB) *gorm.DB {
				return db.Select(userAuthMethodCacheColumns).
					Where("auth_type = ?", "telegram").
					Order("id ASC")
			}).
			First(v).Error
	})
	if err == nil {
		for index := range resp.AuthMethods {
			if err = m.openAuthMethod(&resp.AuthMethods[index]); err != nil {
				return nil, err
			}
		}
	}
	return &resp, err
}

func (m *defaultUserModel) FindOneForReferralPolicy(ctx context.Context, id int64) (*User, error) {
	var resp User
	err := m.QueryNoCacheCtx(ctx, &resp, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Select(userReferralPolicySelectColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *defaultUserModel) FindOneForLogin(ctx context.Context, id int64) (*User, error) {
	userLoginKey := cachekeys.Int(cacheUserLoginIdPrefix, id)
	var resp User
	err := m.QueryCtx(ctx, &resp, userLoginKey, func(conn *gorm.DB, v interface{}) error {
		return conn.Model(&User{}).
			Unscoped().
			Select(userLoginSelectColumns).
			Where("id = ?", id).
			First(v).Error
	})
	return &resp, err
}

func (m *defaultUserModel) Update(ctx context.Context, data *User, tx ...*gorm.DB) error {
	old, err := m.FindOne(ctx, data.Id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	err = m.ExecCtx(ctx, func(conn *gorm.DB) error {
		if len(tx) > 0 {
			conn = tx[0]
		}
		return conn.Save(data).Error
	}, m.getCacheKeys(old)...)
	return err
}

func (m *defaultUserModel) Delete(ctx context.Context, id int64, tx ...*gorm.DB) error {
	data, err := m.findOneUserForCache(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	// Use batch related cache cleaning, including a cache of all relevant data
	defer func() {
		if clearErr := m.BatchClearRelatedCache(ctx, data); clearErr != nil {
			// Record cache cleaning errors, but do not block deletion operations
			logger.Errorf("failed to clear related cache for user %d: %v", id, clearErr.Error())
		}
	}()

	return m.TransactCtx(ctx, func(db *gorm.DB) error {
		if len(tx) > 0 {
			db = tx[0]
		}
		// Soft deletion of user information without any processing of other information (Determine whether to allow login/subscription based on the user's deletion status)
		if err := db.Model(&User{}).Where("id = ?", id).Delete(&User{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (m *defaultUserModel) Transaction(ctx context.Context, fn func(db *gorm.DB) error) error {
	return m.TransactCtx(ctx, fn)
}
