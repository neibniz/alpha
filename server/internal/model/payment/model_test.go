package payment

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type paymentConfigMarshaler interface {
	Marshal() ([]byte, error)
}

func TestPaymentConfigMarshalShapes(t *testing.T) {
	tests := []struct {
		name  string
		value paymentConfigMarshaler
		want  string
	}{
		{
			name:  "stripe",
			value: &StripeConfig{PublicKey: "pk", SecretKey: "sk", WebhookSecret: "whsec", Payment: "card"},
			want:  `{"public_key":"pk","secret_key":"sk","webhook_secret":"whsec","payment":"card"}`,
		},
		{
			name:  "alipay f2f",
			value: &AlipayF2FConfig{AppId: "app", PrivateKey: "private", PublicKey: "public", InvoiceName: "invoice", Sandbox: true},
			want:  `{"app_id":"app","private_key":"private","public_key":"public","invoice_name":"invoice","sandbox":true}`,
		},
		{
			name:  "epay",
			value: &EPayConfig{Pid: "pid", Url: "https://pay.example.test", Key: "key", Type: "alipay"},
			want:  `{"pid":"pid","url":"https://pay.example.test","key":"key","type":"alipay"}`,
		},
		{
			name:  "crypto saas",
			value: &CryptoSaaSConfig{Endpoint: "https://crypto.example.test", AccountID: "acct", SecretKey: "secret", Type: "usdt"},
			want:  `{"endpoint":"https://crypto.example.test","account_id":"acct","secret_key":"secret","type":"usdt"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.Marshal()
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPaymentConfigMarshalNilReceiverKeepsEmptyObjectShape(t *testing.T) {
	var cfg *StripeConfig
	got, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("nil StripeConfig.Marshal() error = %v", err)
	}
	if string(got) != "{}" {
		t.Fatalf("nil StripeConfig.Marshal() = %s, want {}", got)
	}
}

func TestPaymentConfigUnmarshalDirectAlias(t *testing.T) {
	var cfg EPayConfig
	if err := cfg.Unmarshal([]byte(`{"pid":"pid","url":"https://pay.example.test","key":"key","type":"alipay"}`)); err != nil {
		t.Fatalf("EPayConfig.Unmarshal() error = %v", err)
	}
	if cfg.Pid != "pid" || cfg.Url != "https://pay.example.test" || cfg.Key != "key" || cfg.Type != "alipay" {
		t.Fatalf("EPayConfig.Unmarshal() = %#v", cfg)
	}
}

func TestPaymentListColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentListColumns))
	for _, column := range paymentListColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "platform", "icon", "domain", "description", "fee_mode", "fee_percent", "fee_amount", "enable", "token"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("payment list columns must include %q", column)
		}
	}

	if _, ok := columns["config"]; ok {
		t.Fatal("payment list columns must not include config")
	}
}

func TestPaymentAvailableColumnsStayPublic(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentAvailableColumns))
	for _, column := range paymentAvailableColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "platform", "description", "icon", "fee_mode", "fee_percent", "fee_amount"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("payment available columns must include %q", column)
		}
	}

	forbidden := []string{"config", "token", "domain"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("payment available columns must not include %q", column)
		}
	}
}

func TestPaymentPublicOrderColumnsStayLightweight(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentPublicOrderColumns))
	for _, column := range paymentPublicOrderColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "name", "platform", "description", "icon", "fee_mode", "fee_percent", "fee_amount", "enable"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("payment public order columns must include %q", column)
		}
	}

	forbidden := []string{"config", "token", "domain"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("payment public order columns must not include %q", column)
		}
	}
}

func TestPaymentOrderRecordColumnsStayMinimal(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentOrderRecordColumns))
	for _, column := range paymentOrderRecordColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "platform", "token"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("payment order record columns must include %q", column)
		}
	}

	forbidden := []string{"name", "description", "icon", "domain", "config", "fee_mode", "fee_percent", "fee_amount", "enable"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("payment order record columns must not include %q", column)
		}
	}
}

func TestPaymentCheckoutColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentCheckoutColumns))
	for _, column := range paymentCheckoutColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "platform", "domain", "config", "enable", "token"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("payment checkout columns must include %q", column)
		}
	}

	forbidden := []string{"name", "description", "icon", "fee_mode", "fee_percent", "fee_amount"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("payment checkout columns must not include %q", column)
		}
	}
}

func TestPaymentNotifyColumnsStayFocused(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentNotifyColumns))
	for _, column := range paymentNotifyColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "platform", "domain", "config", "token"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("payment notify columns must include %q", column)
		}
	}

	forbidden := []string{"name", "description", "icon", "fee_mode", "fee_percent", "fee_amount", "enable"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("payment notify columns must not include %q", column)
		}
	}
}

func TestPaymentGatewayCheckColumnsStayMinimal(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentGatewayCheckColumns))
	for _, column := range paymentGatewayCheckColumns {
		columns[column] = struct{}{}
	}

	required := []string{"id", "platform", "config"}
	for _, column := range required {
		if _, ok := columns[column]; !ok {
			t.Fatalf("payment gateway check columns must include %q", column)
		}
	}

	forbidden := []string{"name", "description", "icon", "domain", "fee_mode", "fee_percent", "fee_amount", "enable", "token"}
	for _, column := range forbidden {
		if _, ok := columns[column]; ok {
			t.Fatalf("payment gateway check columns must not include %q", column)
		}
	}
}

func TestPaymentCacheColumnsStayMinimal(t *testing.T) {
	if want := []string{"id", "token"}; !reflect.DeepEqual(paymentCacheColumns, want) {
		t.Fatalf("payment cache columns = %#v, want %#v", paymentCacheColumns, want)
	}
}

func TestPaymentAdminDetailColumnsIncludeConfigOnlyOnDemand(t *testing.T) {
	columns := make(map[string]struct{}, len(paymentAdminDetailSelectColumns(false)))
	for _, column := range paymentAdminDetailSelectColumns(false) {
		columns[column] = struct{}{}
	}

	if _, ok := columns["token"]; !ok {
		t.Fatal("payment admin detail columns must include token for cache invalidation")
	}
	if _, ok := columns["config"]; ok {
		t.Fatal("payment admin detail columns must not include config by default")
	}

	withConfig := make(map[string]struct{}, len(paymentAdminDetailSelectColumns(true)))
	for _, column := range paymentAdminDetailSelectColumns(true) {
		withConfig[column] = struct{}{}
	}
	if _, ok := withConfig["config"]; !ok {
		t.Fatal("payment admin detail columns must include config when requested")
	}
}

func TestFindAllPaymentsSelectsOnlyListColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	disabled := false
	if err := db.Create([]*Payment{
		{
			Id:          2,
			Name:        "Stripe",
			Platform:    "stripe",
			Icon:        "stripe.svg",
			Domain:      "https://notify.example.com",
			Config:      `{"secret_key":"sk_live_private"}`,
			Description: "card",
			FeeMode:     3,
			FeePercent:  5,
			FeeAmount:   30,
			Enable:      &enabled,
			Token:       "stripe-token",
		},
		{
			Id:          1,
			Name:        "Balance",
			Platform:    "balance",
			Icon:        "balance.svg",
			Domain:      "https://balance.example.com",
			Config:      `{"private":"config"}`,
			Description: "wallet",
			FeeMode:     0,
			FeePercent:  0,
			FeeAmount:   0,
			Enable:      &disabled,
			Token:       "balance-token",
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("FindAll returned %d rows, want 2", len(got))
	}
	if got[0].Id != 1 || got[0].Name != "Balance" || got[0].Platform != "balance" || got[0].Token != "balance-token" || got[0].Enable == nil || *got[0].Enable {
		t.Fatalf("FindAll first row = %#v, want balance list fields", got[0])
	}
	if got[1].Id != 2 || got[1].Name != "Stripe" || got[1].Platform != "stripe" || got[1].Token != "stripe-token" || got[1].Enable == nil || !*got[1].Enable {
		t.Fatalf("FindAll second row = %#v, want stripe list fields", got[1])
	}
	for _, item := range got {
		if item.Config != "" {
			t.Fatalf("FindAll loaded payment config: %#v", item)
		}
	}
}

func TestFindListByPagePreallocatesPageCapacity(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create([]*Payment{
		{Id: 1, Name: "one", Platform: "one", Config: `{}`, Enable: &enabled, Token: "one-token"},
		{Id: 2, Name: "two", Platform: "two", Config: `{}`, Enable: &enabled, Token: "two-token"},
		{Id: 3, Name: "three", Platform: "three", Config: `{}`, Enable: &enabled, Token: "three-token"},
	}).Error; err != nil {
		t.Fatal(err)
	}

	total, list, err := NewModel(db, nil).FindListByPage(ctx, 1, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(list) != 2 || cap(list) != 2 {
		t.Fatalf("payment total/len/cap = %d/%d/%d, want 3/2/2", total, len(list), cap(list))
	}
}

func TestPaymentListMethodsReturnNonNilEmptySlices(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, nil)
	all, err := model.FindAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if all == nil || len(all) != 0 {
		t.Fatalf("all payments = %#v, want non-nil empty slice", all)
	}

	available, err := model.FindAvailableMethods(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if available == nil || len(available) != 0 {
		t.Fatalf("available payments = %#v, want non-nil empty slice", available)
	}

	total, page, err := model.FindListByPage(ctx, 1, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || page == nil || len(page) != 0 {
		t.Fatalf("payment page = total %d list %#v, want total 0 and non-nil empty slice", total, page)
	}
}

func TestFindPaymentForCacheSelectsOnlyCacheKeys(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&Payment{
		Id:          10,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "icon",
		Domain:      "https://notify.example.com",
		Config:      `{"secret_key":"sk_live_private"}`,
		Description: "card",
		FeeMode:     3,
		FeePercent:  5,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "cache-token",
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewModel(db, nil).(*customPaymentModel).findOneForCache(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 10 || got.Token != "cache-token" {
		t.Fatalf("cache lookup returned %#v, want id/token", got)
	}
	if got.Name != "" || got.Platform != "" || got.Icon != "" || got.Domain != "" || got.Config != "" || got.Description != "" || got.FeeMode != 0 || got.FeePercent != 0 || got.FeeAmount != 0 || got.Enable != nil {
		t.Fatalf("cache lookup loaded non-key fields: %#v", got)
	}
}

func TestFindOneByPaymentTokenSelectsOnlyNotifyColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&Payment{
		Id:          13,
		Name:        "EPay",
		Platform:    "epay",
		Icon:        "icon",
		Domain:      "https://notify.example.com",
		Config:      `{"key":"private"}`,
		Description: "gateway",
		FeeMode:     3,
		FeePercent:  5,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "notify-token",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneByPaymentToken(ctx, "notify-token")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 13 || got.Platform != "epay" || got.Domain != "https://notify.example.com" || got.Config == "" || got.Token != "notify-token" {
		t.Fatalf("notify query dropped required fields: %#v", got)
	}
	if got.Name != "" || got.Description != "" || got.Icon != "" || got.FeeMode != 0 || got.FeePercent != 0 || got.FeeAmount != 0 || got.Enable != nil {
		t.Fatalf("notify query loaded unnecessary fields: %#v", got)
	}
}

func TestFindOneForPublicOrderSelectsOnlyPublicOrderColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&Payment{
		Id:          7,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "icon",
		Domain:      "https://notify.example.com",
		Config:      `{"secret_key":"sk_live_private"}`,
		Description: "card",
		FeeMode:     3,
		FeePercent:  5,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "private-token",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForPublicOrder(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 7 || got.Name != "Stripe" || got.Platform != "stripe" || got.Icon != "icon" || got.Description != "card" || got.FeeMode != 3 || got.FeePercent != 5 || got.FeeAmount != 30 || got.Enable == nil || !*got.Enable {
		t.Fatalf("public order query dropped required fields: %#v", got)
	}
	if got.Config != "" || got.Token != "" || got.Domain != "" {
		t.Fatalf("public order query loaded private/config fields: %#v", got)
	}
}

func TestFindOneForOrderRecordSelectsOnlyOrderRecordColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&Payment{
		Id:          8,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "icon",
		Domain:      "https://notify.example.com",
		Config:      `{"secret_key":"sk_live_private"}`,
		Description: "card",
		FeeMode:     3,
		FeePercent:  5,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "order-token",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForOrderRecord(ctx, 8)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 8 || got.Platform != "stripe" || got.Token != "order-token" {
		t.Fatalf("order record query dropped required fields: %#v", got)
	}
	if got.Name != "" || got.Description != "" || got.Icon != "" || got.Domain != "" || got.Config != "" || got.FeeMode != 0 || got.FeePercent != 0 || got.FeeAmount != 0 || got.Enable != nil {
		t.Fatalf("order record query loaded unnecessary fields: %#v", got)
	}
}

func TestFindOneForCheckoutSelectsOnlyCheckoutColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&Payment{
		Id:          12,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "icon",
		Domain:      "https://notify.example.com",
		Config:      `{"secret_key":"sk_live_private"}`,
		Description: "card",
		FeeMode:     3,
		FeePercent:  5,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "checkout-token",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForCheckout(ctx, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 12 || got.Platform != "stripe" || got.Domain != "https://notify.example.com" || got.Config == "" || got.Enable == nil || !*got.Enable || got.Token != "checkout-token" {
		t.Fatalf("checkout query dropped required fields: %#v", got)
	}
	if got.Name != "" || got.Description != "" || got.Icon != "" || got.FeeMode != 0 || got.FeePercent != 0 || got.FeeAmount != 0 {
		t.Fatalf("checkout query loaded unnecessary fields: %#v", got)
	}
}

func TestFindOneForGatewayCheckSelectsOnlyGatewayColumns(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	if err := db.Create(&Payment{
		Id:          14,
		Name:        "Stripe",
		Platform:    "stripe",
		Icon:        "icon",
		Domain:      "https://notify.example.com",
		Config:      `{"secret_key":"sk_live_private"}`,
		Description: "card",
		FeeMode:     3,
		FeePercent:  5,
		FeeAmount:   30,
		Enable:      &enabled,
		Token:       "gateway-token",
	}).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})

	got, err := NewModel(db, rds).FindOneForGatewayCheck(ctx, 14)
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != 14 || got.Platform != "stripe" || got.Config == "" {
		t.Fatalf("gateway check query dropped required fields: %#v", got)
	}
	if got.Name != "" || got.Description != "" || got.Icon != "" || got.Domain != "" || got.FeeMode != 0 || got.FeePercent != 0 || got.FeeAmount != 0 || got.Enable != nil || got.Token != "" {
		t.Fatalf("gateway check query loaded unnecessary fields: %#v", got)
	}
}

func TestPaymentCacheKeysIncludePublicOrderKey(t *testing.T) {
	keys := (&defaultPaymentModel{}).getCacheKeys(&Payment{Id: 9, Token: "token"})
	if !containsPaymentCacheKey(keys, cachePaymentPublicOrderPrefix+"9") {
		t.Fatalf("payment cache keys = %#v, want public order cache key", keys)
	}
	if !containsPaymentCacheKey(keys, cachePaymentOrderRecordPrefix+"9") {
		t.Fatalf("payment cache keys = %#v, want order record cache key", keys)
	}
	if !containsPaymentCacheKey(keys, cachePaymentCheckoutPrefix+"9") {
		t.Fatalf("payment cache keys = %#v, want checkout cache key", keys)
	}
	if !containsPaymentCacheKey(keys, cachePaymentGatewayCheckPrefix+"9") {
		t.Fatalf("payment cache keys = %#v, want gateway check cache key", keys)
	}
}

func containsPaymentCacheKey(keys []string, want string) bool {
	for _, key := range keys {
		if key == want {
			return true
		}
	}
	return false
}

func TestUpdateFieldsClearsPublicOrderCache(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Payment{}); err != nil {
		t.Fatal(err)
	}

	enabled := true
	current := &Payment{
		Id:        11,
		Name:      "Card",
		Platform:  "stripe",
		Config:    "{}",
		Enable:    &enabled,
		Token:     "cache-token",
		FeeAmount: 99,
	}
	if err := db.Create(current).Error; err != nil {
		t.Fatal(err)
	}

	mr := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rds.Close()
	})
	if err := rds.Set(ctx, cachePaymentPublicOrderPrefix+"11", "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if err := rds.Set(ctx, cachePaymentOrderRecordPrefix+"11", "cached", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}

	model := NewModel(db, rds)
	name := "Updated"
	disabled := false
	feeAmount := int64(0)
	if err := model.UpdateFields(ctx, current, PaymentFieldsUpdate{
		Name:      &name,
		Enable:    &disabled,
		FeeAmount: &feeAmount,
	}); err != nil {
		t.Fatal(err)
	}
	if exists := rds.Exists(ctx, cachePaymentPublicOrderPrefix+"11").Val(); exists != 0 {
		t.Fatalf("public order cache key still exists")
	}
	if exists := rds.Exists(ctx, cachePaymentOrderRecordPrefix+"11").Val(); exists != 0 {
		t.Fatalf("order record cache key still exists")
	}

	var got Payment
	if err := db.Select("id", "name", "platform", "fee_amount", "enable").First(&got, 11).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "Updated" || got.FeeAmount != 0 || got.Enable == nil || *got.Enable {
		t.Fatalf("requested fields were not persisted: %#v", got)
	}
	if got.Platform != "stripe" {
		t.Fatalf("unrequested field changed: %#v", got)
	}
}
