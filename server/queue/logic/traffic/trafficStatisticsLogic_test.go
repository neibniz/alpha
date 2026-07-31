package traffic

import (
	"context"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/model/node"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	trafficModel "github.com/neibniz/alpha/server/internal/model/traffic"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/nodeMultiplier"
	"github.com/neibniz/alpha/server/queue/types"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTrafficStatisticsSubscribeColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(trafficStatisticsSubscribeColumns))
	for _, column := range trafficStatisticsSubscribeColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"user_id",
		"subscribe_id",
		"token",
		"expire_time",
		"traffic",
		"download",
		"upload",
		"traffic_reset_at",
		"status",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("traffic statistics subscribe columns must include %q", column)
		}
	}

	forbidden := []string{
		"order_id",
		"product_snapshot",
		"uuid",
		"note",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("traffic statistics subscribe columns must not include %q", column)
		}
	}
}

func TestTrafficNodeSelectorsNormalizeAndDeduplicate(t *testing.T) {
	nodeIDs, tags := trafficNodeSelectors([]node.Node{
		{Id: 9, Tags: " edge, shared "},
		{Id: 7, Tags: "shared,,premium"},
		{Id: 9, Tags: "edge"},
	})
	if want := []string{"9", "7"}; !reflect.DeepEqual(nodeIDs, want) {
		t.Fatalf("node ids = %#v, want %#v", nodeIDs, want)
	}
	if want := []string{"edge", "shared", "premium"}; !reflect.DeepEqual(tags, want) {
		t.Fatalf("node tags = %#v, want %#v", tags, want)
	}
}

func TestNewlyExhaustedTrafficSubscriptionsOnlyTransitionsActiveNonExpiredRows(t *testing.T) {
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	subscribes := indexTrafficSubscribes([]*user.Subscribe{
		{Id: 1, SubscribeId: 11, Status: 1, Traffic: 100, Download: 70, Upload: 10, ExpireTime: now.Add(time.Hour)},
		{Id: 2, SubscribeId: 12, Status: 1, Traffic: 100, Download: 10, Upload: 10, ExpireTime: now.Add(time.Hour)},
		{Id: 3, SubscribeId: 13, Status: 1, Traffic: 100, Download: 90, Upload: 10, ExpireTime: now.Add(time.Hour)},
		{Id: 4, SubscribeId: 14, Status: 1, Traffic: 100, Download: 70, Upload: 10, ExpireTime: now},
		{Id: 5, SubscribeId: 15, Status: 2, Traffic: 100, Download: 70, Upload: 10, ExpireTime: now.Add(time.Hour)},
	})
	ids, productIDs := newlyExhaustedTrafficSubscriptions(subscribes, []subscribeTrafficDeltaRow{
		{SubscribeID: 1, Download: 20},
		{SubscribeID: 2, Download: 20},
		{SubscribeID: 3, Download: 1},
		{SubscribeID: 4, Download: 20},
		{SubscribeID: 5, Download: 20},
	}, now)
	if want := []int64{1}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("exhausted ids = %#v, want %#v", ids, want)
	}
	if want := []int64{11}; !reflect.DeepEqual(productIDs, want) {
		t.Fatalf("product ids = %#v, want %#v", productIDs, want)
	}
}

func TestQueryTrafficSubscribesPostgresRejectsUnassignedSubscriptions(t *testing.T) {
	dsn := os.Getenv("ALPHA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("ALPHA_TEST_POSTGRES_DSN is not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	enabled := true
	serverID := int64(910001)
	nodeID := int64(910001)
	productIDs := []int64{910001, 910002, 910003}
	subscriptionIDs := []int64{920001, 920002, 920003}

	if err := db.Unscoped().Where("id IN ?", subscriptionIDs).Delete(&user.Subscribe{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Unscoped().Where("id IN ?", productIDs).Delete(&modelSubscribe.Subscribe{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Unscoped().Where("id = ?", nodeID).Delete(&node.Node{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&node.Node{Id: nodeID, ServerId: serverID, Name: "traffic auth", Tags: "edge", Enabled: &enabled, Sort: int(nodeID)}).Error; err != nil {
		t.Fatal(err)
	}
	products := []*modelSubscribe.Subscribe{
		{Id: productIDs[0], Name: "explicit", Nodes: "910001", Sort: productIDs[0]},
		{Id: productIDs[1], Name: "other", Nodes: "910002", Sort: productIDs[1]},
		{Id: productIDs[2], Name: "tagged", NodeTags: "edge", Sort: productIDs[2]},
	}
	if err := db.Create(products).Error; err != nil {
		t.Fatal(err)
	}
	expireAt := time.Now().Add(time.Hour)
	rows := []*user.Subscribe{
		{Id: subscriptionIDs[0], UserId: 1, SubscribeId: productIDs[0], ExpireTime: expireAt, Token: "traffic-auth-1", UUID: "00000000-0000-4000-8000-000000001001", Status: 1},
		{Id: subscriptionIDs[1], UserId: 1, SubscribeId: productIDs[1], ExpireTime: expireAt, Token: "traffic-auth-2", UUID: "00000000-0000-4000-8000-000000001002", Status: 1},
		{Id: subscriptionIDs[2], UserId: 1, SubscribeId: productIDs[2], ExpireTime: expireAt, Token: "traffic-auth-3", UUID: "00000000-0000-4000-8000-000000001003", Status: 1},
	}
	if err := db.Create(rows).Error; err != nil {
		t.Fatal(err)
	}

	logic := NewTrafficStatisticsLogic(&svc.ServiceContext{DB: db})
	got, err := logic.queryTrafficSubscribes(ctx, db, serverID, subscriptionIDs)
	if err != nil {
		t.Fatal(err)
	}
	if got.Len() != 2 || got.Get(subscriptionIDs[0]) == nil || got.Get(subscriptionIDs[2]) == nil {
		t.Fatalf("authorized subscriptions = %#v", got.rows)
	}
	if got.Get(subscriptionIDs[1]) != nil {
		t.Fatalf("unassigned subscription %d was accepted", subscriptionIDs[1])
	}
}

func TestCollectTrafficStatisticsBatchFiltersAndAggregates(t *testing.T) {
	reports, subscribeIDs, totals := collectTrafficStatisticsBatch([]types.UserTraffic{
		{SID: 1, Download: 50, Upload: 40},
		{SID: 1, Download: 100, Upload: 50},
		{SID: 1, Download: 5, Upload: 200},
		{SID: 2, Download: 200, Upload: 0},
		{SID: 0, Download: 500, Upload: 500},
	}, 100, 1200)

	wantReports := []trafficStatisticsReport{
		{SubscribeID: 1, Download: 120, Upload: 60},
		{SubscribeID: 1, Download: 6, Upload: 240},
		{SubscribeID: 2, Download: 240, Upload: 0},
	}
	if !reflect.DeepEqual(reports, wantReports) {
		t.Fatalf("reports = %#v, want %#v", reports, wantReports)
	}

	wantIDs := []int64{1, 2}
	if !reflect.DeepEqual(subscribeIDs, wantIDs) {
		t.Fatalf("subscribeIDs = %#v, want %#v", subscribeIDs, wantIDs)
	}

	wantTotals := map[int64]trafficStatisticsTotal{
		1: {Download: 126, Upload: 300},
		2: {Download: 240, Upload: 0},
	}
	if got := trafficStatisticsTotalsMap(totals); !reflect.DeepEqual(got, wantTotals) {
		t.Fatalf("totals = %#v, want %#v", got, wantTotals)
	}
}

func TestTrafficStatisticsReportsFromTotalsProducesOneRowPerSubscription(t *testing.T) {
	totals := newTrafficStatisticsTotals(4)
	totals.Add(2, 20, 2)
	totals.Add(1, 10, 1)
	totals.Add(2, 30, 3)

	got := trafficStatisticsReportsFromTotals(totals)
	want := []trafficStatisticsReport{
		{SubscribeID: 1, Download: 10, Upload: 1},
		{SubscribeID: 2, Download: 50, Upload: 5},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reports = %#v, want %#v", got, want)
	}
}

func TestAcceptTrafficReportsIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&trafficModel.ReportReceipt{}); err != nil {
		t.Fatal(err)
	}
	service := &svc.ServiceContext{DB: db, Config: config.Config{}}
	service.SetNodeMultiplierManager(nodeMultiplier.NewNodeMultiplierManager(nil))
	logic := NewTrafficStatisticsLogic(service)
	sampledAt := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	reports := []types.TrafficReport{{
		ReportId:        "00000000-0000-4000-8000-000000000001",
		SampledAt:       sampledAt.UnixMilli(),
		IntervalSeconds: 5,
		Sequence:        9,
		Traffic:         []types.UserTraffic{{SID: 7, Download: 20, Upload: 10}},
	}}

	accepted, totals, err := logic.acceptTrafficReports(context.Background(), db, 42, reports, sampledAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(accepted) != 1 || totals.Len() != 1 || !accepted[0].SampledAt.Equal(sampledAt) {
		t.Fatalf("first acceptance = %#v, totals=%#v", accepted, totals)
	}
	accepted, totals, err = logic.acceptTrafficReports(context.Background(), db, 42, reports, sampledAt.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(accepted) != 0 || totals.Len() != 0 {
		t.Fatalf("duplicate acceptance = %#v, totals=%#v", accepted, totals)
	}
	var receipt trafficModel.ReportReceipt
	if err := db.First(&receipt).Error; err != nil {
		t.Fatal(err)
	}
	if receipt.ServerId != 42 || receipt.Sequence != 9 || !receipt.SampledAt.Equal(sampledAt) {
		t.Fatalf("receipt = %#v", receipt)
	}
}

func TestCollectTrafficStatisticsBatchSkipsFilteredWithoutInputSizedCapacity(t *testing.T) {
	logs := make([]types.UserTraffic, viewutil.SmallRowsCapacityThreshold+20)
	for index := range logs {
		logs[index] = types.UserTraffic{SID: int64(index + 1), Download: 1, Upload: 1}
	}
	logs[0] = types.UserTraffic{SID: 99, Download: 200, Upload: 100}

	reports, subscribeIDs, totals := collectTrafficStatisticsBatch(logs, 100, 1000)
	if len(reports) != 1 || reports[0].SubscribeID != 99 {
		t.Fatalf("reports = %#v, want one report for subscribe 99", reports)
	}
	if cap(reports) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("filtered reports cap = %d, want %d", cap(reports), viewutil.SmallRowsCapacityThreshold+1)
	}
	if len(subscribeIDs) != 1 || subscribeIDs[0] != 99 {
		t.Fatalf("subscribe ids = %#v, want [99]", subscribeIDs)
	}
	if cap(subscribeIDs) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("filtered subscribe ids cap = %d, want %d", cap(subscribeIDs), viewutil.SmallRowsCapacityThreshold+1)
	}
	if totals.Len() != 1 || cap(totals.rows) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("totals len/cap = %d/%d, want 1/%d", totals.Len(), cap(totals.rows), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestTrafficStatisticsTotalsUsesLinearPathForSmallInput(t *testing.T) {
	totals := newTrafficStatisticsTotals(smallTrafficStatisticsIndexThreshold)
	for id := int64(1); id <= smallTrafficStatisticsIndexThreshold; id++ {
		if !totals.Add(id, id*10, id) {
			t.Fatalf("Add(%d) reported duplicate", id)
		}
	}
	if totals.byID != nil {
		t.Fatal("small totals should not allocate map index")
	}
	if totals.Add(3, 30, 3) {
		t.Fatal("duplicate total reported new id")
	}
	got, ok := totals.Get(3)
	if !ok || got.Download != 60 || got.Upload != 6 {
		t.Fatalf("total for 3 = %#v, %v", got, ok)
	}
}

func TestTrafficStatisticsTotalsUpgradesLargeInputToMap(t *testing.T) {
	totals := newTrafficStatisticsTotals(smallTrafficStatisticsIndexThreshold + 1)
	for id := int64(1); id <= smallTrafficStatisticsIndexThreshold+1; id++ {
		totals.Add(id, id*10, id)
	}
	if totals.byID == nil {
		t.Fatal("large totals should allocate map index")
	}
	got, ok := totals.Get(int64(smallTrafficStatisticsIndexThreshold + 1))
	if !ok || got.Download != int64((smallTrafficStatisticsIndexThreshold+1)*10) || got.Upload != int64(smallTrafficStatisticsIndexThreshold+1) {
		t.Fatalf("last total = %#v, %v", got, ok)
	}
}

func TestTrafficStatisticsTotalsAvoidsMapForLargeDuplicateInput(t *testing.T) {
	totals := newTrafficStatisticsTotals(smallTrafficStatisticsIndexThreshold + 20)
	for i := 0; i < smallTrafficStatisticsIndexThreshold+20; i++ {
		totals.Add(3, 10, 1)
	}
	if totals.byID != nil {
		t.Fatal("large duplicate totals should not allocate map index")
	}
	if totals.Len() != 1 {
		t.Fatalf("totals len = %d, want 1", totals.Len())
	}
	if cap(totals.rows) != smallTrafficStatisticsIndexThreshold+1 {
		t.Fatalf("totals cap = %d, want small initial cap", cap(totals.rows))
	}
	got, ok := totals.Get(3)
	if !ok || got.Download != int64((smallTrafficStatisticsIndexThreshold+20)*10) || got.Upload != int64(smallTrafficStatisticsIndexThreshold+20) {
		t.Fatalf("total for 3 = %#v, %v", got, ok)
	}
}

func TestTrafficStatisticsTotalsBoundsSparseUpgradeCapacity(t *testing.T) {
	totals := newTrafficStatisticsTotals(smallTrafficStatisticsIndexThreshold + 20)
	for id := int64(1); id <= smallTrafficStatisticsIndexThreshold+1; id++ {
		totals.Add(id, id*10, id)
	}
	if totals.byID == nil {
		t.Fatal("totals should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := totals.Len() * 2; cap(totals.rows) > maxCap {
		t.Fatalf("totals cap = %d, want at most %d", cap(totals.rows), maxCap)
	}
}

func TestTrafficSubscribeIndexUsesLinearPathForSmallInput(t *testing.T) {
	subs := make([]*user.Subscribe, 0, smallTrafficStatisticsIndexThreshold)
	for id := int64(1); id <= smallTrafficStatisticsIndexThreshold; id++ {
		subs = append(subs, &user.Subscribe{Id: id, UserId: id * 10})
	}
	index := indexTrafficSubscribes(subs)
	if index.byID != nil {
		t.Fatal("small subscribe index should not allocate map index")
	}
	if got := index.Get(4); got == nil || got.UserId != 40 {
		t.Fatalf("subscribe 4 = %#v", got)
	}
	index.Add(&user.Subscribe{Id: 4, UserId: 999})
	if got := index.Get(4); got == nil || got.UserId != 40 {
		t.Fatalf("duplicate subscribe changed existing row: %#v", got)
	}
}

func TestTrafficSubscribeIndexUpgradesLargeInputToMap(t *testing.T) {
	subs := make([]*user.Subscribe, 0, smallTrafficStatisticsIndexThreshold+1)
	for id := int64(1); id <= smallTrafficStatisticsIndexThreshold+1; id++ {
		subs = append(subs, &user.Subscribe{Id: id, UserId: id * 10})
	}
	index := indexTrafficSubscribes(subs)
	if index.byID == nil {
		t.Fatal("large subscribe index should allocate map index")
	}
	if index.Len() != smallTrafficStatisticsIndexThreshold+1 {
		t.Fatalf("subscribe index len = %d", index.Len())
	}
	lastID := int64(smallTrafficStatisticsIndexThreshold + 1)
	if got := index.Get(lastID); got == nil || got.UserId != lastID*10 {
		t.Fatalf("last subscribe = %#v", got)
	}
}

func TestTrafficSubscribeIndexAvoidsMapForLargeDuplicateInput(t *testing.T) {
	subs := make([]*user.Subscribe, 0, smallTrafficStatisticsIndexThreshold+20)
	for i := 0; i < cap(subs); i++ {
		subs = append(subs, &user.Subscribe{Id: 4, UserId: int64(i + 1)})
	}
	index := indexTrafficSubscribes(subs)
	if index.byID != nil {
		t.Fatal("large duplicate subscribe index should not allocate map index")
	}
	if index.Len() != 1 {
		t.Fatalf("subscribe index len = %d, want 1", index.Len())
	}
	if cap(index.rows) != smallTrafficStatisticsIndexThreshold+1 {
		t.Fatalf("subscribe index cap = %d, want small initial cap", cap(index.rows))
	}
	if got := index.Get(4); got == nil || got.UserId != 1 {
		t.Fatalf("subscribe 4 = %#v, want first subscribe", got)
	}
}

func TestTrafficSubscribeIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	subs := make([]*user.Subscribe, 0, smallTrafficStatisticsIndexThreshold+20)
	for id := int64(1); id <= smallTrafficStatisticsIndexThreshold+1; id++ {
		subs = append(subs, &user.Subscribe{Id: id, UserId: id * 10})
	}
	for len(subs) < smallTrafficStatisticsIndexThreshold+20 {
		subs = append(subs, &user.Subscribe{Id: 1, UserId: 999})
	}
	index := indexTrafficSubscribes(subs)
	if index.byID == nil {
		t.Fatal("subscribe index should allocate map after actual unique ids exceed threshold")
	}
	if maxCap := index.Len() * 2; cap(index.rows) > maxCap {
		t.Fatalf("subscribe index cap = %d, want at most %d", cap(index.rows), maxCap)
	}
}

func TestBuildTrafficLogRowsSkipsMissingSubscribes(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	rows := buildTrafficLogRows(context.Background(), now, 42, []trafficStatisticsReport{
		{SubscribeID: 2, Download: 20, Upload: 10},
		{SubscribeID: 9, Download: 90, Upload: 9},
		{SubscribeID: 1, Download: 50, Upload: 5},
	}, indexTrafficSubscribes([]*user.Subscribe{
		{Id: 1, UserId: 100},
		{Id: 2, UserId: 200},
	}))

	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].ServerId != 42 || rows[0].SubscribeId != 2 || rows[0].UserId != 200 || rows[0].Download != 20 || rows[0].Upload != 10 || !rows[0].Timestamp.Equal(now) {
		t.Fatalf("first row = %#v", rows[0])
	}
	if rows[1].ServerId != 42 || rows[1].SubscribeId != 1 || rows[1].UserId != 100 || rows[1].Download != 50 || rows[1].Upload != 5 || !rows[1].Timestamp.Equal(now) {
		t.Fatalf("second row = %#v", rows[1])
	}
}

func TestBuildTrafficLogRowsSkipsMissingWithoutInputSizedCapacity(t *testing.T) {
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	reports := make([]trafficStatisticsReport, viewutil.SmallRowsCapacityThreshold+20)
	for index := range reports {
		reports[index] = trafficStatisticsReport{SubscribeID: int64(index + 1), Download: 20, Upload: 10}
	}
	reports[0] = trafficStatisticsReport{SubscribeID: 99, Download: 20, Upload: 10}

	rows := buildTrafficLogRows(context.Background(), now, 42, reports, indexTrafficSubscribes([]*user.Subscribe{
		{Id: 99, UserId: 900},
	}))
	if len(rows) != 1 || rows[0].SubscribeId != 99 {
		t.Fatalf("traffic log rows = %#v, want one row for subscribe 99", rows)
	}
	if cap(rows) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("missing-heavy traffic log rows cap = %d, want %d", cap(rows), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildTrafficEventOutboxRowsUsesStableReportIdentity(t *testing.T) {
	reportID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	rows := buildTrafficLogRows(context.Background(), now, 42, []trafficStatisticsReport{
		{SubscribeID: 7, Download: 20, Upload: 10},
	}, indexTrafficSubscribes([]*user.Subscribe{{Id: 7, UserId: 70}}))
	first := buildTrafficEventOutboxRows(reportID, rows)
	second := buildTrafficEventOutboxRows(reportID, rows)
	if len(first) != 1 || len(second) != 1 || first[0].EventId != second[0].EventId {
		t.Fatalf("stable event ids = %#v, %#v", first, second)
	}
	other := buildTrafficEventOutboxRows(uuid.MustParse("00000000-0000-4000-8000-000000000002"), rows)
	if other[0].EventId == first[0].EventId {
		t.Fatal("different report ids produced the same traffic event id")
	}
	if !first[0].EventTime.Equal(now) {
		t.Fatalf("event time = %v, want sampled time %v", first[0].EventTime, now)
	}
}

func TestBuildSubscribeTrafficDeltaRowsFiltersAndOrders(t *testing.T) {
	got := buildSubscribeTrafficDeltaRows(indexTrafficSubscribes([]*user.Subscribe{
		{Id: 3},
		{Id: 1},
	}), trafficStatisticsTotalsFromMap(map[int64]trafficStatisticsTotal{
		3: {Download: 300, Upload: 30},
		2: {Download: 200, Upload: 20},
		1: {Download: 100, Upload: 10},
		4: {},
	}))
	want := []subscribeTrafficDeltaRow{
		{SubscribeID: 1, Download: 100, Upload: 10},
		{SubscribeID: 3, Download: 300, Upload: 30},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("delta rows = %#v, want %#v", got, want)
	}
}

func TestBuildSubscribeTrafficDeltaRowsSkipsMissingWithoutInputSizedCapacity(t *testing.T) {
	values := make(map[int64]trafficStatisticsTotal, viewutil.SmallRowsCapacityThreshold+20)
	for id := int64(1); id <= int64(viewutil.SmallRowsCapacityThreshold+20); id++ {
		values[id] = trafficStatisticsTotal{Download: id * 10, Upload: id}
	}

	got := buildSubscribeTrafficDeltaRows(indexTrafficSubscribes([]*user.Subscribe{
		{Id: 1},
	}), trafficStatisticsTotalsFromMap(values))
	if len(got) != 1 || got[0].SubscribeID != 1 {
		t.Fatalf("delta rows = %#v, want one row for subscribe 1", got)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("missing-heavy delta rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
}

func TestBuildCurrentTrafficTotalsHonorsResetBarrier(t *testing.T) {
	resetAt := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	subscribes := indexTrafficSubscribes([]*user.Subscribe{
		{Id: 1, TrafficResetAt: resetAt},
		{Id: 2, TrafficResetAt: time.UnixMilli(0)},
	})
	reports := []acceptedTrafficReport{
		{
			SampledAt: resetAt.Add(-5 * time.Second),
			Totals: trafficStatisticsTotalsFromMap(map[int64]trafficStatisticsTotal{
				1: {Download: 100, Upload: 10},
				2: {Download: 200, Upload: 20},
			}),
		},
		{
			SampledAt: resetAt.Add(5 * time.Second),
			Totals: trafficStatisticsTotalsFromMap(map[int64]trafficStatisticsTotal{
				1: {Download: 300, Upload: 30},
			}),
		},
	}
	got := trafficStatisticsTotalsMap(buildCurrentTrafficTotals(reports, subscribes))
	want := map[int64]trafficStatisticsTotal{
		1: {Download: 300, Upload: 30},
		2: {Download: 200, Upload: 20},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("current totals = %#v, want %#v", got, want)
	}
}

func TestTrafficMathAvoidsOverflow(t *testing.T) {
	if !trafficExceedsThreshold(math.MaxInt64, 1, math.MaxInt64-1) {
		t.Fatal("overflow-sized traffic must exceed threshold")
	}
	totals := newTrafficStatisticsTotals(1)
	totals.Add(1, math.MaxInt64, math.MaxInt64)
	totals.Add(1, 1, 1)
	got, _ := totals.Get(1)
	if got.Download != math.MaxInt64 || got.Upload != math.MaxInt64 {
		t.Fatalf("saturated totals = %#v", got)
	}
	reports := trafficStatisticsReportsFromTotals(totals)
	if len(reports) != 1 || reports[0].Download != math.MaxInt64 || reports[0].Upload != math.MaxInt64 {
		t.Fatalf("overflow-sized report was lost: %#v", reports)
	}
	batch := buildDailyTrafficBatch(time.Now().UTC(), 9, indexTrafficSubscribes([]*user.Subscribe{{Id: 1, UserId: 10}}), totals)
	if len(batch.UserRows) != 1 || batch.UserRows[0].Total != math.MaxInt64 {
		t.Fatalf("daily user traffic did not saturate: %#v", batch.UserRows)
	}
	if batch.Server.Download != math.MaxInt64 || batch.Server.Upload != math.MaxInt64 || batch.Server.Total != math.MaxInt64 {
		t.Fatalf("daily server traffic did not saturate: %#v", batch.Server)
	}
	if batch.Stat.Download != math.MaxInt64 || batch.Stat.Upload != math.MaxInt64 || batch.Stat.Total != math.MaxInt64 {
		t.Fatalf("daily traffic statistic did not saturate: %#v", batch.Stat)
	}
}

func TestUpsertDailyTrafficBatchSaturatesStoredCounters(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&trafficModel.DailyUserTraffic{},
		&trafficModel.DailyServerTraffic{},
		&trafficModel.DailyTrafficStat{},
	); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE UNIQUE INDEX uq_daily_user_traffic_test ON daily_user_traffic(date, user_id, subscribe_id)",
		"CREATE UNIQUE INDEX uq_daily_server_traffic_test ON daily_server_traffic(date, server_id)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	date := "2026-07-13"
	nearMax := int64(math.MaxInt64 - 5)
	if err := db.Create(&trafficModel.DailyUserTraffic{Date: date, UserId: 10, SubscribeId: 1, Download: nearMax, Upload: nearMax, Total: nearMax}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&trafficModel.DailyServerTraffic{Date: date, ServerId: 9, Download: nearMax, Upload: nearMax, Total: nearMax}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&trafficModel.DailyTrafficStat{Date: date, Download: nearMax, Upload: nearMax, Total: nearMax}).Error; err != nil {
		t.Fatal(err)
	}

	logic := NewTrafficStatisticsLogic(&svc.ServiceContext{})
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	if err := logic.upsertDailyTrafficBatch(ctx, db, now, dailyTrafficBatch{
		UserRows: []trafficModel.DailyUserTraffic{{Date: date, UserId: 10, SubscribeId: 1, Download: 10, Upload: 10, Total: 20}},
		Server:   trafficModel.DailyServerTraffic{Date: date, ServerId: 9, Download: 10, Upload: 10, Total: 20},
		Stat:     trafficModel.DailyTrafficStat{Date: date, Download: 10, Upload: 10, Total: 20},
	}); err != nil {
		t.Fatal(err)
	}

	var userRow trafficModel.DailyUserTraffic
	var serverRow trafficModel.DailyServerTraffic
	var statRow trafficModel.DailyTrafficStat
	if err := db.Where("date = ? AND user_id = ? AND subscribe_id = ?", date, 10, 1).First(&userRow).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("date = ? AND server_id = ?", date, 9).First(&serverRow).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("date = ?", date).First(&statRow).Error; err != nil {
		t.Fatal(err)
	}
	for name, values := range map[string][3]int64{
		"user":   {userRow.Download, userRow.Upload, userRow.Total},
		"server": {serverRow.Download, serverRow.Upload, serverRow.Total},
		"stat":   {statRow.Download, statRow.Upload, statRow.Total},
	} {
		if values != [3]int64{math.MaxInt64, math.MaxInt64, math.MaxInt64} {
			t.Fatalf("%s counters = %#v, want saturation", name, values)
		}
	}
}

func TestUpdateSubscribeTrafficDeltas(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
CREATE TABLE user_subscribe (
	id integer primary key,
	download integer not null default 0,
	upload integer not null default 0,
	updated_at datetime
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
INSERT INTO user_subscribe (id, download, upload) VALUES
	(1, 10, 1),
	(2, 20, 2),
	(3, 30, 3)
`).Error; err != nil {
		t.Fatal(err)
	}

	err = updateSubscribeTrafficDeltas(ctx, db, []subscribeTrafficDeltaRow{
		{SubscribeID: 1, Download: 100, Upload: 10},
		{SubscribeID: 3, Download: 300, Upload: 30},
	})
	if err != nil {
		t.Fatal(err)
	}

	var rows []user.Subscribe
	if err := db.Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	got := make(map[int64]trafficStatisticsTotal, len(rows))
	for _, row := range rows {
		got[row.Id] = trafficStatisticsTotal{Download: row.Download, Upload: row.Upload}
	}
	want := map[int64]trafficStatisticsTotal{
		1: {Download: 110, Upload: 11},
		2: {Download: 20, Upload: 2},
		3: {Download: 330, Upload: 33},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subscribe traffic = %#v, want %#v", got, want)
	}
}

func TestBuildDailyTrafficBatchAggregatesValidSubscribes(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 30, 0, 0, time.UTC)
	batch := buildDailyTrafficBatch(now, 9, indexTrafficSubscribes([]*user.Subscribe{
		{Id: 3, UserId: 30},
		{Id: 1, UserId: 10},
	}), trafficStatisticsTotalsFromMap(map[int64]trafficStatisticsTotal{
		3: {Download: 300, Upload: 30},
		2: {Download: 200, Upload: 20},
		1: {Download: 100, Upload: 10},
		4: {},
	}))

	if len(batch.UserRows) != 2 {
		t.Fatalf("user rows = %d, want 2", len(batch.UserRows))
	}
	if batch.UserRows[0].SubscribeId != 1 || batch.UserRows[0].UserId != 10 || batch.UserRows[0].Download != 100 || batch.UserRows[0].Upload != 10 || batch.UserRows[0].Total != 110 {
		t.Fatalf("first user row = %#v", batch.UserRows[0])
	}
	if batch.UserRows[1].SubscribeId != 3 || batch.UserRows[1].UserId != 30 || batch.UserRows[1].Download != 300 || batch.UserRows[1].Upload != 30 || batch.UserRows[1].Total != 330 {
		t.Fatalf("second user row = %#v", batch.UserRows[1])
	}
	if batch.Server.Date != "2026-07-06" || batch.Server.ServerId != 9 || batch.Server.Download != 400 || batch.Server.Upload != 40 || batch.Server.Total != 440 {
		t.Fatalf("server row = %#v", batch.Server)
	}
	if batch.Stat.Date != "2026-07-06" || batch.Stat.Download != 400 || batch.Stat.Upload != 40 || batch.Stat.Total != 440 {
		t.Fatalf("stat row = %#v", batch.Stat)
	}
}

func TestBuildDailyTrafficBatchesUsesSampleDates(t *testing.T) {
	subscribes := indexTrafficSubscribes([]*user.Subscribe{{Id: 1, UserId: 10}})
	reports := []acceptedTrafficReport{
		{
			SampledAt: time.Date(2026, 7, 9, 23, 59, 55, 0, time.UTC),
			Totals: trafficStatisticsTotalsFromMap(map[int64]trafficStatisticsTotal{
				1: {Download: 100, Upload: 10},
			}),
		},
		{
			SampledAt: time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
			Totals: trafficStatisticsTotalsFromMap(map[int64]trafficStatisticsTotal{
				1: {Download: 200, Upload: 20},
			}),
		},
	}
	batches := buildDailyTrafficBatches(reports, 9, subscribes)
	if len(batches) != 2 {
		t.Fatalf("batches = %#v", batches)
	}
	if batches[0].Stat.Date != "2026-07-09" || batches[0].Stat.Total != 110 {
		t.Fatalf("first daily batch = %#v", batches[0])
	}
	if batches[1].Stat.Date != "2026-07-10" || batches[1].Stat.Total != 220 {
		t.Fatalf("second daily batch = %#v", batches[1])
	}
}

func trafficStatisticsTotalsFromMap(values map[int64]trafficStatisticsTotal) trafficStatisticsTotals {
	totals := newTrafficStatisticsTotals(len(values))
	for subscribeID, total := range values {
		totals.Add(subscribeID, total.Download, total.Upload)
	}
	return totals
}

func trafficStatisticsTotalsMap(totals trafficStatisticsTotals) map[int64]trafficStatisticsTotal {
	result := make(map[int64]trafficStatisticsTotal, totals.Len())
	for _, row := range totals.rows {
		result[row.SubscribeID] = row.Total
	}
	return result
}
