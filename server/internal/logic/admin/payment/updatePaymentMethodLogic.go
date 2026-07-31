package payment

import (
	"context"

	"github.com/neibniz/alpha/server/internal/jsonview"
	paymentModel "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/paymenturl"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type UpdatePaymentMethodLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewUpdatePaymentMethodLogic Update Payment Method
func NewUpdatePaymentMethodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePaymentMethodLogic {
	return &UpdatePaymentMethodLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdatePaymentMethodLogic) UpdatePaymentMethod(req *types.UpdatePaymentMethodRequest) (resp *types.PaymentConfig, err error) {
	if req == nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "payment method request is required")
	}
	method, err := l.svcCtx.PaymentModel.FindOneForAdmin(l.ctx, req.Id, req.Config != nil)
	if err != nil {
		l.Errorw("find payment method error", logger.Field("id", req.Id), logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method error: %s", err.Error())
	}
	originalPlatform := method.Platform
	update := paymentModel.PaymentFieldsUpdate{}
	if req.Name != nil {
		if *req.Name == "" {
			return nil, errors.Wrapf(xerr.NewErrCodeMsg(xerr.InvalidParams, "payment name is required"), "payment name is required")
		}
		method.Name = *req.Name
		update.Name = &method.Name
	}
	if req.Platform != nil {
		canonicalPlatform, ok := payment.NormalizePlatform(*req.Platform)
		if !ok {
			l.Errorw("unsupported payment platform", logger.Field("platform", *req.Platform))
			return nil, errors.Wrapf(xerr.NewErrCodeMsg(400, "UNSUPPORTED_PAYMENT_PLATFORM"), "unsupported payment platform: %s", *req.Platform)
		}
		method.Platform = canonicalPlatform
		update.Platform = &method.Platform
	}
	if req.Description != nil {
		method.Description = *req.Description
		update.Description = &method.Description
	}
	if req.Icon != nil {
		method.Icon = *req.Icon
		update.Icon = &method.Icon
	}
	if req.Domain != nil {
		normalizedDomain, normalizeErr := paymenturl.NormalizeBaseURL(*req.Domain)
		if normalizeErr != nil {
			return nil, errors.Wrap(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid payment callback domain"), normalizeErr.Error())
		}
		method.Domain = normalizedDomain
		update.Domain = &method.Domain
	}
	if req.Config != nil {
		config, err := parsePaymentPlatformConfig(l.ctx, payment.ParsePlatform(method.Platform), req.Config)
		if err != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "parse payment config error: %v", err.Error())
		}
		method.Config = config
		update.Config = &method.Config
	}
	if req.FeeMode != nil {
		method.FeeMode = *req.FeeMode
		update.FeeMode = &method.FeeMode
	}
	if req.FeePercent != nil {
		method.FeePercent = *req.FeePercent
		update.FeePercent = &method.FeePercent
	}
	if req.FeeAmount != nil {
		method.FeeAmount = *req.FeeAmount
		update.FeeAmount = &method.FeeAmount
	}
	if req.Enable != nil {
		method.Enable = req.Enable
		update.Enable = req.Enable
	}
	changingToBalance := req.Platform != nil &&
		payment.ParsePlatform(originalPlatform) != payment.Balance &&
		payment.ParsePlatform(method.Platform) == payment.Balance
	enablingBalance := req.Enable != nil && *req.Enable &&
		payment.ParsePlatform(method.Platform) == payment.Balance
	if changingToBalance || enablingBalance {
		methods, queryErr := l.svcCtx.PaymentModel.FindAll(l.ctx)
		if queryErr != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query payment methods: %v", queryErr)
		}
		if hasOtherBalancePayment(methods, method.Id, !changingToBalance) {
			return nil, errors.Wrap(
				xerr.NewErrCodeMsg(xerr.InvalidParams, "BALANCE_PAYMENT_ALREADY_EXISTS"),
				"balance payment method already exists",
			)
		}
	}
	if err := l.svcCtx.PaymentModel.UpdateFields(l.ctx, method, update); err != nil {
		l.Errorw("update payment method error", logger.Field("id", req.Id), logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseUpdateError), "update payment method error: %s", err.Error())
	}
	resp = &types.PaymentConfig{}
	resp.Id = method.Id
	resp.Name = method.Name
	resp.Platform = method.Platform
	resp.Description = method.Description
	resp.Icon = method.Icon
	resp.Domain = method.Domain
	resp.FeeMode = method.FeeMode
	resp.FeePercent = method.FeePercent
	resp.FeeAmount = method.FeeAmount
	resp.Enable = method.Enable != nil && *method.Enable
	if req.Config != nil {
		resp.Config = jsonview.RawObjectOrEmpty(method.Config)
	}
	return
}
