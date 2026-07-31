// Package orderLogic provides order processing logic for handling various types of orders
// including subscription purchases, renewals, traffic resets, and balance recharges.
package orderLogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/redislock"

	"github.com/neibniz/alpha/server/internal/logic/telegram"
	"github.com/neibniz/alpha/server/internal/model/membership"
	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/subscribecache"
	"github.com/neibniz/alpha/server/internal/subscriptionstate"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/uuidx"
	"github.com/neibniz/alpha/server/queue/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Order type constants define the different types of orders that can be processed
const (
	OrderTypeSubscribe    = 1 // New subscription purchase
	OrderTypeRenewal      = 2 // Subscription renewal
	OrderTypeResetTraffic = 3 // Traffic quota reset
	OrderTypeRecharge     = 4 // Balance recharge
	OrderTypeMembership   = 5 // Membership card purchase
)

// Order status constants define the lifecycle states of an order
const (
	OrderStatusPending  = 1 // Order created but not paid
	OrderStatusPaid     = 2 // Order paid and ready for processing
	OrderStatusClose    = 3 // Order closed/cancelled
	OrderStatusFailed   = 4 // Order processing failed
	OrderStatusFinished = 5 // Order successfully completed
)

const activateOrderLockTimeout = 10 * time.Minute

var activateOrderMembershipUserColumns = []string{
	"id",
	"member_expired_at",
}

var activateOrderCommissionUserColumns = []string{
	"id",
	"commission",
	"referral_percentage",
	"only_first_purchase",
}

var activateOrderRechargeUserColumns = []string{
	"id",
	"balance",
}

// Predefined error variables for common error conditions
var (
	ErrInvalidOrderStatus = fmt.Errorf("invalid order status")
	ErrInvalidOrderType   = fmt.Errorf("invalid order type")
)

// ActivateOrderLogic handles the activation and processing of paid orders
type ActivateOrderLogic struct {
	svc *svc.ServiceContext // Service context containing dependencies
}

// NewActivateOrderLogic creates a new instance of ActivateOrderLogic
func NewActivateOrderLogic(svc *svc.ServiceContext) *ActivateOrderLogic {
	return &ActivateOrderLogic{
		svc: svc,
	}
}

// ProcessTask is the main entry point for processing order activation tasks.
// It handles the complete workflow of activating a paid order including validation,
// processing based on order type, and finalization.
func (l *ActivateOrderLogic) ProcessTask(ctx context.Context, task *asynq.Task) error {
	payload, err := l.parsePayload(ctx, task.Payload())
	if err != nil {
		return nil // Log and continue
	}

	lockKey := fmt.Sprintf("order:activate:lock:%s", payload.OrderNo)
	lock, acquired, err := redislock.Acquire(ctx, l.svc.Redis, lockKey, activateOrderLockTimeout)
	if err != nil {
		logger.WithContext(ctx).Error("[ActivateOrderLogic] Acquire lock failed",
			logger.Field("error", err.Error()),
			logger.Field("order_no", payload.OrderNo),
		)
		return err
	}
	if !acquired {
		logger.WithContext(ctx).Info("[ActivateOrderLogic] Order is already being activated",
			logger.Field("order_no", payload.OrderNo),
		)
		return nil
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := lock.Release(releaseCtx); err != nil {
			logger.WithContext(ctx).Error("[ActivateOrderLogic] Release lock failed",
				logger.Field("error", err.Error()),
				logger.Field("order_no", payload.OrderNo),
			)
		}
	}()

	orderInfo, err := l.validateAndGetOrder(ctx, payload.OrderNo)
	if err != nil {
		return nil // Log and continue
	}

	if err = l.processOrderByType(ctx, orderInfo); err != nil {
		logger.WithContext(ctx).Error("[ActivateOrderLogic] Process task failed", logger.Field("error", err.Error()))
		return err
	}
	if l.svc.DashboardRealtime != nil {
		l.svc.DashboardRealtime.Publish(realtime.DashboardEventBusiness)
	}

	return nil
}

// parsePayload unMarshals the task payload into a structured format
func (l *ActivateOrderLogic) parsePayload(ctx context.Context, payload []byte) (*types.ForthwithActivateOrderPayload, error) {
	var p types.ForthwithActivateOrderPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		logger.WithContext(ctx).Error("[ActivateOrderLogic] Unmarshal payload failed",
			logger.Field("error", err.Error()),
			logger.Field("payload_bytes", len(payload)),
		)
		return nil, err
	}
	return &p, nil
}

// validateAndGetOrder retrieves an order by order number and validates its status
// Returns error if order is not found or not in paid status
func (l *ActivateOrderLogic) validateAndGetOrder(ctx context.Context, orderNo string) (*order.Order, error) {
	orderInfo, err := l.svc.OrderModel.FindOneForActivation(ctx, orderNo)
	if err != nil {
		logger.WithContext(ctx).Error("Find order failed",
			logger.Field("error", err.Error()),
			logger.Field("order_no", orderNo),
		)
		return nil, err
	}

	if orderInfo.Status != OrderStatusPaid {
		logger.WithContext(ctx).Error("Order status error",
			logger.Field("order_no", orderInfo.OrderNo),
			logger.Field("status", orderInfo.Status),
		)
		return nil, ErrInvalidOrderStatus
	}

	return orderInfo, nil
}

// processOrderByType routes order processing based on the order type
func (l *ActivateOrderLogic) processOrderByType(ctx context.Context, orderInfo *order.Order) error {
	switch orderInfo.Type {
	case OrderTypeSubscribe:
		return l.NewPurchase(ctx, orderInfo)
	case OrderTypeRenewal:
		return l.Renewal(ctx, orderInfo)
	case OrderTypeResetTraffic:
		return l.ResetTraffic(ctx, orderInfo)
	case OrderTypeRecharge:
		return l.Recharge(ctx, orderInfo)
	case OrderTypeMembership:
		return l.Membership(ctx, orderInfo)
	default:
		logger.WithContext(ctx).Error("Order type is invalid", logger.Field("type", orderInfo.Type))
		return ErrInvalidOrderType
	}
}

func (l *ActivateOrderLogic) Membership(ctx context.Context, orderInfo *order.Order) error {
	if orderInfo.UserId == 0 {
		return fmt.Errorf("membership order requires user id")
	}
	var plan membership.Plan
	if !ordersnapshot.ApplyToMembershipPlanModel(orderInfo.ProductSnapshot, &plan) {
		if err := l.svc.DB.WithContext(ctx).
			Model(&membership.Plan{}).
			Select(membership.PlanDurationColumns()).
			Where("id = ?", orderInfo.SubscribeId).
			First(&plan).Error; err != nil {
			logger.WithContext(ctx).Error("Find membership plan failed",
				logger.Field("error", err.Error()),
				logger.Field("plan_id", orderInfo.SubscribeId),
				logger.Field("order_no", orderInfo.OrderNo),
			)
			return err
		}
	}
	durationQuantity := plan.DurationValue
	if durationQuantity <= 0 {
		durationQuantity = 1
	}

	return l.svc.DB.Transaction(func(tx *gorm.DB) error {
		proceed, err := l.lockOrderForActivation(ctx, orderInfo, tx)
		if err != nil || !proceed {
			return err
		}
		var userInfo user.User
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&user.User{}).
			Select(activateOrderMembershipUserColumns).
			Where("id = ?", orderInfo.UserId).
			First(&userInfo).Error; err != nil {
			logger.WithContext(ctx).Error("Find membership user failed",
				logger.Field("error", err.Error()),
				logger.Field("user_id", orderInfo.UserId),
				logger.Field("order_no", orderInfo.OrderNo),
			)
			return err
		}
		base := time.Now()
		if !userInfo.MemberExpiredAt.IsZero() && userInfo.MemberExpiredAt.After(base) {
			base = userInfo.MemberExpiredAt
		}
		userInfo.MemberExpiredAt = tool.AddTime(plan.DurationUnit, durationQuantity, base)
		if err := l.svc.UserModel.UpdateUserMemberExpiredAt(ctx, userInfo.Id, userInfo.MemberExpiredAt, tx); err != nil {
			logger.WithContext(ctx).Error("Update membership failed",
				logger.Field("error", err.Error()),
				logger.Field("user_id", userInfo.Id),
				logger.Field("order_no", orderInfo.OrderNo),
			)
			return err
		}
		return l.finalizeOrder(ctx, orderInfo, tx)
	})
}

func (l *ActivateOrderLogic) lockOrderForActivation(ctx context.Context, orderInfo *order.Order, tx *gorm.DB) (bool, error) {
	if orderInfo == nil || orderInfo.Id <= 0 || tx == nil {
		return false, ErrInvalidOrderStatus
	}
	var locked order.Order
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&order.Order{}).
		Select("id", "status").
		Where("id = ? AND order_no = ?", orderInfo.Id, orderInfo.OrderNo).
		First(&locked).Error; err != nil {
		return false, err
	}
	return locked.Status == OrderStatusPaid, nil
}

func (l *ActivateOrderLogic) finalizeOrder(ctx context.Context, orderInfo *order.Order, tx ...*gorm.DB) error {
	if err := l.svc.OrderModel.UpdateFields(ctx, orderInfo, order.OrderFieldsUpdate{
		Status: order.UpdateValue(uint8(OrderStatusFinished)),
	}, tx...); err != nil {
		logger.WithContext(ctx).Error("Update order status failed",
			logger.Field("error", err.Error()),
			logger.Field("order_no", orderInfo.OrderNo),
		)
		return err
	}
	orderInfo.Status = OrderStatusFinished
	return nil
}

// finalizeCouponAndOrder handles post-processing tasks including coupon updates.
func (l *ActivateOrderLogic) finalizeCouponAndOrder(ctx context.Context, orderInfo *order.Order, tx ...*gorm.DB) error {
	// Update coupon if exists
	if orderInfo.Coupon != "" {
		if err := l.svc.CouponModel.UpdateCount(ctx, orderInfo.Coupon, tx...); err != nil {
			logger.WithContext(ctx).Error("Update coupon status failed",
				logger.Field("error", err.Error()),
				logger.Field("coupon", orderInfo.Coupon),
			)
			return err
		}
	}
	return nil
}

// NewPurchase handles new subscription purchase including user creation,
// subscription setup, commission processing, cache updates, and notifications
func (l *ActivateOrderLogic) NewPurchase(ctx context.Context, orderInfo *order.Order) error {
	if orderInfo.UserId == 0 {
		return fmt.Errorf("new purchase order %s has no user", orderInfo.OrderNo)
	}

	userInfo, err := l.getExistingUser(ctx, orderInfo.UserId)
	if err != nil {
		return err
	}

	sub, err := l.getSubscribeInfoForOrder(ctx, orderInfo)
	if err != nil {
		return err
	}

	var userSub *user.Subscribe
	var activated bool
	if err = l.svc.DB.Transaction(func(tx *gorm.DB) error {
		proceed, lockErr := l.lockOrderForActivation(ctx, orderInfo, tx)
		if lockErr != nil || !proceed {
			return lockErr
		}
		userSub, _, lockErr = l.createUserSubscription(ctx, orderInfo, sub, tx)
		if lockErr != nil {
			return lockErr
		}
		if lockErr = l.applyCommission(ctx, userInfo, orderInfo, tx); lockErr != nil {
			return lockErr
		}
		if lockErr = l.finalizeCouponAndOrder(ctx, orderInfo, tx); lockErr != nil {
			return lockErr
		}
		if lockErr = l.finalizeOrder(ctx, orderInfo, tx); lockErr != nil {
			return lockErr
		}
		activated = true
		return nil
	}); err != nil {
		return err
	}
	if !activated {
		return nil
	}

	// Clear cache
	if err := l.svc.UserModel.ClearSubscribeCache(ctx, userSub); err != nil {
		logger.WithContext(ctx).Error("[Order Queue] Clear user subscription cache failed", logger.Field("error", err.Error()))
	}
	l.clearServerCache(ctx, sub)

	// Send notifications
	l.sendNotifications(ctx, orderInfo, userInfo, sub, userSub, telegram.PurchaseNotify)

	logger.WithContext(ctx).Info("Insert user subscribe success")
	return nil
}

// getExistingUser retrieves user information by user ID
func (l *ActivateOrderLogic) getExistingUser(ctx context.Context, userId int64) (*user.User, error) {
	userInfo, err := l.svc.UserModel.FindOneForOrderActivation(ctx, userId)
	if err != nil {
		logger.WithContext(ctx).Error("Find user failed",
			logger.Field("error", err.Error()),
			logger.Field("user_id", userId),
		)
		return nil, err
	}
	return userInfo, nil
}

// getSubscribeInfo retrieves subscription plan details by subscription ID
func (l *ActivateOrderLogic) getSubscribeInfo(ctx context.Context, subscribeId int64) (*subscribe.Subscribe, error) {
	sub, err := l.svc.SubscribeModel.FindOneForOrderActivation(ctx, subscribeId)
	if err != nil {
		logger.WithContext(ctx).Error("Find subscribe failed",
			logger.Field("error", err.Error()),
			logger.Field("subscribe_id", subscribeId),
		)
		return nil, err
	}
	return sub, nil
}

func (l *ActivateOrderLogic) getSubscribeInfoForOrder(ctx context.Context, orderInfo *order.Order) (*subscribe.Subscribe, error) {
	if snapshotSub, ok := ordersnapshot.SubscribeModel(orderInfo.ProductSnapshot); ok {
		return snapshotSub, nil
	}
	sub, err := l.svc.SubscribeModel.FindOneForOrderActivation(ctx, orderInfo.SubscribeId)
	if err == nil {
		return sub, nil
	}
	return nil, err
}

func (l *ActivateOrderLogic) getSubscribeInfoForUserSubscription(ctx context.Context, userSub *user.Subscribe, orderInfo *order.Order) (*subscribe.Subscribe, error) {
	if orderInfo != nil {
		if snapshotSub, ok := ordersnapshot.SubscribeModel(orderInfo.ProductSnapshot); ok {
			return snapshotSub, nil
		}
	}
	if snapshotSub, ok := ordersnapshot.SubscribeModel(userSub.ProductSnapshot); ok {
		return snapshotSub, nil
	}
	sub, err := l.svc.SubscribeModel.FindOneForOrderActivation(ctx, userSub.SubscribeId)
	if err == nil {
		return sub, nil
	}
	return nil, err
}

// createUserSubscription creates a new user subscription record based on order and subscription plan details
func (l *ActivateOrderLogic) createUserSubscription(ctx context.Context, orderInfo *order.Order, sub *subscribe.Subscribe, tx *gorm.DB) (*user.Subscribe, bool, error) {
	var existing user.Subscribe
	err := tx.WithContext(ctx).Model(&user.Subscribe{}).Where("order_id = ?", orderInfo.Id).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.WithContext(ctx).Error("Find user subscribe by order failed", logger.Field("error", err.Error()))
		return nil, false, err
	}

	now := time.Now()
	expireAt := tool.AddTime(sub.UnitTime, orderInfo.Quantity, now)
	resetCycle := subscriptionstate.NormalizeResetCycle(sub.ResetCycle)
	nextResetAt := subscriptionstate.NextTrafficResetAt(
		resetCycle,
		now,
		now,
		expireAt,
		subscriptionstate.TrafficResetLocation(),
	)
	productName := orderInfo.ProductName
	if productName == "" {
		productName = sub.Name
	}
	subscribeToken, err := uuidx.NewSubscribeToken()
	if err != nil {
		return nil, false, fmt.Errorf("generate subscription token: %w", err)
	}
	userSub := &user.Subscribe{
		UserId:             orderInfo.UserId,
		OrderId:            orderInfo.Id,
		SubscribeId:        orderInfo.SubscribeId,
		ProductName:        productName,
		ProductSnapshot:    orderInfo.ProductSnapshot,
		StartTime:          now,
		ExpireTime:         expireAt,
		Traffic:            sub.Traffic,
		Download:           0,
		Upload:             0,
		ResetCycle:         resetCycle,
		NextTrafficResetAt: nextResetAt,
		Token:              subscribeToken,
		UUID:               uuid.New().String(),
		Status:             1,
	}

	if err := l.svc.UserModel.InsertSubscribe(ctx, userSub, tx); err != nil {
		logger.WithContext(ctx).Error("Insert user subscribe failed", logger.Field("error", err.Error()))
		return nil, false, err
	}

	return userSub, true, nil
}

// applyCommission updates the referrer, order marker and audit log atomically.
func (l *ActivateOrderLogic) applyCommission(ctx context.Context, userInfo *user.User, orderInfo *order.Order, tx *gorm.DB) error {
	if userInfo == nil || userInfo.RefererId <= 0 || orderInfo == nil || orderInfo.Id <= 0 || orderInfo.Commission != 0 || orderInfo.Source == "admin" {
		return nil
	}
	referer := &user.User{}
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&user.User{}).
		Select(activateOrderCommissionUserColumns).
		Where("id = ?", userInfo.RefererId).
		First(referer).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	percentage, ok := l.commissionPercentage(referer, orderInfo.IsNew)
	if !ok {
		return nil
	}
	amount := calculateCommission(orderInfo.Amount-orderInfo.FeeAmount, percentage)
	if amount <= 0 {
		return nil
	}

	marked := tx.WithContext(ctx).
		Model(&order.Order{}).
		Where("id = ? AND commission = 0", orderInfo.Id).
		Update("commission", amount)
	if marked.Error != nil {
		return marked.Error
	}
	if marked.RowsAffected == 0 {
		return nil
	}

	referer.Commission += amount
	if err := l.svc.UserModel.UpdateUserCommission(ctx, referer.Id, referer.Commission, tx); err != nil {
		return err
	}
	commissionType := uint16(log.CommissionTypePurchase)
	if orderInfo.Type == OrderTypeRenewal {
		commissionType = log.CommissionTypeRenewal
	}
	commissionLog := &log.Commission{
		Type:      commissionType,
		Amount:    amount,
		OrderNo:   orderInfo.OrderNo,
		Timestamp: orderInfo.CreatedAt.UnixMilli(),
	}
	content, _ := commissionLog.Marshal()
	if err := tx.WithContext(ctx).Create(&log.SystemLog{
		Type:     log.TypeCommission.Uint8(),
		Date:     time.Now().Format(time.DateOnly),
		ObjectID: referer.Id,
		Content:  string(content),
	}).Error; err != nil {
		return err
	}
	orderInfo.Commission = amount
	return nil
}

func (l *ActivateOrderLogic) commissionPercentage(referer *user.User, isFirstPurchase bool) (uint8, bool) {
	if referer == nil {
		return 0, false
	}
	if referer.ReferralPercentage > 0 {
		if referer.OnlyFirstPurchase != nil && *referer.OnlyFirstPurchase && !isFirstPurchase {
			return 0, false
		}
		return normalizeCommissionPercentage(int64(referer.ReferralPercentage))
	}
	invite := l.svc.ConfigSnapshot().Invite
	if invite.OnlyFirstPurchase && !isFirstPurchase {
		return 0, false
	}
	return normalizeCommissionPercentage(invite.ReferralPercentage)
}

func normalizeCommissionPercentage(percentage int64) (uint8, bool) {
	if percentage <= 0 {
		return 0, false
	}
	if percentage > 100 {
		percentage = 100
	}
	return uint8(percentage), true
}

func calculateCommission(price int64, percentage uint8) int64 {
	if price <= 0 || percentage == 0 {
		return 0
	}
	p := int64(percentage)
	return (price/100)*p + (price%100)*p/100
}

// clearServerCache clears user list cache for all servers associated with the subscription
func (l *ActivateOrderLogic) clearServerCache(ctx context.Context, sub *subscribe.Subscribe) {
	if err := subscribecache.ClearBySubscribe(ctx, l.svc.DB, l.svc.Redis, sub); err != nil {
		logger.WithContext(ctx).Error("[Order Queue] Clear snapshot server user cache failed", logger.Field("error", err.Error()))
	}
	if err := l.svc.SubscribeModel.ClearCache(ctx, sub.Id); err != nil {
		logger.WithContext(ctx).Error("[Order Queue] Clear subscribe cache failed", logger.Field("error", err.Error()))
	}
}

// Renewal handles subscription renewal including subscription extension,
// traffic reset (if configured), commission processing, and notifications
func (l *ActivateOrderLogic) Renewal(ctx context.Context, orderInfo *order.Order) error {
	userInfo, err := l.getExistingUser(ctx, orderInfo.UserId)
	if err != nil {
		return err
	}

	userSub, err := l.getUserSubscription(ctx, orderInfo.SubscribeToken)
	if err != nil {
		return err
	}

	sub, err := l.getSubscribeInfoForUserSubscription(ctx, userSub, orderInfo)
	if err != nil {
		return err
	}

	activated := false
	if err = l.svc.DB.Transaction(func(tx *gorm.DB) error {
		proceed, lockErr := l.lockOrderForActivation(ctx, orderInfo, tx)
		if lockErr != nil || !proceed {
			return lockErr
		}
		lockedUserSub, err := l.getUserSubscriptionForUpdate(ctx, orderInfo.SubscribeToken, tx)
		if err != nil {
			return err
		}
		userSub = lockedUserSub
		if err := l.updateSubscriptionForRenewal(ctx, userSub, sub, orderInfo, tx); err != nil {
			return err
		}
		if err := l.applyCommission(ctx, userInfo, orderInfo, tx); err != nil {
			return err
		}
		if err := l.finalizeCouponAndOrder(ctx, orderInfo, tx); err != nil {
			return err
		}
		if err := l.finalizeOrder(ctx, orderInfo, tx); err != nil {
			return err
		}
		activated = true
		return nil
	}); err != nil {
		return err
	}
	if !activated {
		return nil
	}

	// Clear cache
	if err := l.svc.UserModel.ClearSubscribeCache(ctx, userSub); err != nil {
		logger.WithContext(ctx).Error("[Order Queue] Clear user subscription cache failed", logger.Field("error", err.Error()))
	}
	l.clearServerCache(ctx, sub)

	// Send notifications
	l.sendNotifications(ctx, orderInfo, userInfo, sub, userSub, telegram.RenewalNotify)

	return nil
}

// getUserSubscription retrieves user subscription by token
func (l *ActivateOrderLogic) getUserSubscription(ctx context.Context, token string) (*user.Subscribe, error) {
	userSub, err := l.svc.UserModel.FindOneSubscribeByTokenForRuntime(ctx, token)
	if err != nil {
		logger.WithContext(ctx).Error("Find user subscribe failed", logger.Field("error", err.Error()))
		return nil, err
	}
	return userSub, nil
}

func (l *ActivateOrderLogic) getUserSubscriptionForUpdate(ctx context.Context, token string, tx *gorm.DB) (*user.Subscribe, error) {
	userSub, err := l.svc.UserModel.FindOneSubscribeByTokenForActivationUpdate(ctx, token, tx)
	if err != nil {
		logger.WithContext(ctx).Error("Find user subscribe for update failed", logger.Field("error", err.Error()))
		return nil, err
	}
	return userSub, nil
}

// updateSubscriptionForRenewal updates subscription details for renewal including
// expiration time extension and traffic reset if configured
func (l *ActivateOrderLogic) updateSubscriptionForRenewal(ctx context.Context, userSub *user.Subscribe, sub *subscribe.Subscribe, orderInfo *order.Order, tx ...*gorm.DB) error {
	now := time.Now()
	if userSub.ExpireTime.Before(now) {
		userSub.ExpireTime = now
	}
	today := now.Day()
	resetDay := userSub.ExpireTime.Day()
	update := user.UserSubscribeActivationUpdate{}
	trafficReset := false

	// Reset traffic if enabled
	if (sub.RenewalReset != nil && *sub.RenewalReset) || today == resetDay {
		userSub.Download = 0
		userSub.Upload = 0
		update.Download = user.UpdateValue(int64(0))
		update.Upload = user.UpdateValue(int64(0))
		trafficReset = true
	}

	if userSub.FinishedAt != nil {
		if userSub.FinishedAt.Before(now) && today > resetDay {
			// reset user traffic if finished at is before now
			userSub.Download = 0
			userSub.Upload = 0
			update.Download = user.UpdateValue(int64(0))
			update.Upload = user.UpdateValue(int64(0))
			trafficReset = true
		}

		userSub.FinishedAt = nil
		update.ClearFinishedAt = true
	}

	userSub.ExpireTime = tool.AddTime(sub.UnitTime, orderInfo.Quantity, userSub.ExpireTime)
	userSub.Status = 1
	update.ExpireTime = user.UpdateValue(userSub.ExpireTime)
	update.Status = user.UpdateValue(uint8(1))
	if trafficReset {
		userSub.TrafficResetAt = now
		update.TrafficResetAt = user.UpdateValue(now)
	}
	if orderInfo.ProductSnapshot != "" && orderInfo.ProductSnapshot != "{}" {
		userSub.ProductName = orderInfo.ProductName
		if userSub.ProductName == "" {
			userSub.ProductName = sub.Name
		}
		userSub.ProductSnapshot = orderInfo.ProductSnapshot
		update.ProductName = user.UpdateValue(userSub.ProductName)
		update.ProductSnapshot = user.UpdateValue(userSub.ProductSnapshot)
	}
	resetCycle := subscriptionstate.NormalizeResetCycle(sub.ResetCycle)
	userSub.ResetCycle = resetCycle
	update.ResetCycle = user.UpdateValue(resetCycle)
	nextResetAt := subscriptionstate.NextTrafficResetAt(
		resetCycle,
		userSub.StartTime,
		now,
		userSub.ExpireTime,
		subscriptionstate.TrafficResetLocation(),
	)
	userSub.NextTrafficResetAt = nextResetAt
	if nextResetAt == nil {
		update.ClearNextTrafficResetAt = true
	} else {
		update.NextTrafficResetAt = nextResetAt
	}

	if err := l.svc.UserModel.UpdateSubscribeActivationFields(ctx, userSub, update, tx...); err != nil {
		logger.WithContext(ctx).Error("Update user subscribe failed", logger.Field("error", err.Error()))
		return err
	}

	return nil
}

// ResetTraffic handles traffic quota reset for existing subscriptions
func (l *ActivateOrderLogic) ResetTraffic(ctx context.Context, orderInfo *order.Order) error {
	userInfo, err := l.getExistingUser(ctx, orderInfo.UserId)
	if err != nil {
		return err
	}

	userSub, err := l.getUserSubscription(ctx, orderInfo.SubscribeToken)
	if err != nil {
		return err
	}

	sub, err := l.getSubscribeInfoForUserSubscription(ctx, userSub, orderInfo)
	if err != nil {
		return err
	}

	activated := false
	if err = l.svc.DB.Transaction(func(tx *gorm.DB) error {
		proceed, lockErr := l.lockOrderForActivation(ctx, orderInfo, tx)
		if lockErr != nil || !proceed {
			return lockErr
		}
		lockedUserSub, err := l.getUserSubscriptionForUpdate(ctx, orderInfo.SubscribeToken, tx)
		if err != nil {
			return err
		}
		userSub = lockedUserSub
		// Reset traffic
		userSub.Download = 0
		userSub.Upload = 0
		resetAt := time.Now()
		userSub.TrafficResetAt = resetAt
		userSub.FinishedAt = nil
		userSub.Status = subscriptionstate.StatusActive
		update := user.UserSubscribeActivationUpdate{
			Download:        user.UpdateValue(int64(0)),
			Upload:          user.UpdateValue(int64(0)),
			TrafficResetAt:  user.UpdateValue(resetAt),
			ClearFinishedAt: true,
			Status:          user.UpdateValue(subscriptionstate.StatusActive),
		}
		if (userSub.ProductSnapshot == "" || userSub.ProductSnapshot == "{}") && orderInfo.ProductSnapshot != "" && orderInfo.ProductSnapshot != "{}" {
			userSub.ProductName = orderInfo.ProductName
			if userSub.ProductName == "" {
				userSub.ProductName = sub.Name
			}
			userSub.ProductSnapshot = orderInfo.ProductSnapshot
			update.ProductName = user.UpdateValue(userSub.ProductName)
			update.ProductSnapshot = user.UpdateValue(userSub.ProductSnapshot)
		}

		if err := l.svc.UserModel.UpdateSubscribeActivationFields(ctx, userSub, update, tx); err != nil {
			logger.WithContext(ctx).Error("Update user subscribe failed", logger.Field("error", err.Error()))
			return err
		}

		resetLog := &log.ResetSubscribe{
			Type:      log.ResetSubscribeTypePaid,
			UserId:    userInfo.Id,
			OrderNo:   orderInfo.OrderNo,
			Timestamp: resetAt.UnixMilli(),
		}
		content, _ := resetLog.Marshal()
		if err := tx.Model(&log.SystemLog{}).Create(&log.SystemLog{
			Type:     log.TypeResetSubscribe.Uint8(),
			Date:     resetAt.Format(time.DateOnly),
			ObjectID: userSub.Id,
			EventAt:  resetAt.UnixMilli(),
			Success:  true,
			Content:  string(content),
		}).Error; err != nil {
			logger.WithContext(ctx).Error("[Order Queue]Insert reset subscribe log failed", logger.Field("error", err.Error()))
			return err
		}

		if err := l.finalizeOrder(ctx, orderInfo, tx); err != nil {
			return err
		}
		activated = true
		return nil
	}); err != nil {
		return err
	}
	if !activated {
		return nil
	}

	// Invalidate only after commit so readers cannot repopulate stale data.
	if err := l.svc.UserModel.ClearSubscribeCache(ctx, userSub); err != nil {
		logger.WithContext(ctx).Error("[Order Queue] Clear user subscription cache failed", logger.Field("error", err.Error()))
	}
	l.clearServerCache(ctx, sub)

	// Send notifications
	l.sendNotifications(ctx, orderInfo, userInfo, sub, userSub, telegram.ResetTrafficNotify)

	return nil
}

// Recharge handles balance recharge orders including balance updates,
// transaction logging, and notifications
func (l *ActivateOrderLogic) Recharge(ctx context.Context, orderInfo *order.Order) error {
	// Update balance in transaction
	var userInfo *user.User
	activated := false
	err := l.svc.DB.Transaction(func(tx *gorm.DB) error {
		proceed, lockErr := l.lockOrderForActivation(ctx, orderInfo, tx)
		if lockErr != nil || !proceed {
			return lockErr
		}
		lockedUser := &user.User{}
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&user.User{}).
			Select(activateOrderRechargeUserColumns).
			Where("id = ?", orderInfo.UserId).
			First(lockedUser).Error; err != nil {
			return err
		}
		userInfo = lockedUser
		userInfo.Balance += orderInfo.Price
		if err := l.svc.UserModel.UpdateUserBalance(ctx, userInfo.Id, userInfo.Balance, tx); err != nil {
			return err
		}

		balanceLog := &log.Balance{
			Amount:    orderInfo.Price,
			Type:      log.BalanceTypeRecharge,
			OrderNo:   orderInfo.OrderNo,
			Balance:   userInfo.Balance,
			Timestamp: time.Now().UnixMilli(),
		}
		content, _ := balanceLog.Marshal()

		if err := tx.Model(&log.SystemLog{}).Create(&log.SystemLog{
			Type:     log.TypeBalance.Uint8(),
			Date:     time.Now().Format("2006-01-02"),
			ObjectID: userInfo.Id,
			Content:  string(content),
		}).Error; err != nil {
			return err
		}
		if err := l.finalizeOrder(ctx, orderInfo, tx); err != nil {
			return err
		}
		activated = true
		return nil
	})

	if err != nil {
		logger.WithContext(ctx).Error("[Recharge] Database transaction failed", logger.Field("error", err.Error()))
		return err
	}
	if !activated {
		return nil
	}

	// Send notifications
	l.sendRechargeNotifications(ctx, orderInfo, userInfo)

	return nil
}

// sendNotifications sends both user and admin notifications for order completion
func (l *ActivateOrderLogic) sendNotifications(ctx context.Context, orderInfo *order.Order, userInfo *user.User, sub *subscribe.Subscribe, userSub *user.Subscribe, notifyType string) {
	// Send user notification
	if telegramId, ok := findTelegram(userInfo); ok {
		templateData := l.buildUserNotificationData(orderInfo, sub, userSub)
		if text, err := tool.RenderTemplateToString(notifyType, templateData); err == nil {
			l.sendUserNotifyWithTelegram(telegramId, text)
		}
	}

	// Send admin notification
	adminData := l.buildAdminNotificationData(orderInfo, sub)
	if text, err := tool.RenderTemplateToString(telegram.AdminOrderNotify, adminData); err == nil {
		l.sendAdminNotifyWithTelegram(ctx, text)
	}
}

// sendRechargeNotifications sends specific notifications for balance recharge orders
func (l *ActivateOrderLogic) sendRechargeNotifications(ctx context.Context, orderInfo *order.Order, userInfo *user.User) {
	// Send user notification
	if telegramId, ok := findTelegram(userInfo); ok {
		templateData := map[string]string{
			"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
			"PaymentMethod": orderInfo.Method,
			"Time":          orderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
			"Balance":       fmt.Sprintf("%.2f", float64(userInfo.Balance)/100),
		}
		if text, err := tool.RenderTemplateToString(telegram.RechargeNotify, templateData); err == nil {
			l.sendUserNotifyWithTelegram(telegramId, text)
		}
	}

	// Send admin notification
	adminData := map[string]string{
		"OrderNo":       orderInfo.OrderNo,
		"TradeNo":       orderInfo.TradeNo,
		"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
		"SubscribeName": "余额充值",
		"OrderStatus":   "已支付",
		"OrderTime":     orderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
		"PaymentMethod": orderInfo.Method,
	}
	if text, err := tool.RenderTemplateToString(telegram.AdminOrderNotify, adminData); err == nil {
		l.sendAdminNotifyWithTelegram(ctx, text)
	}
}

// buildUserNotificationData creates template data for user notifications
func (l *ActivateOrderLogic) buildUserNotificationData(orderInfo *order.Order, sub *subscribe.Subscribe, userSub *user.Subscribe) map[string]string {
	data := map[string]string{
		"OrderNo":       orderInfo.OrderNo,
		"SubscribeName": sub.Name,
		"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
	}

	if userSub != nil {
		data["ExpireTime"] = userSub.ExpireTime.Format("2006-01-02 15:04:05")
		data["ResetTime"] = time.Now().Format("2006-01-02 15:04:05")
	}

	return data
}

// buildAdminNotificationData creates template data for admin notifications
func (l *ActivateOrderLogic) buildAdminNotificationData(orderInfo *order.Order, sub *subscribe.Subscribe) map[string]string {
	subscribeName := sub.Name
	if orderInfo.Type == OrderTypeResetTraffic {
		subscribeName = "流量重置"
	}

	return map[string]string{
		"OrderNo":       orderInfo.OrderNo,
		"TradeNo":       orderInfo.TradeNo,
		"SubscribeName": subscribeName,
		"OrderAmount":   fmt.Sprintf("%.2f", float64(orderInfo.Price)/100),
		"OrderStatus":   "已支付",
		"OrderTime":     orderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
		"PaymentMethod": orderInfo.Method,
	}
}

// sendUserNotifyWithTelegram sends a notification message to a user via Telegram
func (l *ActivateOrderLogic) sendUserNotifyWithTelegram(chatId int64, text string) {
	bot := l.svc.TelegramBotClient()
	if bot == nil {
		return
	}
	msg := tgbotapi.NewMessage(chatId, text)
	msg.ParseMode = "markdown"
	if _, err := bot.Send(msg); err != nil {
		logger.Error("Send telegram user message failed", logger.Field("error", err.Error()))
	}
}

// sendAdminNotifyWithTelegram sends a notification message to all admin users via Telegram
func (l *ActivateOrderLogic) sendAdminNotifyWithTelegram(ctx context.Context, text string) {
	bot := l.svc.TelegramBotClient()
	if bot == nil {
		return
	}
	admins, err := l.svc.UserModel.QueryAdminUsers(ctx)
	if err != nil {
		logger.WithContext(ctx).Error("Query admin users failed", logger.Field("error", err.Error()))
		return
	}

	for _, admin := range admins {
		if telegramId, ok := findTelegram(admin); ok {
			msg := tgbotapi.NewMessage(telegramId, text)
			msg.ParseMode = "markdown"
			if _, err := bot.Send(msg); err != nil {
				logger.WithContext(ctx).Error("Send telegram admin message failed", logger.Field("error", err.Error()))
			}
		}
	}
}

// findTelegram extracts Telegram chat ID from user authentication methods.
// Returns the chat ID and a boolean indicating if Telegram auth was found.
func findTelegram(u *user.User) (int64, bool) {
	for _, item := range u.AuthMethods {
		if item.AuthType == "telegram" {
			if telegramId, err := strconv.ParseInt(item.AuthIdentifier, 10, 64); err == nil {
				return telegramId, true
			}
		}
	}
	return 0, false
}
