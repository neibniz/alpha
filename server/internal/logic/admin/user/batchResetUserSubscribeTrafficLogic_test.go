package user

import (
	"context"
	"reflect"
	"testing"
	"time"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateBatchTrafficResetRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *types.BatchResetUserSubscribeTrafficRequest
		wantErr bool
	}{
		{name: "all confirmed", req: &types.BatchResetUserSubscribeTrafficRequest{Scope: "all", Confirmation: resetAllTrafficConfirmation}},
		{name: "all missing confirmation", req: &types.BatchResetUserSubscribeTrafficRequest{Scope: "all"}, wantErr: true},
		{name: "users", req: &types.BatchResetUserSubscribeTrafficRequest{Scope: "users", UserIds: []int64{1}}},
		{name: "users mixed scope", req: &types.BatchResetUserSubscribeTrafficRequest{Scope: "users", UserSubscribeIds: []int64{1}}, wantErr: true},
		{name: "subscriptions", req: &types.BatchResetUserSubscribeTrafficRequest{Scope: "subscriptions", UserSubscribeIds: []int64{1}}},
		{name: "too many users", req: &types.BatchResetUserSubscribeTrafficRequest{Scope: "users", UserIds: make([]int64, maxBatchTrafficResetSelection+1)}, wantErr: true},
		{name: "invalid", req: &types.BatchResetUserSubscribeTrafficRequest{Scope: "everything"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBatchTrafficResetRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizePositiveIDs(t *testing.T) {
	got := normalizePositiveIDs([]int64{3, 1, 3, 0, -1, 2})
	want := []int64{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ids = %#v, want %#v", got, want)
	}
}

func TestResetNextBatchReactivatesOnlyEffectiveExhaustedSubscriptions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelUser.Subscribe{}, &modelLog.SystemLog{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finishedAt := now.Add(-time.Minute)
	rows := []*modelUser.Subscribe{
		{UserId: 1, OrderId: 1, SubscribeId: 1, ProductName: "exhausted", ProductSnapshot: "{}", ExpireTime: now.Add(time.Hour), FinishedAt: &finishedAt, Traffic: 100, Download: 90, Upload: 20, Token: "reset-1", UUID: "reset-uuid-1", Status: 2},
		{UserId: 1, OrderId: 2, SubscribeId: 1, ProductName: "expired", ProductSnapshot: "{}", ExpireTime: now.Add(-time.Hour), FinishedAt: &finishedAt, Traffic: 100, Download: 90, Upload: 20, Token: "reset-2", UUID: "reset-uuid-2", Status: 2},
		{UserId: 1, OrderId: 3, SubscribeId: 1, ProductName: "stopped", ProductSnapshot: "{}", ExpireTime: now.Add(time.Hour), FinishedAt: &finishedAt, Traffic: 100, Download: 90, Upload: 20, Token: "reset-3", UUID: "reset-uuid-3", Status: 5},
		{UserId: 1, OrderId: 4, SubscribeId: 1, ProductName: "active", ProductSnapshot: "{}", ExpireTime: now.Add(time.Hour), Traffic: 100, Download: 50, Upload: 10, Token: "reset-4", UUID: "reset-uuid-4", Status: 1},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	logic := NewBatchResetUserSubscribeTrafficLogic(context.Background(), &svc.ServiceContext{DB: db})
	resetAt := now.Add(time.Second)
	selected, reactivated, err := logic.resetNextBatch(&types.BatchResetUserSubscribeTrafficRequest{
		Scope: batchTrafficResetScopeSubscribes, UserSubscribeIds: ids,
	}, rows[len(rows)-1].Id, resetAt, "test-batch", 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != len(rows) || reactivated != 1 {
		t.Fatalf("selected/reactivated = %d/%d, want %d/1", len(selected), reactivated, len(rows))
	}

	var got []modelUser.Subscribe
	if err := db.Order("id ASC").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	wantStatuses := []uint8{1, 2, 5, 1}
	for i, row := range got {
		if row.Download != 0 || row.Upload != 0 || !row.TrafficResetAt.Equal(resetAt) {
			t.Fatalf("row %d was not reset: %#v", row.Id, row)
		}
		if row.Status != wantStatuses[i] {
			t.Fatalf("row %d status = %d, want %d", row.Id, row.Status, wantStatuses[i])
		}
		if i == 0 && row.FinishedAt != nil {
			t.Fatalf("reactivated row retained finished_at: %#v", row.FinishedAt)
		}
		if i > 0 && i < 3 && row.FinishedAt == nil {
			t.Fatalf("non-reactivated row %d lost finished_at", row.Id)
		}
	}
	var logCount int64
	if err := db.Model(&modelLog.SystemLog{}).Count(&logCount).Error; err != nil {
		t.Fatal(err)
	}
	if logCount != int64(len(rows)) {
		t.Fatalf("audit log count = %d, want %d", logCount, len(rows))
	}
}
