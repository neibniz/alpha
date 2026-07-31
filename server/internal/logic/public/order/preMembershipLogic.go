package order

import (
	"context"

	"github.com/neibniz/alpha/server/internal/model/user"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/constant"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type PreMembershipLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreMembershipLogic {
	return &PreMembershipLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreMembershipLogic) PreMembership(req *types.PreMembershipOrderRequest) (*types.PreOrderResponse, error) {
	u, ok := l.ctx.Value(constant.CtxKeyUser).(*user.User)
	if !ok {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidAccess), "current user is not available")
	}
	plan, err := queryDefaultMembershipPlan(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	price := plan.UnitPrice
	if price < 0 || price > MaxOrderAmount {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "membership price is out of range")
	}

	walletDeduction := quoteOrderWalletDeduction(price, u)
	amount := walletDeduction.Provider
	var feeAmount int64
	if req.Payment != 0 {
		payment, paymentErr := l.svcCtx.PaymentModel.FindOneForPublicOrder(l.ctx, req.Payment)
		if paymentErr != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.PaymentMethodNotFound), "find payment method: %v", paymentErr)
		}
		if err = ensurePaymentEnabled(payment); err != nil {
			return nil, err
		}
		if amount > 0 {
			feeAmount = calculateFee(amount, payment)
		}
	}
	amount += feeAmount
	if amount > MaxOrderAmount {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "membership order amount exceeds maximum limit")
	}

	return &types.PreOrderResponse{
		Price:               price,
		Amount:              amount,
		GiftAmount:          walletDeduction.Gift,
		CommissionDeduction: walletDeduction.Commission,
		FeeAmount:           feeAmount,
	}, nil
}
