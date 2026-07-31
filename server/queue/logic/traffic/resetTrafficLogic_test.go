package traffic

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestResetTrafficDueColumnsUseMaterializedSchedule(t *testing.T) {
	columns := make(map[string]struct{}, len(resetTrafficDueColumns))
	for _, column := range resetTrafficDueColumns {
		columns[column] = struct{}{}
	}
	for _, required := range []string{
		"id",
		"user_id",
		"subscribe_id",
		"product_snapshot",
		"token",
		"start_time",
		"expire_time",
		"reset_cycle",
		"next_traffic_reset_at",
		"status",
	} {
		if _, ok := columns[required]; !ok {
			t.Fatalf("reset traffic columns must include %q", required)
		}
	}
	for _, forbidden := range []string{"traffic", "download", "upload"} {
		if _, ok := columns[forbidden]; ok {
			t.Fatalf("reset traffic columns must not load %q", forbidden)
		}
	}
}

func TestNextTrafficResetAtExprCoversEveryID(t *testing.T) {
	first := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	expr := nextTrafficResetAtExpr([]int64{3, 1, 2}, map[int64]*time.Time{
		3: &first,
		1: nil,
		2: &first,
	})
	if strings.Count(expr.SQL, "WHEN ? THEN ?") != 3 {
		t.Fatalf("expression SQL = %q", expr.SQL)
	}
	wantVars := []any{int64(3), first, int64(1), nil, int64(2), first}
	if !reflect.DeepEqual(expr.Vars, wantVars) {
		t.Fatalf("expression vars = %#v, want %#v", expr.Vars, wantVars)
	}
}

func TestResetTrafficBatchesAreBounded(t *testing.T) {
	if resetTrafficBatchSize <= 0 || resetTrafficBatchSize > 1000 {
		t.Fatalf("reset batch size = %d, want 1..1000", resetTrafficBatchSize)
	}
	if resetTrafficMaxBatches <= 0 {
		t.Fatalf("reset max batches = %d, want positive", resetTrafficMaxBatches)
	}
}

func TestBuildResetTrafficLogRowsIsSuccessfulAndOrdered(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 30, 0, 0, time.UTC)
	rows := buildResetTrafficLogRows([]*user.Subscribe{
		{Id: 3, UserId: 30},
		nil,
		{Id: 1, UserId: 10},
	}, now)

	if len(rows) != 2 || rows[0].ObjectID != 3 || rows[1].ObjectID != 1 {
		t.Fatalf("reset log rows = %#v", rows)
	}
	for i, row := range rows {
		if row.Type != modelLog.TypeResetSubscribe.Uint8() || !row.Success || row.EventAt != now.UnixMilli() {
			t.Fatalf("row[%d] metadata = %#v", i, row)
		}
		var content modelLog.ResetSubscribe
		if err := json.Unmarshal([]byte(row.Content), &content); err != nil {
			t.Fatalf("row[%d] content unmarshal: %v", i, err)
		}
		if content.Type != modelLog.ResetSubscribeTypeAuto || content.Timestamp != now.UnixMilli() {
			t.Fatalf("row[%d] content = %#v", i, content)
		}
	}
}

func TestResetNextBatchPostgresIsIdempotent(t *testing.T) {
	dsn := os.Getenv("ALPHA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ALPHA_TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	dueAt := now.Add(-time.Minute)
	row := &user.Subscribe{
		UserId:             900001,
		OrderId:            0,
		SubscribeId:        900001,
		ProductName:        "reset integration",
		ProductSnapshot:    `{"version":1,"kind":"subscribe","name":"reset integration","reset_cycle":2}`,
		StartTime:          time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		ExpireTime:         time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC),
		Traffic:            1000,
		Download:           400,
		Upload:             300,
		ResetCycle:         2,
		NextTrafficResetAt: &dueAt,
		Token:              "reset-integration-token",
		UUID:               "00000000-0000-4000-8000-000000000099",
		Status:             1,
	}
	if err := db.Unscoped().Where("token = ?", row.Token).Delete(&user.Subscribe{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(row).Error; err != nil {
		t.Fatal(err)
	}

	logic := NewResetTrafficLogic(&svc.ServiceContext{DB: db})
	rows, err := logic.resetNextBatch(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Id != row.Id {
		t.Fatalf("first reset rows = %#v", rows)
	}

	if err := db.Model(&user.Subscribe{}).Where("id = ?", row.Id).
		Updates(map[string]any{"download": int64(11), "upload": int64(7)}).Error; err != nil {
		t.Fatal(err)
	}
	rows, err = logic.resetNextBatch(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("duplicate reset selected %d rows", len(rows))
	}
	var got user.Subscribe
	if err := db.First(&got, row.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Download != 11 || got.Upload != 7 || got.NextTrafficResetAt == nil || !got.NextTrafficResetAt.After(now) {
		t.Fatalf("duplicate reset changed subscription: %#v", got)
	}
}
