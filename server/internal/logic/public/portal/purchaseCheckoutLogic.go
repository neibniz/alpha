package portal

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	orderLogic "github.com/neibniz/alpha/server/internal/logic/public/order"
	"github.com/neibniz/alpha/server/internal/model/log"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	"github.com/neibniz/alpha/server/internal/money"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/paymenturl"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/report"
	"github.com/neibniz/alpha/server/pkg/exchangeRate"

	paymentPlatform "github.com/neibniz/alpha/server/pkg/payment"

	"github.com/neibniz/alpha/server/internal/model/user"
	queueType "github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment/alipay"
	"github.com/neibniz/alpha/server/pkg/payment/epay"
	"github.com/neibniz/alpha/server/pkg/payment/stripe"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

// PurchaseCheckoutLogic handles the checkout process for various payment methods
// including EPay, Stripe, Alipay F2F, and balance payments
type PurchaseCheckoutLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

var balancePaymentUserColumns = []string{
	"id",
	"balance",
}

const settlementCurrencyCNY = "CNY"

// NewPurchaseCheckoutLogic creates a new instance of PurchaseCheckoutLogic
// for handling purchase checkout operations across different payment platforms
func NewPurchaseCheckoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PurchaseCheckoutLogic {
	return &PurchaseCheckoutLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PurchaseCheckout processes the checkout for an order using the specified payment method
// It validates the order, retrieves payment configuration, and routes to the appropriate payment handler
func (l *PurchaseCheckoutLogic) PurchaseCheckout(req *types.CheckoutOrderRequest) (resp *types.CheckoutOrderResponse, err error) {
	if req == nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "checkout request is required")
	}

	// Validate and retrieve order information
	orderInfo, err := l.svcCtx.OrderModel.FindOneForCheckout(l.ctx, req.OrderNo)
	if err != nil {
		l.Logger.Error("[PurchaseCheckout] Find order failed", logger.Field("error", err.Error()), logger.Field("orderNo", req.OrderNo))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.OrderNotExist), "order not exist: %v", req.OrderNo)
	}
	currentUser, ok := authenticatedOrderOwner(l.ctx, orderInfo)
	if !ok {
		l.Errorw("[PurchaseCheckout] order owner mismatch",
			logger.Field("orderNo", req.OrderNo),
			logger.Field("orderUserId", orderInfo.UserId),
		)
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "invalid order access")
	}
	if err = l.enforceSubscriptionExpiry(orderInfo, currentUser); err != nil {
		return nil, err
	}

	if orderInfo.Status == 2 {
		if err = l.enqueueActivation(orderInfo.OrderNo); err != nil {
			return nil, err
		}
		return &types.CheckoutOrderResponse{Type: "processing"}, nil
	}
	// Verify order is in pending payment status (status = 1)
	if orderInfo.Status != 1 {
		l.Logger.Error("[PurchaseCheckout] Order status error", logger.Field("status", orderInfo.Status))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.OrderStatusError), "order status error: %v", orderInfo.Status)
	}
	if orderInfo.Amount == 0 {
		if err = l.completeFreeOrder(orderInfo); err != nil {
			return nil, err
		}
		return &types.CheckoutOrderResponse{Type: "free"}, nil
	}

	// Retrieve payment method configuration
	paymentConfig, err := l.svcCtx.PaymentModel.FindOneForCheckout(l.ctx, orderInfo.PaymentId)
	if err != nil {
		l.Logger.Error("[PurchaseCheckout] Database query error", logger.Field("error", err.Error()), logger.Field("payment", orderInfo.Method))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method error: %v", err.Error())
	}
	if err = ensurePaymentEnabled(paymentConfig); err != nil {
		return nil, err
	}
	// Route to appropriate payment handler based on payment platform
	switch paymentPlatform.ParsePlatform(orderInfo.Method) {
	case paymentPlatform.EPay:
		// Process EPay payment - generates payment URL for redirect
		url, err := l.epayPayment(paymentConfig, orderInfo, req.ReturnUrl)
		if err != nil {
			l.Logger.Error("[PurchaseCheckout] epay error", logger.Field("error", err.Error()))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "epayPayment error: %v", err.Error())
		}
		resp = &types.CheckoutOrderResponse{
			CheckoutUrl: url,
			Type:        "url", // Client should redirect to URL
		}

	case paymentPlatform.Stripe:
		// Process Stripe payment - creates payment sheet for client-side processing
		stripePayment, err := l.stripePayment(paymentConfig.Config, orderInfo, "")
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "stripePayment error: %v", err.Error())
		}
		resp = &types.CheckoutOrderResponse{
			Type:   "stripe", // Client should use Stripe SDK
			Stripe: stripePayment,
		}

	case paymentPlatform.AlipayF2F:
		// Process Alipay Face-to-Face payment - generates QR code
		url, err := l.alipayF2fPayment(paymentConfig, orderInfo)
		if err != nil {
			l.Errorw("[PurchaseCheckout] alipayF2fPayment error", logger.Field("error", err.Error()))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "alipayF2fPayment error: %v", err.Error())
		}
		resp = &types.CheckoutOrderResponse{
			Type:        "qr", // Client should display QR code
			CheckoutUrl: url,
		}

	case paymentPlatform.CryptoSaaS:
		// Process EPay payment - generates payment URL for redirect
		url, err := l.CryptoSaaSPayment(paymentConfig, orderInfo, req.ReturnUrl)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "epayPayment error: %v", err.Error())
		}
		resp = &types.CheckoutOrderResponse{
			CheckoutUrl: url,
			Type:        "url", // Client should redirect to URL
		}

	case paymentPlatform.Balance:
		// Process balance payment - validate user and process payment immediately
		if orderInfo.UserId == 0 {
			l.Errorw("[PurchaseCheckout] user not found", logger.Field("userId", orderInfo.UserId))
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.UserNotExist), "user not found")
		}

		// Process balance payment with gift amount priority logic
		if err = l.balancePayment(currentUser, orderInfo); err != nil {
			return nil, err
		}

		resp = &types.CheckoutOrderResponse{
			Type: "balance", // Payment completed immediately
		}

	default:
		l.Errorw("[PurchaseCheckout] payment method not found", logger.Field("method", orderInfo.Method))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "payment method not found")
	}
	return
}

func (l *PurchaseCheckoutLogic) enforceSubscriptionExpiry(orderInfo *order.Order, u *user.User) error {
	switch orderInfo.Type {
	case orderLogic.OrderTypeSubscribe:
		sub, ok := subscriptionExpiryProductFromSnapshot(orderInfo.ProductSnapshot)
		if !ok {
			var err error
			sub, err = l.svcCtx.SubscribeModel.FindOneForSubscriptionExpiry(l.ctx, orderInfo.SubscribeId)
			if err != nil {
				return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find subscribe error: %v", err.Error())
			}
		}
		return orderLogic.EnforceMembershipSubscriptionExpiryPolicy(l.ctx, l.svcCtx, u, sub, orderInfo.Quantity, nil)
	case orderLogic.OrderTypeRenewal:
		userSub, err := l.svcCtx.UserModel.FindOneSubscribeByTokenForExpiry(l.ctx, orderInfo.SubscribeToken)
		if err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user subscribe error: %v", err.Error())
		}
		sub, ok := subscriptionExpiryProductFromSnapshot(orderInfo.ProductSnapshot)
		if !ok {
			if snapshotSub, snapshotOK := subscriptionExpiryProductFromSnapshot(userSub.ProductSnapshot); snapshotOK {
				sub = snapshotSub
			} else {
				sub, err = l.svcCtx.SubscribeModel.FindOneForSubscriptionExpiry(l.ctx, userSub.SubscribeId)
				if err != nil {
					return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find subscribe error: %v", err.Error())
				}
			}
		}
		return orderLogic.EnforceMembershipSubscriptionExpiryPolicy(l.ctx, l.svcCtx, u, sub, orderInfo.Quantity, &userSub.ExpireTime)
	default:
		return nil
	}
}

func subscriptionExpiryProductFromSnapshot(raw string) (*modelSubscribe.Subscribe, bool) {
	if raw == "" {
		return nil, false
	}
	var snapshot struct {
		Kind     string `json:"kind"`
		Name     string `json:"name"`
		ID       int64  `json:"id,omitempty"`
		UnitTime string `json:"unit_time,omitempty"`
	}
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return nil, false
	}
	if snapshot.Kind != ordersnapshot.KindSubscribe || snapshot.Name == "" {
		return nil, false
	}
	return &modelSubscribe.Subscribe{
		Id:       snapshot.ID,
		UnitTime: snapshot.UnitTime,
	}, true
}

// alipayF2fPayment processes Alipay Face-to-Face payment by generating a QR code
// It handles currency conversion and creates a pre-payment trade for QR code scanning
func (l *PurchaseCheckoutLogic) alipayF2fPayment(pay *payment.Payment, info *order.Order) (string, error) {
	// Parse Alipay F2F configuration from payment settings
	f2FConfig := &payment.AlipayF2FConfig{}
	if err := f2FConfig.Unmarshal([]byte(pay.Config)); err != nil {
		l.Errorw("[PurchaseCheckout] Unmarshal Alipay config error", logger.Field("error", err.Error()))
		return "", errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Unmarshal error: %s", err.Error())
	}

	// Build notification URL only from administrator-controlled configuration.
	notifyURL, err := l.paymentNotifyURL(pay)
	if err != nil {
		return "", err
	}

	// Initialize Alipay client with configuration
	client := alipay.NewClient(alipay.Config{
		AppId:       f2FConfig.AppId,
		PrivateKey:  f2FConfig.PrivateKey,
		PublicKey:   f2FConfig.PublicKey,
		InvoiceName: f2FConfig.InvoiceName,
		NotifyURL:   notifyURL,
	})

	convertAmount, err := l.settlementAmountCNY(info)
	if err != nil {
		return "", err
	}

	// Create pre-payment trade and generate QR code
	QRCode, err := client.PreCreateTrade(l.ctx, alipay.Order{
		OrderNo: info.OrderNo,
		Amount:  convertAmount,
	})
	if err != nil {
		l.Errorw("[PurchaseCheckout] PreCreateTrade error", logger.Field("error", err.Error()))
		return "", errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "PreCreateTrade error: %s", err.Error())
	}
	return QRCode, nil
}

// stripePayment processes Stripe payment by creating a payment sheet
// It supports various payment methods including WeChat Pay and Alipay through Stripe
func (l *PurchaseCheckoutLogic) stripePayment(config string, info *order.Order, identifier string) (*types.StripePayment, error) {
	// Parse Stripe configuration from payment settings
	stripeConfig := &payment.StripeConfig{}

	if err := stripeConfig.Unmarshal([]byte(config)); err != nil {
		l.Errorw("[PurchaseCheckout] Unmarshal Stripe config error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Unmarshal error: %s", err.Error())
	}

	// Initialize Stripe client with API credentials
	client := stripe.NewClient(stripe.Config{
		SecretKey:     stripeConfig.SecretKey,
		PublicKey:     stripeConfig.PublicKey,
		WebhookSecret: stripeConfig.WebhookSecret,
	})

	convertAmount, err := l.settlementAmountCNY(info)
	if err != nil {
		return nil, err
	}

	// Create Stripe payment sheet for client-side processing
	result, err := client.CreatePaymentSheet(&stripe.Order{
		OrderNo:   info.OrderNo,
		Subscribe: strconv.FormatInt(info.SubscribeId, 10),
		Amount:    convertAmount,
		Currency:  "cny",
		Payment:   stripeConfig.Payment,
	},
		&stripe.User{
			UserId: info.UserId,
			Email:  identifier,
		})
	if err != nil {
		l.Errorw("[PurchaseCheckout] CreatePaymentSheet error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "CreatePaymentSheet error: %s", err.Error())
	}

	// Prepare response data for client-side Stripe integration
	stripePayment := &types.StripePayment{
		PublishableKey: stripeConfig.PublicKey,
		ClientSecret:   result.ClientSecret,
		Method:         stripeConfig.Payment,
	}

	// Save Stripe trade number to order for tracking
	info.TradeNo = result.TradeNo
	err = l.svcCtx.OrderModel.UpdateFields(l.ctx, info, order.OrderFieldsUpdate{
		TradeNo: order.UpdateValue(result.TradeNo),
	})
	if err != nil {
		l.Errorw("[PurchaseCheckout] Update order error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Update error: %s", err.Error())
	}
	return stripePayment, nil
}

// epayPayment processes EPay payment by generating a payment URL for redirect
// It handles currency conversion and creates a payment URL for external payment processing
func (l *PurchaseCheckoutLogic) epayPayment(config *payment.Payment, info *order.Order, returnUrl string) (string, error) {
	// Parse EPay configuration from payment settings
	epayConfig := &payment.EPayConfig{}
	if err := epayConfig.Unmarshal([]byte(config.Config)); err != nil {
		l.Errorw("[PurchaseCheckout] Unmarshal EPay config error", logger.Field("error", err.Error()))
		return "", errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Unmarshal error: %s", err.Error())
	}
	// Initialize EPay client with merchant credentials
	client := epay.NewClient(epayConfig.Pid, epayConfig.Url, epayConfig.Key, epayConfig.Type)
	convertAmount, err := l.settlementAmountCNY(info)
	if err != nil {
		return "", err
	}

	// gateway mod
	notifyURL, err := l.paymentNotifyURL(config)
	if err != nil {
		return "", err
	}
	returnURL, err := l.paymentReturnURL(returnUrl)
	if err != nil {
		return "", err
	}

	// Create payment URL for user redirection
	url := client.CreatePayUrl(epay.Order{
		Name:      l.svcCtx.ConfigSnapshot().Site.SiteName,
		Amount:    float64(convertAmount) / 100,
		OrderNo:   info.OrderNo,
		SignType:  "MD5",
		NotifyUrl: notifyURL,
		ReturnUrl: returnURL,
	})
	return url, nil
}

// CryptoSaaSPayment processes CryptoSaaSPayment payment by generating a payment URL for redirect
// It handles currency conversion and creates a payment URL for external payment processing
func (l *PurchaseCheckoutLogic) CryptoSaaSPayment(config *payment.Payment, info *order.Order, returnUrl string) (string, error) {
	// Parse EPay configuration from payment settings
	epayConfig := &payment.CryptoSaaSConfig{}
	if err := epayConfig.Unmarshal([]byte(config.Config)); err != nil {
		l.Errorw("[PurchaseCheckout] Unmarshal EPay config error", logger.Field("error", err.Error()))
		return "", errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "Unmarshal error: %s", err.Error())
	}
	// Initialize EPay client with merchant credentials
	client := epay.NewClient(epayConfig.AccountID, epayConfig.Endpoint, epayConfig.SecretKey, epayConfig.Type)
	convertAmount, err := l.settlementAmountCNY(info)
	if err != nil {
		return "", err
	}

	notifyURL, err := l.paymentNotifyURL(config)
	if err != nil {
		return "", err
	}
	returnURL, err := l.paymentReturnURL(returnUrl)
	if err != nil {
		return "", err
	}
	// Create payment URL for user redirection
	url := client.CreatePayUrl(epay.Order{
		Name:      l.svcCtx.ConfigSnapshot().Site.SiteName,
		Amount:    float64(convertAmount) / 100,
		OrderNo:   info.OrderNo,
		SignType:  "MD5",
		NotifyUrl: notifyURL,
		ReturnUrl: returnURL,
	})
	return url, nil
}

func (l *PurchaseCheckoutLogic) settlementAmountCNY(info *order.Order) (int64, error) {
	if info == nil {
		return 0, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "order is required")
	}
	var convertAmount int64
	if info.SettlementAmount > 0 {
		if info.SettlementCurrency != settlementCurrencyCNY {
			return 0, errors.Wrap(xerr.NewErrCode(xerr.OrderStatusError), "order settlement currency is incompatible")
		}
		convertAmount = info.SettlementAmount
	} else if info.SettlementCurrency != "" {
		return 0, errors.Wrap(xerr.NewErrCode(xerr.OrderStatusError), "order settlement snapshot is invalid")
	} else {
		amount, err := l.queryExchangeRate(settlementCurrencyCNY, info.Amount)
		if err != nil {
			l.Errorw("[PurchaseCheckout] queryExchangeRate error", logger.Field("error", err.Error()))
			return 0, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "queryExchangeRate error: %s", err.Error())
		}
		convertAmount, err = money.MajorToMinor(amount)
		if err != nil {
			return 0, err
		}
	}
	if convertAmount <= 0 {
		return 0, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "settlement amount must be positive")
	}
	compatible, err := l.svcCtx.OrderModel.SetSettlementIfCompatible(l.ctx, info, settlementCurrencyCNY, convertAmount)
	if err != nil {
		return 0, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "save settlement snapshot: %s", err.Error())
	}
	if !compatible {
		return 0, errors.Wrap(xerr.NewErrCode(xerr.OrderStatusError), "order settlement changed during checkout")
	}
	info.SettlementAmount = convertAmount
	info.SettlementCurrency = settlementCurrencyCNY
	return convertAmount, nil
}

func (l *PurchaseCheckoutLogic) paymentNotifyURL(pay *payment.Payment) (string, error) {
	if pay == nil {
		return "", errors.New("payment configuration is required")
	}
	snapshot := l.svcCtx.ConfigSnapshot()
	notifyURL, err := paymenturl.NotifyURL(pay.Domain, snapshot.Site.Host, pay.Platform, pay.Token, report.IsGatewayMode())
	if err != nil {
		return "", errors.Wrap(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid payment callback configuration"), err.Error())
	}
	return notifyURL, nil
}

func (l *PurchaseCheckoutLogic) paymentReturnURL(raw string) (string, error) {
	snapshot := l.svcCtx.ConfigSnapshot()
	trusted := make([]string, 0, len(snapshot.HTTP.AllowedOrigins)+1)
	trusted = append(trusted, snapshot.Site.Host)
	trusted = append(trusted, snapshot.HTTP.AllowedOrigins...)
	returnURL, err := paymenturl.ValidateReturnURL(raw, trusted...)
	if err != nil {
		return "", errors.Wrap(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid payment return URL"), err.Error())
	}
	return returnURL, nil
}

// queryExchangeRate converts the order amount from system currency to target currency
// It retrieves the current exchange rate and performs currency conversion if needed
func (l *PurchaseCheckoutLogic) queryExchangeRate(to string, src int64) (amount float64, err error) {
	// Convert cents to decimal amount
	amount = float64(src) / float64(100)

	// No conversion needed if target currency matches system currency
	currency, rate := l.svcCtx.CurrencyRuntime()
	if to == currency.Unit {
		return amount, nil
	}

	if rate != 0 && to == "CNY" {
		convertedMinor, convertErr := money.ConvertMinor(src, rate)
		if convertErr != nil {
			return 0, convertErr
		}
		return float64(convertedMinor) / 100, nil
	}

	// Never silently treat one currency as another when no rate is available.
	if currency.AccessKey == "" {
		return 0, errors.New("exchange rate is unavailable")
	}

	// Convert currency if system currency differs from target currency
	result, err := exchangeRate.GetExchangeRete(currency.Unit, to, currency.AccessKey, 1)
	if err != nil {
		l.Logger.Error("[PurchaseCheckout] QueryExchangeRate error", logger.Field("error", err.Error()))
		return 0, err
	}
	convertedMinor, err := money.ConvertMinor(src, result)
	if err != nil {
		return 0, err
	}
	if !l.svcCtx.SetExchangeRateForCurrency(currency.Unit, result) {
		return 0, errors.New("currency configuration changed during checkout")
	}
	return float64(convertedMinor) / 100, nil
}

// balancePayment processes balance payment with gift amount priority logic
// It prioritizes using gift amount first, then regular balance, and creates proper audit logs
func (l *PurchaseCheckoutLogic) balancePayment(u *user.User, o *order.Order) error {
	var userInfo user.User
	var err error
	if o.Amount == 0 {
		// No payment required for zero-amount orders
		l.Logger.Info(
			"[PurchaseCheckout] No payment required for zero-amount order",
			logger.Field("orderNo", o.OrderNo),
			logger.Field("userId", u.Id),
		)
		err = l.svcCtx.OrderModel.UpdateOrderStatus(l.ctx, o.OrderNo, 2)
		if err != nil {
			l.Errorw("[PurchaseCheckout] Update order status error",
				logger.Field("error", err.Error()),
				logger.Field("orderNo", o.OrderNo),
				logger.Field("userId", u.Id))
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "Update order status error: %s", err.Error())
		}
		goto activation
	}

	err = l.svcCtx.UserModel.Transaction(l.ctx, func(db *gorm.DB) error {
		var lockedOrder order.Order
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&order.Order{}).
			Select("id", "user_id", "order_no", "amount", "status").
			Where("order_no = ?", o.OrderNo).
			First(&lockedOrder).Error; err != nil {
			return err
		}
		if lockedOrder.Status == 2 {
			return nil
		}
		if lockedOrder.Status != 1 {
			return errors.Wrapf(xerr.NewErrCode(xerr.OrderStatusError), "order status error: %v", lockedOrder.Status)
		}
		if lockedOrder.UserId != u.Id {
			return errors.Wrapf(xerr.NewErrCode(xerr.InvalidAccess), "invalid order access")
		}
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&user.User{}).
			Select(balancePaymentUserColumns).
			Where("id = ?", u.Id).
			First(&userInfo).Error; err != nil {
			return err
		}

		if userInfo.Balance < lockedOrder.Amount {
			return errors.Wrapf(xerr.NewErrCode(xerr.InsufficientBalance),
				"Insufficient balance: required %d, available %d", lockedOrder.Amount, userInfo.Balance)
		}

		balanceUsed := lockedOrder.Amount
		userInfo.Balance -= balanceUsed

		if err = l.svcCtx.UserModel.UpdateUserBalance(l.ctx, userInfo.Id, userInfo.Balance, db); err != nil {
			return err
		}

		// Create balance log if regular balance was used
		if balanceUsed > 0 {
			balanceLog := &log.Balance{
				Amount:    balanceUsed,
				Type:      log.BalanceTypePayment, // Type 3 represents payment deduction
				OrderNo:   o.OrderNo,
				Balance:   userInfo.Balance,
				Timestamp: time.Now().UnixMilli(),
			}
			content, _ := balanceLog.Marshal()
			err = db.Create(&log.SystemLog{
				Type:     log.TypeBalance.Uint8(),
				ObjectID: userInfo.Id,
				Date:     time.Now().Format(time.DateOnly),
				Content:  string(content),
			}).Error
			if err != nil {
				return err
			}
		}

		o.Status = 2
		return l.svcCtx.OrderModel.UpdateFields(l.ctx, &lockedOrder, order.OrderFieldsUpdate{
			Status: order.UpdateValue(uint8(2)),
		}, db)
	})

	if err != nil {
		l.Errorw("[PurchaseCheckout] Balance payment transaction error",
			logger.Field("error", err.Error()),
			logger.Field("orderNo", o.OrderNo),
			logger.Field("userId", u.Id))
		return err
	}

activation:
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventBusiness)
	}
	// Enqueue order activation task for immediate processing
	payload := queueType.ForthwithActivateOrderPayload{
		OrderNo: o.OrderNo,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		l.Errorw("[PurchaseCheckout] Marshal activation payload error", logger.Field("error", err.Error()))
		return err
	}

	task := asynq.NewTask(queueType.ForthwithActivateOrder, bytes)
	_, err = l.svcCtx.Queue.EnqueueContext(l.ctx, task)
	if err != nil {
		l.Errorw("[PurchaseCheckout] Enqueue activation task error", logger.Field("error", err.Error()))
		return err
	}

	l.Logger.Info("[PurchaseCheckout] Balance payment completed successfully",
		logger.Field("orderNo", o.OrderNo),
		logger.Field("userId", u.Id))
	return nil
}

func (l *PurchaseCheckoutLogic) completeFreeOrder(o *order.Order) error {
	paid, err := l.svcCtx.OrderModel.MarkPaid(l.ctx, o.OrderNo, "free")
	if err != nil {
		l.Errorw("[PurchaseCheckout] Mark free order paid error", logger.Field("error", err.Error()), logger.Field("orderNo", o.OrderNo))
		return err
	}
	if !paid && o.Status != 2 {
		return errors.Wrapf(xerr.NewErrCode(xerr.OrderStatusError), "order status changed before free checkout")
	}
	if paid && l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventBusiness)
	}
	return l.enqueueActivation(o.OrderNo)
}

func (l *PurchaseCheckoutLogic) enqueueActivation(orderNo string) error {
	payload := queueType.ForthwithActivateOrderPayload{OrderNo: orderNo}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(queueType.ForthwithActivateOrder, bytes)
	_, err = l.svcCtx.Queue.EnqueueContext(l.ctx, task)
	return err
}
