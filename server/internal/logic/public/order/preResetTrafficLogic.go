package order

import (
	"context"
	"time"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/subscribeview"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type PreResetTrafficLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreResetTrafficLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreResetTrafficLogic {
	return &PreResetTrafficLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreResetTrafficLogic) PreResetTraffic(req *types.PreResetTrafficOrderRequest) (*types.PreResetTrafficOrderResponse, error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is not available")
	}

	userSubscribe, err := l.svcCtx.UserModel.FindOneUserSubscribeForOrderAction(l.ctx, req.UserSubscribeID)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find user subscription: %v", err)
	}
	if userSubscribe.UserId != u.Id {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "subscription does not belong to current user")
	}
	product, err := l.svcCtx.SubscribeModel.FindOneForPublicPurchase(l.ctx, userSubscribe.SubscribeId)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find current product: %v", err)
	}
	if err := ensureResetTrafficActionAllowed(userSubscribe.Status, userSubscribe.ExpireTime, product, time.Now()); err != nil {
		return nil, err
	}

	price := product.Replacement
	amount := price
	walletDeduction := quoteOrderWalletDeduction(amount, u)
	amount = walletDeduction.Provider

	var feeAmount int64
	if amount > 0 && req.Payment != 0 {
		payment, paymentErr := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, req.Payment)
		if paymentErr != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method: %v", paymentErr)
		}
		if err := ensurePaymentEnabled(payment); err != nil {
			return nil, err
		}
		feeAmount = calculateFee(amount, payment)
		amount += feeAmount
	}
	if amount > MaxOrderAmount {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "order amount exceeds maximum limit")
	}
	if err := enforceMembershipOrderPolicies(l.ctx, l.svcCtx, u, amount); err != nil {
		return nil, err
	}

	return &types.PreResetTrafficOrderResponse{
		Price:               price,
		Amount:              amount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		FeeAmount:           feeAmount,
		Product:             subscribeview.PublicSubscribeFromModel(product),
	}, nil
}
