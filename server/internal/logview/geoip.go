package logview

import "github.com/neibniz/alpha/server/internal/svc"

func FromIPLocation(ipLoc *svc.IPLocation) LocationResolver {
	return func(ip, countryShort, region string) Location {
		info := ipLoc.Resolve(ip, countryShort, region)
		return Location{
			CountryShort: info.CountryShort,
			Region:       info.Region,
			Location:     info.Location,
		}
	}
}
