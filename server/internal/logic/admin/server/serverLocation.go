package server

import (
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/svc"
)

func fillServerLocationFromGeoIP(svcCtx *svc.ServiceContext, address string, data *node.Server) {
	if svcCtx == nil || svcCtx.GeoIP == nil || data == nil {
		return
	}
	if data.Country != "" && data.City != "" {
		return
	}
	location := svcCtx.GeoIP.Lookup(address)
	if data.Country == "" {
		if location.Country != "" {
			data.Country = location.Country
		} else {
			data.Country = location.CountryShort
		}
	}
	if data.City == "" {
		data.City = location.City
	}
}
