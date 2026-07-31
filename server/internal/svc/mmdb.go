package svc

import (
	"fmt"
	"net"
	"strings"

	"github.com/neibniz/alpha/server/internal/privacy"
	"github.com/neibniz/alpha/server/pkg/logger"
	"github.com/oschwald/geoip2-golang"
)

type IPLocation struct {
	Path string
	DB   *geoip2.Reader
}

type IPLocationInfo struct {
	Country      string
	CountryShort string
	Region       string
	City         string
	Location     string
}

func NewIPLocation(path string) (*IPLocation, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("GeoIP database path is empty")
	}

	db, err := geoip2.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open GeoIP database %q: %w", path, err)
	}
	return &IPLocation{
		Path: path,
		DB:   db,
	}, nil
}

func (ipLoc *IPLocation) Close() error {
	if ipLoc == nil || ipLoc.DB == nil {
		return nil
	}
	return ipLoc.DB.Close()
}

func (ipLoc *IPLocation) Lookup(ip string) IPLocationInfo {
	if ipLoc == nil || ipLoc.DB == nil {
		return IPLocationInfo{}
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil || parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsUnspecified() {
		return IPLocationInfo{}
	}
	record, err := ipLoc.DB.City(parsed)
	if err != nil {
		logger.Debugf("[GeoIP] lookup failed for %s: %v", privacy.MaskIP(ip), err)
		return IPLocationInfo{}
	}
	info := IPLocationInfo{
		Country:      record.Country.Names["en"],
		CountryShort: record.Country.IsoCode,
	}
	if len(record.Subdivisions) > 0 {
		info.Region = record.Subdivisions[0].Names["en"]
	}
	if record.City.Names != nil {
		info.City = record.City.Names["en"]
	}
	info.Location = FormatIPLocation(info.CountryShort, info.Region)
	return info
}

func (ipLoc *IPLocation) Resolve(ip, countryShort, region string) IPLocationInfo {
	info := IPLocationInfo{
		CountryShort: strings.TrimSpace(countryShort),
		Region:       strings.TrimSpace(region),
	}
	info.Location = FormatIPLocation(info.CountryShort, info.Region)
	if info.CountryShort != "" || info.Region != "" || ipLoc == nil {
		return info
	}
	return ipLoc.Lookup(ip)
}

func (ipLoc *IPLocation) LocationLabel(ip, countryShort, region string) string {
	info := ipLoc.Resolve(ip, countryShort, region)
	return FormatIPLocation(info.CountryShort, info.Region)
}

func FormatIPLocation(countryShort, region string) string {
	parts := make([]string, 0, 2)
	for _, value := range []string{countryShort, region} {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " · ")
}
