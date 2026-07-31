package user

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	modelOrder "github.com/neibniz/alpha/server/internal/model/order"
	modelSubscribe "github.com/neibniz/alpha/server/internal/model/subscribe"
	modelUser "github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/orderaudit"
	"github.com/neibniz/alpha/server/internal/orderpricing"
	"github.com/neibniz/alpha/server/internal/ordersnapshot"
	"github.com/neibniz/alpha/server/internal/productpolicy"
	"github.com/neibniz/alpha/server/internal/realtime"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/timex"
	"github.com/neibniz/alpha/server/pkg/tool"
	"github.com/neibniz/alpha/server/pkg/xerr"
	queue "github.com/neibniz/alpha/server/queue/types"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	issueAmountComplimentary = "complimentary"
	issueAmountCurrentPrice  = "current_price"
	adminOrderSource         = "admin"
)

type IssueUserSubscribeOrderLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type issueUserSubscribeOrderData struct {
	product *modelSubscribe.Subscribe
	quote   types.IssueUserSubscribeOrderQuote
	isNew   bool
}

func NewIssueUserSubscribeOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IssueUserSubscribeOrderLogic {
	return &IssueUserSubscribeOrderLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *IssueUserSubscribeOrderLogic) Quote(req *types.IssueUserSubscribeOrderRequest) (*types.IssueUserSubscribeOrderQuote, error) {
	data, err := l.prepare(req)
	if err != nil {
		return nil, err
	}
	return &data.quote, nil
}

func (l *IssueUserSubscribeOrderLogic) Issue(req *types.IssueUserSubscribeOrderRequest) (*types.IssueUserSubscribeOrderResponse, error) {
	if err := validateIssueUserSubscribeOrderRequest(req); err != nil {
		return nil, err
	}

	payment, err := l.svcCtx.PaymentModel.FindOneForOrderRecord(l.ctx, -1)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.PaymentMethodNotFound), "default order payment method not found: %v", err)
	}
	orderNo, err := tool.GenerateTradeNo()
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate order number")
	}

	var data *issueUserSubscribeOrderData
	var orderInfo *modelOrder.Order
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		data, err = l.lockAndPrepareIssue(tx, req, time.Now().UTC())
		if err != nil {
			return err
		}
		snapshot := ordersnapshot.FromSubscribe(data.product)
		orderInfo = &modelOrder.Order{
			UserId:          req.UserId,
			OrderNo:         orderNo,
			Source:          adminOrderSource,
			Type:            1,
			Quantity:        req.Quantity,
			Price:           data.quote.ListPrice,
			Amount:          data.quote.RecordedAmount,
			Discount:        data.quote.ListPrice - data.quote.RecordedAmount,
			PaymentId:       payment.Id,
			Method:          adminOrderSource,
			TradeNo:         strings.TrimSpace(req.TradeNo),
			Status:          2,
			SubscribeId:     req.SubscribeId,
			ProductName:     snapshot.Name,
			ProductSnapshot: ordersnapshot.Marshal(snapshot),
			IsNew:           data.isNew,
		}
		orderaudit.FillNetwork(l.ctx, l.svcCtx, orderInfo)
		return tx.Create(orderInfo).Error
	})
	if err != nil {
		return nil, errors.Wrapf(err, "issue subscription order")
	}
	if err := l.svcCtx.SubscribeModel.ClearCache(l.ctx, req.SubscribeId); err != nil {
		l.Errorw("[IssueUserSubscribeOrder] clear product cache failed", logger.Field("error", err.Error()), logger.Field("subscribe_id", req.SubscribeId))
	}

	payload, err := json.Marshal(queue.ForthwithActivateOrderPayload{OrderNo: orderInfo.OrderNo})
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "encode activation task: %v", err)
	}
	task := asynq.NewTask(queue.ForthwithActivateOrder, payload, asynq.MaxRetry(10))
	if _, err = l.svcCtx.Queue.EnqueueContext(l.ctx, task); err != nil {
		l.Errorw("[IssueUserSubscribeOrder] activation enqueue failed",
			logger.Field("error", err.Error()),
			logger.Field("order_no", orderInfo.OrderNo),
		)
		if rollbackErr := l.compensateEnqueueFailure(orderInfo, data.product); rollbackErr != nil {
			l.Errorw("[IssueUserSubscribeOrder] enqueue compensation failed",
				logger.Field("error", rollbackErr.Error()),
				logger.Field("order_no", orderInfo.OrderNo),
			)
		}
		if cacheErr := l.svcCtx.SubscribeModel.ClearCache(l.ctx, req.SubscribeId); cacheErr != nil {
			l.Errorw("[IssueUserSubscribeOrder] clear compensated product cache failed", logger.Field("error", cacheErr.Error()))
		}
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "order %s activation could not be queued: %v", orderInfo.OrderNo, err)
	}
	if l.svcCtx.DashboardRealtime != nil {
		l.svcCtx.DashboardRealtime.Publish(realtime.DashboardEventBusiness)
	}
	return &types.IssueUserSubscribeOrderResponse{
		OrderId:          orderInfo.Id,
		OrderNo:          orderInfo.OrderNo,
		OrderStatus:      orderInfo.Status,
		ActivationQueued: true,
		Quote:            data.quote,
	}, nil
}

func (l *IssueUserSubscribeOrderLogic) compensateEnqueueFailure(orderInfo *modelOrder.Order, product *modelSubscribe.Subscribe) error {
	if orderInfo == nil {
		return nil
	}
	return l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&modelOrder.Order{}).
			Where("id = ? AND status = ?", orderInfo.Id, 2).
			Update("status", 4)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		if product != nil && product.Inventory >= 0 {
			return tx.Model(&modelSubscribe.Subscribe{}).
				Where("id = ? AND inventory >= 0", product.Id).
				Update("inventory", gorm.Expr("inventory + 1")).Error
		}
		return nil
	})
}

func (l *IssueUserSubscribeOrderLogic) prepare(req *types.IssueUserSubscribeOrderRequest) (*issueUserSubscribeOrderData, error) {
	if err := validateIssueUserSubscribeOrderRequest(req); err != nil {
		return nil, err
	}
	account, err := l.svcCtx.UserModel.FindOneForAdminCheck(l.ctx, req.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.UserNotExist), "user not found")
		}
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query user: %v", err)
	}
	product, err := l.svcCtx.SubscribeModel.FindOneForPublicPurchase(l.ctx, req.SubscribeId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query subscription product: %v", err)
	}
	quote, err := buildIssueUserSubscribeOrderQuote(account, product, req, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &issueUserSubscribeOrderData{product: product, quote: quote}, nil
}

func validateIssueUserSubscribeOrderRequest(req *types.IssueUserSubscribeOrderRequest) error {
	if req == nil {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "issue request is required")
	}
	if req.UserId <= 0 || req.SubscribeId <= 0 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "user and subscription product are required")
	}
	if req.Quantity < 1 || req.Quantity > orderpricing.MaxQuantity {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid quantity")
	}
	if req.AmountMode != issueAmountComplimentary && req.AmountMode != issueAmountCurrentPrice {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "invalid amount mode")
	}
	if len(req.TradeNo) > 255 {
		return errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "trade number is too long")
	}
	return nil
}

func buildIssueUserSubscribeOrderQuote(account *modelUser.User, product *modelSubscribe.Subscribe, req *types.IssueUserSubscribeOrderRequest, now time.Time) (types.IssueUserSubscribeOrderQuote, error) {
	if account == nil || product == nil || req == nil {
		return types.IssueUserSubscribeOrderQuote{}, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "user, product, and issue request are required")
	}
	if err := productpolicy.ValidateQuantity(product, req.Quantity, orderpricing.MaxQuantity); err != nil {
		return types.IssueUserSubscribeOrderQuote{}, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "invalid quantity: %v", err)
	}
	price, err := orderpricing.ProductQuote(product.UnitPrice, req.Quantity, subscribeview.ParseDiscountLenient(product.Discount))
	if err != nil {
		return types.IssueUserSubscribeOrderQuote{}, errors.Wrapf(xerr.NewErrCode(xerr.InvalidParams), "invalid product price: %v", err)
	}
	recordedAmount := price.Amount
	if req.AmountMode == issueAmountComplimentary {
		recordedAmount = 0
	}
	return types.IssueUserSubscribeOrderQuote{
		ProductId:      product.Id,
		ProductName:    product.Name,
		Quantity:       req.Quantity,
		UnitPrice:      product.UnitPrice,
		ListPrice:      price.ListPrice,
		CurrentAmount:  price.Amount,
		RecordedAmount: recordedAmount,
		Discount:       price.ListPrice - recordedAmount,
		Traffic:        product.Traffic,
		SpeedLimit:     product.SpeedLimit,
		DeviceLimit:    product.DeviceLimit,
		StartTime:      now.UnixMilli(),
		ExpireTime:     timex.UnixMilliOrZero(tool.AddTime(product.UnitTime, req.Quantity, now)),
		AmountMode:     req.AmountMode,
		UserEnabled:    userBoolValue(account.Enable) && !account.DeletedAt.Valid,
	}, nil
}

func (l *IssueUserSubscribeOrderLogic) lockAndPrepareIssue(tx *gorm.DB, req *types.IssueUserSubscribeOrderRequest, now time.Time) (*issueUserSubscribeOrderData, error) {
	var account modelUser.User
	if err := tx.Unscoped().
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "enable", "deleted_at").
		Where("id = ?", req.UserId).
		First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.UserNotExist), "user not found")
		}
		return nil, err
	}
	if !userBoolValue(account.Enable) || account.DeletedAt.Valid {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.UserDisabled), "cannot issue a subscription to a disabled or deleted user")
	}

	var product modelSubscribe.Subscribe
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", req.SubscribeId).
		First(&product).Error; err != nil {
		return nil, err
	}
	quote, err := buildIssueUserSubscribeOrderQuote(&account, &product, req, now)
	if err != nil {
		return nil, err
	}

	activeQuery := tx.Model(&modelUser.Subscribe{}).
		Where("user_id = ? AND status = ?", req.UserId, 1).
		Where("(expire_time = ? OR expire_time > ?)", time.Unix(0, 0).UTC(), now).
		Where("(traffic <= 0 OR used_traffic < traffic)")
	inFlightQuery := tx.Model(&modelOrder.Order{}).
		Where("user_id = ? AND type = ? AND status = ?", req.UserId, 1, 2).
		Where(`NOT EXISTS (SELECT 1 FROM user_subscribe us WHERE us.order_id = "order".id)`)
	if l.svcCtx.ConfigSnapshot().Subscribe.SingleModel {
		var activeCount, inFlightCount int64
		if err := activeQuery.Count(&activeCount).Error; err != nil {
			return nil, err
		}
		if err := inFlightQuery.Count(&inFlightCount).Error; err != nil {
			return nil, err
		}
		if activeCount+inFlightCount > 0 {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.UserSubscribeExist), "user already has an active or activating subscription")
		}
	}
	if product.Quota > 0 {
		var activeCount, inFlightCount int64
		if err := activeQuery.Where("subscribe_id = ?", req.SubscribeId).Count(&activeCount).Error; err != nil {
			return nil, err
		}
		if err := inFlightQuery.Where("subscribe_id = ?", req.SubscribeId).Count(&inFlightCount).Error; err != nil {
			return nil, err
		}
		if activeCount+inFlightCount >= product.Quota {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.SubscribeQuotaLimit), "subscription quota reached for user")
		}
	}
	var priorOrder struct{ Id int64 }
	if err := tx.Model(&modelOrder.Order{}).
		Select("id").
		Where("user_id = ? AND status IN ?", req.UserId, []int64{2, 5}).
		Limit(1).
		Scan(&priorOrder).Error; err != nil {
		return nil, err
	}
	if product.Inventory >= 0 {
		result := tx.Model(&modelSubscribe.Subscribe{}).
			Where("id = ? AND inventory > 0", product.Id).
			Update("inventory", gorm.Expr("inventory - 1"))
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			return nil, errors.Wrap(xerr.NewErrCode(xerr.SubscribeOutOfStock), "subscription product is out of stock")
		}
	}
	return &issueUserSubscribeOrderData{
		product: &product,
		quote:   quote,
		isNew:   priorOrder.Id == 0,
	}, nil
}
