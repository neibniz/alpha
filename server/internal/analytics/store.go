package analytics

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrUnavailable = errors.New("analytics storage is unavailable")

type Store interface {
	Available() bool
	Close() error
	InsertAccessEvents(context.Context, []AccessEvent) error
	QueryAccessEvents(context.Context, AccessFilter) (AccessPage, error)
	InsertTrafficEvents(context.Context, []TrafficEvent) error
	QueryTrafficEvents(context.Context, TrafficFilter) (TrafficPage, error)
	QueryTrafficSeries(context.Context, TrafficSeriesFilter) (TrafficSeries, error)
	InsertForwardConnectionEvents(context.Context, []ForwardConnectionEvent) error
	QueryForwardConnectionEvents(context.Context, ForwardConnectionFilter) (ForwardConnectionPage, error)
}

type AccessEvent struct {
	EventID         uuid.UUID
	EventTime       time.Time
	ServerID        int64
	UserID          int64
	UserSubscribeID int64
	SourceIP        string
	SourcePort      uint16
	DestinationHost string
	DestinationIP   string
	DestinationPort uint16
	Network         string
	Status          string
	OutboundTag     string
	Reason          string
	ParserVersion   uint16
}

type AccessFilter struct {
	Start           time.Time
	End             time.Time
	ServerID        int64
	UserID          int64
	UserSubscribeID int64
	SourceIP        string
	Destination     string
	Status          string
	Limit           int
	CursorTime      time.Time
	CursorID        uuid.UUID
}

type AccessPage struct {
	List    []AccessEvent
	HasMore bool
}

type TrafficEvent struct {
	EventID         uuid.UUID
	EventTime       time.Time
	ServerID        int64
	UserID          int64
	UserSubscribeID int64
	Download        int64
	Upload          int64
}

type TrafficFilter struct {
	Start           time.Time
	End             time.Time
	ServerID        int64
	UserID          int64
	UserSubscribeID int64
	Limit           int
	Offset          int
}

type TrafficPage struct {
	List  []TrafficEvent
	Total int64
}

type TrafficSeriesFilter struct {
	Start           time.Time
	End             time.Time
	Interval        time.Duration
	UserID          int64
	UserSubscribeID int64
}

type TrafficSeriesPoint struct {
	Timestamp time.Time
	Download  int64
	Upload    int64
}

type TrafficSeries struct {
	List     []TrafficSeriesPoint
	Download int64
	Upload   int64
}

type ForwardConnectionEvent struct {
	EventID               uuid.UUID
	ConnectionID          uuid.UUID
	InstanceID            uuid.UUID
	BootID                uuid.UUID
	EventTime             time.Time
	EventType             string
	SourceIP              string
	SourcePort            uint16
	ListenIP              string
	ListenPort            uint16
	ConfiguredTargetHost  string
	ConfiguredTargetPort  uint16
	ResolvedTargetIP      string
	ResolvedTargetPort    uint16
	ForwarderOutboundIP   string
	ForwarderOutboundPort uint16
	RuleID                string
	ServerID              int64
	NodeID                int64
	Inbound               string
	ConfigVersion         string
	ConnectedAt           time.Time
	ClosedAt              time.Time
	DurationMS            int64
	UploadBytes           uint64
	DownloadBytes         uint64
	CloseInitiator        string
	CloseReason           string
	Error                 string
}

type ForwardConnectionFilter struct {
	Start        time.Time
	End          time.Time
	InstanceID   uuid.UUID
	ConnectionID uuid.UUID
	ListenPort   uint16
	SourceIP     string
	Destination  string
	EventType    string
	Limit        int
	CursorTime   time.Time
	CursorID     uuid.UUID
}

type ForwardConnectionPage struct {
	List    []ForwardConnectionEvent
	HasMore bool
}

type unavailableStore struct{}

func NewUnavailableStore() Store { return unavailableStore{} }

func (unavailableStore) Available() bool { return false }
func (unavailableStore) Close() error    { return nil }
func (unavailableStore) InsertAccessEvents(context.Context, []AccessEvent) error {
	return ErrUnavailable
}
func (unavailableStore) QueryAccessEvents(context.Context, AccessFilter) (AccessPage, error) {
	return AccessPage{}, ErrUnavailable
}
func (unavailableStore) InsertTrafficEvents(context.Context, []TrafficEvent) error {
	return ErrUnavailable
}
func (unavailableStore) QueryTrafficEvents(context.Context, TrafficFilter) (TrafficPage, error) {
	return TrafficPage{}, ErrUnavailable
}
func (unavailableStore) QueryTrafficSeries(context.Context, TrafficSeriesFilter) (TrafficSeries, error) {
	return TrafficSeries{}, ErrUnavailable
}
func (unavailableStore) InsertForwardConnectionEvents(context.Context, []ForwardConnectionEvent) error {
	return ErrUnavailable
}
func (unavailableStore) QueryForwardConnectionEvents(context.Context, ForwardConnectionFilter) (ForwardConnectionPage, error) {
	return ForwardConnectionPage{}, ErrUnavailable
}
