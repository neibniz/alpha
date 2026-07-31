package payment

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/neibniz/alpha/server/internal/paymenturl"
	"github.com/neibniz/alpha/server/internal/report"
	"github.com/neibniz/alpha/server/pkg/payment/stripe"
	"gorm.io/gorm"

	"github.com/neibniz/alpha/server/pkg/random"

	paymentModel "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/payment"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type CreatePaymentMethodLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

var (
	paymentConfigRawNull        = []byte("null")
	paymentConfigRawEmptyObject = []byte("{}")
)

// NewCreatePaymentMethodLogic Create Payment Method
func NewCreatePaymentMethodLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePaymentMethodLogic {
	return &CreatePaymentMethodLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePaymentMethodLogic) CreatePaymentMethod(req *types.CreatePaymentMethodRequest) (resp *types.PaymentConfig, err error) {
	if req == nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.InvalidParams), "payment method request is required")
	}
	canonicalPlatform, ok := payment.NormalizePlatform(req.Platform)
	if !ok {
		l.Errorw("unsupported payment platform", logger.Field("mark", req.Platform))
		return nil, errors.Wrapf(xerr.NewErrCodeMsg(400, "UNSUPPORTED_PAYMENT_PLATFORM"), "unsupported payment platform: %s", req.Platform)
	}
	req.Platform = canonicalPlatform
	req.Domain, err = paymenturl.NormalizeBaseURL(req.Domain)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid payment callback domain"), err.Error())
	}
	if payment.ParsePlatform(req.Platform) == payment.Balance {
		methods, queryErr := l.svcCtx.PaymentModel.FindAll(l.ctx)
		if queryErr != nil {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "query payment methods: %v", queryErr)
		}
		if hasOtherBalancePayment(methods, 0, false) {
			return nil, errors.Wrap(
				xerr.NewErrCodeMsg(xerr.InvalidParams, "BALANCE_PAYMENT_ALREADY_EXISTS"),
				"balance payment method already exists",
			)
		}
	}
	config, err := parsePaymentPlatformConfig(l.ctx, payment.ParsePlatform(req.Platform), req.Config)
	if err != nil {
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "parse payment config error: %v", err.Error())
	}
	notifyToken, err := random.SecureKey(24, 1)
	if err != nil {
		return nil, errors.Wrap(xerr.NewErrCode(xerr.ERROR), "generate payment notification token")
	}
	var paymentMethod = &paymentModel.Payment{
		Name:        req.Name,
		Platform:    req.Platform,
		Icon:        req.Icon,
		Domain:      req.Domain,
		Description: req.Description,
		Config:      config,
		FeeMode:     req.FeeMode,
		FeePercent:  req.FeePercent,
		FeeAmount:   req.FeeAmount,
		Enable:      req.Enable,
		Token:       notifyToken,
	}
	err = l.svcCtx.PaymentModel.Transaction(l.ctx, func(tx *gorm.DB) error {
		if req.Platform == "Stripe" {
			var cfg paymentModel.StripeConfig
			if err = cfg.Unmarshal([]byte(paymentMethod.Config)); err != nil {
				l.Errorf("[CreatePaymentMethod] unmarshal stripe config error: %s", err.Error())
				return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "unmarshal stripe config error: %s", err.Error())
			}
			if cfg.SecretKey == "" {
				l.Error("[CreatePaymentMethod] stripe secret key is empty")
				return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "stripe secret key is empty")
			}

			// Create Stripe webhook endpoint
			client := stripe.NewClient(stripe.Config{
				SecretKey: cfg.SecretKey,
				PublicKey: cfg.PublicKey,
			})
			snapshot := l.svcCtx.ConfigSnapshot()
			url, err := paymenturl.NotifyURL(req.Domain, snapshot.Site.Host, paymentMethod.Platform, paymentMethod.Token, report.IsGatewayMode())
			if err != nil {
				return errors.Wrap(xerr.NewErrCodeMsg(xerr.InvalidParams, "invalid payment callback configuration"), err.Error())
			}
			endpoint, err := client.CreateWebhookEndpoint(url)
			if err != nil {
				l.Errorw("[CreatePaymentMethod] create stripe webhook endpoint error", logger.Field("error", err.Error()))
				return errors.Wrapf(xerr.NewErrCode(xerr.ERROR), "create stripe webhook endpoint error: %s", err.Error())
			}
			cfg.WebhookSecret = endpoint.Secret
			content, _ := cfg.Marshal()
			paymentMethod.Config = string(content)
		}
		if err = tx.Model(&paymentModel.Payment{}).Create(paymentMethod).Error; err != nil {
			return errors.Wrapf(xerr.NewErrCode(xerr.DatabaseInsertError), "insert payment method error: %s", err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	paymentResp := paymentConfigFromModel(paymentMethod)
	resp = &paymentResp
	return
}

func parsePaymentPlatformConfig(ctx context.Context, platform payment.Platform, config json.RawMessage) (string, error) {
	data := paymentConfigRaw(config)
	if !json.Valid(data) {
		err := errors.New("invalid payment config json")
		logger.WithContext(ctx).Errorw("parse payment config error", logger.Field("platform", platform), logger.Field("error", err.Error()))
		return "", err
	}

	// 通用处理函数
	handleConfig := func(name string, target interface {
		Unmarshal([]byte) error
		Marshal() ([]byte, error)
	}) (string, error) {
		if err := target.Unmarshal(data); err != nil {
			logger.WithContext(ctx).Errorw("parse "+name+" config error", logger.Field("error", err.Error()))
			return "", err
		}
		content, err := target.Marshal()
		if err != nil {
			logger.WithContext(ctx).Errorw("marshal "+name+" config error", logger.Field("error", err.Error()))
			return "", err
		}
		return string(content), nil
	}

	switch platform {
	case payment.Stripe:
		return handleConfig("Stripe", &paymentModel.StripeConfig{})
	case payment.AlipayF2F:
		return handleConfig("Alipay", &paymentModel.AlipayF2FConfig{})
	case payment.EPay:
		return handleConfig("Epay", &paymentModel.EPayConfig{})
	case payment.CryptoSaaS:
		return handleConfig("CryptoSaaS", &paymentModel.CryptoSaaSConfig{})
	case payment.Balance:
		return "{}", nil
	default:
		return "", errors.New("unsupported payment platform")
	}
}

func paymentConfigRaw(config json.RawMessage) []byte {
	data := bytes.TrimSpace(config)
	if len(data) == 0 || bytes.Equal(data, paymentConfigRawNull) {
		return paymentConfigRawEmptyObject
	}
	return data
}
