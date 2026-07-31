package xray

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"sort"
	"sync"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/panel"
	router "github.com/xtls/xray-core/app/router"
	routercmd "github.com/xtls/xray-core/app/router/command"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/infra/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ipLimitPolicyInterval = 5 * time.Second
	maxBlockDuration      = 5 * time.Minute
)

type blockedSource struct {
	userSubscribeID int64
	email           string
	prefix          netip.Prefix
	expiresAt       time.Time
}

type IPLimitPolicy struct {
	address string

	applyMu  sync.Mutex
	mu       sync.Mutex
	base     *router.Config
	users    map[int64]string
	blocked  map[string]blockedSource
	dirty    bool
	revision uint64
	now      func() time.Time
}

func NewIPLimitPolicy(apiAddress string) *IPLimitPolicy {
	return &IPLimitPolicy{
		address: apiAddress,
		users:   make(map[int64]string),
		blocked: make(map[string]blockedSource),
		now:     time.Now,
	}
}

func (p *IPLimitPolicy) SetBaseConfig(runtimeConfig []byte) error {
	var root struct {
		Routing json.RawMessage `json:"routing"`
	}
	if err := json.Unmarshal(runtimeConfig, &root); err != nil {
		return fmt.Errorf("decode xray runtime routing: %w", err)
	}
	if len(root.Routing) == 0 || string(root.Routing) == "null" {
		root.Routing = json.RawMessage(`{}`)
	}
	var routingConfig conf.RouterConfig
	if err := json.Unmarshal(root.Routing, &routingConfig); err != nil {
		return fmt.Errorf("decode xray routing config: %w", err)
	}
	built, err := routingConfig.Build()
	if err != nil {
		return fmt.Errorf("build xray routing config: %w", err)
	}
	p.mu.Lock()
	p.base = built
	p.mu.Unlock()
	return nil
}

// SetUsers updates the authoritative set of subscriptions for which an IP
// limit is enabled. It returns true when live routing needs to be refreshed.
func (p *IPLimitPolicy) SetUsers(users []panel.XrayUser) bool {
	next := make(map[int64]string, len(users))
	for _, user := range users {
		if user.Enabled && user.DeviceLimit > 0 && user.UserSubscribeID > 0 && user.Email != "" {
			next[user.UserSubscribeID] = user.Email
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	changed := false
	for key, blocked := range p.blocked {
		email, ok := next[blocked.userSubscribeID]
		if !ok {
			delete(p.blocked, key)
			changed = true
			continue
		}
		if email != blocked.email {
			blocked.email = email
			p.blocked[key] = blocked
			changed = true
		}
	}
	p.users = next
	if changed {
		p.markDirtyLocked()
	}
	return changed
}

func (p *IPLimitPolicy) ApplyIPLimitDecisions(ctx context.Context, decisions []panel.IPLimitDecision) error {
	now := p.now().UTC()
	var validationErrors []error

	p.mu.Lock()
	changed := p.pruneLocked(now)
	for _, decision := range decisions {
		if decision.UserSubscribeID <= 0 {
			validationErrors = append(validationErrors, errors.New("IP limit decision has invalid subscription ID"))
			continue
		}
		prefix, err := normalizedPolicyPrefix(decision.SourceNetwork)
		if err != nil {
			validationErrors = append(validationErrors, err)
			continue
		}
		key := blockedSourceKey(decision.UserSubscribeID, prefix)
		if !decision.Blocked {
			if _, ok := p.blocked[key]; ok {
				delete(p.blocked, key)
				changed = true
			}
			continue
		}
		email, enabled := p.users[decision.UserSubscribeID]
		if !enabled {
			if _, ok := p.blocked[key]; ok {
				delete(p.blocked, key)
				changed = true
			}
			continue
		}
		expiresAt := time.UnixMilli(decision.ExpiresAt).UTC()
		if !expiresAt.After(now) {
			if _, ok := p.blocked[key]; ok {
				delete(p.blocked, key)
				changed = true
			}
			continue
		}
		if expiresAt.After(now.Add(maxBlockDuration)) {
			expiresAt = now.Add(maxBlockDuration)
		}
		next := blockedSource{
			userSubscribeID: decision.UserSubscribeID,
			email:           email,
			prefix:          prefix,
			expiresAt:       expiresAt,
		}
		if current, ok := p.blocked[key]; !ok || current != next {
			p.blocked[key] = next
			changed = true
		}
	}
	if changed {
		p.markDirtyLocked()
	}
	p.mu.Unlock()

	if changed {
		if err := p.apply(ctx, false); err != nil {
			return errors.Join(append(validationErrors, err)...)
		}
	}
	return errors.Join(validationErrors...)
}

func (p *IPLimitPolicy) Reapply(ctx context.Context, force bool) error {
	return p.apply(ctx, force)
}

func (p *IPLimitPolicy) Run(ctx context.Context) {
	ticker := time.NewTicker(ipLimitPolicyInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			p.mu.Lock()
			if p.pruneLocked(now.UTC()) {
				p.markDirtyLocked()
			}
			dirty := p.dirty
			p.mu.Unlock()
			if dirty {
				if err := p.apply(ctx, false); err != nil && ctx.Err() == nil {
					log.Printf("refresh xray IP limit routes failed: %v", err)
				}
			}
		}
	}
}

func (p *IPLimitPolicy) apply(ctx context.Context, force bool) error {
	p.applyMu.Lock()
	defer p.applyMu.Unlock()

	p.mu.Lock()
	if p.base == nil {
		p.mu.Unlock()
		return errors.New("xray IP limit policy has no base routing config")
	}
	if !force && !p.dirty {
		p.mu.Unlock()
		return nil
	}
	p.pruneLocked(p.now().UTC())
	routing := buildPolicyRouting(p.base, p.blocked)
	revision := p.revision
	p.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(callCtx, p.address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()
	client := routercmd.NewRoutingServiceClient(conn)
	if _, err := client.AddRule(callCtx, &routercmd.AddRuleRequest{
		Config:       serial.ToTypedMessage(routing),
		ShouldAppend: false,
	}); err != nil {
		return err
	}

	p.mu.Lock()
	if p.revision == revision {
		p.dirty = false
	}
	p.mu.Unlock()
	return nil
}

func buildPolicyRouting(base *router.Config, blocked map[string]blockedSource) *router.Config {
	config := &router.Config{
		DomainStrategy: base.DomainStrategy,
		BalancingRule:  append([]*router.BalancingRule(nil), base.BalancingRule...),
		Rule:           make([]*router.RoutingRule, 0, len(blocked)+len(base.Rule)),
	}
	keys := make([]string, 0, len(blocked))
	for key := range blocked {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := blocked[key]
		addr := item.prefix.Addr()
		config.Rule = append(config.Rule, &router.RoutingRule{
			TargetTag: &router.RoutingRule_Tag{Tag: ipLimitOutboundTag},
			RuleTag:   policyRuleTag(item.userSubscribeID, item.prefix),
			SourceGeoip: []*router.GeoIP{{
				Cidr: []*router.CIDR{{Ip: addr.AsSlice(), Prefix: uint32(item.prefix.Bits())}},
			}},
			UserEmail: []string{item.email},
		})
	}
	config.Rule = append(config.Rule, base.Rule...)
	return config
}

func normalizedPolicyPrefix(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil {
		return netip.Prefix{}, errors.New("IP limit decision has invalid source network")
	}
	prefix = prefix.Masked()
	wantBits := 64
	if prefix.Addr().Is4() {
		wantBits = 32
	}
	if prefix.Bits() != wantBits {
		return netip.Prefix{}, fmt.Errorf("IP limit source network must use /%d", wantBits)
	}
	return prefix, nil
}

func blockedSourceKey(userSubscribeID int64, prefix netip.Prefix) string {
	return fmt.Sprintf("%d:%s", userSubscribeID, prefix.String())
}

func policyRuleTag(userSubscribeID int64, prefix netip.Prefix) string {
	sum := sha256.Sum256([]byte(prefix.String()))
	return fmt.Sprintf("alpha-ip-limit-%d-%s", userSubscribeID, hex.EncodeToString(sum[:6]))
}

func (p *IPLimitPolicy) pruneLocked(now time.Time) bool {
	changed := false
	for key, item := range p.blocked {
		if !item.expiresAt.After(now) {
			delete(p.blocked, key)
			changed = true
		}
	}
	return changed
}

func (p *IPLimitPolicy) markDirtyLocked() {
	p.dirty = true
	p.revision++
}
