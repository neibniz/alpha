package user

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWithdrawalLogSelectColumnsStayLightweight(t *testing.T) {
	want := []string{
		"id",
		"user_id",
		"amount",
		"content",
		"status",
		"reason",
		"created_at",
		"updated_at",
	}
	if !reflect.DeepEqual(withdrawalLogSelectColumns, want) {
		t.Fatalf("withdrawal log select columns = %#v, want %#v", withdrawalLogSelectColumns, want)
	}
	for _, column := range withdrawalLogSelectColumns {
		normalized := strings.ToLower(column)
		for _, forbidden := range []string{"password", "salt", "token", "rules"} {
			if strings.Contains(normalized, forbidden) {
				t.Fatalf("withdrawal log columns must not include %q in %q", forbidden, column)
			}
		}
	}
}

func TestQueryWithdrawalLogFiltersOwnerOrdersAndKeepsEmptyListShape(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelUser.Withdrawal{}); err != nil {
		t.Fatal(err)
	}

	userID := int64(7)
	otherUserID := int64(8)
	now := time.Unix(1700000000, 0).UTC()
	if err := db.Create(&modelUser.User{Id: userID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&modelUser.User{Id: otherUserID}).Error; err != nil {
		t.Fatal(err)
	}
	rows := []modelUser.Withdrawal{
		{Id: 1, UserId: userID, Amount: 100, Content: "old", Status: 0, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{Id: 2, UserId: userID, Amount: 200, Content: "new", Status: 1, Reason: "ok", CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
		{Id: 3, UserId: otherUserID, Amount: 300, Content: "other", Status: 0, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{Id: userID})
	logic := NewQueryWithdrawalLogLogic(ctx, &svc.ServiceContext{DB: db})
	resp, err := logic.QueryWithdrawalLog(&types.QueryWithdrawalLogListRequest{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2", resp.Total)
	}
	if len(resp.List) != 2 {
		t.Fatalf("list length = %d, want 2: %#v", len(resp.List), resp.List)
	}
	if resp.List[0].Id != 2 || resp.List[1].Id != 1 {
		t.Fatalf("withdrawal rows not ordered by id desc: %#v", resp.List)
	}
	if resp.List[0].Amount != 200 || resp.List[0].Reason != "ok" || resp.List[0].CreatedAt != rows[1].CreatedAt.UnixMilli() {
		t.Fatalf("withdrawal row not mapped: %#v", resp.List[0])
	}

	emptyCtx := context.WithValue(context.Background(), constant.CtxKeyUser, &modelUser.User{Id: 99})
	emptyLogic := NewQueryWithdrawalLogLogic(emptyCtx, &svc.ServiceContext{DB: db})
	emptyResp, err := emptyLogic.QueryWithdrawalLog(&types.QueryWithdrawalLogListRequest{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if emptyResp.Total != 0 || emptyResp.List == nil || len(emptyResp.List) != 0 {
		t.Fatalf("empty withdrawal response shape mismatch: %#v", emptyResp)
	}
}
