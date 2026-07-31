package payment

import (
	"context"

	"github.com/neibniz/alpha/server/internal/paymenturl"
	"github.com/neibniz/alpha/server/internal/report"
	paymentPlatform "github.com/neibniz/alpha/server/pkg/payment"

	"github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/pkg/errors"
)

type GetPaymentMethodListLogic struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetPaymentMethodListLogic Get Payment Method List
func NewGetPaymentMethodListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPaymentMethodListLogic {
	return &GetPaymentMethodListLogic{
		Logger: logger.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPaymentMethodListLogic) GetPaymentMethodList(req *types.GetPaymentMethodListRequest) (resp *types.GetPaymentMethodListResponse, err error) {
	total, list, err := l.svcCtx.PaymentModel.FindListByPage(l.ctx, req.Page, req.Size, &payment.Filter{
		Search: req.Search,
		Mark:   req.Platform,
		Enable: req.Enable,
	})
	if err != nil {
		l.Errorw("find payment method list error", logger.Field("error", err.Error()))
		return nil, errors.Wrapf(xerr.NewErrCode(xerr.DatabaseQueryError), "find payment method list error: %s", err.Error())
	}
	resp = &types.GetPaymentMethodListResponse{
		Total: total,
		List:  paymentMethodListItemRows(list, l.svcCtx.ConfigSnapshot().Site.Host, report.IsGatewayMode()),
	}
	return
}

func paymentMethodListItemRows(rows []*payment.Payment, siteURL string, gatewayMode bool) []types.PaymentMethodListItem {
	if len(rows) == 0 {
		return []types.PaymentMethodListItem{}
	}
	collector := viewutil.NewRowsCollector[types.PaymentMethodListItem](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(paymentMethodListItemFromModel(row, paymentNotifyURL(row, siteURL, gatewayMode)))
	}
	return collector.Rows()
}

func paymentNotifyURL(item *payment.Payment, siteURL string, gatewayMode bool) string {
	if item == nil || paymentPlatform.ParsePlatform(item.Platform) == paymentPlatform.Balance {
		return ""
	}
	notifyURL, err := paymenturl.NotifyURL(item.Domain, siteURL, item.Platform, item.Token, gatewayMode)
	if err != nil {
		return ""
	}
	return notifyURL
}
