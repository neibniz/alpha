package subscribe

import (
	"context"
	"reflect"
	"strings"
	"testing"

	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeBatchDeleteSubscribeIDs(t *testing.T) {
	got := normalizeBatchDeleteSubscribeIDs([]int64{4, 0, 4, -1, 2, 3, 2})
	want := []int64{4, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeBatchDeleteSubscribeIDs() = %#v, want %#v", got, want)
	}
}

func TestBatchDeleteSubscribeGroupNormalizesIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelSubscribe.Group{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create([]*modelSubscribe.Group{
		{Id: 7, Name: "first"},
		{Id: 8, Name: "second"},
		{Id: 9, Name: "third"},
	}).Error; err != nil {
		t.Fatal(err)
	}

	logic := NewBatchDeleteSubscribeGroupLogic(context.Background(), &svc.ServiceContext{DB: db})
	if err := logic.BatchDeleteSubscribeGroup(&types.BatchDeleteSubscribeGroupRequest{Ids: []int64{7, 0, 7, -1, 8}}); err != nil {
		t.Fatal(err)
	}

	var remaining []int64
	if err := db.Model(&modelSubscribe.Group{}).Order("id ASC").Pluck("id", &remaining).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(remaining, []int64{9}) {
		t.Fatalf("remaining subscribe group ids = %#v, want [9]", remaining)
	}
}

func TestBuildActiveSubscribeUsageQueryUsesExistenceProbe(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var subscribeID int64
	tx := buildActiveSubscribeUsageQuery(db, 7).Scan(&subscribeID)
	sql := strings.ToLower(tx.Statement.SQL.String())
	if !strings.Contains(sql, "select") || !strings.Contains(sql, "subscribe_id") {
		t.Fatalf("active subscribe usage query should select subscribe_id, got %q", sql)
	}
	if strings.Contains(sql, "count(") {
		t.Fatalf("active subscribe usage query must not count all rows: %q", sql)
	}
	if !strings.Contains(sql, "limit 1") {
		t.Fatalf("active subscribe usage query should limit to one row: %q", sql)
	}
	if !strings.Contains(sql, (&user.Subscribe{}).TableName()) {
		t.Fatalf("active subscribe usage query should target user subscribe table: %q", sql)
	}
}
