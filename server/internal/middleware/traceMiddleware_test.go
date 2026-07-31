package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestRequestAttributesMaskPrivateRequestData(t *testing.T) {
	request := httptest.NewRequest("GET", "https://panel.example/v1/notify/Stripe/abcdef1234567890?token=secret-token&flag=1", nil)
	request.RemoteAddr = "203.0.113.24:54321"
	request.Header.Set("User-Agent", "AlphaClient/1.2.3 (private device model)")

	values := make(map[string]string)
	for _, item := range requestAttributes(request) {
		if item.Value.Type() == attribute.STRING {
			values[string(item.Key)] = item.Value.AsString()
		}
	}
	combined := strings.Join([]string{
		values["url.full"],
		values["url.query"],
		values["user_agent.original"],
		values["client.address"],
	}, " ")
	for _, private := range []string{"abcdef1234567890", "secret-token", "private device model", "203.0.113.24"} {
		if strings.Contains(combined, private) {
			t.Fatalf("trace attributes leaked %q: %q", private, combined)
		}
	}
	if values["client.address"] != "203.0.113.*" {
		t.Fatalf("masked client address = %q", values["client.address"])
	}
}

func TestRequestAttributesHandlesMalformedProtocolAndRemoteAddress(t *testing.T) {
	request := httptest.NewRequest("GET", "/health", nil)
	request.Proto = "HTTP"
	request.RemoteAddr = "not-an-address"
	_ = requestAttributes(request)
}
