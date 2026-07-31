package main

import (
	"log/slog"

	"github.com/neibniz/alpha/forwared/logger"
)

// 判断是否允许连入
func IsIPAllowed(ip string) bool {
	if IP2Location_db == nil {
		return true
	}
	recored, err := IP2Location_db.Get_all(ip)
	if err != nil {
		logger.Logger.Warn("GeoIP lookup failed", slog.Any("error", err))
		return true // 如果查询失败，默认允许连接
	}
	if recored.Country_short == "CN" && recored.Region == "Xinjiang Uygur" {
		return false
	}
	return true
}
