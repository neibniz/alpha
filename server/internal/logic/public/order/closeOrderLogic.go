package order

import (
	"context"
	"encoding/json"
	"time"

	"github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/pkg/constant"
	paymentPlatform "github.com/neibniz/alpha/server/pkg/payment"
	"github.com/neibniz/alpha/server/pkg/payment/epay"
	"github.com/neibniz/alpha/server/pkg/payment/stripe"
	queue "github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neibniz/alpha/server/internal/model/order"
	paymentModel "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment/alipay"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CloseOrderLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCloseOrderLogic Close order
func NewCloseOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CloseOrderLogic {
	return &CloseOrderLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CloseOrderLogic) CloseOrder(req *types.CloseOrderRequest) error {
	return l.closeOrder(req, 0, false)
}

func (l *CloseOrderLogic) CloseOrderForCurrentUser(req *types.CloseOrderRequest) error {
	currentUser, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok || currentUser == nil || currentUser.Id <= 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is not available")
	}
	return l.closeOrder(req, currentUser.Id, true)
}

func (l *CloseOrderLogic) closeOrder(req *types.CloseOrderRequest, userID int64, scoped bool) error {
	// Find order information by order number
	var orderInfo *order.Order
	var err error
	if scoped {
		orderInfo, err = l.svcCtx.OrderModel.FindOneForCloseByUser(l.ctx, req.OrderNo, userID)
	} else {
		orderInfo, err = l.svcCtx.OrderModel.FindOneForClose(l.ctx, req.OrderNo)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if scoped {
				return errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "order is not accessible")
			}
			return nil
		}
		l.Errorw("[CloseOrder] Find order info failed",
			logger.Field("error", err.Error()),
		)
		return errors.Wrap(xerr.NewErrCode(xerr.DatabaseQueryError), "find order for close")
	}
	// If the order status is not 1, it means that the order has been closed or paid
	if orderInfo.Status != 1 {
		l.Infow("[CloseOrder] Order status is not 1",
			logger.Field("orderNo", req.OrderNo),
			logger.Field("status", orderInfo.Status),
		)
		return nil
	}
	if l.confirmationPayment(orderInfo) {
		paid, err := l.svcCtx.OrderModel.MarkPaid(l.ctx, orderInfo.OrderNo, orderInfo.TradeNo)
		if err != nil {
			return err
		}
		if paid {
			payload := queue.ForthwithActivateOrderPayload{OrderNo: orderInfo.OrderNo}
			bytes, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			task := asynq.NewTask(queue.ForthwithActivateOrder, bytes)
			_, err = l.svcCtx.Queue.EnqueueContext(l.ctx, task)
			return err
		}
		return nil
	}

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		updated, err := l.svcCtx.OrderModel.UpdateStatusIfCurrent(l.ctx, orderInfo, 1, 3, tx)
		if err != nil {
			l.Errorw("[CloseOrder] Update order status failed",
				logger.Field("error", err.Error()),
				logger.Field("orderNo", req.OrderNo),
			)
			return err
		}
		if !updated {
			return nil
		}
		// If User ID is 0, it means that the order is a guest order and does not need to be refunded, the order can be deleted directly
		if orderInfo.UserId == 0 {
			err = tx.Model(&order.Order{}).Where("order_no = ?", req.OrderNo).Delete(&order.Order{}).Error
			if err != nil {
				l.Errorw("[CloseOrder] Delete order failed",
					logger.Field("error", err.Error()),
					logger.Field("orderNo", req.OrderNo),
				)
				return err
			}
			return nil
		}
		// Restore each prepaid wallet source recorded on the order. The status
		// transition above makes this refund idempotent.
		if orderInfo.CommissionDeduction > 0 || orderInfo.GiftAmount > 0 {
			var userInfo user.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Model(&user.User{}).
				Select(orderWalletUserColumns).
				Where("id = ?", orderInfo.UserId).
				First(&userInfo).Error; err != nil {
				l.Errorw("[CloseOrder] Find user info failed",
					logger.Field("error", err.Error()),
					logger.Field("user_id", orderInfo.UserId),
				)
				return err
			}
			now := time.Now()
			if orderInfo.CommissionDeduction > 0 {
				userInfo.Commission += orderInfo.CommissionDeduction
				if err := l.svcCtx.UserModel.UpdateUserCommission(l.ctx, userInfo.Id, userInfo.Commission, tx); err != nil {
					return err
				}
				commissionLog := log.Commission{
					Type:      log.CommissionTypePaymentRefund,
					OrderNo:   orderInfo.OrderNo,
					Amount:    orderInfo.CommissionDeduction,
					Balance:   userInfo.Commission,
					Timestamp: now.UnixMilli(),
				}
				content, marshalErr := commissionLog.Marshal()
				if marshalErr != nil {
					return marshalErr
				}
				if err := tx.WithContext(l.ctx).Create(&log.SystemLog{
					Type:     log.TypeCommission.Uint8(),
					Date:     now.Format(time.DateOnly),
					ObjectID: userInfo.Id,
					Content:  string(content),
				}).Error; err != nil {
					return err
				}
			}

			if orderInfo.GiftAmount > 0 {
				userInfo.GiftAmount += orderInfo.GiftAmount
				if err := l.svcCtx.UserModel.UpdateUserGiftAmount(l.ctx, userInfo.Id, userInfo.GiftAmount, tx); err != nil {
					return err
				}
				giftLog := log.Gift{
					Type:        log.GiftTypeIncrease,
					OrderNo:     orderInfo.OrderNo,
					SubscribeId: 0,
					Amount:      orderInfo.GiftAmount,
					Balance:     userInfo.GiftAmount,
					Remark:      "Order cancellation refund",
					Timestamp:   now.UnixMilli(),
				}
				content, marshalErr := giftLog.Marshal()
				if marshalErr != nil {
					return marshalErr
				}
				if err := tx.WithContext(l.ctx).Create(&log.SystemLog{
					Type:     log.TypeGift.Uint8(),
					Date:     now.Format(time.DateOnly),
					ObjectID: userInfo.Id,
					Content:  string(content),
				}).Error; err != nil {
					return err
				}
			}
			// update user cache
			if err = l.svcCtx.UserModel.UpdateUserCache(l.ctx, &userInfo); err != nil {
				return err
			}
		}

		if orderInfo.Type != OrderTypeSubscribe {
			return nil
		}

		if e := restoreSubscribeInventory(l.ctx, tx, orderInfo.SubscribeId); e != nil {
			l.Errorw("[CloseOrder] Restore subscribe inventory failed",
				logger.Field("error", e.Error()),
				logger.Field("subscribeId", orderInfo.SubscribeId),
			)
			return e
		}

		return nil
	})
	if err != nil {
		logger.Errorf("[CloseOrder] Transaction failed: %v", err.Error())
		return err
	}
	return nil
}

// confirmationPayment Determine whether the payment is successful
//
//nolint:unused
func (l *CloseOrderLogic) confirmationPayment(order *order.Order) bool {
	if !l.shouldConfirmGatewayPayment(order) {
		return false
	}

	paymentConfig, err := l.svcCtx.PaymentModel.FindOneForGatewayCheck(l.ctx, order.PaymentId)
	if err != nil {
		l.Errorw("[CloseOrder] Find payment config failed", logger.Field("error", err.Error()), logger.Field("paymentMark", order.Method))
		return false
	}
	switch order.Method {
	case AlipayF2f:
		if order.TradeNo != "" && l.queryAlipay(paymentConfig, order.TradeNo) {
			return true
		}
	case StripeAlipay:
		if order.TradeNo != "" && l.queryStripe(paymentConfig, order.TradeNo) {
			return true
		}
	case StripeWeChatPay:
		if order.TradeNo != "" && l.queryStripe(paymentConfig, order.TradeNo) {
			return true
		}
	default:
		switch paymentPlatform.ParsePlatform(order.Method) {
		case paymentPlatform.EPay, paymentPlatform.CryptoSaaS:
			if l.queryEPay(paymentConfig, order.OrderNo) {
				return true
			}
		default:
			l.Infow("[CloseOrder] Unsupported payment method", logger.Field("paymentMethod", order.Method))
		}
	}
	return false
}

func (l *CloseOrderLogic) shouldConfirmGatewayPayment(order *order.Order) bool {
	if order == nil {
		return false
	}
	switch order.Method {
	case AlipayF2f, StripeAlipay, StripeWeChatPay:
		return order.TradeNo != ""
	default:
		switch paymentPlatform.ParsePlatform(order.Method) {
		case paymentPlatform.EPay, paymentPlatform.CryptoSaaS:
			return true
		default:
			l.Infow("[CloseOrder] Unsupported payment method", logger.Field("paymentMethod", order.Method))
			return false
		}
	}
}

func (l *CloseOrderLogic) queryEPay(paymentConfig *paymentModel.Payment, orderNo string) bool {
	switch paymentPlatform.ParsePlatform(paymentConfig.Platform) {
	case paymentPlatform.EPay:
		config := paymentModel.EPayConfig{}
		if err := json.Unmarshal([]byte(paymentConfig.Config), &config); err != nil {
			l.Errorw("[CloseOrder] Unmarshal payment config failed", logger.Field("error", err.Error()), logger.Field("payment_id", paymentConfig.Id), logger.Field("platform", paymentConfig.Platform))
			return false
		}
		client := epay.NewClient(config.Pid, config.Url, config.Key, config.Type)
		return client.QueryOrderStatus(orderNo)
	case paymentPlatform.CryptoSaaS:
		config := paymentModel.CryptoSaaSConfig{}
		if err := json.Unmarshal([]byte(paymentConfig.Config), &config); err != nil {
			l.Errorw("[CloseOrder] Unmarshal payment config failed", logger.Field("error", err.Error()), logger.Field("payment_id", paymentConfig.Id), logger.Field("platform", paymentConfig.Platform))
			return false
		}
		client := epay.NewClient(config.AccountID, config.Endpoint, config.SecretKey, config.Type)
		if client.QueryOrderStatus(orderNo) {
			return true
		}
	default:
		return false
	}
	return false
}

// queryAlipay Query Alipay payment status
//
//nolint:unused
func (l *CloseOrderLogic) queryAlipay(paymentConfig *paymentModel.Payment, TradeNo string) bool {
	config := paymentModel.AlipayF2FConfig{}
	if err := json.Unmarshal([]byte(paymentConfig.Config), &config); err != nil {
		l.Errorw("[CloseOrder] Unmarshal payment config failed", logger.Field("error", err.Error()), logger.Field("payment_id", paymentConfig.Id), logger.Field("platform", paymentConfig.Platform))
		return false
	}
	client := alipay.NewClient(alipay.Config{
		AppId:       config.AppId,
		PrivateKey:  config.PrivateKey,
		PublicKey:   config.PublicKey,
		InvoiceName: config.InvoiceName,
	})
	status, err := client.QueryTrade(l.ctx, TradeNo)
	if err != nil {
		l.Errorw("[CloseOrder] Query trade failed", logger.Field("error", err.Error()), logger.Field("TradeNo", TradeNo))
		return false
	}
	if status == alipay.Success || status == alipay.Finished {
		return true
	}
	return false
}

// queryStripe Query Stripe payment status
//
//nolint:unused
func (l *CloseOrderLogic) queryStripe(paymentConfig *paymentModel.Payment, TradeNo string) bool {
	config := paymentModel.StripeConfig{}
	if err := json.Unmarshal([]byte(paymentConfig.Config), &config); err != nil {
		l.Errorw("[CloseOrder] Unmarshal payment config failed", logger.Field("error", err.Error()), logger.Field("payment_id", paymentConfig.Id), logger.Field("platform", paymentConfig.Platform))
		return false
	}
	client := stripe.NewClient(stripe.Config{
		PublicKey:     config.PublicKey,
		SecretKey:     config.SecretKey,
		WebhookSecret: config.WebhookSecret,
	})
	status, err := client.QueryOrderStatus(TradeNo)
	if err != nil {
		l.Errorw("[CloseOrder] Query order status failed", logger.Field("error", err.Error()), logger.Field("TradeNo", TradeNo))
		return false
	}
	return status
}
