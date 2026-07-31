package paymentview

import (
	"github.com/neibniz/alpha/server/internal/jsonview"
	modelPayment "github.com/neibniz/alpha/server/internal/model/payment"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func MethodFromModel(item *modelPayment.Payment) types.PaymentMethod {
	if item == nil {
		return types.PaymentMethod{}
	}
	return types.PaymentMethod{
		Id:          item.Id,
		Name:        item.Name,
		Platform:    item.Platform,
		Description: item.Description,
		Icon:        item.Icon,
		FeeMode:     item.FeeMode,
		FeePercent:  item.FeePercent,
		FeeAmount:   item.FeeAmount,
	}
}

func MethodRows(rows []*modelPayment.Payment) []types.PaymentMethod {
	if len(rows) == 0 {
		return []types.PaymentMethod{}
	}
	collector := viewutil.NewRowsCollector[types.PaymentMethod](len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		collector.Add(MethodFromModel(row))
	}
	return collector.Rows()
}

func ConfigFromModel(item *modelPayment.Payment) types.PaymentConfig {
	if item == nil {
		return types.PaymentConfig{}
	}
	return types.PaymentConfig{
		Id:          item.Id,
		Name:        item.Name,
		Platform:    item.Platform,
		Description: item.Description,
		Icon:        item.Icon,
		Domain:      item.Domain,
		Config:      jsonview.RawObjectOrEmpty(item.Config),
		FeeMode:     item.FeeMode,
		FeePercent:  item.FeePercent,
		FeeAmount:   item.FeeAmount,
		Enable:      BoolValue(item.Enable),
	}
}

func ListItemFromModel(item *modelPayment.Payment, notifyURL string) types.PaymentMethodListItem {
	if item == nil {
		return types.PaymentMethodListItem{}
	}
	return types.PaymentMethodListItem{
		Id:          item.Id,
		Name:        item.Name,
		Platform:    item.Platform,
		Description: item.Description,
		Icon:        item.Icon,
		FeeMode:     item.FeeMode,
		FeePercent:  item.FeePercent,
		FeeAmount:   item.FeeAmount,
		Enable:      BoolValue(item.Enable),
		NotifyURL:   notifyURL,
	}
}

func BoolValue(value *bool) bool {
	return value != nil && *value
}
