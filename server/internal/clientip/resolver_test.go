package clientip

import (
	"net/http/httptest"
	"testing"
)

func testResolver(t *testing.T, options Options) *Resolver {
	t.Helper()
	resolver, err := NewResolver(options)
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

func TestResolverIgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	resolver := testResolver(t, Options{TrustedProxies: []string{"127.0.0.1"}})
	request := httptest.NewRequest("GET", "https://api.example.test/v1/common/heartbeat", nil)
	request.RemoteAddr = "198.51.100.20:4312"
	request.Header.Set("X-Forwarded-For", "203.0.113.99")
	request.Header.Set("X-Real-IP", "203.0.113.98")

	result := resolver.Resolve(request)
	if result.String() != "198.51.100.20" || result.Source != SourceDirect || !result.Verified {
		t.Fatalf("unexpected direct result: %#v", result)
	}
}

func TestResolverWalksTrustedXForwardedForFromRight(t *testing.T) {
	resolver := testResolver(t, Options{TrustedProxies: []string{
		"127.0.0.1",
		"203.0.113.0/24",
	}})
	request := httptest.NewRequest("GET", "https://api.example.test/v1/common/heartbeat", nil)
	request.RemoteAddr = "127.0.0.1:4312"
	request.Header.Set("X-Forwarded-For", "198.51.100.42, 203.0.113.10")

	result := resolver.Resolve(request)
	if result.String() != "198.51.100.42" || result.Source != SourceXForwardedFor || !result.Verified {
		t.Fatalf("unexpected forwarded result: %#v", result)
	}
}

func TestResolverRejectsMalformedForwardingChain(t *testing.T) {
	resolver := testResolver(t, Options{TrustedProxies: []string{"127.0.0.1"}})
	request := httptest.NewRequest("GET", "https://api.example.test/v1/common/heartbeat", nil)
	request.RemoteAddr = "127.0.0.1:4312"
	request.Header.Set("X-Forwarded-For", "198.51.100.42, not-an-ip")

	result := resolver.Resolve(request)
	if result.String() != "127.0.0.1" || result.Source != SourceDirect {
		t.Fatalf("malformed chain should fall back to peer: %#v", result)
	}
}

func TestResolverFallsBackToXRealIPFromTrustedProxy(t *testing.T) {
	resolver := testResolver(t, Options{TrustedProxies: []string{"127.0.0.1"}})
	request := httptest.NewRequest("GET", "https://api.example.test/v1/common/heartbeat", nil)
	request.RemoteAddr = "127.0.0.1:4312"
	request.Header.Set("X-Real-IP", "198.51.100.42")

	result := resolver.Resolve(request)
	if result.String() != "198.51.100.42" || result.Source != SourceXRealIP || !result.Verified {
		t.Fatalf("unexpected X-Real-IP result: %#v", result)
	}
}

func TestResolverAcceptsCloudflareEdgeHeader(t *testing.T) {
	resolver := testResolver(t, Options{})
	request := httptest.NewRequest("GET", "https://api.example.test/v1/common/heartbeat", nil)
	request.RemoteAddr = "203.0.113.7:443"
	request.Header.Set("CF-Connecting-IP", "198.51.100.55")

	result := resolver.Resolve(request)
	if result.String() != "198.51.100.55" || result.Source != SourceCloudflare || result.Verified {
		t.Fatalf("unexpected Cloudflare result: %#v", result)
	}
}

func TestResolverAcceptsUnverifiedEdgeHeaderAcrossDeploymentBoundaries(t *testing.T) {
	resolver := testResolver(t, Options{TrustedProxies: []string{"127.0.0.1"}})
	request := httptest.NewRequest("GET", "https://api.example.test/v1/subscribe/config", nil)
	request.RemoteAddr = "192.0.2.40:443"
	request.Header.Set(EdgeClientHeader, "2001:db8::42")

	result := resolver.Resolve(request)
	if result.String() != "2001:db8::42" || result.Source != SourceEdgeHeader || result.Verified {
		t.Fatalf("unexpected edge header result: %#v", result)
	}
}

func TestResolverPreservesEdgeHeaderAcrossAnotherProxy(t *testing.T) {
	resolver := testResolver(t, Options{TrustedProxies: []string{"127.0.0.1"}})
	request := httptest.NewRequest("GET", "https://api.example.test/v1/common/heartbeat", nil)
	request.RemoteAddr = "127.0.0.1:4312"
	request.Header.Set("X-Forwarded-For", "198.51.100.42")
	request.Header.Set(EdgeClientHeader, "203.0.113.99")

	result := resolver.Resolve(request)
	if result.String() != "203.0.113.99" || result.Source != SourceEdgeHeader || result.Verified {
		t.Fatalf("edge header was not preserved: %#v", result)
	}
}

func TestResolverRejectsUnsafeOrInvalidConfiguration(t *testing.T) {
	for _, options := range []Options{
		{TrustedProxies: []string{"0.0.0.0/0"}},
	} {
		if _, err := NewResolver(options); err == nil {
			t.Fatalf("expected invalid options to fail: %#v", options)
		}
	}
}

func TestContextCarriesResolvedClientIP(t *testing.T) {
	request := httptest.NewRequest("GET", "https://api.example.test/", nil)
	request.RemoteAddr = "192.0.2.9:4321"
	result := Result{Address: parseRemoteAddress("198.51.100.9"), Source: SourceEdgeHeader, Verified: false}
	request = request.WithContext(WithResult(request.Context(), result))

	if got := Address(request); got != "198.51.100.9" {
		t.Fatalf("context address = %q", got)
	}
	stored, ok := FromContext(request.Context())
	if !ok || stored.Source != SourceEdgeHeader || stored.Verified {
		t.Fatalf("context result = %#v, %v", stored, ok)
	}
}
