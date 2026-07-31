package clientip

import (
	"net/netip"
	"strings"
)

func parseAddressChain(raw string) ([]netip.Addr, bool) {
	if raw == "" || len(raw) > maxForwardedHeaderBytes {
		return nil, false
	}
	parts := strings.Split(raw, ",")
	if len(parts) == 0 || len(parts) > maxForwardedHops {
		return nil, false
	}
	chain := make([]netip.Addr, 0, len(parts))
	for _, part := range parts {
		address, ok := parseAddressValue(part)
		if !ok {
			return nil, false
		}
		chain = append(chain, address)
	}
	return chain, true
}
