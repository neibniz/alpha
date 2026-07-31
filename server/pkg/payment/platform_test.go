package payment

import "testing"

func TestPlatformStringUsesStableNames(t *testing.T) {
	tests := []struct {
		name string
		in   Platform
		want string
	}{
		{name: "stripe", in: Stripe, want: "Stripe"},
		{name: "alipay", in: AlipayF2F, want: "AlipayF2F"},
		{name: "epay", in: EPay, want: "EPay"},
		{name: "balance", in: Balance, want: "balance"},
		{name: "crypto saas", in: CryptoSaaS, want: "CryptoSaaS"},
		{name: "unsupported", in: UNSUPPORTED, want: "unsupported"},
		{name: "unknown", in: Platform(99), want: "unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePlatformKeepsSupportedNames(t *testing.T) {
	if got := ParsePlatform("CryptoSaaS"); got != CryptoSaaS {
		t.Fatalf("ParsePlatform(CryptoSaaS) = %v, want CryptoSaaS", got)
	}
	if got := ParsePlatform("missing"); got != UNSUPPORTED {
		t.Fatalf("ParsePlatform(missing) = %v, want UNSUPPORTED", got)
	}
}

func TestParsePlatformAcceptsLegacyAliases(t *testing.T) {
	tests := map[string]Platform{
		" stripe ":    Stripe,
		"epay":        EPay,
		"alipay_f2f":  AlipayF2F,
		"crypto_saas": CryptoSaaS,
		"BALANCE":     Balance,
	}
	for input, want := range tests {
		if got := ParsePlatform(input); got != want {
			t.Fatalf("ParsePlatform(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestNormalizePlatformReturnsCanonicalName(t *testing.T) {
	if got, ok := NormalizePlatform("epay"); !ok || got != "EPay" {
		t.Fatalf("NormalizePlatform(epay) = %q, %v, want EPay, true", got, ok)
	}
	if got, ok := NormalizePlatform("missing"); ok || got != "" {
		t.Fatalf("NormalizePlatform(missing) = %q, %v, want empty, false", got, ok)
	}
}

func TestCompatiblePlatformNamesIncludesLegacyRows(t *testing.T) {
	got := CompatiblePlatformNames("EPay")
	if len(got) != 2 || got[0] != "EPay" || got[1] != "epay" {
		t.Fatalf("CompatiblePlatformNames(EPay) = %#v", got)
	}
}

func TestGetSupportedPlatformsKeepsPaymentShape(t *testing.T) {
	platforms := GetSupportedPlatforms()
	if len(platforms) != 4 {
		t.Fatalf("payment platform count = %d, want 4", len(platforms))
	}
	for _, platform := range platforms {
		if platform.Platform == "" || platform.PlatformFieldDescription == nil {
			t.Fatalf("invalid payment platform = %#v", platform)
		}
	}
	cryptoFields := platforms[len(platforms)-1].PlatformFieldDescription
	if cryptoFields["type"] == "" {
		t.Fatal("CryptoSaaS payment type field is missing")
	}
}
