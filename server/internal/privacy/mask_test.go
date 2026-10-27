package privacy

import (
	"net"
	"testing"
)

var maskTestSink string

func TestMaskIP(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ipv4", in: "203.0.113.42", want: "203.0.113.*"},
		{name: "ipv6", in: "2001:db8:85a3:8d3:1319:8a2e:370:7348", want: "2001:db8:85a3:8d3:****:****:****:****"},
		{name: "invalid", in: "device-fingerprint-abcdef", want: "devi...cdef"},
		{name: "already masked", in: "203.0.113.*", want: "203.0.113.*"},
		{name: "redacted", in: "***", want: "***"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskIP(tt.in); got != tt.want {
				t.Fatalf("MaskIP(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeIPNetwork(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "ipv4", in: "203.0.113.42", want: "203.0.113.42/32", ok: true},
		{name: "mapped ipv4", in: "::ffff:203.0.113.42", want: "203.0.113.42/32", ok: true},
		{name: "ipv6", in: "2001:db8:85a3:8d3:1319:8a2e:370:7348", want: "2001:db8:85a3:8d3::/64", ok: true},
		{name: "ipv4 scope", in: "203.0.113.42/32", want: "203.0.113.42/32", ok: true},
		{name: "mapped ipv4 scope", in: "::ffff:203.0.113.42/128", want: "203.0.113.42/32", ok: true},
		{name: "ipv6 scope", in: "2001:db8:85a3:8d3::/64", want: "2001:db8:85a3:8d3::/64", ok: true},
		{name: "ipv6 host bits", in: " 2001:db8:85a3:8d3::42/64 ", want: "2001:db8:85a3:8d3::/64", ok: true},
		{name: "wide ipv4 scope", in: "203.0.113.0/24", ok: false},
		{name: "wide ipv6 scope", in: "2001:db8::/48", ok: false},
		{name: "narrow ipv6 scope", in: "2001:db8::1/128", ok: false},
		{name: "wide mapped scope", in: "::ffff:203.0.113.42/120", ok: false},
		{name: "invalid", in: "not-an-ip", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NormalizeIPNetwork(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("NormalizeIPNetwork(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.ok)
			}
			if ok {
				if again, valid := NormalizeIPNetwork(got); !valid || again != got {
					t.Fatalf("normalizing a scope twice changed %q to (%q, %v)", got, again, valid)
				}
			}
		})
	}
}

func TestMaskUserAgent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "browser", in: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15", want: "Mozilla/***"},
		{name: "client with product name", in: "Clash Verge/v1.7.7 Darwin/23.0", want: "Clash Verge/***"},
		{name: "single product", in: "sing-box/1.11.0", want: "sing-box/***"},
		{name: "unicode whitespace", in: "Clash\u00a0Meta/1.0 Darwin/23.0", want: "Clash Meta/***"},
		{name: "parenthesized only", in: "(Macintosh; Intel Mac OS X)", want: "***"},
		{name: "empty", in: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskUserAgent(tt.in); got != tt.want {
				t.Fatalf("MaskUserAgent(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMaskUserAgentSingleProductAllocationBudget(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		maskTestSink = MaskUserAgent("sing-box/1.11.0")
	})
	if allocs > 1 {
		t.Fatalf("MaskUserAgent single product allocs = %v, want <= 1", allocs)
	}
}

func TestMaskIPHelperAllocationBudget(t *testing.T) {
	v4 := net.IPv4(203, 0, 113, 42).To4()
	v6 := net.ParseIP("2001:db8:85a3:8d3:1319:8a2e:370:7348").To16()

	v4Allocs := testing.AllocsPerRun(100, func() {
		maskTestSink = maskIPv4(v4)
	})
	if v4Allocs > 1 {
		t.Fatalf("maskIPv4 allocs = %v, want <= 1", v4Allocs)
	}

	v6Allocs := testing.AllocsPerRun(100, func() {
		maskTestSink = maskIPv6(v6)
	})
	if v6Allocs > 1 {
		t.Fatalf("maskIPv6 allocs = %v, want <= 1", v6Allocs)
	}
}

func TestSanitizeURLAndQuery(t *testing.T) {
	got := SanitizeURL("/v1/subscribe/config?token=abcdef1234567890&flag=clash&email=user@example.com")
	want := "/v1/subscribe/config?email=%2A%2A%2A&flag=clash&token=%2A%2A%2A"
	if got != want {
		t.Fatalf("SanitizeURL() = %q, want %q", got, want)
	}

	got = SanitizeURL("/notify/stripe/abcdef1234567890")
	want = "/notify/stripe/abcd...7890"
	if got != want {
		t.Fatalf("SanitizeURL(path token) = %q, want %q", got, want)
	}
}

func TestSanitizeHostMasksTokenBearingSubdomainAndIP(t *testing.T) {
	if got := SanitizeHost("abcdef1234567890abcdef.user.example.com:8443"); got != "abcd...cdef.user.example.com:8443" {
		t.Fatalf("SanitizeHost(token) = %q", got)
	}
	if got := SanitizeHost("203.0.113.24:8080"); got != "203.0.113.*:8080" {
		t.Fatalf("SanitizeHost(ip) = %q", got)
	}
	if got := SanitizeHost("user.example.com"); got != "user.example.com" {
		t.Fatalf("SanitizeHost(domain) = %q", got)
	}
}

func TestMaskTokenPrefix(t *testing.T) {
	if got := MaskTokenPrefix("abcdefgh"); got != "abcd****" {
		t.Fatalf("MaskTokenPrefix() = %q", got)
	}
}

func TestMaskMessageContentJSON(t *testing.T) {
	got := string(MaskMessageContentJSON([]byte(`{"code":123456,"nested":{"verification_code":"654321","ok":true}}`)))
	want := `{"code":"***","nested":{"ok":true,"verification_code":"***"}}`
	if got != want {
		t.Fatalf("MaskMessageContentJSON() = %q, want %q", got, want)
	}

	got = string(MaskMessageContentJSON([]byte(`"Your code is 123456"`)))
	want = `"***"`
	if got != want {
		t.Fatalf("MaskMessageContentJSON(string) = %q, want %q", got, want)
	}
}

func TestMaskJSONRedactsNestedNetworkIdentity(t *testing.T) {
	got := string(MaskJSON([]byte(`{"login_ip":"203.0.113.42","nested":{"client_ip":"198.51.100.8","user_agent":"AlphaClient/1.0"}}`)))
	want := `{"login_ip":"***","nested":{"client_ip":"***","user_agent":"***"}}`
	if got != want {
		t.Fatalf("MaskJSON(network identity) = %q, want %q", got, want)
	}
}

func TestMaskLogFieldJSON(t *testing.T) {
	tests := []struct {
		key  string
		raw  string
		want string
	}{
		{key: "ip", raw: `"203.0.113.42"`, want: `"203.0.113.*"`},
		{key: "user-agent", raw: `"Clash Verge/v1.7.7 Darwin/23.0"`, want: `"Clash Verge/***"`},
		{key: "query", raw: `"token=abcdef1234567890&flag=clash"`, want: `"flag=clash\u0026token=%2A%2A%2A"`},
		{key: "request", raw: `"GET example.com/v1/subscribe/config?token=abcdef1234567890"`, want: `"GET example.com/v1/subscribe/config?token=%2A%2A%2A"`},
		{key: "email", raw: `"user@example.com"`, want: `"u**r@example.com"`},
		{key: "token", raw: `"abcdef1234567890"`, want: `"***"`},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := string(MaskLogFieldJSON(tt.key, []byte(tt.raw))); got != tt.want {
				t.Fatalf("MaskLogFieldJSON() = %q, want %q", got, tt.want)
			}
		})
	}

	got := string(MaskLogFieldJSON("payload", []byte(`{"token":"abcdef","nested":{"email":"user@example.com"}}`)))
	want := `{"nested":{"email":"***"},"token":"***"}`
	if got != want {
		t.Fatalf("MaskLogFieldJSON(object) = %q, want %q", got, want)
	}

	got = string(MaskLogFieldJSON("payload", []byte(`"{\"auth_identifier\":\"user@example.com\",\"client_ip\":\"203.0.113.42\"}"`)))
	want = `"{\"auth_identifier\":\"***\",\"client_ip\":\"***\"}"`
	if got != want {
		t.Fatalf("MaskLogFieldJSON(encoded payload) = %q, want %q", got, want)
	}
}

func TestMaskLogFieldJSONKeepsUnchangedJSONShape(t *testing.T) {
	if got := string(MaskLogFieldJSON("message", []byte(`"already\u003cescaped"`))); got != `"already\u003cescaped"` {
		t.Fatalf("unchanged string raw message = %q", got)
	}
	if got := string(MaskLogFieldJSON("optional", []byte(`null`))); got != `null` {
		t.Fatalf("null raw message = %q, want null", got)
	}
	if got := string(MaskLogFieldJSON("count", []byte(`123`))); got != `123` {
		t.Fatalf("number raw message = %q, want 123", got)
	}
}

func TestNormalizeKeyMatchesLegacyShape(t *testing.T) {
	tests := map[string]string{
		" token ":       "token",
		"Access_Token":  "accesstoken",
		"client-secret": "clientsecret",
		"user agent":    "useragent",
		"AlreadyPlain":  "alreadyplain",
	}
	for input, want := range tests {
		if got := normalizeKey(input); got != want {
			t.Fatalf("normalizeKey(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeKeyPlainLowercaseDoesNotAllocate(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_ = normalizeKey("token")
	})
	if allocs != 0 {
		t.Fatalf("normalizeKey plain lowercase allocs = %v, want 0", allocs)
	}
}
