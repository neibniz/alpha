package device

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDeviceWebSocketRejectsCrossOriginBrowser(t *testing.T) {
	dm := &DeviceManager{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dm.AddDevice(w, r, "session", 7, "device", 1)
	}))
	t.Cleanup(server.Close)

	header := http.Header{}
	header.Set("Origin", "https://attacker.example")
	conn, response, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1), header)
	if conn != nil {
		_ = conn.Close()
	}
	if response != nil {
		_ = response.Body.Close()
	}
	if err == nil {
		t.Fatal("cross-origin websocket handshake should be rejected")
	}
}

func TestRemoveDeviceUsesDeviceIdentity(t *testing.T) {
	oldDevice := &Device{
		Session:      "old-session",
		DeviceID:     "phone",
		CreatedAt:    time.Now().Add(-time.Minute),
		LastPingTime: time.Now().Add(-time.Minute),
	}
	newDevice := &Device{
		Session:      "new-session",
		DeviceID:     "phone",
		CreatedAt:    time.Now(),
		LastPingTime: time.Now(),
	}
	dm := &DeviceManager{
		userDevices: map[int64]*userDeviceSet{7: newUserDeviceSet(oldDevice, newDevice)},
		totalOnline: 2,
	}

	dm.removeDevice(7, oldDevice)

	set := dm.userDevices[7]
	devices := set.ordered
	if len(devices) != 1 || devices[0] != newDevice {
		t.Fatalf("removeDevice removed the replacement device: %#v", devices)
	}
	if got := set.lookup("phone"); got != newDevice {
		t.Fatalf("device lookup points to %#v, want replacement device", got)
	}
	if got := atomic.LoadInt64(&dm.totalOnline); got != 1 {
		t.Fatalf("totalOnline = %d, want 1", got)
	}
}

func TestRemoveDeviceMissingKeepsDeviceSlice(t *testing.T) {
	now := time.Now()
	first := &Device{Session: "first", DeviceID: "first", CreatedAt: now, LastPingTime: now}
	second := &Device{Session: "second", DeviceID: "second", CreatedAt: now, LastPingTime: now}
	missing := &Device{Session: "missing", DeviceID: "missing", CreatedAt: now, LastPingTime: now}
	set := newUserDeviceSet(first, second)
	before := set.ordered
	dm := &DeviceManager{
		userDevices: map[int64]*userDeviceSet{8: set},
		totalOnline: 2,
		OnDeviceOffline: func(userID int64, deviceID, session string, createAt time.Time) {
			t.Fatalf("unexpected offline callback for %s", deviceID)
		},
	}

	dm.removeDevice(8, missing)

	after := dm.userDevices[8].ordered
	if len(after) != 2 || after[0] != first || after[1] != second {
		t.Fatalf("devices changed: %#v", after)
	}
	if &after[0] != &before[0] {
		t.Fatal("device slice was rebuilt even though no device was removed")
	}
	if got := atomic.LoadInt64(&dm.totalOnline); got != 2 {
		t.Fatalf("totalOnline = %d, want 2", got)
	}
}

func TestKickDeviceUpdatesTypedDeviceMap(t *testing.T) {
	first := &Device{Session: "a", DeviceID: "first", CreatedAt: time.Now(), LastPingTime: time.Now()}
	second := &Device{Session: "b", DeviceID: "second", CreatedAt: time.Now(), LastPingTime: time.Now()}
	kicked := make([]string, 0, 1)
	dm := &DeviceManager{
		userDevices: map[int64]*userDeviceSet{9: newUserDeviceSet(first, second)},
		totalOnline: 2,
		OnDeviceKicked: func(userID int64, deviceID, session string, operator Operator) {
			if userID != 9 || operator != Admin {
				t.Fatalf("unexpected kick callback: userID=%d operator=%d", userID, operator)
			}
			kicked = append(kicked, deviceID)
		},
	}

	dm.KickDevice(9, "first")

	set := dm.userDevices[9]
	devices := set.ordered
	if len(devices) != 1 || devices[0] != second {
		t.Fatalf("remaining devices = %#v, want second only", devices)
	}
	if got := set.lookup("first"); got != nil {
		t.Fatal("kicked device remained in device index")
	}
	if got := set.lookup("second"); got != second {
		t.Fatalf("device lookup points to %#v, want second", got)
	}
	if len(kicked) != 1 || kicked[0] != "first" {
		t.Fatalf("kicked callbacks = %#v, want [first]", kicked)
	}
	if got := atomic.LoadInt64(&dm.totalOnline); got != 1 {
		t.Fatalf("totalOnline = %d, want 1", got)
	}
}

func TestKickDeviceMissingKeepsDeviceSlice(t *testing.T) {
	now := time.Now()
	first := &Device{Session: "first", DeviceID: "first", CreatedAt: now, LastPingTime: now}
	second := &Device{Session: "second", DeviceID: "second", CreatedAt: now, LastPingTime: now}
	set := newUserDeviceSet(first, second)
	before := set.ordered
	dm := &DeviceManager{
		userDevices: map[int64]*userDeviceSet{10: set},
		totalOnline: 2,
		OnDeviceKicked: func(userID int64, deviceID, session string, operator Operator) {
			t.Fatalf("unexpected kick callback for %s", deviceID)
		},
	}

	dm.KickDevice(10, "missing")

	after := dm.userDevices[10].ordered
	if len(after) != 2 || after[0] != first || after[1] != second {
		t.Fatalf("devices changed: %#v", after)
	}
	if &after[0] != &before[0] {
		t.Fatal("device slice was rebuilt even though no device was kicked")
	}
	if got := atomic.LoadInt64(&dm.totalOnline); got != 2 {
		t.Fatalf("totalOnline = %d, want 2", got)
	}
}

func TestRemoveHeartbeatTimeoutDevices(t *testing.T) {
	now := time.Now()
	stale := &Device{Session: "stale", DeviceID: "stale", CreatedAt: now.Add(-time.Hour), LastPingTime: now.Add(-time.Minute)}
	active := &Device{Session: "active", DeviceID: "active", CreatedAt: now, LastPingTime: now}
	offline := make(chan string, 1)
	dm := &DeviceManager{
		userDevices:      map[int64]*userDeviceSet{11: newUserDeviceSet(stale, active)},
		totalOnline:      2,
		heartbeatTimeout: 30,
		OnDeviceOffline: func(userID int64, deviceID, session string, createAt time.Time) {
			if userID != 11 {
				t.Fatalf("unexpected offline userID = %d", userID)
			}
			offline <- deviceID
		},
	}

	dm.removeHeartbeatTimeoutDevices(now)

	set := dm.userDevices[11]
	devices := set.ordered
	if len(devices) != 1 || devices[0] != active {
		t.Fatalf("remaining devices = %#v, want active only", devices)
	}
	if got := set.lookup("stale"); got != nil {
		t.Fatal("stale device remained in device index")
	}
	if got := set.lookup("active"); got != active {
		t.Fatalf("device lookup points to %#v, want active", got)
	}
	if got := atomic.LoadInt64(&dm.totalOnline); got != 1 {
		t.Fatalf("totalOnline = %d, want 1", got)
	}
	select {
	case got := <-offline:
		if got != "stale" {
			t.Fatalf("offline callback = %q, want stale", got)
		}
	case <-time.After(time.Second):
		t.Fatal("offline callback was not called")
	}
}

func TestRemoveHeartbeatTimeoutDevicesNoTimeoutKeepsDeviceSlice(t *testing.T) {
	now := time.Now()
	first := &Device{Session: "first", DeviceID: "first", CreatedAt: now, LastPingTime: now}
	second := &Device{Session: "second", DeviceID: "second", CreatedAt: now, LastPingTime: now}
	set := newUserDeviceSet(first, second)
	before := set.ordered
	dm := &DeviceManager{
		userDevices:      map[int64]*userDeviceSet{12: set},
		totalOnline:      2,
		heartbeatTimeout: 30,
		OnDeviceOffline: func(userID int64, deviceID, session string, createAt time.Time) {
			t.Fatalf("unexpected offline callback for %s", deviceID)
		},
	}

	dm.removeHeartbeatTimeoutDevices(now)

	after := dm.userDevices[12].ordered
	if len(after) != 2 || after[0] != first || after[1] != second {
		t.Fatalf("devices changed: %#v", after)
	}
	if &after[0] != &before[0] {
		t.Fatal("device slice was rebuilt even though no device timed out")
	}
	if got := atomic.LoadInt64(&dm.totalOnline); got != 2 {
		t.Fatalf("totalOnline = %d, want 2", got)
	}
}

func TestUpdateHeartbeatUsesDeviceLookup(t *testing.T) {
	now := time.Now()
	first := &Device{Session: "a", DeviceID: "first", CreatedAt: now, LastPingTime: now.Add(-time.Minute)}
	second := &Device{Session: "b", DeviceID: "second", CreatedAt: now, LastPingTime: now.Add(-time.Minute)}
	dm := &DeviceManager{
		userDevices: map[int64]*userDeviceSet{12: newUserDeviceSet(first, second)},
		totalOnline: 2,
	}

	firstBefore := first.LastPingTime
	secondBefore := second.LastPingTime

	dm.UpdateHeartbeat(12, "second")

	if !first.LastPingTime.Equal(firstBefore) {
		t.Fatal("heartbeat updated the wrong device")
	}
	if !second.LastPingTime.After(secondBefore) {
		t.Fatal("heartbeat did not update the indexed device")
	}
}

func TestUserDeviceSetLookupFallsBackToOrderedDevices(t *testing.T) {
	device := &Device{Session: "a", DeviceID: "phone", CreatedAt: time.Now(), LastPingTime: time.Now()}
	set := &userDeviceSet{ordered: []*Device{device}}

	if got := set.lookup("phone"); got != device {
		t.Fatalf("lookup = %#v, want device from ordered list", got)
	}
	if got := set.lookup("missing"); got != nil {
		t.Fatalf("lookup missing = %#v, want nil", got)
	}
}

func TestUserDeviceSetSmallInputAvoidsMapIndex(t *testing.T) {
	first := &Device{Session: "a", DeviceID: "phone", CreatedAt: time.Now(), LastPingTime: time.Now()}
	second := &Device{Session: "b", DeviceID: "phone", CreatedAt: time.Now(), LastPingTime: time.Now()}
	set := newUserDeviceSet(first, second)

	if set.byID != nil {
		t.Fatal("small user device set should not allocate map index")
	}
	if got := set.lookup("phone"); got != second {
		t.Fatalf("duplicate device lookup = %#v, want newest device", got)
	}
}

func TestUserDeviceSetLargeInputBuildsMapIndex(t *testing.T) {
	devices := make([]*Device, 0, smallUserDeviceSetIndexThreshold+1)
	for i := 0; i <= smallUserDeviceSetIndexThreshold; i++ {
		devices = append(devices, &Device{
			Session:      "session",
			DeviceID:     "device-" + string(rune('a'+i)),
			CreatedAt:    time.Now(),
			LastPingTime: time.Now(),
		})
	}
	set := newUserDeviceSet(devices...)

	if set.byID == nil {
		t.Fatal("large user device set should build map index")
	}
	last := devices[len(devices)-1]
	if got := set.lookup(last.DeviceID); got != last {
		t.Fatalf("large device lookup = %#v, want last device", got)
	}
	if cap(set.ordered) != smallUserDeviceSetIndexThreshold+1 {
		t.Fatalf("large device ordered cap = %d, want %d", cap(set.ordered), smallUserDeviceSetIndexThreshold+1)
	}
}

func TestUserDeviceSetDoesNotPreallocateForNilInput(t *testing.T) {
	devices := make([]*Device, smallUserDeviceSetIndexThreshold+20)
	set := newUserDeviceSet(devices...)

	if set.byID != nil {
		t.Fatal("nil-only user device set should not allocate map index")
	}
	if len(set.ordered) != 0 || cap(set.ordered) != 0 {
		t.Fatalf("nil-only user device ordered len/cap = %d/%d, want 0/0", len(set.ordered), cap(set.ordered))
	}
}

func TestUserDeviceSetLargeInputUsesCapacityHintAfterIndexUpgrade(t *testing.T) {
	devices := make([]*Device, 0, smallUserDeviceSetIndexThreshold+20)
	for i := 0; i < smallUserDeviceSetIndexThreshold+20; i++ {
		devices = append(devices, &Device{
			Session:      "session",
			DeviceID:     "device-" + string(rune('a'+i)),
			CreatedAt:    time.Now(),
			LastPingTime: time.Now(),
		})
	}
	set := newUserDeviceSet(devices...)

	if set.byID == nil {
		t.Fatal("large user device set should allocate map index")
	}
	if cap(set.ordered) != smallUserDeviceSetIndexThreshold+20 {
		t.Fatalf("large device ordered cap = %d, want %d", cap(set.ordered), smallUserDeviceSetIndexThreshold+20)
	}
}

func TestUserDeviceSetBoundsSparseIndexCapacity(t *testing.T) {
	devices := make([]*Device, smallUserDeviceSetIndexThreshold+40)
	for i := 0; i <= smallUserDeviceSetIndexThreshold; i++ {
		devices[i] = &Device{
			Session:      "session",
			DeviceID:     "device-" + string(rune('a'+i)),
			CreatedAt:    time.Now(),
			LastPingTime: time.Now(),
		}
	}
	set := newUserDeviceSet(devices...)

	if set.byID == nil {
		t.Fatal("sparse large user device set should allocate map index")
	}
	if maxCap := len(set.ordered) * 2; cap(set.ordered) > maxCap {
		t.Fatalf("sparse device ordered cap = %d, want <= %d", cap(set.ordered), maxCap)
	}
	last := devices[smallUserDeviceSetIndexThreshold]
	if got := set.lookup(last.DeviceID); got != last {
		t.Fatalf("sparse device lookup = %#v, want last device", got)
	}
}
