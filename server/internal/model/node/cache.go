package node

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

type (
	customCacheLogicModel interface {
		StatusCache(ctx context.Context, serverId int64) (Status, error)
		UpdateStatusCache(ctx context.Context, serverId int64, status *Status) error
		OnlineUserSubscribe(ctx context.Context, serverId int64, protocol string) (OnlineUserSubscribe, error)
		UpdateOnlineUserSubscribe(ctx context.Context, serverId int64, protocol string, subscribe OnlineUserSubscribe) error
		OnlineUserSubscribeGlobal(ctx context.Context) (int64, error)
		UpdateOnlineUserSubscribeGlobal(ctx context.Context, subscribe OnlineUserSubscribe) error
		UpdateOnlineUsers(ctx context.Context, serverId int64, update OnlineUsersUpdate) (int64, error)
	}

	Status struct {
		Cpu       float64 `json:"cpu"`
		Mem       float64 `json:"mem"`
		Disk      float64 `json:"disk"`
		UpdatedAt int64   `json:"updated_at"`
	}

	OnlineUserSubscribe map[int64][]string

	OnlineUsersUpdate struct {
		Mode       string
		InstanceID string
		Seq        int64
		Active     []int64
		Inactive   []int64
		TTLSeconds int64
	}
)

// Marshal  to json string
func (s *Status) Marshal() string {
	if s == nil {
		return "{}"
	}
	data, _ := json.Marshal(s)
	return string(data)
}

// Unmarshal from json string
func (s *Status) Unmarshal(data string) error {
	raw := strings.TrimSpace(data)
	if raw == "" || raw == "null" {
		*s = Status{}
		return nil
	}
	return json.Unmarshal([]byte(raw), s)
}

const (
	Expiry                                = 300 * time.Second              // Cache expiry time in seconds
	StatusCacheKey                        = "node:status:%d"               // Node status cache key format (Server ID and protocol) Example: node:status:1:shadowsocks
	OnlineUserCacheKeyWithSubscribe       = "node:online:subscribe:%d:%s"  // Online user subscribe cache key format (Server ID and protocol) Example: node:online:subscribe:1:shadowsocks
	OnlineUserSubscribeCacheKeyWithGlobal = "node:online:subscribe:global" // Online user global subscribe cache key
	OnlineUserCacheKeyWithServer          = "node:online:server:%d"        // Online user cache key format (Server ID)
	OnlineUserSeqCacheKeyWithServer       = "node:online:server:%d:seq"    // Online user sequence key format (Server ID)
	onlineUserMinTTLSeconds               = 30
	onlineUserDefaultTTLSeconds           = 180
	onlineUserMaxTTLSeconds               = 3600
	onlineUserMemberLinearLimit           = 8
)

var updateOnlineUsersScript = redis.NewScript(`
local serverKey = KEYS[1]
local seqKey = KEYS[2]
local now = tonumber(ARGV[1])
local ttlSeconds = tonumber(ARGV[2])
local expireAt = tonumber(ARGV[3])
local mode = ARGV[4]
local instanceID = ARGV[5]
local seq = tonumber(ARGV[6])
local seqValue = ARGV[7]
local activeCount = tonumber(ARGV[8])
local cursor = 9

redis.call("ZREMRANGEBYSCORE", serverKey, "-inf", tostring(now))

if seq > 0 then
	local current = redis.call("GET", seqKey)
	local currentInstanceID = ""
	local currentSeq = 0
	if current then
		local sep = string.find(current, ":", 1, true)
		if sep then
			currentInstanceID = string.sub(current, 1, sep - 1)
			currentSeq = tonumber(string.sub(current, sep + 1)) or 0
		else
			currentSeq = tonumber(current) or 0
		end
	end

	local skip = false
	if mode == "full" and instanceID ~= "" and currentInstanceID ~= instanceID then
		skip = false
	elseif instanceID ~= "" and currentInstanceID ~= "" and instanceID ~= currentInstanceID then
		skip = true
	elseif currentSeq >= seq then
		skip = true
	end

	if skip then
		return redis.call("ZCARD", serverKey)
	end
end

if mode == "full" then
	redis.call("DEL", serverKey)
end

for i = 1, activeCount do
	redis.call("ZADD", serverKey, expireAt, ARGV[cursor])
	cursor = cursor + 1
end

local inactiveCount = tonumber(ARGV[cursor])
cursor = cursor + 1
for i = 1, inactiveCount do
	redis.call("ZREM", serverKey, ARGV[cursor])
	cursor = cursor + 1
end

if seq > 0 then
	redis.call("SET", seqKey, seqValue, "EX", ttlSeconds * 2)
end
redis.call("EXPIRE", serverKey, ttlSeconds * 2)
return redis.call("ZCARD", serverKey)
`)

func statusCacheKey(serverID int64) string {
	return "node:status:" + strconv.FormatInt(serverID, 10)
}

func onlineUserSubscribeCacheKey(serverID int64, protocol string) string {
	return "node:online:subscribe:" + strconv.FormatInt(serverID, 10) + ":" + protocol
}

func onlineUserServerCacheKey(serverID int64) string {
	return "node:online:server:" + strconv.FormatInt(serverID, 10)
}

func onlineUserServerSeqCacheKey(serverID int64) string {
	return "node:online:server:" + strconv.FormatInt(serverID, 10) + ":seq"
}

func onlineUserScore(value int64) string {
	return strconv.FormatInt(value, 10)
}

// UpdateStatusCache Update server status to cache
func (m *customServerModel) UpdateStatusCache(ctx context.Context, serverId int64, status *Status) error {
	key := statusCacheKey(serverId)
	return m.Cache.Set(ctx, key, status.Marshal(), Expiry).Err()

}

// DeleteStatusCache Delete server status from cache
func (m *customServerModel) DeleteStatusCache(ctx context.Context, serverId int64) error {
	key := statusCacheKey(serverId)
	return m.Cache.Del(ctx, key).Err()
}

// StatusCache Get server status from cache
func (m *customServerModel) StatusCache(ctx context.Context, serverId int64) (Status, error) {
	var status Status
	key := statusCacheKey(serverId)

	result, err := m.Cache.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return status, nil
		}
		return status, err
	}
	if result == "" {
		return status, nil
	}
	err = status.Unmarshal(result)
	return status, err
}

// OnlineUserSubscribe Get online user subscribe
func (m *customServerModel) OnlineUserSubscribe(ctx context.Context, serverId int64, protocol string) (OnlineUserSubscribe, error) {
	key := onlineUserSubscribeCacheKey(serverId, protocol)
	result, err := m.Cache.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return OnlineUserSubscribe{}, nil
		}
		return nil, err
	}
	if result == "" {
		return OnlineUserSubscribe{}, nil
	}
	var subscribe OnlineUserSubscribe
	err = json.Unmarshal([]byte(result), &subscribe)
	return subscribe, err
}

// UpdateOnlineUserSubscribe Update online user subscribe
func (m *customServerModel) UpdateOnlineUserSubscribe(ctx context.Context, serverId int64, protocol string, subscribe OnlineUserSubscribe) error {
	key := onlineUserSubscribeCacheKey(serverId, protocol)
	data, err := json.Marshal(subscribe)
	if err != nil {
		return err
	}
	return m.Cache.Set(ctx, key, data, Expiry).Err()
}

// DeleteOnlineUserSubscribe Delete online user subscribe
func (m *customServerModel) DeleteOnlineUserSubscribe(ctx context.Context, serverId int64, protocol string) error {
	key := onlineUserSubscribeCacheKey(serverId, protocol)
	return m.Cache.Del(ctx, key).Err()
}

// OnlineUserSubscribeGlobal Get global online user subscribe count
func (m *customServerModel) OnlineUserSubscribeGlobal(ctx context.Context) (int64, error) {
	now := time.Now().Unix()
	// Clear expired data
	if err := m.Cache.ZRemRangeByScore(ctx, OnlineUserSubscribeCacheKeyWithGlobal, "-inf", onlineUserScore(now)).Err(); err != nil {
		return 0, err
	}
	return m.Cache.ZCard(ctx, OnlineUserSubscribeCacheKeyWithGlobal).Result()
}

// UpdateOnlineUserSubscribeGlobal Update global online user subscribe count
func (m *customServerModel) UpdateOnlineUserSubscribeGlobal(ctx context.Context, subscribe OnlineUserSubscribe) error {
	now := time.Now()
	expireTime := now.Add(5 * time.Minute).Unix() // set expire time 5 minutes later

	pipe := m.Cache.Pipeline()

	// Clear expired data
	pipe.ZRemRangeByScore(ctx, OnlineUserSubscribeCacheKeyWithGlobal, "-inf", onlineUserScore(now.Unix()))
	// Add or update each subscribe with new expire time
	for sub := range subscribe {
		// Use ZAdd to add or update the member with new score (expire time)
		pipe.ZAdd(ctx, OnlineUserSubscribeCacheKeyWithGlobal, redis.Z{
			Score:  float64(expireTime),
			Member: sub,
		})
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (m *customServerModel) UpdateOnlineUsers(ctx context.Context, serverId int64, update OnlineUsersUpdate) (int64, error) {
	serverKey := onlineUserServerCacheKey(serverId)
	seqKey := onlineUserServerSeqCacheKey(serverId)
	now := time.Now().Unix()
	ttlSeconds := normalizeOnlineUserTTL(update.TTLSeconds)
	expireAt := now + ttlSeconds
	activeMembers, inactiveMembers := normalizeOnlineUserMembers(update.Active, update.Inactive)

	args := make([]interface{}, 0, 9+len(activeMembers)+len(inactiveMembers))
	args = append(args,
		now,
		ttlSeconds,
		expireAt,
		update.Mode,
		update.InstanceID,
		update.Seq,
		formatOnlineUserSeq(update.InstanceID, update.Seq),
		0,
	)
	activeCountIndex := len(args) - 1
	for _, member := range activeMembers {
		args = append(args, member)
	}
	args[activeCountIndex] = len(args) - activeCountIndex - 1
	inactiveCountIndex := len(args)
	args = append(args, 0)
	for _, member := range inactiveMembers {
		args = append(args, member)
	}
	args[inactiveCountIndex] = len(args) - inactiveCountIndex - 1

	count, err := updateOnlineUsersScript.Run(ctx, m.Cache, []string{serverKey, seqKey}, args...).Int64()
	if err != nil {
		return 0, err
	}
	if err := m.syncOnlineUsersGlobal(ctx, now, expireAt, ttlSeconds, update.Mode, activeMembers, inactiveMembers); err != nil {
		return 0, err
	}
	return count, nil
}

func normalizeOnlineUserMembers(active, inactive []int64) ([]string, []string) {
	inactiveSet := newOnlineUserMemberSet(len(inactive))
	for _, id := range inactive {
		inactiveSet.Add(id)
	}

	activeSet := newOnlineUserMemberSet(len(active))
	for _, id := range active {
		if id <= 0 {
			continue
		}
		if inactiveSet.Contains(id) {
			continue
		}
		activeSet.Add(id)
	}
	return activeSet.StringMembers(), inactiveSet.StringMembers()
}

type onlineUserMemberSet struct {
	ids     []int64
	index   map[int64]struct{}
	capHint int
}

func newOnlineUserMemberSet(capHint int) onlineUserMemberSet {
	if capHint <= 0 {
		return onlineUserMemberSet{}
	}
	return onlineUserMemberSet{capHint: capHint}
}

func (s *onlineUserMemberSet) Add(id int64) bool {
	if id <= 0 {
		return false
	}
	if s.index != nil {
		if _, ok := s.index[id]; ok {
			return false
		}
		s.index[id] = struct{}{}
		s.ensureIDs()
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
	if len(s.ids) > onlineUserMemberLinearLimit {
		nextCap := onlineUserMemberLookupCap(len(s.ids), s.capHint)
		if nextCap > cap(s.ids) {
			ids := make([]int64, len(s.ids), nextCap)
			copy(ids, s.ids)
			s.ids = ids
		}
		s.index = make(map[int64]struct{}, nextCap)
		for _, existing := range s.ids {
			s.index[existing] = struct{}{}
		}
	}
	return true
}

func (s *onlineUserMemberSet) ensureIDs() {
	if s.ids != nil {
		return
	}
	capHint := s.capHint
	if capHint <= 0 {
		capHint = onlineUserMemberLinearLimit
	} else if capHint > onlineUserMemberLinearLimit+1 {
		capHint = onlineUserMemberLinearLimit + 1
	}
	s.ids = make([]int64, 0, capHint)
}

func onlineUserMemberLookupCap(uniqueCount, capHint int) int {
	if capHint <= uniqueCount {
		return uniqueCount
	}
	if bounded := uniqueCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (s onlineUserMemberSet) Contains(id int64) bool {
	if id <= 0 {
		return false
	}
	if s.index != nil {
		_, ok := s.index[id]
		return ok
	}
	for _, existing := range s.ids {
		if existing == id {
			return true
		}
	}
	return false
}

func (s onlineUserMemberSet) StringMembers() []string {
	if len(s.ids) == 0 {
		return []string{}
	}
	members := make([]string, 0, len(s.ids))
	for _, id := range s.ids {
		members = append(members, strconv.FormatInt(id, 10))
	}
	return members
}

func shouldSkipOnlineUsersUpdate(mode, instanceID, currentInstanceID string, seq, currentSeq int64) bool {
	if seq <= 0 {
		return false
	}
	if mode == "full" && instanceID != "" && currentInstanceID != instanceID {
		return false
	}
	if instanceID != "" && currentInstanceID != "" && instanceID != currentInstanceID {
		return true
	}
	return currentSeq >= seq
}

func parseOnlineUserSeq(value string) (string, int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, nil
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) == 2 {
		seq, err := strconv.ParseInt(parts[1], 10, 64)
		return parts[0], seq, err
	}
	seq, err := strconv.ParseInt(value, 10, 64)
	return "", seq, err
}

func formatOnlineUserSeq(instanceID string, seq int64) string {
	if instanceID == "" {
		return strconv.FormatInt(seq, 10)
	}
	return instanceID + ":" + strconv.FormatInt(seq, 10)
}

func (m *customServerModel) syncOnlineUsersGlobal(ctx context.Context, now, expireAt, ttlSeconds int64, mode string, activeMembers, inactiveMembers []string) error {
	if mode == "delta" && len(inactiveMembers) == 0 {
		pipe := m.Cache.Pipeline()
		cutoff := onlineUserScore(now)
		pipe.ZRemRangeByScore(ctx, OnlineUserSubscribeCacheKeyWithGlobal, "-inf", cutoff)
		for _, member := range activeMembers {
			pipe.ZAdd(ctx, OnlineUserSubscribeCacheKeyWithGlobal, redis.Z{
				Score:  float64(expireAt),
				Member: member,
			})
		}
		pipe.Expire(ctx, OnlineUserSubscribeCacheKeyWithGlobal, time.Duration(ttlSeconds*2)*time.Second)
		_, err := pipe.Exec(ctx)
		return err
	}
	return m.rebuildOnlineUsersGlobal(ctx, now)
}

func (m *customServerModel) rebuildOnlineUsersGlobal(ctx context.Context, now int64) error {
	var serverIDs []int64
	if err := m.WithContext(ctx).Model(&Server{}).Pluck("id", &serverIDs).Error; err != nil {
		return err
	}
	pipe := m.Cache.TxPipeline()
	cutoff := onlineUserScore(now)
	keys := make([]string, 0, len(serverIDs))
	for _, serverID := range serverIDs {
		if serverID <= 0 {
			continue
		}
		key := onlineUserServerCacheKey(serverID)
		keys = append(keys, key)
		pipe.ZRemRangeByScore(ctx, key, "-inf", cutoff)
	}
	if len(keys) == 0 {
		pipe.Del(ctx, OnlineUserSubscribeCacheKeyWithGlobal)
		_, err := pipe.Exec(ctx)
		return err
	}
	pipe.ZUnionStore(ctx, OnlineUserSubscribeCacheKeyWithGlobal, &redis.ZStore{
		Keys:      keys,
		Aggregate: "MAX",
	})
	pipe.ZRemRangeByScore(ctx, OnlineUserSubscribeCacheKeyWithGlobal, "-inf", cutoff)
	pipe.Expire(ctx, OnlineUserSubscribeCacheKeyWithGlobal, time.Duration(onlineUserMaxTTLSeconds*2)*time.Second)
	_, err := pipe.Exec(ctx)
	return err
}

func normalizeOnlineUserTTL(ttlSeconds int64) int64 {
	if ttlSeconds < onlineUserMinTTLSeconds {
		return onlineUserDefaultTTLSeconds
	}
	if ttlSeconds > onlineUserMaxTTLSeconds {
		return onlineUserMaxTTLSeconds
	}
	return ttlSeconds
}

// DeleteOnlineUserSubscribeGlobal Delete global online user subscribe count
func (m *customServerModel) DeleteOnlineUserSubscribeGlobal(ctx context.Context) error {
	return m.Cache.Del(ctx, OnlineUserSubscribeCacheKeyWithGlobal).Err()
}
