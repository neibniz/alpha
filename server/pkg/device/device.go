package device

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/neibniz/alpha/server/internal/privacy"

	"github.com/gorilla/websocket"
)

type Operator int

const (
	MaxDevices Operator = iota
	Admin
	SubscribeUpdate = "subscribe_update"
)

// Device represents a device structure
type Device struct {
	Session      string
	DeviceID     string
	Conn         *websocket.Conn
	CreatedAt    time.Time
	LastPingTime time.Time
	writeMu      sync.Mutex
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{}

// DeviceManager manages devices
type DeviceManager struct {
	devicesMu        sync.RWMutex
	userDevices      map[int64]*userDeviceSet
	totalOnline      int64 // total online devices
	heartbeatTimeout int   // heartbeat timeout (seconds)
	checkInterval    int   // heartbeat check interval (seconds)

	// event callbacks
	OnDeviceOnline  func(userID int64, deviceID, session string)
	OnDeviceOffline func(userID int64, deviceID, session string, createAt time.Time)
	OnDeviceKicked  func(userID int64, deviceID, session string, operator Operator)
	OnMessage       func(userID int64, deviceID, session string, message string)
}

type userDeviceSet struct {
	ordered []*Device
	byID    map[string]*Device
	capHint int
}

const smallUserDeviceSetIndexThreshold = 8

func newUserDeviceSet(devices ...*Device) *userDeviceSet {
	set := &userDeviceSet{capHint: len(devices)}
	for _, device := range devices {
		if device == nil {
			continue
		}
		set.ensureOrdered()
		set.ordered = append(set.ordered, device)
	}
	set.rebuildIndex()
	return set
}

func (set *userDeviceSet) ensureIndex() {
	if set == nil || set.byID != nil {
		return
	}
	if len(set.ordered) <= smallUserDeviceSetIndexThreshold {
		return
	}
	set.rebuildIndex()
}

func (set *userDeviceSet) rebuildIndex() {
	if set == nil {
		return
	}
	if len(set.ordered) <= smallUserDeviceSetIndexThreshold {
		set.byID = nil
		return
	}
	nextCap := userDeviceSetLookupCap(len(set.ordered), set.capHint)
	if nextCap != cap(set.ordered) {
		ordered := make([]*Device, len(set.ordered), nextCap)
		copy(ordered, set.ordered)
		set.ordered = ordered
	}
	set.byID = make(map[string]*Device, nextCap)
	for _, device := range set.ordered {
		if device != nil {
			set.byID[device.DeviceID] = device
		}
	}
}

func (set *userDeviceSet) replaceOrdered(devices []*Device) {
	set.capHint = len(devices)
	set.ordered = devices
	set.rebuildIndex()
}

func (set *userDeviceSet) ensureOrdered() {
	if set == nil || set.ordered != nil {
		return
	}
	set.ordered = make([]*Device, 0, userDeviceSetInitialCap(set.capHint))
}

func userDeviceSetInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallUserDeviceSetIndexThreshold+1 {
		return smallUserDeviceSetIndexThreshold + 1
	}
	return capHint
}

func userDeviceSetLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func (set *userDeviceSet) lookup(deviceID string) *Device {
	if set == nil {
		return nil
	}
	if set.byID != nil {
		return set.byID[deviceID]
	}
	for i := len(set.ordered) - 1; i >= 0; i-- {
		device := set.ordered[i]
		if device != nil && device.DeviceID == deviceID {
			return device
		}
	}
	return nil
}

func (dm *DeviceManager) ensureDeviceMapLocked() {
	if dm.userDevices == nil {
		dm.userDevices = make(map[int64]*userDeviceSet)
	}
}

func (dm *DeviceManager) deviceSetLocked(userID int64) *userDeviceSet {
	if dm.userDevices == nil {
		return nil
	}
	set := dm.userDevices[userID]
	if set != nil {
		set.ensureIndex()
	}
	return set
}

func (dm *DeviceManager) ensureDeviceSetLocked(userID int64) *userDeviceSet {
	dm.ensureDeviceMapLocked()
	set := dm.userDevices[userID]
	if set == nil {
		set = newUserDeviceSet()
		dm.userDevices[userID] = set
		return set
	}
	set.ensureIndex()
	return set
}

func closeDeviceConn(device *Device) {
	if device != nil && device.Conn != nil {
		_ = device.Conn.Close()
	}
}

func writeDeviceMessage(device *Device, messageType int, payload []byte) error {
	if device == nil || device.Conn == nil {
		return fmt.Errorf("device connection is nil")
	}
	device.writeMu.Lock()
	defer device.writeMu.Unlock()
	return device.Conn.WriteMessage(messageType, payload)
}

// Listen to WebSocket data
func (dm *DeviceManager) listenToDevice(userID int64, device *Device) {
	defer func() {
		dm.removeDevice(userID, device) // remove device when disconnected
	}()

	for {
		_, msg, err := device.Conn.ReadMessage()
		if err != nil {
			zap.S().Infof("Device %s (User %d) disconnected: %v", privacy.MaskIdentifier(device.DeviceID), userID, err)
			break
		}

		message := string(msg)
		if message == "ping" || message == "heartbeat" {
			dm.UpdateHeartbeat(userID, device.DeviceID)
			continue
		}

		// Trigger message callback
		if dm.OnMessage != nil {
			go dm.OnMessage(userID, device.DeviceID, device.Session, message)
		}
	}
}

// UpdateHeartbeat updates device heartbeat
func (dm *DeviceManager) UpdateHeartbeat(userID int64, deviceID string) {
	var target *Device

	dm.devicesMu.Lock()
	if set := dm.deviceSetLocked(userID); set != nil {
		target = set.lookup(deviceID)
	}
	if target != nil {
		target.LastPingTime = time.Now()
	}
	dm.devicesMu.Unlock()

	if target != nil {
		if err := writeDeviceMessage(target, websocket.TextMessage, []byte("ping")); err != nil {
			zap.S().Infof("✅ Heartbeat updated: Device %s (User %d) err: %s", privacy.MaskIdentifier(deviceID), userID, err.Error())
		}
	}
}

// AddDevice **Add: Device connects WebSocket and is added to the manager**
func (dm *DeviceManager) AddDevice(w http.ResponseWriter, r *http.Request, session string, userID int64, deviceID string, maxDevices int) {
	// **Upgrade WebSocket connection**
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.S().Infof("WebSocket upgrade failed: %v", err)
		return
	}

	newDevice := &Device{
		Session:      session,
		DeviceID:     deviceID,
		Conn:         conn,
		CreatedAt:    time.Now(),
		LastPingTime: time.Now(),
	}

	//不限制设备数量
	if maxDevices < 1 {
		maxDevices = 99
	}

	var kicked []*Device
	var replaced []*Device

	dm.devicesMu.Lock()
	set := dm.ensureDeviceSetLocked(userID)

	devices := set.ordered
	activeDevices := devices
	if set.lookup(deviceID) != nil {
		activeDevices = make([]*Device, 0, len(devices)+1)
		for _, d := range devices {
			if d.DeviceID == deviceID {
				replaced = append(replaced, d)
				continue
			}
			activeDevices = append(activeDevices, d)
		}
	}

	// **If exceeding the limit, kick out the earliest device**
	if len(replaced) == 0 && len(activeDevices) >= maxDevices {
		kicked = append(kicked, activeDevices[0])
		activeDevices = activeDevices[1:]
	}

	// Add new device
	activeDevices = append(activeDevices, newDevice)
	set.replaceOrdered(activeDevices)
	atomic.AddInt64(&dm.totalOnline, int64(1-len(replaced)-len(kicked)))
	dm.devicesMu.Unlock()

	for _, d := range replaced {
		closeDeviceConn(d)
	}
	for _, d := range kicked {
		if dm.OnDeviceKicked != nil {
			dm.OnDeviceKicked(userID, d.DeviceID, d.Session, MaxDevices)
		}
		closeDeviceConn(d)
	}

	// Trigger online event
	if dm.OnDeviceOnline != nil {
		go dm.OnDeviceOnline(userID, deviceID, session)
	}

	// Start listening
	go dm.listenToDevice(userID, newDevice)
}

// removeDevice removes a device
func (dm *DeviceManager) removeDevice(userID int64, target *Device) {
	if target == nil {
		return
	}

	var removed *Device

	dm.devicesMu.Lock()
	set := dm.deviceSetLocked(userID)
	if set == nil {
		dm.devicesMu.Unlock()
		return
	}
	devices := set.ordered
	var activeDevices []*Device
	for index, d := range devices {
		if d == target {
			if activeDevices == nil {
				activeDevices = make([]*Device, 0, len(devices)-1)
				activeDevices = append(activeDevices, devices[:index]...)
			}
			removed = d
			continue
		}
		if activeDevices != nil {
			activeDevices = append(activeDevices, d)
		}
	}
	if removed != nil {
		if len(activeDevices) == 0 {
			delete(dm.userDevices, userID)
		} else {
			set.replaceOrdered(activeDevices)
		}
		atomic.AddInt64(&dm.totalOnline, -1)
	}
	dm.devicesMu.Unlock()

	if removed == nil {
		return
	}

	closeDeviceConn(removed)
	if dm.OnDeviceOffline != nil {
		go dm.OnDeviceOffline(userID, removed.DeviceID, removed.Session, removed.CreatedAt)
	}
}

// KickDevice kicks a device (supports individual device or entire user)
func (dm *DeviceManager) KickDevice(userID int64, deviceID string) {
	var kicked []*Device

	dm.devicesMu.Lock()
	set := dm.deviceSetLocked(userID)
	if set == nil || len(set.ordered) == 0 {
		dm.devicesMu.Unlock()
		zap.S().Infof("⚠️ User %d has no online devices, unable to kick out", userID)
		return
	}

	devices := set.ordered
	var activeDevices []*Device

	if deviceID == "" {
		kicked = append(kicked, devices...)
	} else if set.lookup(deviceID) != nil {
		activeDevices = make([]*Device, 0, len(devices)-1)
		for _, d := range devices {
			if d.DeviceID == deviceID {
				kicked = append(kicked, d)
			} else {
				activeDevices = append(activeDevices, d)
			}
		}
	} else {
		dm.devicesMu.Unlock()
		return
	}

	// Update user's device mapping
	if len(activeDevices) == 0 {
		delete(dm.userDevices, userID)
	} else {
		set.replaceOrdered(activeDevices)
	}
	atomic.AddInt64(&dm.totalOnline, -int64(len(kicked)))
	dm.devicesMu.Unlock()

	for _, d := range kicked {
		// Trigger kick event callback
		if dm.OnDeviceKicked != nil {
			dm.OnDeviceKicked(userID, d.DeviceID, d.Session, Admin)
		}
		// Close WebSocket connection
		closeDeviceConn(d)
		zap.S().Infof("❌ Device %s (User %d) kicked out", privacy.MaskIdentifier(d.DeviceID), userID)
	}
}

// StartHeartbeatCheck periodically checks for heartbeat timeout devices
func (dm *DeviceManager) StartHeartbeatCheck() {
	ticker := time.NewTicker(time.Duration(dm.checkInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		dm.removeHeartbeatTimeoutDevices(time.Now())
		//zap.S().Infof("Total online devices: %d\n", dm.totalOnline)
	}
}

func (dm *DeviceManager) removeHeartbeatTimeoutDevices(now time.Time) {
	type offlineDevice struct {
		userID int64
		device *Device
	}

	timeout := time.Duration(dm.heartbeatTimeout) * time.Second
	var offline []offlineDevice

	dm.devicesMu.Lock()
	for uid, set := range dm.userDevices {
		if set == nil {
			delete(dm.userDevices, uid)
			continue
		}
		set.ensureIndex()
		devices := set.ordered
		var activeDevices []*Device
		for index, d := range devices {
			if now.Sub(d.LastPingTime) > timeout {
				if activeDevices == nil {
					activeDevices = make([]*Device, 0, len(devices)-1)
					activeDevices = append(activeDevices, devices[:index]...)
				}
				offline = append(offline, offlineDevice{userID: uid, device: d})
				continue
			}
			if activeDevices != nil {
				activeDevices = append(activeDevices, d)
			}
		}

		if activeDevices == nil {
			continue
		}
		if len(activeDevices) == 0 {
			delete(dm.userDevices, uid)
		} else {
			set.replaceOrdered(activeDevices)
		}
	}
	atomic.AddInt64(&dm.totalOnline, -int64(len(offline)))
	dm.devicesMu.Unlock()

	for _, item := range offline {
		zap.S().Infof("⚠️ Device %s (User %d) heartbeat timeout, removed", privacy.MaskIdentifier(item.device.DeviceID), item.userID)
		closeDeviceConn(item.device)

		if dm.OnDeviceOffline != nil {
			go dm.OnDeviceOffline(item.userID, item.device.DeviceID, item.device.Session, item.device.CreatedAt)
		}
	}
}

// NewDeviceManager creates a new device manager
func NewDeviceManager(heartbeatTimeout, checkInterval int) *DeviceManager {
	dm := &DeviceManager{
		heartbeatTimeout: heartbeatTimeout,
		checkInterval:    checkInterval,
		userDevices:      make(map[int64]*userDeviceSet),
	}
	go dm.StartHeartbeatCheck()
	return dm
}

// SendToDevice sends a message to a specific device
func (dm *DeviceManager) SendToDevice(userID int64, deviceID string, message string) error {
	var target *Device
	var devices []*Device

	dm.devicesMu.RLock()
	if set := dm.userDevices[userID]; set != nil {
		if deviceID == "" {
			devices = append([]*Device(nil), set.ordered...)
		} else {
			target = set.lookup(deviceID)
		}
	}
	dm.devicesMu.RUnlock()

	if deviceID == "" {
		if len(devices) == 0 {
			return fmt.Errorf("device %s (User %d) is offline", privacy.MaskIdentifier(deviceID), userID)
		}
		for _, d := range devices {
			if err := writeDeviceMessage(d, websocket.TextMessage, []byte(message)); err != nil {
				return err
			}
		}
		return nil
	}

	if target == nil {
		return fmt.Errorf("device %s (User %d) is offline", privacy.MaskIdentifier(deviceID), userID)
	}
	return writeDeviceMessage(target, websocket.TextMessage, []byte(message))
}

// Broadcast sends a message to all devices
func (dm *DeviceManager) Broadcast(message string) {
	go func(message string) {
		dm.devicesMu.RLock()
		devices := make([]*Device, 0)
		for _, set := range dm.userDevices {
			if set != nil {
				devices = append(devices, set.ordered...)
			}
		}
		dm.devicesMu.RUnlock()

		for _, d := range devices {
			_ = writeDeviceMessage(d, websocket.TextMessage, []byte(message))
		}
	}(message)

}

// Gracefully shut down all WebSocket connections
func (dm *DeviceManager) Shutdown(ctx context.Context) {
	<-ctx.Done()
	zap.S().Infof("🔴 Shutting down all WebSocket connections...")

	type onlineDevice struct {
		userID int64
		device *Device
	}
	devices := make([]onlineDevice, 0)

	dm.devicesMu.Lock()
	for uid, set := range dm.userDevices {
		if set == nil {
			continue
		}
		for _, d := range set.ordered {
			devices = append(devices, onlineDevice{userID: uid, device: d})
		}
	}
	dm.userDevices = make(map[int64]*userDeviceSet)
	atomic.StoreInt64(&dm.totalOnline, 0)
	dm.devicesMu.Unlock()

	for _, item := range devices {
		closeDeviceConn(item.device)
		zap.S().Infof("✅ Closed device %s (User %d)", privacy.MaskIdentifier(item.device.DeviceID), item.userID)
	}
}
