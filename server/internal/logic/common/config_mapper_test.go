package common

import (
	"reflect"
	"testing"

	runtimeconfig "github.com/neibniz/alpha/server/internal/config"
	modelAuth "github.com/neibniz/alpha/server/internal/model/auth"
	"github.com/neibniz/alpha/server/internal/types"
)

func TestRuntimeConfigMappersExposePublicGlobalConfigShape(t *testing.T) {
	site := siteConfigFromRuntime(runtimeconfig.SiteConfig{
		Host:       "https://alpha.example",
		SiteName:   "Alpha",
		SiteDesc:   "Panel",
		SiteLogo:   "/logo.png",
		Keywords:   "alpha,panel",
		CustomHTML: "<meta>",
		CustomData: `{"theme":"dark"}`,
	})
	if site.Host != "https://alpha.example" || site.CustomData != `{"theme":"dark"}` {
		t.Fatalf("site config mismatch: %+v", site)
	}

	subscribe := subscribeConfigFromRuntime(runtimeconfig.SubscribeConfig{
		SingleModel:     true,
		SubscribePath:   "/v1/subscribe/config",
		SubscribeDomain: "sub.example",
		PanDomain:       true,
		UserAgentLimit:  true,
		UserAgentList:   "Clash\nsing-box, Clash",
	})
	if !subscribe.SingleModel || subscribe.SubscribeDomain != "sub.example" || !reflect.DeepEqual(subscribe.UserAgentList, []string{"clash", "sing-box"}) {
		t.Fatalf("subscribe config mismatch: %+v", subscribe)
	}

	email := emailAuthConfigFromRuntime(runtimeconfig.EmailConfig{
		Enable:             true,
		EnableVerify:       true,
		EnableDomainSuffix: true,
		DomainSuffixList:   "@Example.com\nalpha.dev, example.com",
	})
	if !email.Enable || !email.EnableVerify || !reflect.DeepEqual(email.DomainSuffixList, []string{"example.com", "alpha.dev"}) {
		t.Fatalf("email config mismatch: %+v", email)
	}

	register := registerConfigFromRuntime(runtimeconfig.RegisterConfig{
		StopRegister:            true,
		EnableTrial:             true,
		TrialSubscribe:          7,
		EnableIpRegisterLimit:   true,
		IpRegisterLimit:         3,
		IpRegisterLimitDuration: 60,
	})
	wantRegister := types.PubilcRegisterConfig{
		StopRegister:            true,
		EnableIpRegisterLimit:   true,
		IpRegisterLimit:         3,
		IpRegisterLimitDuration: 60,
	}
	if register != wantRegister {
		t.Fatalf("register config mismatch: got %+v want %+v", register, wantRegister)
	}

	verify := verifyConfigFromRuntime(runtimeconfig.Verify{
		TurnstileSiteKey:    "site",
		TurnstileSecret:     "secret",
		LoginVerify:         true,
		RegisterVerify:      true,
		ResetPasswordVerify: true,
	})
	wantVerify := types.VeifyConfig{
		TurnstileSiteKey:          "site",
		EnableLoginVerify:         true,
		EnableRegisterVerify:      true,
		EnableResetPasswordVerify: true,
	}
	if verify != wantVerify {
		t.Fatalf("verify config mismatch: got %+v want %+v", verify, wantVerify)
	}

	invite := inviteConfigFromRuntime(runtimeconfig.InviteConfig{
		ForcedInvite:       true,
		ReferralPercentage: 15,
		OnlyFirstPurchase:  true,
	})
	if invite.ReferralPercentage != 15 || !invite.ForcedInvite || !invite.OnlyFirstPurchase {
		t.Fatalf("invite config mismatch: %+v", invite)
	}
}

func TestPublicConfigListFromStringKeepsStableArrayShape(t *testing.T) {
	for _, value := range []string{"", "  ", "\n,\r"} {
		got := publicConfigListFromString(value)
		if got == nil || len(got) != 0 {
			t.Fatalf("publicConfigListFromString(%q) = %#v, want non-nil empty slice", value, got)
		}
	}
}

func TestPublicConfigListFromStringNormalizesInSinglePass(t *testing.T) {
	got := publicConfigListFromString(" @ Example.com \nALPHA.dev, example.com\rBeta.dev ")
	want := []string{"example.com", "alpha.dev", "beta.dev"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("publicConfigListFromString() = %#v, want %#v", got, want)
	}
}

func TestPublicConfigListCollectorUsesLinearPathForSmallInput(t *testing.T) {
	var collector publicConfigListCollector
	collector.Add("clash")
	collector.Add("sing-box")
	collector.Add("clash")

	if collector.index != nil {
		t.Fatal("small public config list should not allocate map")
	}
	if got := collector.Items(); !reflect.DeepEqual(got, []string{"clash", "sing-box"}) {
		t.Fatalf("collector items = %#v", got)
	}
}

func TestPublicConfigListCollectorUpgradesLargeInputToMap(t *testing.T) {
	var collector publicConfigListCollector
	for i := 0; i <= publicConfigListLinearLimit; i++ {
		collector.Add(string(rune('a' + i)))
	}
	collector.Add("a")

	if collector.index == nil {
		t.Fatal("large public config list should allocate map")
	}
	if got := len(collector.Items()); got != publicConfigListLinearLimit+1 {
		t.Fatalf("collector item count = %d, want %d", got, publicConfigListLinearLimit+1)
	}
}

func TestPublicConfigListFromStringSmallInputAllocationBudget(t *testing.T) {
	allocs := testing.AllocsPerRun(1000, func() {
		_ = publicConfigListFromString("clash,sing-box,clash")
	})
	if allocs > 1 {
		t.Fatalf("publicConfigListFromString small input allocations = %v, want <= 1", allocs)
	}
}

func TestGlobalAuthMethodsFromModelsExposesOnlyEnabledGitHub(t *testing.T) {
	enabled := true
	disabled := false
	methods := globalAuthMethodsFromModels([]*modelAuth.Auth{
		nil,
		{Method: "email", Enabled: &enabled},
		{Method: "google", Enabled: &disabled},
		{Method: "github", Enabled: &enabled},
		{Method: "device", Enabled: &enabled},
		{Method: "mobile", Enabled: &enabled},
	})

	if !reflect.DeepEqual(methods, []string{"github"}) {
		t.Fatalf("methods = %#v, want enabled supported oauth methods only", methods)
	}
}

func TestGlobalAuthMethodsFromModelsKeepsStableEmptyShapes(t *testing.T) {
	methods := globalAuthMethodsFromModels(nil)
	if methods == nil || len(methods) != 0 {
		t.Fatalf("empty methods should be non-nil empty slice: %#v", methods)
	}
}

func TestGlobalAuthMethodsFromModelsKeepsEmptyShapeWhenAllDisabled(t *testing.T) {
	disabled := false
	methods := globalAuthMethodsFromModels([]*modelAuth.Auth{
		{Method: "email", Enabled: &disabled},
		{Method: "github", Enabled: &disabled},
	})
	if methods == nil || len(methods) != 0 {
		t.Fatalf("disabled methods should return non-nil empty slice: %#v", methods)
	}
}

func TestGlobalAuthMethodsFromModelsDeduplicatesResponseMethods(t *testing.T) {
	enabled := true
	methods := globalAuthMethodsFromModels([]*modelAuth.Auth{
		{Method: "", Enabled: &enabled},
		{Method: "email", Enabled: &enabled},
		{Method: "email", Enabled: &enabled},
		{Method: "github", Enabled: &enabled},
		{Method: "github", Enabled: &enabled},
		{Method: "device", Enabled: &enabled},
	})

	if !reflect.DeepEqual(methods, []string{"github"}) {
		t.Fatalf("methods = %#v, want unique supported oauth methods", methods)
	}
}

func TestAuthMethodCollectorUsesLinearPathForSmallInput(t *testing.T) {
	collector := newAuthMethodCollector(3)
	collector.Add("email")
	collector.Add("google")
	collector.Add("email")

	if collector.index != nil {
		t.Fatal("small auth method list should not allocate map")
	}
	if got := collector.Items(); !reflect.DeepEqual(got, []string{"email", "google"}) {
		t.Fatalf("collector items = %#v, want stable unique methods", got)
	}
}

func TestAuthMethodCollectorUpgradesLargeInputToMap(t *testing.T) {
	collector := newAuthMethodCollector(16)
	for i := 0; i <= authMethodLinearLimit; i++ {
		collector.Add(string(rune('a' + i)))
	}
	collector.Add("a")

	if collector.index == nil {
		t.Fatal("large auth method list should allocate map")
	}
	if got := len(collector.Items()); got != authMethodLinearLimit+1 {
		t.Fatalf("collector item count = %d, want %d", got, authMethodLinearLimit+1)
	}
}
