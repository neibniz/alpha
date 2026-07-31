package xray

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strconv"
	"time"
)

type OnlineUsersUpdate struct {
	Mode       string  `json:"mode"`
	InstanceID string  `json:"instance_id"`
	Seq        int64   `json:"seq"`
	Active     []int64 `json:"active,omitempty"`
	Inactive   []int64 `json:"inactive,omitempty"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

type ActiveUserTracker struct {
	ttl             time.Duration
	instanceID      string
	seq             int64
	lastSeen        map[int64]time.Time
	reported        map[int64]struct{}
	pendingActive   map[int64]struct{}
	pendingInactive map[int64]struct{}
}

func NewActiveUserTracker(ttl time.Duration) *ActiveUserTracker {
	if ttl <= 0 {
		ttl = 3 * time.Minute
	}
	return &ActiveUserTracker{
		ttl:             ttl,
		instanceID:      newOnlineUsersInstanceID(),
		lastSeen:        map[int64]time.Time{},
		reported:        map[int64]struct{}{},
		pendingActive:   map[int64]struct{}{},
		pendingInactive: map[int64]struct{}{},
	}
}

func (t *ActiveUserTracker) Observe(ids []int64, now time.Time) {
	if t == nil {
		return
	}
	t.cleanup(now)
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		t.lastSeen[id] = now
		t.pendingActive[id] = struct{}{}
		delete(t.pendingInactive, id)
	}
}

func (t *ActiveUserTracker) Count(now time.Time) int64 {
	if t == nil {
		return 0
	}
	t.cleanup(now)
	return int64(len(t.lastSeen))
}

func (t *ActiveUserTracker) Delta(now time.Time) (OnlineUsersUpdate, bool) {
	if t == nil {
		return OnlineUsersUpdate{}, false
	}
	t.cleanup(now)
	active := sortedKeys(t.pendingActive)
	inactive := sortedKeys(t.pendingInactive)
	if len(active) == 0 && len(inactive) == 0 {
		return OnlineUsersUpdate{}, false
	}
	t.seq++
	return OnlineUsersUpdate{
		Mode:       "delta",
		InstanceID: t.instanceID,
		Seq:        t.seq,
		Active:     active,
		Inactive:   inactive,
		TTLSeconds: int64(t.ttl.Seconds()),
	}, true
}

func (t *ActiveUserTracker) Full(now time.Time) OnlineUsersUpdate {
	if t == nil {
		return OnlineUsersUpdate{}
	}
	t.cleanup(now)
	t.seq++
	return OnlineUsersUpdate{
		Mode:       "full",
		InstanceID: t.instanceID,
		Seq:        t.seq,
		Active:     sortedKeys(t.lastSeen),
		TTLSeconds: int64(t.ttl.Seconds()),
	}
}

func (t *ActiveUserTracker) MarkSent(update OnlineUsersUpdate) {
	if t == nil {
		return
	}
	if update.Mode == "full" {
		clear(t.reported)
		for _, id := range update.Active {
			t.reported[id] = struct{}{}
		}
		clear(t.pendingActive)
		clear(t.pendingInactive)
		return
	}
	for _, id := range update.Active {
		t.reported[id] = struct{}{}
		delete(t.pendingActive, id)
	}
	for _, id := range update.Inactive {
		delete(t.reported, id)
		delete(t.pendingInactive, id)
	}
}

func (t *ActiveUserTracker) cleanup(now time.Time) {
	for id, lastSeen := range t.lastSeen {
		if now.Sub(lastSeen) <= t.ttl {
			continue
		}
		delete(t.lastSeen, id)
		delete(t.pendingActive, id)
		if _, ok := t.reported[id]; ok {
			t.pendingInactive[id] = struct{}{}
		}
	}
}

func newOnlineUsersInstanceID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func sortedKeys[T any](values map[int64]T) []int64 {
	keys := make([]int64, 0, len(values))
	for id := range values {
		keys = append(keys, id)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}
