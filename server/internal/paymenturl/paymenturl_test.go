package paymenturl

import "testing"

func TestNotifyURLUsesTrustedConfiguredBase(t *testing.T) {
	tests := []struct {
		name          string
		paymentDomain string
		siteURL       string
		gateway       bool
		want          string
	}{
		{name: "payment domain", paymentDomain: "https://pay.example.test/base/", siteURL: "https://site.example.test", want: "https://pay.example.test/base/v1/notify/EPay/token"},
		{name: "site fallback", siteURL: "https://site.example.test/", want: "https://site.example.test/v1/notify/EPay/token"},
		{name: "gateway", siteURL: "https://site.example.test", gateway: true, want: "https://site.example.test/api/v1/notify/EPay/token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NotifyURL(test.paymentDomain, test.siteURL, "EPay", "token", test.gateway)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("NotifyURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNotifyURLRejectsUntrustedOrMalformedInputs(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		siteURL  string
		platform string
		token    string
	}{
		{name: "no configured base", platform: "EPay", token: "token"},
		{name: "credentials", domain: "https://user@pay.example.test", platform: "EPay", token: "token"},
		{name: "query", domain: "https://pay.example.test?next=evil", platform: "EPay", token: "token"},
		{name: "non-http", domain: "javascript:alert(1)", platform: "EPay", token: "token"},
		{name: "platform path injection", domain: "https://pay.example.test", platform: "../EPay", token: "token"},
		{name: "token path injection", domain: "https://pay.example.test", platform: "EPay", token: "../token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NotifyURL(test.domain, test.siteURL, test.platform, test.token, false); err == nil {
				t.Fatal("NotifyURL() should reject unsafe input")
			}
		})
	}
}

func TestValidateReturnURLRestrictsOrigin(t *testing.T) {
	trusted := []string{"https://user.example.test/app", "https://admin.example.test"}
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "empty"},
		{name: "site path and query", raw: "https://user.example.test/orders/1?paid=1#receipt"},
		{name: "default port", raw: "https://user.example.test:443/orders/1"},
		{name: "allowed origin", raw: "https://admin.example.test/orders/1"},
		{name: "attacker", raw: "https://evil.example.test/orders/1", wantErr: true},
		{name: "subdomain", raw: "https://evil.user.example.test/orders/1", wantErr: true},
		{name: "credentials", raw: "https://user@user.example.test/orders/1", wantErr: true},
		{name: "relative", raw: "/orders/1", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateReturnURL(test.raw, trusted...)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateReturnURL() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestNormalizeBaseURL(t *testing.T) {
	got, err := NormalizeBaseURL(" https://pay.example.test/base/// ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://pay.example.test/base" {
		t.Fatalf("NormalizeBaseURL() = %q", got)
	}
	if got, err := NormalizeBaseURL("  "); err != nil || got != "" {
		t.Fatalf("blank NormalizeBaseURL() = %q, %v", got, err)
	}
}
