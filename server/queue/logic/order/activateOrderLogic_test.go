package orderLogic

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	modelCoupon "github.com/neibniz/alpha/server/internal/model/coupon"
	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestActivateOrderMembershipUserColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "member_expired_at"}
	if !reflect.DeepEqual(activateOrderMembershipUserColumns, want) {
		t.Fatalf("membership user columns = %#v, want %#v", activateOrderMembershipUserColumns, want)
	}
}

func TestActivateOrderCommissionUserColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "commission", "referral_percentage", "only_first_purchase"}
	if !reflect.DeepEqual(activateOrderCommissionUserColumns, want) {
		t.Fatalf("commission user columns = %#v, want %#v", activateOrderCommissionUserColumns, want)
	}
}

func TestActivateOrderRechargeUserColumnsStayLightweight(t *testing.T) {
	want := []string{"id", "balance"}
	if !reflect.DeepEqual(activateOrderRechargeUserColumns, want) {
		t.Fatalf("recharge user columns = %#v, want %#v", activateOrderRechargeUserColumns, want)
	}
}

func TestFinalizeOrderUpdatesOnlyStatus(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&modelOrder.Order{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Now().UTC()
	if err := db.Create(&modelOrder.Order{
		Id:              11,
		UserId:          7,
		OrderNo:         "alpha-order-11",
		Type:            OrderTypeSubscribe,
		Quantity:        2,
		Price:           1200,
		Amount:          1900,
		Method:          "stripe",
		TradeNo:         "trade-11",
		Status:          OrderStatusPaid,
		SubscribeId:     10,
		ProductName:     "Pro",
		ProductSnapshot: `{"name":"Pro"}`,
		IsNew:           true,
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

	logic := NewActivateOrderLogic(&svc.ServiceContext{
		OrderModel: modelOrder.NewModel(db, rds),
	})
	narrowOrder := &modelOrder.Order{Id: 11, OrderNo: "alpha-order-11"}
	if err := logic.finalizeOrder(ctx, narrowOrder); err != nil {
		t.Fatalf("finalize order: %v", err)
	}
	if narrowOrder.Status != OrderStatusFinished {
		t.Fatalf("narrow order status = %d, want %d", narrowOrder.Status, OrderStatusFinished)
	}

	var got modelOrder.Order
	if err := db.First(&got, 11).Error; err != nil {
		t.Fatalf("find order: %v", err)
	}
	if got.Status != OrderStatusFinished {
		t.Fatalf("stored status = %d, want %d", got.Status, OrderStatusFinished)
	}
	if got.Amount != 1900 || got.Price != 1200 || got.Method != "stripe" || got.TradeNo != "trade-11" || got.ProductName != "Pro" || got.ProductSnapshot != `{"name":"Pro"}` || !got.IsNew || !got.CreatedAt.Equal(now) {
		t.Fatalf("finalize order changed unrelated fields: %#v", got)
	}
}

func TestFinalizeCouponAndOrderReturnsAtomicUsageError(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelCoupon.Coupon{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&modelCoupon.Coupon{Code: "FULL", Count: 1, UsedCount: 1}).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	logic := NewActivateOrderLogic(&svc.ServiceContext{CouponModel: modelCoupon.NewModel(db, rds)})

	if err := logic.finalizeCouponAndOrder(ctx, &modelOrder.Order{Coupon: "FULL"}); err == nil {
		t.Fatal("expected exhausted coupon update to fail")
	}
}

func TestRechargeActivationIsDatabaseIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelOrder.Order{}, &modelLog.SystemLog{}); err != nil {
		t.Fatal(err)
	}
	userInfo := modelUser.User{Id: 7, Balance: 100}
	orderInfo := modelOrder.Order{
		Id:      11,
		UserId:  userInfo.Id,
		OrderNo: "recharge-idempotent",
		Type:    OrderTypeRecharge,
		Price:   250,
		Status:  OrderStatusPaid,
	}
	if err := db.Create(&userInfo).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&orderInfo).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	logic := NewActivateOrderLogic(&svc.ServiceContext{
		DB:         db,
		Redis:      rds,
		OrderModel: modelOrder.NewModel(db, rds),
		UserModel:  modelUser.NewModel(db, rds),
	})

	if err := logic.Recharge(ctx, &orderInfo); err != nil {
		t.Fatal(err)
	}
	if err := logic.Recharge(ctx, &orderInfo); err != nil {
		t.Fatal(err)
	}

	var storedUser modelUser.User
	if err := db.First(&storedUser, userInfo.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedUser.Balance != 350 {
		t.Fatalf("balance = %d, want 350", storedUser.Balance)
	}
	var balanceLogs int64
	if err := db.Model(&modelLog.SystemLog{}).Where("object_id = ?", userInfo.Id).Count(&balanceLogs).Error; err != nil {
		t.Fatal(err)
	}
	if balanceLogs != 1 {
		t.Fatalf("balance log count = %d, want 1", balanceLogs)
	}
}

func TestApplyCommissionIsTransactionalAndIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&modelUser.User{}, &modelOrder.Order{}, &modelLog.SystemLog{}); err != nil {
		t.Fatal(err)
	}
	onlyFirst := false
	referer := modelUser.User{
		Id:                 3,
		Commission:         100,
		ReferralPercentage: 20,
		OnlyFirstPurchase:  &onlyFirst,
		MemberExpiredAt:    time.Now().Add(time.Hour),
	}
	if err := db.Create(&referer).Error; err != nil {
		t.Fatal(err)
	}
	orderInfo := modelOrder.Order{
		Id:        11,
		UserId:    7,
		OrderNo:   "commission-once",
		Type:      OrderTypeSubscribe,
		Amount:    1055,
		FeeAmount: 55,
		Status:    OrderStatusPaid,
		IsNew:     true,
		CreatedAt: time.Now().UTC(),
	}
	if err := db.Create(&orderInfo).Error; err != nil {
		t.Fatal(err)
	}
	mini := miniredis.RunT(t)
	rds := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rds.Close() })
	logic := NewActivateOrderLogic(&svc.ServiceContext{
		DB:        db,
		Redis:     rds,
		UserModel: modelUser.NewModel(db, rds),
	})
	buyer := &modelUser.User{Id: 7, RefererId: referer.Id}

	for range 2 {
		attempt := orderInfo
		if err := db.Transaction(func(tx *gorm.DB) error {
			return logic.applyCommission(ctx, buyer, &attempt, tx)
		}); err != nil {
			t.Fatal(err)
		}
	}
	var storedReferer modelUser.User
	if err := db.First(&storedReferer, referer.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedReferer.Commission != 300 {
		t.Fatalf("referer commission = %d, want 300", storedReferer.Commission)
	}
	var storedOrder modelOrder.Order
	if err := db.First(&storedOrder, orderInfo.Id).Error; err != nil {
		t.Fatal(err)
	}
	if storedOrder.Commission != 200 {
		t.Fatalf("order commission marker = %d, want 200", storedOrder.Commission)
	}
	var logCount int64
	if err := db.Model(&modelLog.SystemLog{}).
		Where("type = ? AND object_id = ?", modelLog.TypeCommission.Uint8(), referer.Id).
		Count(&logCount).Error; err != nil {
		t.Fatal(err)
	}
	if logCount != 1 {
		t.Fatalf("commission log count = %d, want 1", logCount)
	}
}

func TestCalculateCommissionUsesOverflowSafeIntegerMath(t *testing.T) {
	if got := calculateCommission(1001, 20); got != 200 {
		t.Fatalf("calculateCommission(1001, 20) = %d, want 200", got)
	}
	if got := calculateCommission(math.MaxInt64, 100); got != math.MaxInt64 {
		t.Fatalf("calculateCommission(MaxInt64, 100) = %d", got)
	}
	if got := calculateCommission(-1, 20); got != 0 {
		t.Fatalf("calculateCommission(-1, 20) = %d, want 0", got)
	}
}
