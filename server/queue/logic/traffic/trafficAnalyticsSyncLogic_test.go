package traffic

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/analytics"
	trafficModel "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTrafficAnalyticsOutboxColumnsStayLightweight(t *testing.T) {
	want := []string{
		"id", "event_id", "server_id", "user_id", "user_subscribe_id",
		"download", "upload", "event_time",
	}
	if !reflect.DeepEqual(trafficAnalyticsOutboxColumns, want) {
		t.Fatalf("outbox columns = %#v, want %#v", trafficAnalyticsOutboxColumns, want)
	}
}

func TestTrafficAnalyticsSyncBatchDeletesOnlyAfterSuccessfulInsert(t *testing.T) {
	db := newTrafficAnalyticsSyncTestDB(t)
	rows := []trafficModel.EventOutbox{
		newTrafficAnalyticsOutboxRow(1, 10),
		newTrafficAnalyticsOutboxRow(2, 20),
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	store := &trafficAnalyticsSyncTestStore{available: true}
	logic := NewTrafficAnalyticsSyncLogic(&svc.ServiceContext{DB: db, Analytics: store})

	count, err := logic.syncBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || len(store.events) != 2 {
		t.Fatalf("sync count/events = %d/%d, want 2/2", count, len(store.events))
	}
	if store.events[0].EventID != rows[0].EventId || store.events[1].Download != rows[1].Download {
		t.Fatalf("inserted events = %#v", store.events)
	}
	if remaining := trafficAnalyticsOutboxCount(t, db); remaining != 0 {
		t.Fatalf("remaining outbox rows = %d, want 0", remaining)
	}
}

func TestTrafficAnalyticsSyncBatchRetainsRowsOnInsertFailure(t *testing.T) {
	db := newTrafficAnalyticsSyncTestDB(t)
	row := newTrafficAnalyticsOutboxRow(1, 10)
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("clickhouse unavailable")
	store := &trafficAnalyticsSyncTestStore{available: true, insertErr: wantErr}
	logic := NewTrafficAnalyticsSyncLogic(&svc.ServiceContext{DB: db, Analytics: store})

	count, err := logic.syncBatch(context.Background())
	if count != 1 || !errors.Is(err, wantErr) {
		t.Fatalf("sync count/error = %d/%v, want 1/%v", count, err, wantErr)
	}
	if remaining := trafficAnalyticsOutboxCount(t, db); remaining != 1 {
		t.Fatalf("remaining outbox rows = %d, want 1", remaining)
	}
}

func TestTrafficAnalyticsSyncProcessUsesDistributedLock(t *testing.T) {
	db := newTrafficAnalyticsSyncTestDB(t)
	row := newTrafficAnalyticsOutboxRow(1, 10)
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	insertStarted := make(chan struct{})
	releaseInsert := make(chan struct{})
	store := &trafficAnalyticsSyncTestStore{
		available:     true,
		insertStarted: insertStarted,
		releaseInsert: releaseInsert,
	}
	logic := NewTrafficAnalyticsSyncLogic(&svc.ServiceContext{DB: db, Redis: redisClient, Analytics: store})

	firstDone := make(chan error, 1)
	go func() { firstDone <- logic.ProcessTask(context.Background(), nil) }()
	select {
	case <-insertStarted:
	case <-time.After(time.Second):
		t.Fatal("first sync did not reach analytics insert")
	}
	if err := logic.ProcessTask(context.Background(), nil); err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}
	if calls := store.insertCallCount(); calls != 1 {
		t.Fatalf("analytics insert calls while locked = %d, want 1", calls)
	}
	close(releaseInsert)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("first sync did not finish")
	}
}

func newTrafficAnalyticsSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&trafficModel.EventOutbox{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func newTrafficAnalyticsOutboxRow(id, download int64) trafficModel.EventOutbox {
	return trafficModel.EventOutbox{
		Id:              id,
		EventId:         uuid.New(),
		ServerId:        3,
		UserId:          4,
		UserSubscribeId: 5,
		Download:        download,
		Upload:          download / 2,
		EventTime:       time.UnixMilli(1_700_000_000_000 + id).UTC(),
		CreatedAt:       time.Now().UTC(),
	}
}

func trafficAnalyticsOutboxCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&trafficModel.EventOutbox{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

type trafficAnalyticsSyncTestStore struct {
	mu            sync.Mutex
	available     bool
	insertErr     error
	events        []analytics.TrafficEvent
	insertCalls   int
	insertStarted chan struct{}
	releaseInsert chan struct{}
}

func (s *trafficAnalyticsSyncTestStore) Available() bool { return s.available }
func (s *trafficAnalyticsSyncTestStore) Close() error    { return nil }
func (s *trafficAnalyticsSyncTestStore) InsertAccessEvents(context.Context, []analytics.AccessEvent) error {
	return nil
}
func (s *trafficAnalyticsSyncTestStore) QueryAccessEvents(context.Context, analytics.AccessFilter) (analytics.AccessPage, error) {
	return analytics.AccessPage{}, nil
}
func (s *trafficAnalyticsSyncTestStore) InsertTrafficEvents(_ context.Context, events []analytics.TrafficEvent) error {
	s.mu.Lock()
	s.events = append(s.events, events...)
	s.insertCalls++
	insertErr := s.insertErr
	insertStarted := s.insertStarted
	releaseInsert := s.releaseInsert
	s.mu.Unlock()
	if insertStarted != nil {
		select {
		case <-insertStarted:
		default:
			close(insertStarted)
		}
	}
	if releaseInsert != nil {
		<-releaseInsert
	}
	return insertErr
}

func (s *trafficAnalyticsSyncTestStore) insertCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.insertCalls
}
func (s *trafficAnalyticsSyncTestStore) QueryTrafficEvents(context.Context, analytics.TrafficFilter) (analytics.TrafficPage, error) {
	return analytics.TrafficPage{}, nil
}
func (s *trafficAnalyticsSyncTestStore) QueryTrafficSeries(context.Context, analytics.TrafficSeriesFilter) (analytics.TrafficSeries, error) {
	return analytics.TrafficSeries{}, nil
}
func (s *trafficAnalyticsSyncTestStore) InsertForwardConnectionEvents(context.Context, []analytics.ForwardConnectionEvent) error {
	return nil
}
func (s *trafficAnalyticsSyncTestStore) QueryForwardConnectionEvents(context.Context, analytics.ForwardConnectionFilter) (analytics.ForwardConnectionPage, error) {
	return analytics.ForwardConnectionPage{}, nil
}
