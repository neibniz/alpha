package user

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/cachekeys"
	"github.com/neibniz/alpha/server/pkg/logger"
)

const (
	smallModelCacheKeySetThreshold = 8
	userModelCacheKeyHint          = 8
	subscribeModelCacheKeyHint     = 5
	deviceModelCacheKeyHint        = 2
	authMethodModelCacheKeyHint    = 7
)

var emptyModelCacheKeys = []string{}

type CacheKeyGenerator interface {
	GetCacheKeys() []string
}

type CacheManager interface {
	ClearCache(ctx context.Context, keys ...string) error
	ClearModelCache(ctx context.Context, models ...CacheKeyGenerator) error
}

type UserCacheManager struct {
	model *defaultUserModel
}

func NewUserCacheManager(model *defaultUserModel) *UserCacheManager {
	return &UserCacheManager{
		model: model,
	}
}

func (c *UserCacheManager) ClearCache(ctx context.Context, keys ...string) error {
	keys = uniqueCacheKeys(keys)
	if len(keys) == 0 {
		return nil
	}
	return c.model.CachedConn.DelCacheCtx(ctx, keys...)
}

func (c *UserCacheManager) ClearModelCache(ctx context.Context, models ...CacheKeyGenerator) error {
	keys := cacheKeysForModels(models...)
	if len(keys) == 0 {
		return nil
	}
	return c.model.CachedConn.DelCacheCtx(ctx, keys...)
}

func cacheKeysForModels(models ...CacheKeyGenerator) []string {
	if len(models) == 0 {
		return emptyModelCacheKeys
	}
	allKeys := newModelCacheKeyCollector(len(models) * 4)
	for _, model := range models {
		addModelCacheKeys(&allKeys, model)
	}
	if allKeys.Len() == 0 {
		return emptyModelCacheKeys
	}
	return allKeys.Keys()
}

func cacheKeysForTypedModels[T interface {
	CacheKeyGenerator
	comparable
}](capHint int, models ...T) []string {
	if len(models) == 0 {
		return emptyModelCacheKeys
	}
	var zero T
	allKeys := newModelCacheKeyCollector(capHint)
	for _, model := range models {
		if model == zero {
			continue
		}
		for _, key := range model.GetCacheKeys() {
			allKeys.Add(key)
		}
	}
	if allKeys.Len() == 0 {
		return emptyModelCacheKeys
	}
	return allKeys.Keys()
}

func clearTypedModelCache[T interface {
	CacheKeyGenerator
	comparable
}](ctx context.Context, m *defaultUserModel, capHint int, models ...T) error {
	keys := cacheKeysForTypedModels(capHint, models...)
	if len(keys) == 0 {
		return nil
	}
	return m.CachedConn.DelCacheCtx(ctx, keys...)
}

func addModelCacheKeys(collector *modelCacheKeyCollector, model CacheKeyGenerator) {
	if collector == nil || model == nil {
		return
	}
	for _, key := range model.GetCacheKeys() {
		collector.Add(key)
	}
}

type modelCacheKeyCollector struct {
	keys    []string
	seen    map[string]struct{}
	capHint int
}

func newModelCacheKeyCollector(capHint int) modelCacheKeyCollector {
	return modelCacheKeyCollector{capHint: capHint}
}

func (c *modelCacheKeyCollector) Add(key string) bool {
	if c == nil || key == "" {
		return false
	}
	if c.seen != nil {
		if _, ok := c.seen[key]; ok {
			return false
		}
		c.seen[key] = struct{}{}
		c.keys = append(c.keys, key)
		return true
	}
	for _, existing := range c.keys {
		if existing == key {
			return false
		}
	}
	c.ensureKeys()
	c.keys = append(c.keys, key)
	if len(c.keys) > smallModelCacheKeySetThreshold {
		c.upgradeLookup()
	}
	return true
}

func (c modelCacheKeyCollector) Len() int {
	return len(c.keys)
}

func (c modelCacheKeyCollector) Keys() []string {
	return c.keys
}

func (c *modelCacheKeyCollector) ensureKeys() {
	if c.keys != nil {
		return
	}
	c.keys = make([]string, 0, modelCacheKeyInitialCap(c.capHint))
}

func (c *modelCacheKeyCollector) upgradeLookup() {
	if c.seen != nil {
		return
	}
	nextCap := modelCacheKeyLookupCap(len(c.keys), c.capHint)
	if nextCap > cap(c.keys) {
		keys := make([]string, len(c.keys), nextCap)
		copy(keys, c.keys)
		c.keys = keys
	}
	c.seen = make(map[string]struct{}, nextCap)
	for _, existing := range c.keys {
		c.seen[existing] = struct{}{}
	}
}

func modelCacheKeyInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallModelCacheKeySetThreshold+1 {
		return smallModelCacheKeySetThreshold + 1
	}
	return capHint
}

func modelCacheKeyLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func appendUniqueModelCacheKey(keys []string, key string, seen map[string]struct{}) ([]string, map[string]struct{}) {
	collector := modelCacheKeyCollector{
		keys:    keys,
		seen:    seen,
		capHint: cap(keys),
	}
	collector.Add(key)
	return collector.keys, collector.seen
}

func (u *User) GetCacheKeys() []string {
	if u == nil {
		return []string{}
	}
	keys := make([]string, 0, userModelCacheKeyHint)
	keys = append(keys,
		cachekeys.Int(cacheUserIdPrefix, u.Id),
		cachekeys.Int(cacheUserContextPrefix, u.Id),
		cachekeys.Int(cacheUserLoginIdPrefix, u.Id),
		cachekeys.Int(cacheUserSubscribeAccessPrefix, u.Id),
		cachekeys.Int(cacheUserAdminCheckPrefix, u.Id),
	)

	for _, auth := range u.AuthMethods {
		if auth.AuthType == "email" {
			identifier := auth.AuthIdentifier
			if auth.AuthIdentifierBlindIndex != "" {
				identifier = auth.AuthIdentifierBlindIndex
			}
			keys = append(keys, cachekeys.String(cacheUserEmailPrefix, identifier))
			keys = append(keys, cachekeys.String(cacheUserEmailLookupPrefix, identifier))
			keys = append(keys, cachekeys.String(cacheUserLoginEmailPrefix, identifier))
			break
		}
	}
	return keys
}

func (s *Subscribe) GetCacheKeys() []string {
	if s == nil {
		return []string{}
	}
	keys := make([]string, 0, subscribeModelCacheKeyHint)

	if s.Token != "" {
		keys = append(keys, cachekeys.String(cacheUserSubscribeTokenPrefix, s.Token))
		keys = append(keys, cachekeys.String(cacheUserSubscribeRuntimeTokenPrefix, s.Token))
		keys = append(keys, cachekeys.String(cacheUserSubscribeExpiryTokenPrefix, s.Token))
	}
	if s.UserId != 0 {
		keys = append(keys, cachekeys.Int(cacheUserSubscribeUserPrefix, s.UserId))
	}
	if s.Id != 0 {
		keys = append(keys, cachekeys.Int(cacheUserSubscribeIdPrefix, s.Id))
	}
	return keys
}

func (d *Device) GetCacheKeys() []string {
	if d == nil {
		return []string{}
	}
	keys := make([]string, 0, deviceModelCacheKeyHint)

	if d.Id != 0 {
		keys = append(keys, cachekeys.Int(cacheUserDeviceIdPrefix, d.Id))
	}
	identifier := d.Identifier
	if d.IdentifierBlindIndex != "" {
		identifier = d.IdentifierBlindIndex
	}
	if identifier != "" {
		keys = append(keys, cachekeys.String(cacheUserDeviceNumberPrefix, identifier))
	}
	return keys
}

func (a *AuthMethods) GetCacheKeys() []string {
	if a == nil {
		return []string{}
	}
	keys := make([]string, 0, authMethodModelCacheKeyHint)

	if a.UserId != 0 {
		keys = append(keys, cachekeys.Int(cacheUserIdPrefix, a.UserId))
		keys = append(keys, cachekeys.Int(cacheUserContextPrefix, a.UserId))
		keys = append(keys, cachekeys.Int(cacheUserLoginIdPrefix, a.UserId))
		keys = append(keys, cachekeys.Int(cacheUserSubscribeAccessPrefix, a.UserId))
	}
	if a.AuthType == "email" && a.AuthIdentifier != "" {
		identifier := a.AuthIdentifier
		if a.AuthIdentifierBlindIndex != "" {
			identifier = a.AuthIdentifierBlindIndex
		}
		keys = append(keys, cachekeys.String(cacheUserEmailPrefix, identifier))
		keys = append(keys, cachekeys.String(cacheUserEmailLookupPrefix, identifier))
		keys = append(keys, cachekeys.String(cacheUserLoginEmailPrefix, identifier))
	}
	return keys
}

func (m *defaultUserModel) GetCacheManager() *UserCacheManager {
	return NewUserCacheManager(m)
}

func (m *defaultUserModel) ClearUserCache(ctx context.Context, users ...*User) error {
	return clearTypedModelCache(ctx, m, len(users)*userModelCacheKeyHint, users...)
}

func (m *defaultUserModel) ClearSubscribeCacheByModels(ctx context.Context, subscribes ...*Subscribe) error {
	return clearTypedModelCache(ctx, m, len(subscribes)*subscribeModelCacheKeyHint, subscribes...)
}

func (m *defaultUserModel) ClearDeviceCache(ctx context.Context, devices ...*Device) error {
	return clearTypedModelCache(ctx, m, len(devices)*deviceModelCacheKeyHint, devices...)
}

func (m *defaultUserModel) ClearAuthMethodCache(ctx context.Context, authMethods ...*AuthMethods) error {
	return clearTypedModelCache(ctx, m, len(authMethods)*authMethodModelCacheKeyHint, authMethods...)
}

func (m *defaultUserModel) BatchClearRelatedCache(ctx context.Context, user *User) error {
	if user == nil {
		return nil
	}

	cacheManager := m.GetCacheManager()

	allModels := make([]CacheKeyGenerator, 0, 1+len(user.AuthMethods)+len(user.UserDevices))
	allModels = append(allModels, user)

	for _, auth := range user.AuthMethods {
		allModels = append(allModels, &auth)
	}

	for _, device := range user.UserDevices {
		allModels = append(allModels, &device)
	}

	subscribes, err := m.QueryUserSubscribeRuntime(ctx, user.Id)
	if err != nil {
		logger.Errorf("failed to query user subscribes for cache clearing: %v", err)
	} else {
		for _, sub := range subscribes {
			subModel := &Subscribe{
				Id:          sub.Id,
				UserId:      sub.UserId,
				Token:       sub.Token,
				SubscribeId: sub.SubscribeId,
			}
			allModels = append(allModels, subModel)
		}
	}

	return cacheManager.ClearModelCache(ctx, allModels...)
}

func (m *defaultUserModel) CacheInvalidationHandler(ctx context.Context, operation string, modelType string, model interface{}) error {
	switch operation {
	case "create", "update", "delete":
		switch modelType {
		case "user":
			if user, ok := model.(*User); ok {
				return m.BatchClearRelatedCache(ctx, user)
			}
		case "subscribe":
			if subscribe, ok := model.(*Subscribe); ok {
				return m.ClearSubscribeCacheByModels(ctx, subscribe)
			}
		case "device":
			if device, ok := model.(*Device); ok {
				return m.ClearDeviceCache(ctx, device)
			}
		case "authmethod":
			if authMethod, ok := model.(*AuthMethods); ok {
				return m.ClearAuthMethodCache(ctx, authMethod)
			}
		}
	}
	return nil
}

func (m *customUserModel) GetRelatedCacheKeys(ctx context.Context, modelType string, modelId int64) ([]string, error) {
	var keys []string

	switch modelType {
	case "user":
		user, err := m.findOneUserForCache(ctx, modelId)
		if err != nil {
			return nil, err
		}
		userKeys := user.GetCacheKeys()

		auths, err := m.FindUserAuthMethods(ctx, modelId)
		authKeysCap := 0
		if err == nil {
			authKeysCap = len(auths) * 7
		}

		subscribes, err := m.QueryUserSubscribeRuntime(ctx, modelId)
		subscribeKeysCap := 0
		if err == nil {
			subscribeKeysCap = len(subscribes) * 5
		}

		keys = make([]string, 0, len(userKeys)+authKeysCap+subscribeKeysCap)
		keys = append(keys, userKeys...)
		if authKeysCap > 0 {
			for _, auth := range auths {
				keys = append(keys, auth.GetCacheKeys()...)
			}
		}
		if subscribeKeysCap > 0 {
			for _, sub := range subscribes {
				subModel := &Subscribe{
					Id:          sub.Id,
					UserId:      sub.UserId,
					Token:       sub.Token,
					SubscribeId: sub.SubscribeId,
				}
				keys = append(keys, subModel.GetCacheKeys()...)
			}
		}

	case "subscribe":
		subscribe, err := m.findOneSubscribeForCache(ctx, modelId)
		if err != nil {
			return nil, err
		}
		keys = append(keys, subscribe.GetCacheKeys()...)

	case "device":
		device, err := m.findOneDeviceForCache(ctx, modelId)
		if err != nil {
			return nil, err
		}
		keys = append(keys, device.GetCacheKeys()...)
	}

	return uniqueCacheKeys(keys), nil
}
