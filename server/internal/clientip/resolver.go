package clientip

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

const (
	EdgeClientHeader        = "X-Alpha-Client-IP"
	maxForwardedHeaderBytes = 4096
	maxForwardedHops        = 32
)

type Options struct {
	TrustedProxies []string
}

type Resolver struct {
	trustedProxies []netip.Prefix
}

func NewResolver(options Options) (*Resolver, error) {
	trustedProxies := options.TrustedProxies
	if len(trustedProxies) == 0 {
		trustedProxies = []string{"127.0.0.1", "::1"}
	}
	trusted, err := parsePrefixes("trusted proxy", trustedProxies)
	if err != nil {
		return nil, err
	}
	return &Resolver{
		trustedProxies: trusted,
	}, nil
}

func (r *Resolver) Resolve(req *http.Request) Result {
	if req == nil {
		return Result{Source: SourceUnknown}
	}
	peer := parseRemoteAddress(req.RemoteAddr)
	fallback := Result{Address: peer, Source: SourceDirect, Verified: peer.IsValid()}

	if r == nil || !peer.IsValid() {
		return fallback
	}
	if address, ok := parseAddressValue(req.Header.Get(EdgeClientHeader)); ok {
		return Result{Address: address, Source: SourceEdgeHeader, Verified: false}
	}
	if address, ok := parseAddressValue(req.Header.Get("CF-Connecting-IP")); ok {
		return Result{Address: address, Source: SourceCloudflare, Verified: false}
	}
	if containsPrefix(r.trustedProxies, peer) {
		if chain, ok := parseAddressChain(strings.Join(req.Header.Values("X-Forwarded-For"), ",")); ok {
			return Result{
				Address:  firstUntrustedFromRight(chain, r.trustedProxies),
				Source:   SourceXForwardedFor,
				Verified: true,
			}
		}
		if address, ok := parseAddressValue(req.Header.Get("X-Real-IP")); ok {
			return Result{Address: address, Source: SourceXRealIP, Verified: true}
		}
	}
	return fallback
}

func parsePrefixes(kind string, values []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			address, addressErr := netip.ParseAddr(value)
			if addressErr != nil {
				return nil, fmt.Errorf("parse %s %q: %w", kind, value, err)
			}
			address = address.Unmap()
			bits := 128
			if address.Is4() {
				bits = 32
			}
			prefix = netip.PrefixFrom(address, bits)
		}
		prefix = prefix.Masked()
		if prefix.Bits() == 0 {
			return nil, fmt.Errorf("%s %q trusts every address", kind, value)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}

func containsPrefix(prefixes []netip.Prefix, address netip.Addr) bool {
	address = address.Unmap()
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func firstUntrustedFromRight(chain []netip.Addr, trusted []netip.Prefix) netip.Addr {
	for index := len(chain) - 1; index >= 0; index-- {
		if !containsPrefix(trusted, chain[index]) {
			return chain[index].Unmap()
		}
	}
	return chain[0].Unmap()
}

func parseRemoteAddress(value string) netip.Addr {
	value = strings.TrimSpace(value)
	if value == "" {
		return netip.Addr{}
	}
	if addressPort, err := netip.ParseAddrPort(value); err == nil {
		return validAddress(addressPort.Addr())
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		if address, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
			return validAddress(address)
		}
	}
	address, err := netip.ParseAddr(strings.Trim(value, "[]"))
	if err != nil {
		return netip.Addr{}
	}
	return validAddress(address)
}

func parseAddressValue(value string) (netip.Addr, bool) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 || strings.Contains(value, ",") {
		return netip.Addr{}, false
	}
	address := parseRemoteAddress(value)
	return address, address.IsValid()
}

func validAddress(address netip.Addr) netip.Addr {
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() {
		return netip.Addr{}
	}
	return address.Unmap()
}
