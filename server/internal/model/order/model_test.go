package order

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOrderCacheColumnsStayMinimal(t *testing.T) {
	if want := []string{"id", "order_no"}; !reflect.DeepEqual(orderCacheColumns, want) {
		t.Fatalf("order cache columns = %#v, want %#v", orderCacheColumns, want)
	}
}

func TestInsertStoresMaskedClientIP(t *testing.T) {
	db := newOrderListTestDB(t)
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	data := &Order{OrderNo: "alpha-private-order", ClientIP: "203.0.113.42"}
	if err := NewModel(db, rds).Insert(context.Background(), data); err != nil {
		t.Fatal(err)
	}

	var stored Order
	if err := db.Where("order_no = ?", data.OrderNo).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ClientIP != "203.0.113.*" {
		t.Fatalf("stored client IP = %q, want masked value", stored.ClientIP)
	}
}

func TestFindOneByRequestIDScopesUserAndOrderType(t *testing.T) {
	db := newOrderListTestDB(t)
	rows := []Order{
		{UserId: 7, OrderNo: "alpha-idempotent-purchase", Type: 1, RequestId: "550e8400-e29b-41d4-a716-446655440000"},
		{UserId: 7, OrderNo: "alpha-idempotent-renewal", Type: 2, RequestId: "550e8400-e29b-41d4-a716-446655440000"},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindOneByRequestID(context.Background(), 7, 2, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatal(err)
	}
	if got.OrderNo != "alpha-idempotent-renewal" || got.Type != 2 || got.UserId != 7 {
		t.Fatalf("idempotent lookup returned %#v", got)
	}
}

func TestIsUserEligibleForNewOrderStopsAtFirstCompletedOrder(t *testing.T) {
	db := newOrderListTestDB(t)
	rows := []Order{
		{Id: 1, UserId: 7, OrderNo: "pending", Status: 1},
		{Id: 2, UserId: 7, OrderNo: "paid", Status: 2},
		{Id: 3, UserId: 7, OrderNo: "finished", Status: 5},
		{Id: 4, UserId: 8, OrderNo: "other-user", Status: 2},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	model := NewModel(db, nil)

	eligible, err := model.IsUserEligibleForNewOrder(context.Background(), 7)
	if err != nil || eligible {
		t.Fatalf("user 7 eligible/error = %v/%v, want false/nil", eligible, err)
	}
	eligible, err = model.IsUserEligibleForNewOrder(context.Background(), 9)
	if err != nil || !eligible {
		t.Fatalf("user 9 eligible/error = %v/%v, want true/nil", eligible, err)
	}
}

func TestFindOrderForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db := newOrderListTestDB(t)
	now := time.Unix(1700000000, 0)
	if err := db.Create(&Order{
		Id:              12,
		ParentId:        1,
		UserId:          2,
		OrderNo:         "alpha-order-12",
		Type:            1,
		Quantity:        2,
		Price:           1000,
		Amount:          1800,
		GiftAmount:      100,
		Discount:        200,
		Coupon:          "SAVE",
		CouponDiscount:  50,
		Commission:      10,
		PaymentId:       3,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade",
		Status:          2,
		SubscribeId:     4,
		SubscribeToken:  "subscribe-token",
		ProductName:     "product",
		ProductSnapshot: `{"heavy":true}`,
		ClientIP:        "203.0.113.7",
		CountryShort:    "US",
		Region:          "CA",
		Location:        "US · CA",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).(*customOrderModel).findOneForCache(ctx, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 12 || got.OrderNo != "alpha-order-12" {
		t.Fatalf("cache lookup returned %#v, want id/order_no", got)
	}
	if got.UserId != 0 || got.Amount != 0 || got.Method != "" || got.TradeNo != "" || got.ProductSnapshot != "" || got.ClientIP != "" || got.IsNew || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("cache lookup loaded non-key fields: %#v", got)
	}
}

func TestAdminOrderListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(adminOrderListColumns))
	for _, column := range adminOrderListColumns {
		columns[column] = struct{}{}
	}

	required := []string{
		"id",
		"user_id",
		"order_no",
		"type",
		"quantity",
		"price",
		"amount",
		"gift_amount",
		"discount",
		"coupon_discount",
		"payment_id",
		"fee_amount",
		"trade_no",
		"status",
		"subscribe_id",
		"product_name",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("admin order list columns must include %q", column)
		}
	}

	forbidden := []string{
		"coupon",
		"commission",
		"product_snapshot",
		"subscribe_token",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
		"created_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("admin order list columns must not include heavy/private column %q", column)
		}
	}
}

func TestAdminOrderListPaymentColumnsStayLightweight(t *testing.T) {
	columns := columnSet(adminOrderListPaymentColumns)
	for _, column := range []string{"id", "name", "platform"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("admin order list payment columns must include %q", column)
		}
	}
	for _, column := range []string{"description", "icon", "fee_mode", "fee_percent", "fee_amount", "domain", "config", "token", "enable", "created_at", "updated_at"} {
		if _, ok := columns[column]; ok {
			t.Fatalf("admin order list payment columns must not include heavy/private column %q", column)
		}
	}
}

func TestPublicOrderListColumnsStayLightweight(t *testing.T) {
	columns := columnSet(publicOrderListColumns)

	required := []string{
		"id",
		"user_id",
		"order_no",
		"type",
		"quantity",
		"price",
		"amount",
		"gift_amount",
		"discount",
		"coupon",
		"coupon_discount",
		"payment_id",
		"method",
		"fee_amount",
		"trade_no",
		"status",
		"subscribe_id",
		"product_name",
		"created_at",
		"updated_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("public order list columns must include %q", column)
		}
	}

	forbidden := []string{
		"subscribe_token",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"commission",
		"is_new",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("public order list columns must not include heavy/private column %q", column)
		}
	}
}

func TestPublicOrderDetailColumnsStayLightweight(t *testing.T) {
	columns := columnSet(publicOrderDetailColumns)

	for _, column := range publicOrderListColumns {
		if _, ok := columns[column]; !ok {
			t.Fatalf("public order detail columns must include public list column %q", column)
		}
	}
	if _, ok := columns["product_snapshot"]; !ok {
		t.Fatal("public order detail columns must include product_snapshot")
	}

	forbidden := []string{
		"subscribe_token",
		"client_ip",
		"country_short",
		"region",
		"location",
		"commission",
		"is_new",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("public order detail columns must not include private/admin-only column %q", column)
		}
	}
}

func TestInternalOrderDetailColumnsStayMinimal(t *testing.T) {
	columns := columnSet(internalOrderDetailColumns)

	for _, column := range []string{"id", "order_no", "quantity", "amount", "gift_amount"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("internal order detail columns must include %q", column)
		}
	}

	forbidden := []string{
		"user_id",
		"subscribe_id",
		"subscribe_token",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"commission",
		"payment_id",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("internal order detail columns must not include unrelated/private column %q", column)
		}
	}
}

func TestInternalOrderSubOrderColumnsStayMinimal(t *testing.T) {
	columns := columnSet(internalOrderSubOrderColumns)

	for _, column := range []string{"id", "parent_id", "quantity", "amount", "gift_amount", "status"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("internal sub-order columns must include %q", column)
		}
	}

	forbidden := []string{
		"order_no",
		"user_id",
		"subscribe_id",
		"subscribe_token",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"commission",
		"payment_id",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("internal sub-order columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderStatusUpdateColumnsStayMinimal(t *testing.T) {
	columns := columnSet(orderStatusUpdateColumns)

	for _, column := range []string{"id", "order_no", "status"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order status update columns must include %q", column)
		}
	}

	forbidden := []string{
		"user_id",
		"amount",
		"payment_id",
		"method",
		"trade_no",
		"subscribe_id",
		"subscribe_token",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order status update columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderCheckoutColumnsStayFocused(t *testing.T) {
	columns := columnSet(orderCheckoutColumns)

	required := []string{
		"id",
		"user_id",
		"order_no",
		"type",
		"quantity",
		"amount",
		"payment_id",
		"method",
		"status",
		"subscribe_id",
		"subscribe_token",
		"product_snapshot",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order checkout columns must include %q", column)
		}
	}

	forbidden := []string{
		"price",
		"discount",
		"coupon",
		"coupon_discount",
		"commission",
		"fee_amount",
		"trade_no",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order checkout columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderActivationColumnsStayFocused(t *testing.T) {
	columns := columnSet(orderActivationColumns)

	required := []string{
		"id",
		"user_id",
		"order_no",
		"type",
		"quantity",
		"price",
		"amount",
		"fee_amount",
		"coupon",
		"commission",
		"method",
		"trade_no",
		"status",
		"subscribe_id",
		"subscribe_token",
		"product_name",
		"product_snapshot",
		"is_new",
		"created_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order activation columns must include %q", column)
		}
	}

	forbidden := []string{
		"parent_id",
		"gift_amount",
		"discount",
		"coupon_discount",
		"payment_id",
		"client_ip",
		"country_short",
		"region",
		"location",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order activation columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderPurchaseQueryColumnsStayFocused(t *testing.T) {
	columns := columnSet(orderPurchaseQueryColumns)

	required := []string{
		"id",
		"order_no",
		"quantity",
		"price",
		"amount",
		"discount",
		"coupon",
		"coupon_discount",
		"payment_id",
		"fee_amount",
		"status",
		"subscribe_id",
		"product_snapshot",
		"created_at",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order purchase query columns must include %q", column)
		}
	}

	forbidden := []string{
		"user_id",
		"type",
		"gift_amount",
		"commission",
		"method",
		"trade_no",
		"subscribe_token",
		"product_name",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order purchase query columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderBalanceCheckoutColumnsStayMinimal(t *testing.T) {
	columns := columnSet(orderBalanceCheckoutColumns)

	for _, column := range []string{"id", "user_id", "order_no", "amount", "status"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order balance checkout columns must include %q", column)
		}
	}

	forbidden := []string{
		"payment_id",
		"method",
		"settlement_amount",
		"settlement_currency",
		"trade_no",
		"subscribe_id",
		"subscribe_token",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order balance checkout columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderPaymentNotifyColumnsStayMinimal(t *testing.T) {
	columns := columnSet(orderPaymentNotifyColumns)

	for _, column := range []string{"id", "order_no", "amount", "settlement_amount", "settlement_currency", "status"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order payment notify columns must include %q", column)
		}
	}

	forbidden := []string{
		"user_id",
		"type",
		"payment_id",
		"method",
		"trade_no",
		"gift_amount",
		"subscribe_id",
		"subscribe_token",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order payment notify columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderCloseColumnsStayFocused(t *testing.T) {
	columns := columnSet(orderCloseColumns)

	required := []string{
		"id",
		"user_id",
		"order_no",
		"type",
		"payment_id",
		"method",
		"trade_no",
		"gift_amount",
		"status",
		"subscribe_id",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order close columns must include %q", column)
		}
	}

	forbidden := []string{
		"amount",
		"price",
		"discount",
		"coupon",
		"coupon_discount",
		"commission",
		"fee_amount",
		"subscribe_token",
		"product_name",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order close columns must not include unrelated/private column %q", column)
		}
	}
}

func TestOrderListPaymentColumnsStayLightweight(t *testing.T) {
	columns := columnSet(orderListPaymentColumns)

	required := []string{
		"id",
		"name",
		"platform",
		"icon",
		"description",
		"fee_mode",
		"fee_percent",
		"fee_amount",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order list payment columns must include %q", column)
		}
	}

	forbidden := []string{
		"domain",
		"config",
		"token",
		"enable",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order list payment columns must not include private/config column %q", column)
		}
	}
}

func TestPublicOrderListSubscribeColumnsStayMinimal(t *testing.T) {
	if want := []string{"id", "name"}; !reflect.DeepEqual(publicOrderListSubscribeColumns, want) {
		t.Fatalf("public order list subscribe columns = %#v, want %#v", publicOrderListSubscribeColumns, want)
	}
}

func TestOrderTypeUsesSubscribeSummaryOnlyForSubscriptionOrders(t *testing.T) {
	for _, orderType := range []uint8{orderTypeSubscribe, orderTypeRenewal, orderTypeResetTraffic} {
		if !orderTypeUsesSubscribeSummary(orderType) {
			t.Fatalf("order type %d should use subscribe summary", orderType)
		}
	}
	for _, orderType := range []uint8{4, 5, 0, 99} {
		if orderTypeUsesSubscribeSummary(orderType) {
			t.Fatalf("order type %d should not use subscribe summary", orderType)
		}
	}
}

func TestOrderListSubscribeIndexUsesLinearPathForSmallInput(t *testing.T) {
	rows := []modelSubscribe.Subscribe{
		{Id: 0, Name: "Invalid"},
		{Id: 1, Name: "Basic"},
		{Id: 2, Name: "Premium"},
	}
	index := newOrderListSubscribeIndex(rows)
	if index.byID != nil {
		t.Fatal("small subscribe index should not allocate map index")
	}
	if got := index.Find(2); got == nil || got.Name != "Premium" {
		t.Fatalf("Find(2) = %#v", got)
	}
	if got := index.Find(9); got != nil {
		t.Fatalf("Find(9) = %#v, want nil", got)
	}
	if got := index.Find(0); got != nil {
		t.Fatalf("Find(0) = %#v, want nil", got)
	}
}

func TestOrderListSubscribeIndexUpgradesLargeInputToMap(t *testing.T) {
	rows := make([]modelSubscribe.Subscribe, 0, smallOrderListSubscribeIndexThreshold+1)
	for id := int64(1); id <= smallOrderListSubscribeIndexThreshold+1; id++ {
		rows = append(rows, modelSubscribe.Subscribe{Id: id, Name: "Plan"})
	}
	index := newOrderListSubscribeIndex(rows)
	if index.byID == nil {
		t.Fatal("large subscribe index should allocate map index")
	}
	lastID := int64(smallOrderListSubscribeIndexThreshold + 1)
	if got := index.Find(lastID); got == nil || got.Id != lastID {
		t.Fatalf("Find(%d) = %#v", lastID, got)
	}
}

func TestOrderListSubscribeIndexAvoidsMapForLargeDuplicateInput(t *testing.T) {
	rows := make([]modelSubscribe.Subscribe, 0, smallOrderListSubscribeIndexThreshold+20)
	for i := 0; i < cap(rows); i++ {
		rows = append(rows, modelSubscribe.Subscribe{Id: 2, Name: "Plan"})
	}
	index := newOrderListSubscribeIndex(rows)
	if index.byID != nil {
		t.Fatal("large duplicate subscribe index should not allocate map index")
	}
	if len(index.rows) != 1 {
		t.Fatalf("subscribe index rows = %d, want 1", len(index.rows))
	}
	if cap(index.rows) != smallOrderListSubscribeIndexThreshold+1 {
		t.Fatalf("subscribe index cap = %d, want small initial cap", cap(index.rows))
	}
	if got := index.Find(2); got == nil || got.Id != 2 {
		t.Fatalf("Find(2) = %#v", got)
	}
}

func TestOrderListSubscribeIndexBoundsSparseUpgradeCapacity(t *testing.T) {
	rows := make([]modelSubscribe.Subscribe, 0, smallOrderListSubscribeIndexThreshold+20)
	for id := int64(1); id <= smallOrderListSubscribeIndexThreshold+1; id++ {
		rows = append(rows, modelSubscribe.Subscribe{Id: id, Name: "Plan"})
	}
	for len(rows) < smallOrderListSubscribeIndexThreshold+20 {
		rows = append(rows, modelSubscribe.Subscribe{Id: 1, Name: "Duplicate"})
	}
	index := newOrderListSubscribeIndex(rows)
	if index.byID == nil {
		t.Fatal("subscribe index should allocate map after actual unique subscribes exceed threshold")
	}
	if len(index.rows) != smallOrderListSubscribeIndexThreshold+1 {
		t.Fatalf("subscribe index rows = %d, want %d", len(index.rows), smallOrderListSubscribeIndexThreshold+1)
	}
	if maxCap := (smallOrderListSubscribeIndexThreshold + 1) * 2; cap(index.rows) > maxCap {
		t.Fatalf("subscribe index cap = %d, want <= %d", cap(index.rows), maxCap)
	}
}

func TestOrderListInt64CollectorBoundsSparseUpgradeCapacity(t *testing.T) {
	collector := newOrderListInt64Collector(smallOrderListSubscribeIndexThreshold + 40)
	for id := int64(1); id <= smallOrderListSubscribeIndexThreshold+1; id++ {
		collector.Add(id)
	}
	for i := 0; i < smallOrderListSubscribeIndexThreshold+40; i++ {
		collector.Add(1)
	}

	got := collector.Values()
	if len(got) != smallOrderListSubscribeIndexThreshold+1 {
		t.Fatalf("subscribe ids len = %d, want %d", len(got), smallOrderListSubscribeIndexThreshold+1)
	}
	if maxCap := (smallOrderListSubscribeIndexThreshold + 1) * 2; cap(got) > maxCap {
		t.Fatalf("subscribe ids cap = %d, want <= %d", cap(got), maxCap)
	}
}

func TestCollectPublicOrderListSubscribeIDsSkipsInvalidWithoutInputSizedCapacity(t *testing.T) {
	list := make([]*Details, smallOrderListSubscribeIndexThreshold+20)
	list[1] = &Details{Type: orderTypeSubscribe, SubscribeId: 7}
	list[2] = &Details{Type: orderTypeRenewal, SubscribeId: 7}
	list[3] = &Details{Type: 4, SubscribeId: 8}
	list[smallOrderListSubscribeIndexThreshold+10] = &Details{Type: orderTypeResetTraffic, SubscribeId: 7}

	got := collectPublicOrderListSubscribeIDs(list)
	want := []int64{7}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectPublicOrderListSubscribeIDs() = %#v, want %#v", got, want)
	}
	if cap(got) != smallOrderListSubscribeIndexThreshold+1 {
		t.Fatalf("subscribe id cap = %d, want small initial cap", cap(got))
	}
}

func TestHydratePublicOrderListSubscribesNormalizesIDs(t *testing.T) {
	ctx := context.Background()
	db := newOrderListTestDB(t)
	if err := db.Create([]*modelSubscribe.Subscribe{
		{Id: 1, Name: "Basic"},
		{Id: 2, Name: "Premium"},
	}).Error; err != nil {
		t.Fatal(err)
	}

	var queries []string
	queryDB := db.Session(&gorm.Session{
		Logger: orderSubscribeQueryLogger{queries: &queries},
	})
	list := []*Details{
		{Type: orderTypeSubscribe, SubscribeId: 1},
		{Type: orderTypeRenewal, SubscribeId: 1},
		{Type: 4, SubscribeId: 1},
		{Type: orderTypeResetTraffic, SubscribeId: 2},
		{Type: orderTypeSubscribe},
		nil,
	}

	if err := hydratePublicOrderListSubscribes(ctx, queryDB, list); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 {
		t.Fatalf("subscribe queries = %d, want 1: %#v", len(queries), queries)
	}
	if strings.Contains(queries[0], "1,1") {
		t.Fatalf("subscribe query kept duplicate ids: %s", queries[0])
	}
	if list[0].Subscribe == nil || list[0].Subscribe.Name != "Basic" {
		t.Fatalf("subscribe order not hydrated: %#v", list[0].Subscribe)
	}
	if list[1].Subscribe == nil || list[1].Subscribe.Name != "Basic" {
		t.Fatalf("renewal order not hydrated: %#v", list[1].Subscribe)
	}
	if list[2].Subscribe != nil {
		t.Fatalf("non-subscribe order should not hydrate summary: %#v", list[2].Subscribe)
	}
	if list[3].Subscribe == nil || list[3].Subscribe.Name != "Premium" {
		t.Fatalf("reset-traffic order not hydrated: %#v", list[3].Subscribe)
	}
}

func TestOrderDetailSubscribeColumnsStayLightweight(t *testing.T) {
	columns := columnSet(orderDetailSubscribeColumns)

	required := []string{
		"id",
		"name",
		"unit_price",
		"unit_time",
		"traffic",
		"speed_limit",
		"device_limit",
		"show_original_price",
	}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order detail subscribe columns must include %q", column)
		}
	}

	forbidden := []string{
		"language",
		"description",
		"discount",
		"replacement",
		"inventory",
		"quota",
		"nodes",
		"node_tags",
		"show",
		"user_visible",
		"sell",
		"sort",
		"deduction_ratio",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
		"renewal_reset",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order detail subscribe columns must not include heavy selector/discount column %q", column)
		}
	}
}

func TestFindOneForStatusUpdateSelectsOnlyStatusColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:              501,
		UserId:          7,
		OrderNo:         "alpha-order-501",
		Type:            1,
		Quantity:        2,
		Amount:          1900,
		GiftAmount:      100,
		PaymentId:       9,
		Method:          "stripe",
		TradeNo:         "trade-501",
		Status:          1,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductSnapshot: `{"name":"Snapshot"}`,
		ClientIP:        "203.0.113.8",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneForStatusUpdate(context.Background(), 501)
	if err != nil {
		t.Fatalf("find status update order: %v", err)
	}
	if got.Id != 501 || got.OrderNo != "alpha-order-501" || got.Status != 1 {
		t.Fatalf("status update query dropped required fields: %#v", got)
	}
	if got.Amount != 0 || got.PaymentId != 0 || got.Method != "" || got.TradeNo != "" || got.SubscribeId != 0 || got.ProductSnapshot != "" || got.ClientIP != "" || got.IsNew {
		t.Fatalf("status update query loaded unrelated/private fields: %#v", got)
	}
}

func TestFindOneForCheckoutSelectsOnlyCheckoutColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:              504,
		UserId:          7,
		OrderNo:         "alpha-order-504",
		Type:            1,
		Quantity:        2,
		Price:           1200,
		Amount:          1900,
		GiftAmount:      100,
		Discount:        50,
		Coupon:          "WELCOME",
		CouponDiscount:  20,
		Commission:      60,
		PaymentId:       9,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-504",
		Status:          1,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro",
		ProductSnapshot: `{"name":"Snapshot"}`,
		ClientIP:        "203.0.113.8",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneForCheckout(context.Background(), "alpha-order-504")
	if err != nil {
		t.Fatalf("find checkout order: %v", err)
	}
	if got.Id != 504 || got.UserId != 7 || got.OrderNo != "alpha-order-504" || got.Type != 1 || got.Quantity != 2 || got.Amount != 1900 || got.PaymentId != 9 || got.Method != "stripe" || got.Status != 1 || got.SubscribeId != 10 || got.SubscribeToken != "sub-secret-token" || got.ProductSnapshot == "" {
		t.Fatalf("checkout query dropped required fields: %#v", got)
	}
	if got.Price != 0 || got.GiftAmount != 0 || got.Discount != 0 || got.Coupon != "" || got.CouponDiscount != 0 || got.Commission != 0 || got.FeeAmount != 0 || got.TradeNo != "" || got.ProductName != "" || got.ClientIP != "" || got.IsNew {
		t.Fatalf("checkout query loaded unrelated/private fields: %#v", got)
	}
}

func TestFindOneForActivationSelectsOnlyActivationColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:              510,
		ParentId:        99,
		UserId:          7,
		OrderNo:         "alpha-order-510",
		Type:            1,
		Quantity:        2,
		Price:           1200,
		Amount:          1900,
		GiftAmount:      100,
		Discount:        50,
		Coupon:          "WELCOME",
		CouponDiscount:  20,
		Commission:      60,
		PaymentId:       9,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-510",
		Status:          2,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro",
		ProductSnapshot: `{"name":"Snapshot"}`,
		ClientIP:        "203.0.113.8",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneForActivation(context.Background(), "alpha-order-510")
	if err != nil {
		t.Fatalf("find activation order: %v", err)
	}
	if got.Id != 510 || got.UserId != 7 || got.OrderNo != "alpha-order-510" || got.Type != 1 || got.Quantity != 2 || got.Price != 1200 || got.Amount != 1900 || got.FeeAmount != 30 || got.Coupon != "WELCOME" || got.Commission != 60 || got.Method != "stripe" || got.TradeNo != "trade-510" || got.Status != 2 || got.SubscribeId != 10 || got.SubscribeToken != "sub-secret-token" || got.ProductName != "Pro" || got.ProductSnapshot == "" || !got.IsNew || !got.CreatedAt.Equal(now) {
		t.Fatalf("activation query dropped required fields: %#v", got)
	}
	if got.ParentId != 0 || got.GiftAmount != 0 || got.Discount != 0 || got.CouponDiscount != 0 || got.PaymentId != 0 || got.ClientIP != "" || got.CountryShort != "" || got.Region != "" || got.Location != "" || !got.UpdatedAt.IsZero() {
		t.Fatalf("activation query loaded unrelated/private fields: %#v", got)
	}
}

func TestFindOneForPurchaseQuerySelectsOnlyPurchaseQueryColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:              507,
		UserId:          7,
		OrderNo:         "alpha-order-507",
		Type:            1,
		Quantity:        2,
		Price:           1200,
		Amount:          1900,
		GiftAmount:      100,
		Discount:        50,
		Coupon:          "WELCOME",
		CouponDiscount:  20,
		Commission:      60,
		PaymentId:       9,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-507",
		Status:          1,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro",
		ProductSnapshot: `{"name":"Snapshot"}`,
		ClientIP:        "203.0.113.8",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneForPurchaseQuery(context.Background(), "alpha-order-507")
	if err != nil {
		t.Fatalf("find purchase query order: %v", err)
	}
	if got.Id != 507 || got.OrderNo != "alpha-order-507" || got.Quantity != 2 || got.Price != 1200 || got.Amount != 1900 || got.Discount != 50 || got.Coupon != "WELCOME" || got.CouponDiscount != 20 || got.PaymentId != 9 || got.FeeAmount != 30 || got.Status != 1 || got.SubscribeId != 10 || got.ProductSnapshot == "" || !got.CreatedAt.Equal(now) {
		t.Fatalf("purchase query dropped required fields: %#v", got)
	}
	if got.UserId != 0 || got.Type != 0 || got.GiftAmount != 0 || got.Commission != 0 || got.Method != "" || got.TradeNo != "" || got.SubscribeToken != "" || got.ProductName != "" || got.ClientIP != "" || got.IsNew || !got.UpdatedAt.IsZero() {
		t.Fatalf("purchase query loaded unrelated/private fields: %#v", got)
	}
}

func TestFindOneForPaymentNotifySelectsOnlyNotifyColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:                 508,
		UserId:             7,
		OrderNo:            "alpha-order-508",
		Type:               1,
		Quantity:           2,
		Price:              1200,
		Amount:             1900,
		GiftAmount:         100,
		Discount:           50,
		Coupon:             "WELCOME",
		CouponDiscount:     20,
		Commission:         60,
		PaymentId:          9,
		Method:             "stripe",
		FeeAmount:          30,
		SettlementAmount:   13800,
		SettlementCurrency: "CNY",
		TradeNo:            "trade-508",
		Status:             1,
		SubscribeId:        10,
		SubscribeToken:     "sub-secret-token",
		ProductName:        "Pro",
		ProductSnapshot:    `{"name":"Snapshot"}`,
		ClientIP:           "203.0.113.8",
		CountryShort:       "US",
		Region:             "Colorado",
		Location:           "US - Colorado",
		IsNew:              true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneForPaymentNotify(context.Background(), "alpha-order-508")
	if err != nil {
		t.Fatalf("find payment notify order: %v", err)
	}
	if got.Id != 508 || got.OrderNo != "alpha-order-508" || got.Amount != 1900 || got.SettlementAmount != 13800 || got.SettlementCurrency != "CNY" || got.Status != 1 {
		t.Fatalf("payment notify query dropped required fields: %#v", got)
	}
	if got.UserId != 0 || got.Type != 0 || got.PaymentId != 0 || got.Method != "" || got.TradeNo != "" || got.GiftAmount != 0 || got.SubscribeId != 0 || got.ProductSnapshot != "" || got.ClientIP != "" || got.IsNew {
		t.Fatalf("payment notify query loaded unrelated/private fields: %#v", got)
	}
}

func TestSetSettlementIfCompatibleIsImmutableAndPendingOnly(t *testing.T) {
	db := newOrderListTestDB(t)
	ctx := context.Background()
	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	if err := db.Create(&Order{Id: 511, OrderNo: "alpha-order-511", Amount: 100, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	model := NewModel(db, rds)
	current := &Order{Id: 511, OrderNo: "alpha-order-511", Status: 1}

	for attempt := 0; attempt < 2; attempt++ {
		updated, err := model.SetSettlementIfCompatible(ctx, current, "CNY", 725)
		if err != nil || !updated {
			t.Fatalf("compatible settlement attempt %d = %v, %v", attempt, updated, err)
		}
	}
	updated, err := model.SetSettlementIfCompatible(ctx, current, "CNY", 800)
	if err != nil {
		t.Fatal(err)
	}
	if updated {
		t.Fatal("conflicting settlement amount must not overwrite the snapshot")
	}

	var stored Order
	if err := db.First(&stored, 511).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SettlementAmount != 725 || stored.SettlementCurrency != "CNY" {
		t.Fatalf("stored settlement = %d %q", stored.SettlementAmount, stored.SettlementCurrency)
	}
	if err := db.Model(&Order{}).Where("id = ?", 511).Update("status", 2).Error; err != nil {
		t.Fatal(err)
	}
	updated, err = model.SetSettlementIfCompatible(ctx, current, "CNY", 725)
	if err != nil || updated {
		t.Fatalf("paid order settlement update = %v, %v; want false, nil", updated, err)
	}
}

func TestFindOneForCloseSelectsOnlyCloseColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:              509,
		UserId:          7,
		OrderNo:         "alpha-order-509",
		Type:            1,
		Quantity:        2,
		Price:           1200,
		Amount:          1900,
		GiftAmount:      100,
		Discount:        50,
		Coupon:          "WELCOME",
		CouponDiscount:  20,
		Commission:      60,
		PaymentId:       9,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-509",
		Status:          1,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro",
		ProductSnapshot: `{"name":"Snapshot"}`,
		ClientIP:        "203.0.113.8",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneForClose(context.Background(), "alpha-order-509")
	if err != nil {
		t.Fatalf("find close order: %v", err)
	}
	if got.Id != 509 || got.UserId != 7 || got.OrderNo != "alpha-order-509" || got.Type != 1 || got.PaymentId != 9 || got.Method != "stripe" || got.TradeNo != "trade-509" || got.GiftAmount != 100 || got.Status != 1 || got.SubscribeId != 10 {
		t.Fatalf("close query dropped required fields: %#v", got)
	}
	if got.Amount != 0 || got.Price != 0 || got.Discount != 0 || got.Coupon != "" || got.CouponDiscount != 0 || got.Commission != 0 || got.FeeAmount != 0 || got.SubscribeToken != "" || got.ProductName != "" || got.ProductSnapshot != "" || got.ClientIP != "" || got.IsNew {
		t.Fatalf("close query loaded unrelated/private fields: %#v", got)
	}
}

func TestOrderUnsubscribeRefundColumnsStayMinimal(t *testing.T) {
	columns := columnSet(orderUnsubscribeRefundColumns)

	for _, column := range []string{"id", "order_no", "method", "gift_amount"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("order unsubscribe refund columns must include %q", column)
		}
	}

	forbidden := []string{
		"user_id",
		"type",
		"quantity",
		"price",
		"amount",
		"discount",
		"coupon",
		"coupon_discount",
		"commission",
		"payment_id",
		"fee_amount",
		"trade_no",
		"status",
		"subscribe_id",
		"subscribe_token",
		"product_name",
		"product_snapshot",
		"client_ip",
		"country_short",
		"region",
		"location",
		"is_new",
		"created_at",
		"updated_at",
	}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("order unsubscribe refund columns must not include unrelated/private column %q", column)
		}
	}
}

func TestFindOneForUnsubscribeRefundSelectsOnlyRefundColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:              512,
		UserId:          7,
		OrderNo:         "alpha-order-512",
		Type:            1,
		Quantity:        2,
		Price:           1200,
		Amount:          1900,
		GiftAmount:      100,
		Discount:        50,
		Coupon:          "WELCOME",
		CouponDiscount:  20,
		Commission:      60,
		PaymentId:       9,
		Method:          "balance",
		FeeAmount:       30,
		TradeNo:         "trade-512",
		Status:          2,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro",
		ProductSnapshot: `{"name":"Snapshot"}`,
		ClientIP:        "203.0.113.8",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneForUnsubscribeRefund(context.Background(), 512)
	if err != nil {
		t.Fatalf("find unsubscribe refund order: %v", err)
	}
	if got.Id != 512 || got.OrderNo != "alpha-order-512" || got.Method != "balance" || got.GiftAmount != 100 {
		t.Fatalf("unsubscribe refund query dropped required fields: %#v", got)
	}
	if got.UserId != 0 || got.Type != 0 || got.Quantity != 0 || got.Price != 0 || got.Amount != 0 || got.Discount != 0 || got.Coupon != "" || got.CouponDiscount != 0 || got.Commission != 0 || got.PaymentId != 0 || got.FeeAmount != 0 || got.TradeNo != "" || got.Status != 0 || got.SubscribeId != 0 || got.SubscribeToken != "" || got.ProductName != "" || got.ProductSnapshot != "" || got.ClientIP != "" || got.IsNew || !got.CreatedAt.IsZero() || !got.UpdatedAt.IsZero() {
		t.Fatalf("unsubscribe refund query loaded unrelated/private fields: %#v", got)
	}
}

func TestUpdateAdminStatusFieldsUpdatesOnlyRequestedFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	if err := db.Create(&Order{
		Id:              502,
		UserId:          7,
		OrderNo:         "alpha-order-502",
		Type:            1,
		Quantity:        2,
		Amount:          1900,
		GiftAmount:      100,
		PaymentId:       8,
		Method:          "old-platform",
		TradeNo:         "old-trade",
		Status:          1,
		SubscribeId:     10,
		ProductSnapshot: `{"name":"Snapshot"}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range []string{cacheOrderIdPrefix + "502", cacheOrderNoPrefix + "alpha-order-502"} {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	current := &Order{Id: 502, OrderNo: "alpha-order-502"}
	if err := model.UpdateAdminStatusFields(ctx, current, OrderFieldsUpdate{
		PaymentID: UpdateValue(int64(9)),
		Method:    UpdateValue("stripe"),
		TradeNo:   UpdateValue("new-trade"),
		Status:    UpdateValue(uint8(2)),
	}); err != nil {
		t.Fatalf("update admin status fields: %v", err)
	}

	var got Order
	if err := db.First(&got, 502).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}
	if got.PaymentId != 9 || got.Method != "stripe" || got.TradeNo != "new-trade" || got.Status != 2 {
		t.Fatalf("status fields not updated: %#v", got)
	}
	if got.Amount != 1900 || got.GiftAmount != 100 || got.ProductSnapshot != `{"name":"Snapshot"}` {
		t.Fatalf("unrelated fields changed: %#v", got)
	}
	for _, key := range []string{cacheOrderIdPrefix + "502", cacheOrderNoPrefix + "alpha-order-502"} {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateFieldsUpdatesOnlyRequestedFieldsAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db := newOrderListTestDB(t)
	if err := db.Create(&Order{
		Id:              505,
		UserId:          7,
		OrderNo:         "alpha-order-505",
		Amount:          1900,
		TradeNo:         "old-trade",
		Status:          1,
		ProductSnapshot: `{"name":"Snapshot"}`,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range []string{cacheOrderIdPrefix + "505", cacheOrderNoPrefix + "alpha-order-505"} {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	current := &Order{Id: 505, OrderNo: "alpha-order-505"}
	if err := model.UpdateFields(ctx, current, OrderFieldsUpdate{TradeNo: UpdateValue("new-trade")}); err != nil {
		t.Fatalf("update fields: %v", err)
	}

	var got Order
	if err := db.First(&got, 505).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}
	if got.TradeNo != "new-trade" || got.Status != 1 || got.Amount != 1900 || got.ProductSnapshot != `{"name":"Snapshot"}` {
		t.Fatalf("unexpected order after field update: %#v", got)
	}
	for _, key := range []string{cacheOrderIdPrefix + "505", cacheOrderNoPrefix + "alpha-order-505"} {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateOrderStatusUsesLightweightIdentityAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db := newOrderListTestDB(t)
	if err := db.Create(&Order{
		Id:              503,
		OrderNo:         "alpha-order-503",
		Amount:          1900,
		ProductSnapshot: `{"name":"Snapshot"}`,
		Status:          1,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range []string{cacheOrderIdPrefix + "503", cacheOrderNoPrefix + "alpha-order-503"} {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	if err := NewModel(db, rds).UpdateOrderStatus(ctx, "alpha-order-503", 3); err != nil {
		t.Fatalf("update order status: %v", err)
	}
	var got Order
	if err := db.First(&got, 503).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}
	if got.Status != 3 || got.Amount != 1900 || got.ProductSnapshot != `{"name":"Snapshot"}` {
		t.Fatalf("unexpected order after status update: %#v", got)
	}
	for _, key := range []string{cacheOrderIdPrefix + "503", cacheOrderNoPrefix + "alpha-order-503"} {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestUpdateStatusIfCurrentConditionallyUpdatesAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db := newOrderListTestDB(t)
	if err := db.Create(&Order{
		Id:              510,
		OrderNo:         "alpha-order-510",
		Amount:          1900,
		ProductSnapshot: `{"name":"Snapshot"}`,
		Status:          1,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range []string{cacheOrderIdPrefix + "510", cacheOrderNoPrefix + "alpha-order-510"} {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	model := NewModel(db, rds)
	current := &Order{Id: 510, OrderNo: "alpha-order-510"}
	updated, err := model.UpdateStatusIfCurrent(ctx, current, 1, 3)
	if err != nil {
		t.Fatalf("update status if current: %v", err)
	}
	if !updated {
		t.Fatalf("updated = false, want true")
	}
	var got Order
	if err := db.First(&got, 510).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}
	if got.Status != 3 || got.Amount != 1900 || got.ProductSnapshot != `{"name":"Snapshot"}` {
		t.Fatalf("unexpected order after conditional status update: %#v", got)
	}
	for _, key := range []string{cacheOrderIdPrefix + "510", cacheOrderNoPrefix + "alpha-order-510"} {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}

	updated, err = model.UpdateStatusIfCurrent(ctx, current, 1, 4)
	if err != nil {
		t.Fatalf("second update status if current: %v", err)
	}
	if updated {
		t.Fatalf("updated = true for stale status")
	}
}

func TestMarkPaidUsesLightweightIdentityAndClearsCache(t *testing.T) {
	ctx := context.Background()
	db := newOrderListTestDB(t)
	if err := db.Create(&Order{
		Id:              506,
		OrderNo:         "alpha-order-506",
		Amount:          1900,
		ProductSnapshot: `{"name":"Snapshot"}`,
		Status:          1,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	for _, key := range []string{cacheOrderIdPrefix + "506", cacheOrderNoPrefix + "alpha-order-506"} {
		if err := rds.Set(ctx, key, "cached", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
	}

	paid, err := NewModel(db, rds).MarkPaid(ctx, "alpha-order-506", "trade-506")
	if err != nil {
		t.Fatalf("mark paid: %v", err)
	}
	if !paid {
		t.Fatalf("paid = false, want true")
	}
	var got Order
	if err := db.First(&got, 506).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}
	if got.Status != 2 || got.TradeNo != "trade-506" || got.Amount != 1900 || got.ProductSnapshot != `{"name":"Snapshot"}` {
		t.Fatalf("unexpected order after mark paid: %#v", got)
	}
	for _, key := range []string{cacheOrderIdPrefix + "506", cacheOrderNoPrefix + "alpha-order-506"} {
		if exists := rds.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("cache key %q still exists", key)
		}
	}
}

func TestQueryOrderListByPageUsesLightweightColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	enabled := true
	now := time.Now().UTC()

	if err := db.Create(&modelPayment.Payment{
		Id:          9,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "stripe.svg",
		Domain:      "https://pay.example.test",
		Config:      `{"secret_key":"sk_test_private"}`,
		Description: "Credit card",
		FeeMode:     3,
		FeePercent:  250,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "payment-token",
	}).Error; err != nil {
		t.Fatalf("create payment: %v", err)
	}

	if err := db.Create(&modelSubscribe.Subscribe{
		Id:                10,
		Name:              "Pro",
		Language:          "en",
		Description:       "Pro plan",
		UnitPrice:         1200,
		UnitTime:          "month",
		Discount:          `[{"quantity":12,"discount":0.75}]`,
		Replacement:       10,
		Inventory:         100,
		Traffic:           1024,
		SpeedLimit:        200,
		DeviceLimit:       5,
		Quota:             2,
		Nodes:             "1,2,3",
		NodeTags:          "premium,edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              9,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
	}).Error; err != nil {
		t.Fatalf("create subscribe: %v", err)
	}

	if err := db.Create(&Order{
		Id:              101,
		UserId:          7,
		OrderNo:         "alpha-order-101",
		Type:            1,
		Quantity:        1,
		Price:           1200,
		Amount:          1230,
		GiftAmount:      20,
		Discount:        0,
		Coupon:          "WELCOME",
		CouponDiscount:  100,
		Commission:      66,
		PaymentId:       9,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-101",
		Status:          2,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro snapshot",
		ProductSnapshot: `{"name":"Pro snapshot"}`,
		ClientIP:        "203.0.113.7",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	model := NewModel(db, nil)
	total, list, err := model.QueryOrderListByPage(context.Background(), 1, 10, 0, 7, 0, "")
	if err != nil {
		t.Fatalf("query order list: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	item := list[0]
	if item.ProductName != "Pro snapshot" {
		t.Fatalf("product name not loaded: %#v", item)
	}
	if item.ProductSnapshot != "" {
		t.Fatalf("product snapshot should not be loaded for order list: %#v", item)
	}
	if item.ClientIP != "" || item.CountryShort != "" || item.Region != "" || item.Location != "" {
		t.Fatalf("private location fields should not be loaded: %#v", item)
	}
	if item.SubscribeToken != "" || item.Commission != 0 || item.IsNew {
		t.Fatalf("private or admin-only fields should not be loaded: %#v", item)
	}
	if item.Payment == nil {
		t.Fatal("payment preload is nil")
	}
	if item.Payment.Name != "Stripe" || item.Payment.Platform != "stripe" || item.Payment.FeeMode != 3 || item.Payment.FeePercent != 250 || item.Payment.FeeAmount != 30 {
		t.Fatalf("payment public fields not loaded: %#v", item.Payment)
	}
	if item.Payment.Domain != "" || item.Payment.Config != "" || item.Payment.Token != "" || item.Payment.Enable != nil {
		t.Fatalf("payment private/config fields should not be loaded: %#v", item.Payment)
	}
	if item.Subscribe == nil || item.Subscribe.Name != "Pro" {
		t.Fatalf("subscribe name preload should remain available for order list display: %#v", item.Subscribe)
	}
	if item.Subscribe.Description != "" || item.Subscribe.UnitPrice != 0 || item.Subscribe.Traffic != 0 || item.Subscribe.AllowRenewal != nil {
		t.Fatalf("order list subscribe should only load id/name: %#v", item.Subscribe)
	}
}

func TestQueryAdminOrderListByPageUsesLightweightColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	enabled := true
	now := time.Now().UTC()

	if err := db.Create(&modelPayment.Payment{
		Id:          19,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "stripe.svg",
		Domain:      "https://pay.example.test",
		Config:      `{"secret_key":"sk_test_private"}`,
		Description: "Credit card",
		FeeMode:     3,
		FeePercent:  250,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "payment-token",
	}).Error; err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if err := db.Create(&Order{
		Id:              111,
		UserId:          7,
		OrderNo:         "alpha-order-admin-111",
		Type:            1,
		Quantity:        1,
		Price:           1200,
		Amount:          1230,
		GiftAmount:      20,
		Discount:        0,
		Coupon:          "WELCOME",
		CouponDiscount:  100,
		Commission:      66,
		PaymentId:       19,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-111",
		Status:          2,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro snapshot",
		ProductSnapshot: `{"name":"Pro snapshot"}`,
		ClientIP:        "203.0.113.7",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	total, list, err := NewModel(db, nil).QueryAdminOrderListByPage(context.Background(), 1, 10, 0, 7, 0, "")
	if err != nil {
		t.Fatalf("query admin order list: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("total/len = %d/%d, want 1/1", total, len(list))
	}
	item := list[0]
	if item.Id != 111 || item.OrderNo != "alpha-order-admin-111" || item.Amount != 1230 || item.ProductName != "Pro snapshot" || item.UpdatedAt.IsZero() {
		t.Fatalf("admin order list dropped required fields: %#v", item)
	}
	if item.Coupon != "" || item.Commission != 0 || item.ProductSnapshot != "" || item.SubscribeToken != "" || item.ClientIP != "" || item.CountryShort != "" || item.Region != "" || item.Location != "" || item.IsNew || !item.CreatedAt.IsZero() {
		t.Fatalf("admin order list loaded redundant/private fields: %#v", item)
	}
	if item.Payment == nil {
		t.Fatal("payment preload is nil")
	}
	if item.Payment.Id != 19 || item.Payment.Name != "Stripe" || item.Payment.Platform != "stripe" {
		t.Fatalf("payment summary fields not loaded: %#v", item.Payment)
	}
	if item.Payment.Description != "" || item.Payment.Icon != "" || item.Payment.FeeMode != 0 || item.Payment.FeePercent != 0 || item.Payment.FeeAmount != 0 || item.Payment.Domain != "" || item.Payment.Config != "" || item.Payment.Token != "" || item.Payment.Enable != nil {
		t.Fatalf("admin payment summary loaded heavy/private fields: %#v", item.Payment)
	}
}

func TestQueryOrderListByPageFiltersOwnedOrdersByStatusAndSearch(t *testing.T) {
	db := newOrderListTestDB(t)
	orders := []Order{
		{Id: 1201, UserId: 7, OrderNo: "alpha-owned-pending", TradeNo: "pending-ref", Status: 1},
		{Id: 1202, UserId: 7, OrderNo: "alpha-owned-paid", TradeNo: "trade-search", Status: 2},
		{Id: 1203, UserId: 8, OrderNo: "alpha-other-paid", TradeNo: "trade-search", Status: 2},
	}
	if err := db.Create(&orders).Error; err != nil {
		t.Fatalf("create orders: %v", err)
	}

	total, list, err := NewModel(db, nil).QueryOrderListByPage(
		context.Background(), 1, 10, 2, 7, 0, "trade-search",
	)
	if err != nil {
		t.Fatalf("query order list: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].OrderNo != "alpha-owned-paid" {
		t.Fatalf("filtered orders = %d/%#v, want owned paid order", total, list)
	}
}

func TestFindOneAdminDetailsLoadsDetailWithoutPrivateFields(t *testing.T) {
	db := newOrderListTestDB(t)
	enabled := true
	now := time.Now().UTC()
	if err := db.Create(&modelPayment.Payment{
		Id: 29, Name: "Stripe", Platform: "stripe", Config: `{"secret":"private"}`, Token: "private-token", Enable: &enabled,
	}).Error; err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if err := db.Create(&Order{
		Id: 1210, UserId: 7, OrderNo: "alpha-admin-detail", Type: 1, Quantity: 2,
		Price: 2000, Amount: 1800, Coupon: "DETAIL", Commission: 77, PaymentId: 29,
		Method: "stripe", TradeNo: "trade-detail", Status: 2, ProductName: "Pro snapshot",
		ProductSnapshot: `{"version":1,"kind":"subscribe","name":"Pro snapshot"}`,
		SubscribeToken:  "private-subscribe-token", ClientIP: "203.0.113.8", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	got, err := NewModel(db, nil).FindOneAdminDetails(context.Background(), 1210)
	if err != nil {
		t.Fatalf("find admin detail: %v", err)
	}
	if got.Coupon != "DETAIL" || got.Commission != 77 || got.Method != "stripe" || got.ProductSnapshot == "" || got.CreatedAt.IsZero() {
		t.Fatalf("admin detail dropped required fields: %#v", got)
	}
	if got.SubscribeToken != "" || got.ClientIP != "" || got.CountryShort != "" || got.Region != "" || got.Location != "" {
		t.Fatalf("admin detail loaded private fields: %#v", got)
	}
	if got.Payment == nil || got.Payment.Name != "Stripe" || got.Payment.Platform != "stripe" {
		t.Fatalf("payment not loaded: %#v", got.Payment)
	}
	if got.Payment.Config != "" || got.Payment.Token != "" || got.Payment.Enable != nil {
		t.Fatalf("payment secrets loaded: %#v", got.Payment)
	}
}

func TestQueryOrderListByPageHydratesSubscribeOnlyForSubscriptionOrders(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()

	if err := db.Create(&[]modelSubscribe.Subscribe{
		{Id: 10, Name: "Subscription Plan"},
		{Id: 55, Name: "Membership ID Collision"},
	}).Error; err != nil {
		t.Fatalf("create subscribes: %v", err)
	}
	if err := db.Create(&[]Order{
		{
			Id:          101,
			UserId:      7,
			OrderNo:     "alpha-order-subscribe",
			Type:        orderTypeSubscribe,
			Status:      2,
			SubscribeId: 10,
			ProductName: "Subscription Plan",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			Id:          102,
			UserId:      7,
			OrderNo:     "alpha-order-membership",
			Type:        5,
			Status:      2,
			SubscribeId: 55,
			ProductName: "Membership Plan",
			CreatedAt:   now.Add(time.Minute),
			UpdatedAt:   now.Add(time.Minute),
		},
	}).Error; err != nil {
		t.Fatalf("create orders: %v", err)
	}

	total, list, err := NewModel(db, nil).QueryOrderListByPage(context.Background(), 1, 10, 0, 7, 0, "")
	if err != nil {
		t.Fatalf("query order list: %v", err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("total/len = %d/%d, want 2/2", total, len(list))
	}
	if list[0].Id != 102 || list[0].Subscribe != nil {
		t.Fatalf("membership order should not hydrate subscribe summary: %#v", list[0])
	}
	if list[1].Id != 101 || list[1].Subscribe == nil || list[1].Subscribe.Name != "Subscription Plan" {
		t.Fatalf("subscription order did not hydrate subscribe summary: %#v", list[1])
	}
}

func TestOrderListByPageEmptyResultKeepsStableSlice(t *testing.T) {
	db := newOrderListTestDB(t)
	model := NewModel(db, nil)

	adminTotal, adminList, err := model.QueryAdminOrderListByPage(context.Background(), -1, 0, 0, 7, 0, "")
	if err != nil {
		t.Fatalf("query admin order list: %v", err)
	}
	if adminTotal != 0 || adminList == nil || len(adminList) != 0 {
		t.Fatalf("admin total/list = %d/%#v, want 0/non-nil empty", adminTotal, adminList)
	}

	publicTotal, publicList, err := model.QueryOrderListByPage(context.Background(), -1, 0, 0, 7, 0, "")
	if err != nil {
		t.Fatalf("query public order list: %v", err)
	}
	if publicTotal != 0 || publicList == nil || len(publicList) != 0 {
		t.Fatalf("public total/list = %d/%#v, want 0/non-nil empty", publicTotal, publicList)
	}
}

func TestOrderListByPagePreallocatesPageCapacity(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()
	for i := int64(1); i <= 3; i++ {
		if err := db.Create(&Order{
			Id:          200 + i,
			UserId:      7,
			OrderNo:     "alpha-order-capacity-" + strconv.FormatInt(i, 10),
			Type:        5,
			Status:      2,
			ProductName: "Capacity Plan",
			CreatedAt:   now.Add(time.Duration(i) * time.Minute),
			UpdatedAt:   now.Add(time.Duration(i) * time.Minute),
		}).Error; err != nil {
			t.Fatalf("create order %d: %v", i, err)
		}
	}

	model := NewModel(db, nil)
	total, publicList, err := model.QueryOrderListByPage(context.Background(), 1, 2, 0, 7, 0, "")
	if err != nil {
		t.Fatalf("query public order list: %v", err)
	}
	if total != 3 || len(publicList) != 2 || cap(publicList) != 2 {
		t.Fatalf("public total/len/cap = %d/%d/%d, want 3/2/2", total, len(publicList), cap(publicList))
	}

	total, adminList, err := model.QueryAdminOrderListByPage(context.Background(), 1, 10, 0, 7, 0, "")
	if err != nil {
		t.Fatalf("query admin order list: %v", err)
	}
	if total != 3 || len(adminList) != 3 || cap(adminList) != 3 {
		t.Fatalf("admin total/len/cap = %d/%d/%d, want 3/3/3", total, len(adminList), cap(adminList))
	}
}

func TestFindOneDetailsByOrderNoUsesLightweightPublicColumns(t *testing.T) {
	db := newOrderListTestDB(t)
	enabled := true
	now := time.Now().UTC()

	if err := db.Create(&modelPayment.Payment{
		Id:          9,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "stripe.svg",
		Domain:      "https://pay.example.test",
		Config:      `{"secret_key":"sk_test_private"}`,
		Description: "Credit card",
		FeeMode:     3,
		FeePercent:  250,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "payment-token",
	}).Error; err != nil {
		t.Fatalf("create payment: %v", err)
	}

	if err := db.Create(&modelSubscribe.Subscribe{
		Id:                10,
		Name:              "Pro",
		Language:          "en",
		Description:       "Pro plan",
		UnitPrice:         1200,
		UnitTime:          "month",
		Discount:          `[{"quantity":12,"discount":0.75}]`,
		Replacement:       10,
		Inventory:         100,
		Traffic:           1024,
		SpeedLimit:        200,
		DeviceLimit:       5,
		Quota:             2,
		Nodes:             "1,2,3",
		NodeTags:          "premium,edge",
		Show:              &enabled,
		UserVisible:       &enabled,
		Sell:              &enabled,
		Sort:              9,
		AllowRenewal:      &enabled,
		AllowResetTraffic: &enabled,
		ResetCycle:        2,
		RenewalReset:      &enabled,
		ShowOriginalPrice: true,
	}).Error; err != nil {
		t.Fatalf("create subscribe: %v", err)
	}

	if err := db.Create(&Order{
		Id:              101,
		UserId:          7,
		OrderNo:         "alpha-order-101",
		Type:            1,
		Quantity:        1,
		Price:           1200,
		Amount:          1230,
		GiftAmount:      20,
		Discount:        0,
		Coupon:          "WELCOME",
		CouponDiscount:  100,
		Commission:      66,
		PaymentId:       9,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-101",
		Status:          2,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro snapshot",
		ProductSnapshot: `{"name":"Pro snapshot"}`,
		ClientIP:        "203.0.113.7",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	model := NewModel(db, nil)
	item, err := model.FindOneDetailsByOrderNo(context.Background(), "alpha-order-101")
	if err != nil {
		t.Fatalf("find order detail: %v", err)
	}

	if item.ProductName != "Pro snapshot" || item.ProductSnapshot == "" {
		t.Fatalf("product snapshot fields not loaded: %#v", item)
	}
	if item.ClientIP != "" || item.CountryShort != "" || item.Region != "" || item.Location != "" {
		t.Fatalf("private location fields should not be loaded: %#v", item)
	}
	if item.SubscribeToken != "" || item.Commission != 0 || item.IsNew {
		t.Fatalf("private or admin-only fields should not be loaded: %#v", item)
	}
	if item.Payment == nil {
		t.Fatal("payment preload is nil")
	}
	if item.Payment.Name != "Stripe" || item.Payment.Platform != "stripe" || item.Payment.FeeMode != 3 || item.Payment.FeePercent != 250 || item.Payment.FeeAmount != 30 {
		t.Fatalf("payment public fields not loaded: %#v", item.Payment)
	}
	if item.Payment.Domain != "" || item.Payment.Config != "" || item.Payment.Token != "" || item.Payment.Enable != nil {
		t.Fatalf("payment private/config fields should not be loaded: %#v", item.Payment)
	}
	if item.Subscribe == nil || item.Subscribe.Name != "Pro" {
		t.Fatalf("subscribe preload should remain available for legacy order detail compatibility: %#v", item.Subscribe)
	}
	if item.Subscribe.UnitPrice != 1200 || item.Subscribe.UnitTime != "month" || item.Subscribe.Traffic != 1024 || item.Subscribe.SpeedLimit != 200 || item.Subscribe.DeviceLimit != 5 || !item.Subscribe.ShowOriginalPrice {
		t.Fatalf("subscribe detail fields not loaded: %#v", item.Subscribe)
	}
	if item.Subscribe.Description != "" || item.Subscribe.Discount != "" || item.Subscribe.Nodes != "" || item.Subscribe.NodeTags != "" || item.Subscribe.AllowRenewal != nil {
		t.Fatalf("subscribe non-payment fields should not be loaded: %#v", item.Subscribe)
	}
}

func TestFindOneDetailsUsesInternalAmountColumnsOnly(t *testing.T) {
	db := newOrderListTestDB(t)
	now := time.Now().UTC()

	if err := db.Create(&Order{
		Id:              201,
		UserId:          7,
		OrderNo:         "alpha-order-201",
		Type:            1,
		Quantity:        2,
		Price:           1000,
		Amount:          1900,
		GiftAmount:      100,
		Discount:        100,
		Coupon:          "WELCOME",
		CouponDiscount:  100,
		Commission:      66,
		PaymentId:       9,
		Method:          "stripe",
		FeeAmount:       30,
		TradeNo:         "trade-201",
		Status:          2,
		SubscribeId:     10,
		SubscribeToken:  "sub-secret-token",
		ProductName:     "Pro snapshot",
		ProductSnapshot: `{"name":"Pro snapshot"}`,
		ClientIP:        "203.0.113.7",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create parent order: %v", err)
	}

	if err := db.Create(&Order{
		Id:              202,
		ParentId:        201,
		UserId:          7,
		OrderNo:         "alpha-order-202",
		Type:            2,
		Quantity:        3,
		Amount:          3300,
		GiftAmount:      300,
		Commission:      88,
		PaymentId:       9,
		Status:          2,
		SubscribeId:     10,
		SubscribeToken:  "sub-child-secret-token",
		ProductSnapshot: `{"name":"Child snapshot"}`,
		ClientIP:        "203.0.113.8",
		CountryShort:    "US",
		Region:          "Colorado",
		Location:        "US - Colorado",
		IsNew:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create child order: %v", err)
	}

	model := NewModel(db, nil)
	item, err := model.FindOneDetails(context.Background(), 201)
	if err != nil {
		t.Fatalf("find internal order detail: %v", err)
	}

	if item.OrderNo != "alpha-order-201" || item.Quantity != 2 || item.Amount != 1900 || item.GiftAmount != 100 {
		t.Fatalf("internal amount fields not loaded: %#v", item)
	}
	if item.UserId != 0 || item.SubscribeId != 0 || item.ProductName != "" || item.ProductSnapshot != "" || item.PaymentId != 0 {
		t.Fatalf("unrelated root fields should not be loaded: %#v", item)
	}
	if item.Subscribe != nil || item.Payment != nil {
		t.Fatalf("internal detail should not preload subscribe/payment: %#v", item)
	}
	if item.ClientIP != "" || item.CountryShort != "" || item.Region != "" || item.Location != "" {
		t.Fatalf("private location fields should not be loaded: %#v", item)
	}
	if item.SubscribeToken != "" || item.Commission != 0 || item.IsNew {
		t.Fatalf("private or unrelated root fields should not be loaded: %#v", item)
	}
	if len(item.SubOrders) != 1 {
		t.Fatalf("len(suborders) = %d, want 1", len(item.SubOrders))
	}
	subOrder := item.SubOrders[0]
	if subOrder.ParentId != 201 || subOrder.Quantity != 3 || subOrder.Amount != 3300 || subOrder.GiftAmount != 300 || subOrder.Status != 2 {
		t.Fatalf("internal sub-order amount fields not loaded: %#v", subOrder)
	}
	if subOrder.OrderNo != "" || subOrder.UserId != 0 || subOrder.SubscribeId != 0 || subOrder.ProductSnapshot != "" || subOrder.PaymentId != 0 {
		t.Fatalf("unrelated sub-order fields should not be loaded: %#v", subOrder)
	}
	if subOrder.ClientIP != "" || subOrder.CountryShort != "" || subOrder.Region != "" || subOrder.Location != "" {
		t.Fatalf("private sub-order location fields should not be loaded: %#v", subOrder)
	}
	if subOrder.SubscribeToken != "" || subOrder.Commission != 0 || subOrder.IsNew {
		t.Fatalf("private or unrelated sub-order fields should not be loaded: %#v", subOrder)
	}
}

func newOrderListTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Order{}, &modelPayment.Payment{}, &modelSubscribe.Subscribe{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestUserScopedOrderLookupsRejectOtherOwners(t *testing.T) {
	db := newOrderListTestDB(t)
	if err := db.Create(&Order{Id: 3001, UserId: 7, OrderNo: "owned-order", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	model := NewModel(db, nil)

	if _, err := model.FindOneDetailsByOrderNoForUser(context.Background(), "owned-order", 7); err != nil {
		t.Fatalf("owner detail lookup failed: %v", err)
	}
	if _, err := model.FindOneDetailsByOrderNoForUser(context.Background(), "owned-order", 8); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("other-user detail lookup error = %v, want record not found", err)
	}
	if _, err := model.FindOneForCloseByUser(context.Background(), "owned-order", 7); err != nil {
		t.Fatalf("owner close lookup failed: %v", err)
	}
	if _, err := model.FindOneForCloseByUser(context.Background(), "owned-order", 8); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("other-user close lookup error = %v, want record not found", err)
	}
}

func columnSet(columns []string) map[string]struct{} {
	result := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		result[column] = struct{}{}
	}
	return result
}

type orderSubscribeQueryLogger struct {
	queries *[]string
}

func (l orderSubscribeQueryLogger) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (l orderSubscribeQueryLogger) Info(context.Context, string, ...interface{}) {}

func (l orderSubscribeQueryLogger) Warn(context.Context, string, ...interface{}) {}

func (l orderSubscribeQueryLogger) Error(context.Context, string, ...interface{}) {}

func (l orderSubscribeQueryLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	if strings.Contains(sql, "FROM `subscribe`") || strings.Contains(sql, `FROM "subscribe"`) {
		*l.queries = append(*l.queries, sql)
	}
}
