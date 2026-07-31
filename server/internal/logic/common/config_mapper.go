package common

import (
	"strings"

	runtimeconfig "github.com/neibniz/alpha/server/internal/config"
	"github.com/neibniz/alpha/server/internal/types"
)

func siteConfigFromRuntime(cfg runtimeconfig.SiteConfig) types.SiteConfig {
	return types.SiteConfig{
		Host:       cfg.Host,
		SiteName:   cfg.SiteName,
		SiteDesc:   cfg.SiteDesc,
		SiteLogo:   cfg.SiteLogo,
		Keywords:   cfg.Keywords,
		CustomHTML: cfg.CustomHTML,
		CustomData: cfg.CustomData,
	}
}

func subscribeConfigFromRuntime(cfg runtimeconfig.SubscribeConfig) types.PublicSubscribeConfig {
	return types.PublicSubscribeConfig{
		SingleModel:     cfg.SingleModel,
		SubscribePath:   cfg.SubscribePath,
		SubscribeDomain: cfg.SubscribeDomain,
		PanDomain:       cfg.PanDomain,
		UserAgentLimit:  cfg.UserAgentLimit,
		UserAgentList:   publicConfigListFromString(cfg.UserAgentList),
	}
}

func emailAuthConfigFromRuntime(cfg runtimeconfig.EmailConfig) types.PublicEmailAuthenticateConfig {
	return types.PublicEmailAuthenticateConfig{
		Enable:             cfg.Enable,
		EnableVerify:       cfg.EnableVerify,
		EnableDomainSuffix: cfg.EnableDomainSuffix,
		DomainSuffixList:   publicConfigListFromString(cfg.DomainSuffixList),
	}
}

func registerConfigFromRuntime(cfg runtimeconfig.RegisterConfig) types.PubilcRegisterConfig {
	return types.PubilcRegisterConfig{
		StopRegister:            cfg.StopRegister,
		EnableIpRegisterLimit:   cfg.EnableIpRegisterLimit,
		IpRegisterLimit:         cfg.IpRegisterLimit,
		IpRegisterLimitDuration: cfg.IpRegisterLimitDuration,
	}
}

func verifyConfigFromRuntime(cfg runtimeconfig.Verify) types.VeifyConfig {
	return types.VeifyConfig{
		TurnstileSiteKey:          cfg.TurnstileSiteKey,
		EnableLoginVerify:         cfg.LoginVerify,
		EnableRegisterVerify:      cfg.RegisterVerify,
		EnableResetPasswordVerify: cfg.ResetPasswordVerify,
	}
}

func inviteConfigFromRuntime(cfg runtimeconfig.InviteConfig) types.InviteConfig {
	return types.InviteConfig{
		ForcedInvite:       cfg.ForcedInvite,
		ReferralPercentage: cfg.ReferralPercentage,
		OnlyFirstPurchase:  cfg.OnlyFirstPurchase,
	}
}

func publicConfigListFromString(value string) []string {
	var list publicConfigListCollector
	forEachPublicConfigListItem(value, list.Add)
	return list.Items()
}

func forEachPublicConfigListItem(value string, visit func(string)) {
	start := 0
	for i := 0; i <= len(value); i++ {
		if i < len(value) && value[i] != ',' && value[i] != '\n' && value[i] != '\r' {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value[start:i]), "@"))
		if item != "" {
			visit(strings.ToLower(item))
		}
		start = i + 1
	}
}

const publicConfigListLinearLimit = 8

type publicConfigListCollector struct {
	items []string
	index map[string]struct{}
}

func (c *publicConfigListCollector) Add(item string) {
	if item == "" {
		return
	}
	if c.index != nil {
		if _, ok := c.index[item]; ok {
			return
		}
		c.index[item] = struct{}{}
		c.items = append(c.items, item)
		return
	}
	for _, existing := range c.items {
		if existing == item {
			return
		}
	}
	if c.items == nil {
		c.items = make([]string, 0, publicConfigListLinearLimit)
	}
	c.items = append(c.items, item)
	if len(c.items) > publicConfigListLinearLimit {
		c.index = make(map[string]struct{}, len(c.items))
		for _, existing := range c.items {
			c.index[existing] = struct{}{}
		}
	}
}

func (c publicConfigListCollector) Items() []string {
	if len(c.items) == 0 {
		return []string{}
	}
	return c.items
}
