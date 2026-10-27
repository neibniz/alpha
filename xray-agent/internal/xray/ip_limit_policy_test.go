package xray

import (
	"net/netip"
	"testing"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
	router "github.com/xtls/xray-core/app/router"
)

func TestBuildPolicyRoutingPlacesScopedBlocksBeforeBaseRules(t *testing.T) {
	baseRule := &router.RoutingRule{TargetTag: &router.RoutingRule_Tag{Tag: "direct"}}
	base := &router.Config{Rule: []*router.RoutingRule{baseRule}}
	prefix := netip.MustParsePrefix("203.0.113.7/32")
	routing := buildPolicyRouting(base, map[string]blockedSource{
		"blocked": {userSubscribeID: 12, email: "u_12", prefix: prefix},
	})
	if len(routing.Rule) != 2 || routing.Rule[1] != baseRule {
		t.Fatalf("routing rules = %+v", routing.Rule)
	}
	block := routing.Rule[0]
	if block.GetTag() != ipLimitOutboundTag || len(block.UserEmail) != 1 || block.UserEmail[0] != "u_12" {
		t.Fatalf("block rule = %+v", block)
	}
	if got := block.SourceIp[0].GetCustom().Cidr; got.Prefix != 32 || netip.AddrFrom4([4]byte(got.Ip)) != prefix.Addr() {
		t.Fatalf("block CIDR = %+v", got)
	}
}

func TestBuildPolicyRoutingPreservesIPv6SourceScope(t *testing.T) {
	prefix := netip.MustParsePrefix("2001:db8:1:2::/64")
	routing := buildPolicyRouting(&router.Config{}, map[string]blockedSource{
		"blocked": {userSubscribeID: 12, email: "u_12", prefix: prefix},
	})
	block := routing.Rule[0]
	if len(block.SourceIp) != 1 || block.SourceIp[0].GetCustom() == nil {
		t.Fatalf("source IP rules = %+v", block.SourceIp)
	}
	got := block.SourceIp[0].GetCustom().Cidr
	addr, ok := netip.AddrFromSlice(got.Ip)
	if !ok || addr != prefix.Addr() || got.Prefix != 64 {
		t.Fatalf("block CIDR = %+v", got)
	}
	if block.GetTag() != ipLimitOutboundTag || len(block.UserEmail) != 1 || block.UserEmail[0] != "u_12" {
		t.Fatalf("block rule = %+v", block)
	}
}

func TestIPLimitPolicyValidatesScopeAndClearsDisabledUsers(t *testing.T) {
	policy := NewIPLimitPolicy("127.0.0.1:1")
	policy.now = func() time.Time { return time.Unix(100, 0) }
	policy.SetUsers([]panel.XrayUser{{UserSubscribeID: 12, Email: "u_12", DeviceLimit: 1, Enabled: true}})

	policy.mu.Lock()
	prefix := netip.MustParsePrefix("2001:db8:1:2::/64")
	policy.blocked[blockedSourceKey(12, prefix)] = blockedSource{userSubscribeID: 12, email: "u_12", prefix: prefix, expiresAt: time.Unix(200, 0)}
	policy.mu.Unlock()

	if !policy.SetUsers([]panel.XrayUser{{UserSubscribeID: 12, Email: "u_12", DeviceLimit: 0, Enabled: true}}) {
		t.Fatal("disabling the limit should mark live routing dirty")
	}
	policy.mu.Lock()
	defer policy.mu.Unlock()
	if len(policy.blocked) != 0 || !policy.dirty {
		t.Fatalf("blocked = %+v, dirty = %v", policy.blocked, policy.dirty)
	}
}

func TestNormalizedPolicyPrefixRejectsBroadNetworks(t *testing.T) {
	if _, err := normalizedPolicyPrefix("203.0.113.0/24"); err == nil {
		t.Fatal("IPv4 /24 should be rejected")
	}
	if _, err := normalizedPolicyPrefix("2001:db8::/48"); err == nil {
		t.Fatal("IPv6 /48 should be rejected")
	}
	if got, err := normalizedPolicyPrefix("2001:db8:1:2::9/64"); err != nil || got.String() != "2001:db8:1:2::/64" {
		t.Fatalf("normalized IPv6 prefix = %v, err = %v", got, err)
	}
}
