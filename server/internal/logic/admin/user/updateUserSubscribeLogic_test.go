package user

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdjustedSubscriptionStatus(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	tests := []struct {
		name     string
		current  uint8
		expireAt time.Time
		traffic  int64
		download int64
		upload   int64
		want     uint8
	}{
		{name: "pending remains pending", current: subscriptionstate.StatusPending, expireAt: future, traffic: 100, want: subscriptionstate.StatusPending},
		{name: "deducted remains deducted", current: subscriptionstate.StatusDeducted, expireAt: future, traffic: 100, want: subscriptionstate.StatusDeducted},
		{name: "stopped remains stopped", current: subscriptionstate.StatusStopped, expireAt: future, traffic: 100, want: subscriptionstate.StatusStopped},
		{name: "expired extension reactivates", current: subscriptionstate.StatusExpired, expireAt: future, traffic: 100, download: 20, upload: 30, want: subscriptionstate.StatusActive},
		{name: "exhausted counter reduction reactivates", current: subscriptionstate.StatusExhausted, expireAt: future, traffic: 100, download: 20, upload: 30, want: subscriptionstate.StatusActive},
		{name: "active over quota exhausts", current: subscriptionstate.StatusActive, expireAt: future, traffic: 100, download: 70, upload: 30, want: subscriptionstate.StatusExhausted},
		{name: "active expiry wins over quota", current: subscriptionstate.StatusActive, expireAt: past, traffic: 100, download: 100, want: subscriptionstate.StatusExpired},
		{name: "unlimited quota stays active", current: subscriptionstate.StatusExhausted, expireAt: future, traffic: 0, download: 500, upload: 500, want: subscriptionstate.StatusActive},
		{name: "unknown status is preserved", current: 99, expireAt: future, traffic: 100, want: 99},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := adjustedSubscriptionStatus(test.current, test.expireAt, test.traffic, test.download, test.upload, now)
			if got != test.want {
				t.Fatalf("adjustedSubscriptionStatus() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestAdjustedSubscriptionFinishedAt(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	existing := now.Add(-time.Hour)

	tests := []struct {
		name      string
		current   *modelUser.Subscribe
		next      uint8
		wantSet   bool
		wantClear bool
	}{
		{
			name:      "active clears terminal timestamp",
			current:   &modelUser.Subscribe{Status: subscriptionstate.StatusExpired, FinishedAt: &existing},
			next:      subscriptionstate.StatusActive,
			wantClear: true,
		},
		{
			name:    "new exhaustion records timestamp",
			current: &modelUser.Subscribe{Status: subscriptionstate.StatusActive},
			next:    subscriptionstate.StatusExhausted,
			wantSet: true,
		},
		{
			name:    "same terminal status preserves timestamp",
			current: &modelUser.Subscribe{Status: subscriptionstate.StatusExpired, FinishedAt: &existing},
			next:    subscriptionstate.StatusExpired,
		},
		{
			name:    "manual stopped state is untouched",
			current: &modelUser.Subscribe{Status: subscriptionstate.StatusStopped, FinishedAt: &existing},
			next:    subscriptionstate.StatusStopped,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, clear := adjustedSubscriptionFinishedAt(test.current, test.next, now)
			if clear != test.wantClear {
				t.Fatalf("clear = %v, want %v", clear, test.wantClear)
			}
			if (got != nil) != test.wantSet {
				t.Fatalf("timestamp set = %v, want %v", got != nil, test.wantSet)
			}
			if got != nil && !got.Equal(now) {
				t.Fatalf("timestamp = %v, want %v", got, now)
			}
		})
	}
}

func TestUpdateUserSubscribeUsesTrafficBarrierAndWritesAudit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update-user-subscribe?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.Subscribe{}, &subscribe.Subscribe{}, &log.SystemLog{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	finishedAt := now.Add(-time.Hour)
	product := &subscribe.Subscribe{Id: 3, Name: "Stable", ResetCycle: 2}
	if err := db.Create(product).Error; err != nil {
		t.Fatal(err)
	}
	record := &modelUser.Subscribe{
		Id:              10,
		UserId:          20,
		OrderId:         30,
		SubscribeId:     product.Id,
		ProductName:     product.Name,
		ProductSnapshot: "{}",
		StartTime:       now.Add(-24 * time.Hour),
		ExpireTime:      now.Add(-time.Hour),
		FinishedAt:      &finishedAt,
		Traffic:         100,
		Download:        80,
		Upload:          20,
		Token:           "adjustment-token",
		UUID:            "adjustment-uuid",
		Status:          subscriptionstate.StatusExpired,
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	svcCtx := &svc.ServiceContext{
		DB:             db,
		Redis:          rds,
		UserModel:      modelUser.NewModel(db, rds),
		SubscribeModel: subscribe.NewModel(db, rds),
	}
	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{Id: 99})
	traffic := int64(200)
	expiredAt := now.Add(48 * time.Hour).UnixMilli()
	upload := int64(0)
	download := int64(0)
	started := time.Now().UTC()

	err = NewUpdateUserSubscribeLogic(ctx, svcCtx).UpdateUserSubscribe(&types.UpdateUserSubscribeRequest{
		UserSubscribeId: record.Id,
		SubscribeId:     product.Id,
		Traffic:         &traffic,
		ExpiredAt:       &expiredAt,
		Upload:          &upload,
		Download:        &download,
	})
	if err != nil {
		t.Fatal(err)
	}

	var got modelUser.Subscribe
	if err := db.First(&got, record.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != subscriptionstate.StatusActive || got.FinishedAt != nil {
		t.Fatalf("subscription was not reactivated consistently: %#v", got)
	}
	if got.Traffic != traffic || got.Download != 0 || got.Upload != 0 {
		t.Fatalf("traffic adjustment was not persisted: %#v", got)
	}
	if got.TrafficResetAt.Before(started) {
		t.Fatalf("traffic reset barrier was not advanced: %v < %v", got.TrafficResetAt, started)
	}

	var audit log.SystemLog
	if err := db.Where("type = ? AND object_id = ?", log.TypeAdjustSubscribe.Uint8(), record.Id).First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	var content log.SubscribeAdjustment
	if err := content.Unmarshal([]byte(audit.Content)); err != nil {
		t.Fatal(err)
	}
	if content.UserId != record.UserId || content.OperatorUserId != 99 {
		t.Fatalf("unexpected audit actor data: %#v", content)
	}
	if content.Before.Status != subscriptionstate.StatusExpired || content.After.Status != subscriptionstate.StatusActive {
		t.Fatalf("unexpected audit state transition: %#v", content)
	}
}

func TestUpdateUserSubscribeRequestValidationRequiresExplicitZeroValues(t *testing.T) {
	zero := int64(0)
	valid := &types.UpdateUserSubscribeRequest{
		UserSubscribeId: 1,
		SubscribeId:     2,
		Traffic:         &zero,
		ExpiredAt:       &zero,
		Upload:          &zero,
		Download:        &zero,
	}
	if err := (&svc.ServiceContext{}).Validate(valid); err != nil {
		t.Fatalf("explicit zero values should be valid: %v", err)
	}

	missing := *valid
	missing.Download = nil
	if err := (&svc.ServiceContext{}).Validate(&missing); err == nil {
		t.Fatal("missing download must be rejected")
	}

	negative := int64(-1)
	invalid := *valid
	invalid.Traffic = &negative
	if err := (&svc.ServiceContext{}).Validate(&invalid); err == nil {
		t.Fatal("negative traffic must be rejected")
	}
}
