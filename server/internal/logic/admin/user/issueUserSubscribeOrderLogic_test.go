package user

import (
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/config"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateIssueUserSubscribeOrderRequest(t *testing.T) {
	valid := &types.IssueUserSubscribeOrderRequest{
		UserId:      1,
		SubscribeId: 2,
		Quantity:    1,
		AmountMode:  issueAmountCurrentPrice,
	}
	if err := validateIssueUserSubscribeOrderRequest(valid); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	for name, req := range map[string]*types.IssueUserSubscribeOrderRequest{
		"nil":          nil,
		"missing user": {SubscribeId: 2, Quantity: 1, AmountMode: issueAmountCurrentPrice},
		"quantity":     {UserId: 1, SubscribeId: 2, Quantity: 0, AmountMode: issueAmountCurrentPrice},
		"amount mode":  {UserId: 1, SubscribeId: 2, Quantity: 1, AmountMode: "custom"},
		"trade number": {UserId: 1, SubscribeId: 2, Quantity: 1, AmountMode: issueAmountCurrentPrice, TradeNo: string(make([]byte, 256))},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateIssueUserSubscribeOrderRequest(req); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestBuildIssueUserSubscribeOrderQuoteHonorsAmountMode(t *testing.T) {
	enabled := true
	now := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	account := &modelUser.User{Id: 1, Enable: &enabled}
	product := &modelSubscribe.Subscribe{
		Id:          2,
		Name:        "Pro",
		UnitPrice:   1000,
		UnitTime:    "month",
		Discount:    `[{"quantity":2,"discount":80}]`,
		Traffic:     100,
		SpeedLimit:  200,
		DeviceLimit: 3,
		MinQuantity: 1,
	}
	req := &types.IssueUserSubscribeOrderRequest{UserId: 1, SubscribeId: 2, Quantity: 2, AmountMode: issueAmountCurrentPrice}
	quote, err := buildIssueUserSubscribeOrderQuote(account, product, req, now)
	if err != nil {
		t.Fatal(err)
	}
	if quote.ListPrice != 2000 || quote.CurrentAmount != 1600 || quote.RecordedAmount != 1600 || quote.Discount != 400 {
		t.Fatalf("current-price quote = %#v", quote)
	}
	req.AmountMode = issueAmountComplimentary
	quote, err = buildIssueUserSubscribeOrderQuote(account, product, req, now)
	if err != nil {
		t.Fatal(err)
	}
	if quote.CurrentAmount != 1600 || quote.RecordedAmount != 0 || quote.Discount != 2000 {
		t.Fatalf("complimentary quote = %#v", quote)
	}
}

func TestLockAndPrepareIssueUsesLockedCurrentProductTerms(t *testing.T) {
	db := newIssueUserSubscribeOrderTestDB(t)
	userID, productID := seedIssueUserSubscribeOrderTestData(t, db, 2, 0)
	logic := NewIssueUserSubscribeOrderLogic(t.Context(), &svc.ServiceContext{DB: db, Config: config.Config{}})
	req := &types.IssueUserSubscribeOrderRequest{
		UserId: userID, SubscribeId: productID, Quantity: 1, AmountMode: issueAmountCurrentPrice,
	}

	var first *issueUserSubscribeOrderData
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		first, err = logic.lockAndPrepareIssue(tx, req, time.Now().UTC())
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if first.quote.RecordedAmount != 100 || !first.isNew {
		t.Fatalf("first prepared order = %#v", first)
	}
	if err := db.Model(&modelSubscribe.Subscribe{}).Where("id = ?", productID).UpdateColumn("unit_price", 250).Error; err != nil {
		t.Fatal(err)
	}

	var second *issueUserSubscribeOrderData
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		second, err = logic.lockAndPrepareIssue(tx, req, time.Now().UTC())
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if second.quote.UnitPrice != 250 || second.quote.RecordedAmount != 250 {
		t.Fatalf("second prepared order used stale terms: %#v", second.quote)
	}
	var inventory int64
	if err := db.Model(&modelSubscribe.Subscribe{}).Where("id = ?", productID).Pluck("inventory", &inventory).Error; err != nil {
		t.Fatal(err)
	}
	if inventory != 0 {
		t.Fatalf("inventory = %d, want 0", inventory)
	}
}

func TestLockAndPrepareIssueCountsOnlyEffectiveActiveSubscriptions(t *testing.T) {
	db := newIssueUserSubscribeOrderTestDB(t)
	userID, productID := seedIssueUserSubscribeOrderTestData(t, db, -1, 1)
	logic := NewIssueUserSubscribeOrderLogic(t.Context(), &svc.ServiceContext{DB: db, Config: config.Config{}})
	now := time.Now().UTC()
	for _, sub := range []*modelUser.Subscribe{
		{UserId: userID, OrderId: 10, SubscribeId: productID, ProductName: "expired", ProductSnapshot: "{}", ExpireTime: now.Add(-time.Hour), Traffic: 0, Token: "expired-token", UUID: "expired-uuid", Status: 1},
		{UserId: userID, OrderId: 11, SubscribeId: productID, ProductName: "exhausted", ProductSnapshot: "{}", ExpireTime: now.Add(time.Hour), Traffic: 100, Token: "exhausted-token", UUID: "exhausted-uuid", Status: 1},
	} {
		if err := db.Create(sub).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Exec("UPDATE user_subscribe SET used_traffic = traffic WHERE token = ?", "exhausted-token").Error; err != nil {
		t.Fatal(err)
	}
	req := &types.IssueUserSubscribeOrderRequest{UserId: userID, SubscribeId: productID, Quantity: 1, AmountMode: issueAmountCurrentPrice}
	if err := db.Transaction(func(tx *gorm.DB) error {
		_, err := logic.lockAndPrepareIssue(tx, req, now)
		return err
	}); err != nil {
		t.Fatalf("expired and exhausted subscriptions should not consume quota: %v", err)
	}

	active := &modelUser.Subscribe{
		UserId: userID, OrderId: 12, SubscribeId: productID, ProductName: "active", ProductSnapshot: "{}",
		ExpireTime: now.Add(time.Hour), Traffic: 0, Token: "active-token", UUID: "active-uuid", Status: 1,
	}
	if err := db.Create(active).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		_, err := logic.lockAndPrepareIssue(tx, req, now)
		return err
	}); err == nil {
		t.Fatal("effective active subscription should consume product quota")
	}
}

func newIssueUserSubscribeOrderTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelSubscribe.Subscribe{}, &modelOrder.Order{}, &modelUser.Subscribe{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("ALTER TABLE user_subscribe ADD COLUMN used_traffic integer NOT NULL DEFAULT 0").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func seedIssueUserSubscribeOrderTestData(t *testing.T, db *gorm.DB, inventory, quota int64) (int64, int64) {
	t.Helper()
	enabled := true
	account := &modelUser.User{Password: "test", Enable: &enabled}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	product := &modelSubscribe.Subscribe{
		Name: "Current product", UnitPrice: 100, UnitTime: "month", Discount: "[]",
		Inventory: inventory, Quota: quota, MinQuantity: 1,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatal(err)
	}
	return account.Id, product.Id
}
