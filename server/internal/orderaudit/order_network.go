package orderaudit

import (
	"context"
	"strings"

	"github.com/neibniz/alpha/server/internal/model/order"
	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/constant"
)

func FillNetwork(ctx context.Context, svcCtx *svc.ServiceContext, item *order.Order) {
	if item == nil || strings.TrimSpace(item.ClientIP) != "" {
		return
	}
	ip, _ := ctx.Value(constant.CtxKeyClientIP).(string)
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return
	}
	item.ClientIP = ip
	if svcCtx == nil || svcCtx.GeoIP == nil {
		return
	}
	location := svcCtx.GeoIP.Lookup(ip)
	item.CountryShort = location.CountryShort
	item.Region = location.Region
	item.Location = location.Location
}
