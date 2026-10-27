package xraytemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/neibniz/alpha/server/adapter"
	"github.com/neibniz/alpha/server/internal/model/node"
	"github.com/neibniz/alpha/server/internal/viewutil"
	"github.com/neibniz/alpha/server/pkg/tool"
)

type UserInfo struct {
	Password string
	UUID     string
}

type Ref map[string]map[string]ConfigObject

type renderBindingPlan struct {
	entries []renderBindingPlanEntry
	counts  ConfigBucketCounts
	ref     Ref
}

type renderBindingPlanEntry struct {
	binding   *node.ServerXrayTemplate
	variables ConfigObject
}

type RenderBindingConfigOptions struct {
	TemplateType            string
	SubscriptionEnabledOnly bool
	bindingFilter           func(*node.ServerXrayTemplate) bool
}

type RenderedBindingConfig struct {
	Binding *node.ServerXrayTemplate
	Config  ConfigObject
}

type RenderedSubscriptionProxy struct {
	Binding *node.ServerXrayTemplate
	Proxy   *adapter.Proxy
}

func BindingMatchesInbound(binding *node.ServerXrayTemplate, inboundAlias string) bool {
	inboundAlias = strings.TrimSpace(inboundAlias)
	if inboundAlias == "" {
		return false
	}
	for _, candidate := range BindingInboundAliases(binding) {
		if candidate == inboundAlias {
			return true
		}
	}
	return false
}

func BindingInboundAliases(binding *node.ServerXrayTemplate) []string {
	if binding == nil || binding.Template == nil || binding.Template.Type != node.XrayTemplateInbound {
		return nil
	}
	return bindingInboundAliases(binding, jsonObject(binding.Template.Config))
}

func BindingInboundAliasesFromConfig(binding *node.ServerXrayTemplate, cfg ConfigObject) []string {
	if binding == nil || binding.Template == nil || binding.Template.Type != node.XrayTemplateInbound {
		return nil
	}
	return bindingInboundAliases(binding, cfg)
}

func bindingInboundAliases(binding *node.ServerXrayTemplate, cfg ConfigObject) []string {
	out := make([]string, 0, 6)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if containsInboundAlias(out, value) {
			return
		}
		out = append(out, value)
	}
	addNameVariants := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		add(value)
		trimmed := strings.TrimSuffix(value, " Inbound")
		add(trimmed)
		if trimmed == value {
			add(value + " Inbound")
		}
	}
	addNameVariants(binding.Alias)
	addNameVariants(binding.Template.Name)
	if len(cfg) > 0 {
		add(stringValue(cfg, "tag"))
	}
	return out
}

func containsInboundAlias(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

type InboundBindingIndex struct {
	rows    []inboundBindingAliasEntry
	byAlias map[string]int
	capHint int
}

type inboundBindingAliasEntry struct {
	alias    string
	bindings []*node.ServerXrayTemplate
}

const smallInboundBindingAliasIndexThreshold = 8

func NewInboundBindingIndex(bindings []*node.ServerXrayTemplate) *InboundBindingIndex {
	index := &InboundBindingIndex{capHint: len(bindings) * 2}
	for _, binding := range bindings {
		if binding == nil || binding.Template == nil || binding.Template.Type != node.XrayTemplateInbound {
			continue
		}
		if binding.SubscriptionEnabled != nil && !*binding.SubscriptionEnabled {
			continue
		}
		for _, alias := range BindingInboundAliases(binding) {
			index.add(alias, binding)
		}
	}
	return index
}

func (i *InboundBindingIndex) add(alias string, binding *node.ServerXrayTemplate) {
	if i == nil || binding == nil {
		return
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return
	}
	if i.byAlias != nil {
		if pos, ok := i.byAlias[alias]; ok {
			i.rows[pos].bindings = append(i.rows[pos].bindings, binding)
			return
		}
		i.byAlias[alias] = len(i.rows)
		i.rows = append(i.rows, inboundBindingAliasEntry{alias: alias, bindings: []*node.ServerXrayTemplate{binding}})
		return
	}
	for pos := range i.rows {
		if i.rows[pos].alias == alias {
			i.rows[pos].bindings = append(i.rows[pos].bindings, binding)
			return
		}
	}
	i.ensureRows()
	i.rows = append(i.rows, inboundBindingAliasEntry{alias: alias, bindings: []*node.ServerXrayTemplate{binding}})
	if len(i.rows) > smallInboundBindingAliasIndexThreshold {
		i.upgradeLookup()
	}
}

func (i *InboundBindingIndex) ensureRows() {
	if i.rows != nil {
		return
	}
	i.rows = make([]inboundBindingAliasEntry, 0, inboundBindingIndexInitialCap(i.capHint))
}

func (i *InboundBindingIndex) upgradeLookup() {
	if i.byAlias != nil {
		return
	}
	nextCap := renderLookupCap(len(i.rows), i.capHint)
	if nextCap > cap(i.rows) {
		rows := make([]inboundBindingAliasEntry, len(i.rows), nextCap)
		copy(rows, i.rows)
		i.rows = rows
	}
	i.byAlias = make(map[string]int, nextCap)
	for pos, row := range i.rows {
		i.byAlias[row.alias] = pos
	}
}

func inboundBindingIndexInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallInboundBindingAliasIndexThreshold+1 {
		return smallInboundBindingAliasIndexThreshold + 1
	}
	return capHint
}

func (i *InboundBindingIndex) Match(inboundAlias string) []*node.ServerXrayTemplate {
	if i == nil || len(i.rows) == 0 {
		return nil
	}
	inboundAlias = strings.TrimSpace(inboundAlias)
	if inboundAlias == "" {
		return nil
	}
	if i.byAlias != nil {
		pos, ok := i.byAlias[inboundAlias]
		if !ok {
			return nil
		}
		return i.rows[pos].bindings
	}
	for pos := range i.rows {
		if i.rows[pos].alias == inboundAlias {
			return i.rows[pos].bindings
		}
	}
	return nil
}

func BuildRef(server *node.Server, bindings []*node.ServerXrayTemplate) Ref {
	ref := newEmptyRef()
	for index, binding := range bindings {
		if binding == nil || binding.Template == nil {
			continue
		}
		alias := renderBindingAlias(binding, index)
		vars := Variables(binding)
		ref[binding.Template.Type][alias] = renderRefVariables(server, alias, vars)
	}
	return ref
}

func newRenderBindingPlan(server *node.Server, bindings []*node.ServerXrayTemplate) renderBindingPlan {
	return newRenderBindingPlanWithOptions(server, bindings, RenderBindingConfigOptions{})
}

func newRenderBindingPlanWithOptions(server *node.Server, bindings []*node.ServerXrayTemplate, opts RenderBindingConfigOptions) renderBindingPlan {
	entries := viewutil.NewRowsCollector[renderBindingPlanEntry](len(bindings))
	plan := renderBindingPlan{
		ref: newEmptyRef(),
	}
	for index, binding := range bindings {
		if binding == nil || binding.Template == nil {
			continue
		}
		alias := renderBindingAlias(binding, index)
		vars := Variables(binding)
		plan.ref[binding.Template.Type][alias] = renderRefVariables(server, alias, cloneConfigObject(vars))
		if opts.include(binding) {
			entries.Add(renderBindingPlanEntry{
				binding:   binding,
				variables: vars,
			})
			plan.counts.Add(binding.Template.Type)
		}
	}
	plan.entries = entries.Rows()
	return plan
}

func (o RenderBindingConfigOptions) include(binding *node.ServerXrayTemplate) bool {
	if binding == nil || binding.Template == nil {
		return false
	}
	if o.TemplateType != "" && binding.Template.Type != o.TemplateType {
		return false
	}
	if o.SubscriptionEnabledOnly && binding.SubscriptionEnabled != nil && !*binding.SubscriptionEnabled {
		return false
	}
	if o.bindingFilter != nil && !o.bindingFilter(binding) {
		return false
	}
	return true
}

func newEmptyRef() Ref {
	return Ref{
		node.XrayTemplateInbound:  {},
		node.XrayTemplateOutbound: {},
		node.XrayTemplateDNS:      {},
		node.XrayTemplateRouting:  {},
		node.XrayTemplateGeodata:  {},
	}
}

func renderBindingAlias(binding *node.ServerXrayTemplate, index int) string {
	if binding == nil || binding.Template == nil {
		return ""
	}
	if binding.Alias != "" {
		return binding.Alias
	}
	return fmt.Sprintf("%s_%d", binding.Template.Type, index+1)
}

func renderRefVariables(server *node.Server, alias string, vars ConfigObject) ConfigObject {
	if vars == nil {
		vars = ConfigObject{}
	}
	if _, ok := vars["tag"]; !ok && alias != "" {
		vars["tag"] = alias
	}
	vars["alias"] = alias
	vars["server_id"] = server.Id
	vars["server_name"] = server.Name
	vars["server_address"] = server.Address
	return vars
}

func cloneConfigObject(source ConfigObject) ConfigObject {
	if len(source) == 0 {
		return ConfigObject{}
	}
	cloned := make(ConfigObject, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func Variables(binding *node.ServerXrayTemplate) ConfigObject {
	vars := ConfigObject{}
	if binding == nil {
		return vars
	}
	if binding.Template != nil {
		merge(vars, jsonObject(binding.Template.DefaultVariables))
		merge(vars, ConfigVariables(binding))
	}
	merge(vars, jsonObject(binding.Variables))
	return vars
}

func ConfigVariables(binding *node.ServerXrayTemplate) ConfigObject {
	if binding == nil || binding.Template == nil {
		return ConfigObject{}
	}
	cfg := jsonObject(binding.Template.Config)
	return ExtractConfigVariables(cfg, binding.Template.Type)
}

func ExtractConfigVariables(cfg ConfigObject, templateType string) ConfigObject {
	vars := ConfigObject{}
	if len(cfg) == 0 {
		return vars
	}
	copyKey(vars, cfg, "tag", "tag")
	copyKey(vars, cfg, "listen", "listen")
	copyKey(vars, cfg, "port", "port")
	copyKey(vars, cfg, "protocol", "protocol")
	copyKey(vars, cfg, "sendThrough", "sendThrough")

	switch templateType {
	case node.XrayTemplateDNS:
		extractDNSVariables(vars, cfg)
		return vars
	case node.XrayTemplateRouting:
		extractRoutingVariables(vars, cfg)
		return vars
	case node.XrayTemplateGeodata:
		extractGeodataVariables(vars, cfg)
		return vars
	}

	if settings, ok := cfg["settings"].(ConfigObject); ok {
		extractSettingsVariables(vars, settings, stringValue(cfg, "protocol"), templateType)
	}
	if stream, ok := cfg["streamSettings"].(ConfigObject); ok {
		extractStreamVariables(vars, stream, templateType)
	}
	return vars
}

func SubscriptionVariables(binding *node.ServerXrayTemplate) ConfigObject {
	vars := Variables(binding)
	if binding != nil {
		merge(vars, jsonObject(binding.SubscriptionVariables))
	}
	return vars
}

func RenderBindingConfig(server *node.Server, binding *node.ServerXrayTemplate, ref Ref) (ConfigObject, error) {
	return renderBindingConfigWithVariables(server, binding, ref, nil)
}

func RenderBindingConfigList(server *node.Server, bindings []*node.ServerXrayTemplate, opts RenderBindingConfigOptions) ([]RenderedBindingConfig, error) {
	plan := newRenderBindingPlanWithOptions(server, bindings, opts)
	if len(plan.entries) == 0 {
		return []RenderedBindingConfig{}, nil
	}
	result := make([]RenderedBindingConfig, 0, len(plan.entries))
	for _, entry := range plan.entries {
		cfg, err := renderBindingConfigWithVariables(server, entry.binding, plan.ref, entry.variables)
		if err != nil {
			return nil, err
		}
		result = append(result, RenderedBindingConfig{
			Binding: entry.binding,
			Config:  cfg,
		})
	}
	return result, nil
}

func RenderSubscriptionProxyList(server *node.Server, bindings []*node.ServerXrayTemplate, selected []*node.ServerXrayTemplate, user UserInfo) ([]RenderedSubscriptionProxy, error) {
	selectedSet := newSelectedBindingSet(selected)
	plan := newRenderBindingPlanWithOptions(server, bindings, RenderBindingConfigOptions{
		TemplateType:            node.XrayTemplateInbound,
		SubscriptionEnabledOnly: true,
		bindingFilter:           selectedSet.Contains,
	})
	if len(plan.entries) == 0 {
		return []RenderedSubscriptionProxy{}, nil
	}
	result := viewutil.NewRowsCollector[RenderedSubscriptionProxy](len(plan.entries))
	for _, entry := range plan.entries {
		cfg, err := renderBindingConfigWithVariables(server, entry.binding, plan.ref, entry.variables)
		if err != nil {
			return nil, err
		}
		proxy, err := renderSubscriptionProxyWithConfigAndVariables(server, entry.binding, plan.ref, user, cfg, entry.variables)
		if err != nil {
			return nil, err
		}
		if proxy == nil {
			continue
		}
		result.Add(RenderedSubscriptionProxy{
			Binding: entry.binding,
			Proxy:   proxy,
		})
	}
	return result.Rows(), nil
}

func RenderConfigBucketsAndSubscriptionProxies(server *node.Server, bindings []*node.ServerXrayTemplate, user UserInfo) (ConfigBuckets, []RenderedSubscriptionProxy, error) {
	plan := newRenderBindingPlan(server, bindings)
	buckets := NewConfigBucketsWithCounts(plan.counts)
	proxies := viewutil.NewRowsCollector[RenderedSubscriptionProxy](plan.counts.Inbounds)
	for _, entry := range plan.entries {
		cfg, err := renderBindingConfigWithVariables(server, entry.binding, plan.ref, entry.variables)
		if err != nil {
			return ConfigBuckets{}, nil, err
		}
		buckets.Add(entry.binding.Template.Type, cfg)
		if entry.binding.Template.Type != node.XrayTemplateInbound {
			continue
		}
		if entry.binding.SubscriptionEnabled != nil && !*entry.binding.SubscriptionEnabled {
			continue
		}
		proxy, err := renderSubscriptionProxyWithConfigAndVariables(server, entry.binding, plan.ref, user, cfg, entry.variables)
		if err != nil {
			return ConfigBuckets{}, nil, err
		}
		if proxy == nil {
			continue
		}
		proxies.Add(RenderedSubscriptionProxy{
			Binding: entry.binding,
			Proxy:   proxy,
		})
	}
	return buckets, proxies.Rows(), nil
}

func renderBindingConfigWithVariables(server *node.Server, binding *node.ServerXrayTemplate, ref Ref, vars ConfigObject) (ConfigObject, error) {
	if binding == nil || binding.Template == nil {
		return nil, nil
	}
	source := binding.Template.Config
	if binding.Template.ConfigTemplate != "" {
		source = binding.Template.ConfigTemplate
	}
	if vars == nil {
		vars = Variables(binding)
	}
	cfg, err := renderJSON(source, Context(server, binding, vars, ref, UserInfo{}))
	if err != nil {
		return nil, err
	}
	data, ok := cfg.(ConfigObject)
	if !ok {
		return nil, fmt.Errorf("xray template config must be a JSON object")
	}
	overridden := applyConfigVariableOverrides(data, vars, binding.Template.Type)
	if err := normalizeAndValidateXrayConfig(overridden, binding.Template.Type); err != nil {
		return nil, err
	}
	return overridden, nil
}

type selectedBindingSet struct {
	rows    []*node.ServerXrayTemplate
	index   map[*node.ServerXrayTemplate]struct{}
	capHint int
}

const smallSelectedBindingSetThreshold = 8

func newSelectedBindingSet(bindings []*node.ServerXrayTemplate) selectedBindingSet {
	set := selectedBindingSet{capHint: len(bindings)}
	for _, binding := range bindings {
		set.Add(binding)
	}
	return set
}

func (s *selectedBindingSet) Add(binding *node.ServerXrayTemplate) {
	if binding == nil {
		return
	}
	if s.index != nil {
		if _, ok := s.index[binding]; ok {
			return
		}
		s.index[binding] = struct{}{}
		s.ensureRows()
		s.rows = append(s.rows, binding)
		return
	}
	for _, existing := range s.rows {
		if existing == binding {
			return
		}
	}
	s.ensureRows()
	s.rows = append(s.rows, binding)
	if len(s.rows) > smallSelectedBindingSetThreshold {
		s.upgradeLookup()
	}
}

func (s *selectedBindingSet) ensureRows() {
	if s.rows != nil {
		return
	}
	s.rows = make([]*node.ServerXrayTemplate, 0, selectedBindingSetInitialCap(s.capHint))
}

func (s *selectedBindingSet) upgradeLookup() {
	if s.index != nil {
		return
	}
	nextCap := renderLookupCap(len(s.rows), s.capHint)
	if nextCap > cap(s.rows) {
		rows := make([]*node.ServerXrayTemplate, len(s.rows), nextCap)
		copy(rows, s.rows)
		s.rows = rows
	}
	s.index = make(map[*node.ServerXrayTemplate]struct{}, nextCap)
	for _, existing := range s.rows {
		s.index[existing] = struct{}{}
	}
}

func renderLookupCap(rowCount, capHint int) int {
	if capHint <= rowCount {
		return rowCount
	}
	if bounded := rowCount * 2; capHint > bounded {
		return bounded
	}
	return capHint
}

func selectedBindingSetInitialCap(capHint int) int {
	if capHint <= 0 {
		return 0
	}
	if capHint > smallSelectedBindingSetThreshold+1 {
		return smallSelectedBindingSetThreshold + 1
	}
	return capHint
}

func (s selectedBindingSet) Contains(binding *node.ServerXrayTemplate) bool {
	if binding == nil {
		return false
	}
	if s.index != nil {
		_, ok := s.index[binding]
		return ok
	}
	for _, existing := range s.rows {
		if existing == binding {
			return true
		}
	}
	return false
}

func ValidateTemplateConfig(template *node.XrayTemplate) error {
	if template == nil {
		return nil
	}
	server := &node.Server{
		Id:      1,
		Name:    "preview",
		Address: "127.0.0.1",
	}
	binding := &node.ServerXrayTemplate{
		Id:        1,
		ServerId:  1,
		Template:  template,
		Alias:     template.Name,
		Variables: "{}",
	}
	return validateTemplateConfigWithServer(server, binding)
}

func validateTemplateConfigWithServer(server *node.Server, binding *node.ServerXrayTemplate) error {
	ref := BuildRef(server, []*node.ServerXrayTemplate{binding})
	_, err := RenderBindingConfig(server, binding, ref)
	return err
}

func RenderSubscriptionProxy(server *node.Server, binding *node.ServerXrayTemplate, ref Ref, user UserInfo) (*adapter.Proxy, error) {
	return renderSubscriptionProxyWithConfigAndVariables(server, binding, ref, user, nil, nil)
}

func renderSubscriptionProxyWithConfigAndVariables(server *node.Server, binding *node.ServerXrayTemplate, ref Ref, user UserInfo, cfg ConfigObject, baseVars ConfigObject) (*adapter.Proxy, error) {
	if binding == nil || binding.Template == nil || binding.Template.Type != node.XrayTemplateInbound {
		return nil, nil
	}
	if binding.SubscriptionEnabled != nil && !*binding.SubscriptionEnabled {
		return nil, nil
	}
	vars := subscriptionVariablesFromBase(binding, baseVars)
	ctx := Context(server, binding, vars, ref, user)

	data := ConfigObject{}
	var err error
	metaSource := binding.Template.SubscriptionMeta
	if metaSource != "" && metaSource != "{}" {
		meta, err := renderJSON(metaSource, ctx)
		if err != nil {
			return nil, err
		}
		if metaData, ok := meta.(ConfigObject); ok {
			merge(data, metaData)
		}
	}
	if cfg == nil {
		cfg, err = renderBindingConfigWithVariables(server, binding, ref, baseVars)
		if err != nil {
			return nil, err
		}
	}
	merge(data, inferSubscriptionMeta(server, cfg, user))
	merge(data, jsonObject(binding.SubscriptionVariables))
	if enabled, ok := data["enabled"].(bool); ok && !enabled {
		return nil, nil
	}

	name := stringValue(data, "name")
	if binding.SubscriptionName != "" {
		name, err = renderString(binding.SubscriptionName, ctx)
		if err != nil {
			return nil, err
		}
	}
	if name == "" {
		alias := binding.Alias
		if alias == "" {
			alias = fmt.Sprintf("%s-%d", binding.Template.Type, binding.Id)
		}
		name = fmt.Sprintf("%s-%s", server.Name, alias)
	}
	proxy := &adapter.Proxy{
		Name:                    name,
		Server:                  firstString(data, "server", server.Address),
		Port:                    uint16Value(data, "port"),
		Type:                    firstString(data, "type", stringValue(vars, "type")),
		Security:                stringValue(data, "security"),
		SNI:                     stringValue(data, "sni"),
		AllowInsecure:           boolValue(data, "allow_insecure"),
		Fingerprint:             stringValue(data, "fingerprint"),
		PinnedPeerCertSha256:    stringValue(data, "pinned_peer_cert_sha256"),
		VerifyPeerCertByName:    stringValue(data, "verify_peer_cert_by_name"),
		RealityPublicKey:        firstString(data, "public_key", stringValue(data, "reality_public_key")),
		RealityShortId:          firstString(data, "short_id", stringValue(data, "reality_short_id")),
		Transport:               stringValue(data, "transport"),
		Host:                    stringValue(data, "host"),
		Path:                    stringValue(data, "path"),
		ServiceName:             stringValue(data, "service_name"),
		GrpcAuthority:           stringValue(data, "grpc_authority"),
		GrpcMultiMode:           boolValue(data, "grpc_multi_mode"),
		GrpcUserAgent:           stringValue(data, "grpc_user_agent"),
		GrpcIdleTimeout:         intValue(data, "grpc_idle_timeout"),
		GrpcHealthCheckTimeout:  intValue(data, "grpc_health_check_timeout"),
		GrpcPermitWithoutStream: boolValue(data, "grpc_permit_without_stream"),
		GrpcInitialWindowsSize:  intValue(data, "grpc_initial_windows_size"),
		Method:                  stringValue(data, "method"),
		ServerKey:               stringValue(data, "server_key"),
		Username:                stringValue(data, "username"),
		HeaderType:              stringValue(data, "header_type"),
		Obfs:                    stringValue(data, "obfs"),
		ObfsHost:                stringValue(data, "obfs_host"),
		ObfsPath:                stringValue(data, "obfs_path"),
		TLS:                     boolValue(data, "tls"),
		EchEnable:               boolValue(data, "ech_enable"),
		EchServerName:           stringValue(data, "ech_server_name"),
		Flow:                    stringValue(data, "flow"),
		HopPorts:                stringValue(data, "hop_ports"),
		HopInterval:             intValue(data, "hop_interval"),
		ObfsPassword:            stringValue(data, "obfs_password"),
		UpMbps:                  intValue(data, "up_mbps"),
		DownMbps:                intValue(data, "down_mbps"),
		HysteriaUDPIdleTimeout:  firstInt(data, "hysteria_udp_idle_timeout", "udp_idle_timeout"),
		HysteriaMasquerade:      stringOrJSONValue(data["hysteria_masquerade"]),
		DisableSNI:              boolValue(data, "disable_sni"),
		ReduceRtt:               boolValue(data, "reduce_rtt"),
		UDPRelayMode:            stringValue(data, "udp_relay_mode"),
		CongestionController:    stringValue(data, "congestion_controller"),
		XhttpMode:               stringValue(data, "xhttp_mode"),
		XhttpExtra:              stringOrJSONValue(data["xhttp_extra"]),
		Encryption:              stringValue(data, "encryption"),
		EncryptionMode:          firstString(data, "encryption_mode", stringValue(data, "encryptionMode")),
		EncryptionRtt:           stringValue(data, "encryption_rtt"),
		EncryptionClientPadding: stringValue(data, "encryption_client_padding"),
		EncryptionPassword:      stringValue(data, "encryption_password"),
		RealityServerAddr:       stringValue(data, "reality_server_addr"),
		RealityServerPort:       intValue(data, "reality_server_port"),
	}
	if proxy.Type == "" {
		proxy.Type = "vless"
	}
	if strings.EqualFold(strings.TrimSpace(proxy.Type), "vless") &&
		strings.EqualFold(strings.TrimSpace(proxy.Security), "reality") &&
		strings.EqualFold(strings.TrimSpace(proxy.Transport), "grpc") {
		proxy.Flow = "none"
	}
	if proxy.Port == 0 {
		proxy.Port = uint16Value(vars, "port")
	}
	if proxy.Server == "" {
		proxy.Server = server.Address
	}
	if proxy.Method == "" {
		proxy.Method = stringValue(vars, "method")
	}
	if proxy.TLS || proxy.Security == "tls" || proxy.Security == "reality" {
		proxy.TLS = true
	}
	if proxy.EncryptionMode == "" {
		proxy.EncryptionMode = stringValue(vars, "encryption_mode")
	}
	proxy.Encryption_Mode = proxy.EncryptionMode
	proxy.ServerKey = firstString(ConfigObject{"server_key": proxy.ServerKey}, "server_key", stringValue(vars, "server_key"))
	if binding.Alias != "" {
		proxy.Tags = []string{binding.Alias}
	}
	proxy.RealityPrivateKey = stringValue(vars, "reality_private_key")
	if proxy.RealityPublicKey == "" {
		proxy.RealityPublicKey = stringValue(vars, "reality_public_key")
	}
	if proxy.RealityPublicKey == "" && proxy.RealityPrivateKey != "" {
		proxy.RealityPublicKey = deriveRealityPublicKey(proxy.RealityPrivateKey)
	}
	if proxy.RealityShortId == "" {
		proxy.RealityShortId = stringValue(vars, "reality_short_id")
	}
	return proxy, nil
}

func subscriptionVariablesFromBase(binding *node.ServerXrayTemplate, baseVars ConfigObject) ConfigObject {
	if baseVars == nil {
		return SubscriptionVariables(binding)
	}
	vars := cloneConfigObject(baseVars)
	if binding != nil {
		merge(vars, jsonObject(binding.SubscriptionVariables))
	}
	return vars
}

func deriveRealityPublicKey(privateKey string) string {
	publicKey, _, err := tool.Curve25519Genkey(false, privateKey)
	if err != nil {
		return ""
	}
	return publicKey
}

func inferSubscriptionMeta(server *node.Server, data ConfigObject, user UserInfo) ConfigObject {
	if data == nil {
		return ConfigObject{}
	}
	meta := ConfigObject{
		"server": server.Address,
		"port":   data["port"],
		"type":   data["protocol"],
	}
	protocol := stringValue(data, "protocol")
	if protocol == "vless" || protocol == "vmess" || protocol == "tuic" {
		meta["server_key"] = user.UUID
	}
	if protocol == "hysteria" {
		meta["type"] = "hysteria2"
		meta["password"] = firstNonEmpty(user.Password, user.UUID)
		meta["server_key"] = firstNonEmpty(user.Password, user.UUID)
	}
	if settings, ok := data["settings"].(ConfigObject); ok {
		if method := stringValue(settings, "method"); method != "" {
			meta["method"] = method
		}
		if password := stringValue(settings, "password"); password != "" {
			meta["password"] = password
		}
		if key := firstString(settings, "serverKey", stringValue(settings, "server_key")); key != "" {
			meta["server_key"] = key
		}
		if flow := stringValue(settings, "flow"); flow != "" {
			meta["flow"] = flow
		}
		if accounts, ok := settings["accounts"].([]any); ok && len(accounts) > 0 {
			if account, ok := accounts[0].(ConfigObject); ok {
				if user := stringValue(account, "user"); user != "" {
					meta["username"] = user
				}
			}
		}
		if clients, ok := settings[userListField(protocol)].([]any); ok && len(clients) > 0 {
			if client, ok := clients[0].(ConfigObject); ok {
				if flow := stringValue(client, "flow"); flow != "" {
					meta["flow"] = flow
				}
				if method := stringValue(client, "method"); method != "" {
					meta["method"] = method
				}
			}
		}
	}
	stream, ok := data["streamSettings"].(ConfigObject)
	if !ok {
		return meta
	}
	if network := stringValue(stream, "network"); network != "" {
		meta["transport"] = network
	}
	if security := stringValue(stream, "security"); security != "" {
		meta["security"] = security
		meta["tls"] = security == "tls" || security == "reality"
	}
	if tls, ok := stream["tlsSettings"].(ConfigObject); ok {
		if serverName := stringValue(tls, "serverName"); serverName != "" {
			meta["sni"] = serverName
		}
		if fingerprint := stringValue(tls, "fingerprint"); fingerprint != "" {
			meta["fingerprint"] = fingerprint
		}
		if allowInsecure, ok := tls["allowInsecure"].(bool); ok {
			meta["allow_insecure"] = allowInsecure
		}
		if pinned := stringValue(tls, "pinnedPeerCertSha256"); pinned != "" {
			meta["pinned_peer_cert_sha256"] = pinned
		}
		if verifyName := stringValue(tls, "verifyPeerCertByName"); verifyName != "" {
			meta["verify_peer_cert_by_name"] = verifyName
		}
		if ech, ok := tls["echSettings"].(ConfigObject); ok {
			if enabled, ok := ech["enable"].(bool); ok {
				meta["ech_enable"] = enabled
			}
			if serverName := stringValue(ech, "serverName"); serverName != "" {
				meta["ech_server_name"] = serverName
			}
		}
	}
	if reality, ok := stream["realitySettings"].(ConfigObject); ok {
		if password := firstString(reality, "password", stringValue(reality, "publicKey")); password != "" {
			meta["public_key"] = password
			meta["reality_public_key"] = password
		}
		if shortID := stringValue(reality, "shortId"); shortID != "" {
			meta["short_id"] = shortID
			meta["reality_short_id"] = shortID
		}
		if serverNames, ok := reality["serverNames"].([]any); ok && len(serverNames) > 0 {
			if value, ok := serverNames[0].(string); ok {
				meta["sni"] = value
			}
		}
		if shortIds, ok := reality["shortIds"].([]any); ok && len(shortIds) > 0 {
			if value, ok := shortIds[0].(string); ok {
				meta["short_id"] = value
			}
		}
	}
	if ws, ok := stream["wsSettings"].(ConfigObject); ok {
		if path := stringValue(ws, "path"); path != "" {
			meta["path"] = path
		}
		if headers, ok := ws["headers"].(ConfigObject); ok {
			if host := stringValue(headers, "Host"); host != "" {
				meta["host"] = host
			}
		}
	}
	if raw, ok := stream["rawSettings"].(ConfigObject); ok {
		if header, ok := raw["header"].(ConfigObject); ok {
			if typ := stringValue(header, "type"); typ != "" {
				meta["header_type"] = typ
			}
			if request, ok := header["request"].(ConfigObject); ok {
				if path := firstStringFromList(request, "path"); path != "" {
					meta["obfs_path"] = path
				}
				if headers, ok := request["headers"].(ConfigObject); ok {
					if host := firstStringFromList(headers, "Host"); host != "" {
						meta["obfs_host"] = host
					}
				}
			}
		}
	}
	if kcp, ok := stream["kcpSettings"].(ConfigObject); ok {
		if header, ok := kcp["header"].(ConfigObject); ok {
			if typ := stringValue(header, "type"); typ != "" {
				meta["header_type"] = typ
				meta["obfs"] = typ
			}
		}
	}
	if grpc, ok := stream["grpcSettings"].(ConfigObject); ok {
		if serviceName := stringValue(grpc, "serviceName"); serviceName != "" {
			meta["service_name"] = serviceName
		}
		if authority := stringValue(grpc, "authority"); authority != "" {
			meta["grpc_authority"] = authority
		}
		if multiMode, ok := grpc["multiMode"].(bool); ok {
			meta["grpc_multi_mode"] = multiMode
		}
		if userAgent := stringValue(grpc, "user_agent"); userAgent != "" {
			meta["grpc_user_agent"] = userAgent
		}
		if idleTimeout := intValue(grpc, "idle_timeout"); idleTimeout != 0 {
			meta["grpc_idle_timeout"] = idleTimeout
		}
		if healthCheckTimeout := intValue(grpc, "health_check_timeout"); healthCheckTimeout != 0 {
			meta["grpc_health_check_timeout"] = healthCheckTimeout
		}
		if permitWithoutStream, ok := grpc["permit_without_stream"].(bool); ok {
			meta["grpc_permit_without_stream"] = permitWithoutStream
		}
		if initialWindowsSize := intValue(grpc, "initial_windows_size"); initialWindowsSize != 0 {
			meta["grpc_initial_windows_size"] = initialWindowsSize
		}
	}
	if hysteria, ok := stream["hysteriaSettings"].(ConfigObject); ok {
		if auth := stringValue(hysteria, "auth"); auth != "" {
			meta["password"] = auth
		}
		if version := intValue(hysteria, "version"); version != 0 {
			meta["hysteria_version"] = version
		}
		if udpIdleTimeout := intValue(hysteria, "udpIdleTimeout"); udpIdleTimeout != 0 {
			meta["udp_idle_timeout"] = udpIdleTimeout
			meta["hysteria_udp_idle_timeout"] = udpIdleTimeout
		}
		if masquerade, ok := hysteria["masquerade"]; ok && masquerade != nil {
			meta["hysteria_masquerade"] = masquerade
		}
	}
	if xhttp, ok := stream["xhttpSettings"].(ConfigObject); ok {
		if host := stringValue(xhttp, "host"); host != "" {
			meta["host"] = host
		}
		if path := stringValue(xhttp, "path"); path != "" {
			meta["path"] = path
		}
		if mode := stringValue(xhttp, "mode"); mode != "" {
			meta["xhttp_mode"] = mode
		}
		if extra, ok := xhttp["extra"]; ok && extra != nil {
			meta["xhttp_extra"] = extra
		}
	}
	if upgrade, ok := stream["httpupgradeSettings"].(ConfigObject); ok {
		if host := stringValue(upgrade, "host"); host != "" {
			meta["host"] = host
		}
		if path := stringValue(upgrade, "path"); path != "" {
			meta["path"] = path
		}
	}
	return meta
}

func Context(server *node.Server, binding *node.ServerXrayTemplate, vars ConfigObject, ref Ref, user UserInfo) ConfigObject {
	serverData := ConfigObject{
		"id":      server.Id,
		"name":    server.Name,
		"address": server.Address,
		"country": server.Country,
		"city":    server.City,
	}
	templateData := ConfigObject{
		"id":   binding.Template.Id,
		"name": binding.Template.Name,
		"type": binding.Template.Type,
	}
	bindingData := ConfigObject{
		"id":    binding.Id,
		"alias": binding.Alias,
		"sort":  binding.Sort,
	}
	return ConfigObject{
		"Server":    serverData,
		"server":    serverData,
		"Template":  templateData,
		"template":  templateData,
		"Binding":   bindingData,
		"binding":   bindingData,
		"Vars":      vars,
		"vars":      vars,
		"Ref":       ref,
		"ref":       ref,
		"UserInfo":  user,
		"user_info": user,
	}
}

func renderJSON(source string, data any) (any, error) {
	rendered, err := renderString(source, data)
	if err != nil {
		return nil, err
	}
	if rendered == "" {
		rendered = "{}"
	}
	var out any
	if err := json.Unmarshal([]byte(rendered), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func applyConfigVariableOverrides(data ConfigObject, vars ConfigObject, templateType string) ConfigObject {
	if len(vars) == 0 {
		return data
	}
	setIfVar(data, vars, "tag", "tag")
	if templateType == node.XrayTemplateInbound {
		setIfVar(data, vars, "listen", "listen")
		setIfVar(data, vars, "port", "port")
	}
	setIfVar(data, vars, "protocol", "protocol")
	setIfVar(data, vars, "sendThrough", "sendThrough")

	switch templateType {
	case node.XrayTemplateDNS:
		applyDNSVariableOverrides(data, vars)
		return data
	case node.XrayTemplateRouting:
		applyRoutingVariableOverrides(data, vars)
		return data
	case node.XrayTemplateGeodata:
		applyGeodataVariableOverrides(data, vars)
		return data
	}

	if settings, ok := data["settings"].(ConfigObject); ok {
		if templateType == node.XrayTemplateInbound {
			normalizeInboundUserList(settings, stringValue(data, "protocol"))
		}
		if raw, ok := vars["settings"].(ConfigObject); ok {
			merge(settings, raw)
			if templateType == node.XrayTemplateInbound {
				normalizeInboundUserList(settings, stringValue(data, "protocol"))
			}
		}
		setIfVar(settings, vars, "address", "address")
		setIfVar(settings, vars, "out_address", "address")
		if templateType == node.XrayTemplateOutbound {
			setIfVar(settings, vars, "port", "port")
		}
		setIfVar(settings, vars, "out_port", "port")
		setIfVar(settings, vars, "dokodemo_address", "address")
		setIfVar(settings, vars, "dokodemo_port", "port")
		setIfVar(settings, vars, "method", "method")
		setIfVar(settings, vars, "ss_method", "method")
		setIfVar(settings, vars, "password", "password")
		setIfVar(settings, vars, "out_password", "password")
		setIfVar(settings, vars, "ss_password", "password")
		setIfVar(settings, vars, "out_user", "user")
		setIfVar(settings, vars, "out_pass", "pass")
		setIfVar(settings, vars, "id", "id")
		setIfVar(settings, vars, "out_id", "id")
		setIfVar(settings, vars, "out_encryption", "encryption")
		setIfVar(settings, vars, "out_security", "security")
		if templateType == node.XrayTemplateInbound && stringValue(data, "protocol") == "vless" {
			setFirstClientIfVar(settings, vars, "flow", "flow")
			delete(settings, "flow")
		} else {
			setIfVar(settings, vars, "flow", "flow")
		}
		setIfVar(settings, vars, "out_flow", "flow")
		setIfVar(settings, vars, "hysteria_version", "version")
	}

	if raw, ok := vars["streamSettings"].(ConfigObject); ok {
		stream := ensureMap(data, "streamSettings")
		merge(stream, raw)
	}
	stream, ok := data["streamSettings"].(ConfigObject)
	if !ok {
		return data
	}
	setIfVar(stream, vars, "network", "network")
	setIfVar(stream, vars, "transport", "network")
	setIfVar(stream, vars, "security", "security")

	if stringValue(stream, "security") == "tls" {
		tls := ensureMap(stream, "tlsSettings")
		setIfVar(tls, vars, "sni", "serverName")
		setIfVar(tls, vars, "server_name", "serverName")
		setIfVar(tls, vars, "fingerprint", "fingerprint")
		setIfVar(tls, vars, "pinned_peer_cert_sha256", "pinnedPeerCertSha256")
		setIfVar(tls, vars, "verify_peer_cert_by_name", "verifyPeerCertByName")
	}
	if stringValue(stream, "security") == "reality" {
		reality := ensureMap(stream, "realitySettings")
		if templateType == node.XrayTemplateInbound {
			setIfVar(reality, vars, "reality_target", "target")
			setIfVar(reality, vars, "reality_private_key", "privateKey")
			setStringListIfVar(reality, vars, "reality_server_names", "serverNames")
			setStringListIfVar(reality, vars, "sni", "serverNames")
			setStringListIfVar(reality, vars, "server_name", "serverNames")
			setStringListIfVar(reality, vars, "reality_short_ids", "shortIds")
			setStringListIfVar(reality, vars, "reality_short_id", "shortIds")
		}
		if templateType == node.XrayTemplateOutbound {
			setIfVar(reality, vars, "reality_public_key", "password")
			setIfVar(reality, vars, "reality_short_id", "shortId")
			setIfVar(reality, vars, "sni", "serverName")
			setIfVar(reality, vars, "server_name", "serverName")
		}
	}

	switch stringValue(stream, "network") {
	case "ws":
		ws := ensureMap(stream, "wsSettings")
		setIfVar(ws, vars, "path", "path")
		if value, ok := lookupVar(vars, "host"); ok {
			headers := ensureMap(ws, "headers")
			headers["Host"] = value
		}
	case "grpc":
		grpc := ensureMap(stream, "grpcSettings")
		setIfVar(grpc, vars, "service_name", "serviceName")
		setIfVar(grpc, vars, "grpc_service_name", "serviceName")
		setIfVar(grpc, vars, "grpc_authority", "authority")
		setIfVar(grpc, vars, "grpc_multi_mode", "multiMode")
		setIfVar(grpc, vars, "grpc_user_agent", "user_agent")
		setIfVar(grpc, vars, "grpc_idle_timeout", "idle_timeout")
		setIfVar(grpc, vars, "grpc_health_check_timeout", "health_check_timeout")
		setIfVar(grpc, vars, "grpc_permit_without_stream", "permit_without_stream")
		setIfVar(grpc, vars, "grpc_initial_windows_size", "initial_windows_size")
	case "xhttp":
		xhttp := ensureMap(stream, "xhttpSettings")
		setIfVar(xhttp, vars, "host", "host")
		setIfVar(xhttp, vars, "path", "path")
		setIfVar(xhttp, vars, "xhttp_mode", "mode")
		setJSONIfVar(xhttp, vars, "xhttp_extra", "extra")
	case "hysteria":
		hysteria := ensureMap(stream, "hysteriaSettings")
		setIfVar(hysteria, vars, "hysteria_version", "version")
		setIfVar(hysteria, vars, "udp_idle_timeout", "udpIdleTimeout")
		setJSONIfVar(hysteria, vars, "hysteria_masquerade", "masquerade")
	case "httpupgrade":
		upgrade := ensureMap(stream, "httpupgradeSettings")
		setIfVar(upgrade, vars, "host", "host")
		setIfVar(upgrade, vars, "path", "path")
	}
	return data
}

func normalizeVlessFlowForRuntime(data ConfigObject, templateType string) {
	if stringValue(data, "protocol") != "vless" {
		return
	}
	settings, ok := data["settings"].(ConfigObject)
	if !ok {
		return
	}
	stream, _ := data["streamSettings"].(ConfigObject)
	grpcReality := strings.EqualFold(strings.TrimSpace(stringValue(stream, "network")), "grpc") &&
		strings.EqualFold(strings.TrimSpace(stringValue(stream, "security")), "reality")
	flowDisabled := func(user ConfigObject) bool {
		return grpcReality || strings.EqualFold(strings.TrimSpace(stringValue(user, "flow")), "none")
	}

	if templateType == node.XrayTemplateInbound {
		delete(settings, "flow")
		for _, field := range []string{"clients", "users"} {
			switch users := settings[field].(type) {
			case []any:
				for _, item := range users {
					if user, ok := item.(ConfigObject); ok && flowDisabled(user) {
						delete(user, "flow")
					}
				}
			case []ConfigObject:
				for _, user := range users {
					if flowDisabled(user) {
						delete(user, "flow")
					}
				}
			}
		}
		return
	}
	if templateType == node.XrayTemplateOutbound &&
		(grpcReality || strings.EqualFold(strings.TrimSpace(stringValue(settings, "flow")), "none")) {
		delete(settings, "flow")
	}
}

func normalizeAndValidateXrayConfig(data ConfigObject, templateType string) error {
	if data == nil {
		return fmt.Errorf("xray template config must be a JSON object")
	}
	if templateType == node.XrayTemplateRouting {
		return validateRoutingConfig(data)
	}
	if templateType == node.XrayTemplateInbound || templateType == node.XrayTemplateOutbound {
		if err := validateHysteriaConfig(data); err != nil {
			return err
		}
	}
	normalizeVlessFlowForRuntime(data, templateType)
	if templateType != node.XrayTemplateInbound {
		return nil
	}
	if err := validateHysteriaServerCertificate(data); err != nil {
		return err
	}
	stream, ok := data["streamSettings"].(ConfigObject)
	if !ok || stringValue(stream, "security") != "reality" {
		return nil
	}
	reality, ok := stream["realitySettings"].(ConfigObject)
	if !ok {
		return fmt.Errorf("inbound reality requires streamSettings.realitySettings")
	}
	if stringValue(reality, "target") == "" && stringValue(reality, "dest") == "" {
		return fmt.Errorf("inbound reality requires realitySettings.target, for example ebay.com:443")
	}
	if stringValue(reality, "privateKey") == "" {
		return fmt.Errorf("inbound reality requires realitySettings.privateKey")
	}
	if len(stringListValue(reality["serverNames"])) == 0 && stringValue(reality, "serverName") != "" {
		reality["serverNames"] = []any{stringValue(reality, "serverName")}
		delete(reality, "serverName")
	}
	if len(stringListValue(reality["serverNames"])) == 0 {
		return fmt.Errorf("inbound reality requires realitySettings.serverNames")
	}
	if len(stringListValue(reality["shortIds"])) == 0 && stringValue(reality, "shortId") != "" {
		reality["shortIds"] = []any{stringValue(reality, "shortId")}
		delete(reality, "shortId")
	}
	if len(stringListValue(reality["shortIds"])) == 0 {
		return fmt.Errorf("inbound reality requires realitySettings.shortIds")
	}
	delete(reality, "password")
	delete(reality, "publicKey")
	return nil
}

func validateHysteriaConfig(data ConfigObject) error {
	protocolHysteria := stringValue(data, "protocol") == "hysteria"
	if protocolHysteria {
		settings, ok := data["settings"].(ConfigObject)
		if !ok {
			return fmt.Errorf("hysteria requires settings")
		}
		if intValue(settings, "version") != 2 {
			return fmt.Errorf("hysteria settings.version must be 2")
		}
	}
	stream, ok := data["streamSettings"].(ConfigObject)
	if !ok {
		if protocolHysteria {
			return fmt.Errorf("hysteria requires streamSettings")
		}
		return nil
	}
	if network := stringValue(stream, "network"); network != "hysteria" {
		if !protocolHysteria {
			return nil
		}
		return fmt.Errorf("hysteria requires streamSettings.network to be hysteria")
	}
	if security := stringValue(stream, "security"); security != "tls" {
		return fmt.Errorf("hysteria transport requires streamSettings.security to be tls")
	}
	hysteria, ok := stream["hysteriaSettings"].(ConfigObject)
	if !ok {
		return fmt.Errorf("hysteria requires streamSettings.hysteriaSettings")
	}
	if intValue(hysteria, "version") != 2 {
		return fmt.Errorf("hysteria streamSettings.hysteriaSettings.version must be 2")
	}
	return nil
}

func validateHysteriaServerCertificate(data ConfigObject) error {
	stream, ok := data["streamSettings"].(ConfigObject)
	if !ok || stringValue(stream, "network") != "hysteria" {
		return nil
	}
	tls, _ := stream["tlsSettings"].(ConfigObject)
	certificates, _ := tls["certificates"].([]any)
	hasLines := func(value any) bool {
		for _, line := range stringListValue(value) {
			if strings.TrimSpace(line) != "" {
				return true
			}
		}
		return false
	}
	for _, item := range certificates {
		certificate, ok := item.(ConfigObject)
		if !ok {
			continue
		}
		usage := strings.TrimSpace(stringValue(certificate, "usage"))
		if usage != "" && usage != "encipherment" && usage != "issue" {
			continue
		}
		hasCertificate := strings.TrimSpace(stringValue(certificate, "certificateFile")) != "" || hasLines(certificate["certificate"])
		hasKey := strings.TrimSpace(stringValue(certificate, "keyFile")) != "" || hasLines(certificate["key"])
		if hasCertificate && hasKey {
			return nil
		}
	}
	return fmt.Errorf("hysteria inbound requires a TLS server certificate and private key in tlsSettings.certificates")
}

func stringListValue(value any) []string {
	switch typed := value.(type) {
	case []any:
		result := viewutil.NewRowsCollector[string](len(typed))
		for _, item := range typed {
			if text := fmt.Sprint(item); text != "" {
				result.Add(text)
			}
		}
		return result.Rows()
	case []string:
		result := viewutil.NewRowsCollector[string](len(typed))
		for _, item := range typed {
			if item != "" {
				result.Add(item)
			}
		}
		return result.Rows()
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	default:
		if value == nil {
			return nil
		}
		text := fmt.Sprint(value)
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func ensureMap(data ConfigObject, key string) ConfigObject {
	if value, ok := data[key].(ConfigObject); ok {
		return value
	}
	value := ConfigObject{}
	data[key] = value
	return value
}

func lookupVar(vars ConfigObject, keys ...string) (any, bool) {
	for _, key := range keys {
		value, ok := vars[key]
		if ok && value != nil && value != "" {
			return value, true
		}
	}
	return nil, false
}

func setIfVar(data ConfigObject, vars ConfigObject, varKey string, configKey string) {
	if value, ok := lookupVar(vars, varKey); ok {
		data[configKey] = value
	}
}

func setFirstClientIfVar(settings ConfigObject, vars ConfigObject, varKey string, configKey string) {
	value, ok := lookupVar(vars, varKey)
	if !ok {
		return
	}
	clients, _ := settings["clients"].([]any)
	if len(clients) == 0 {
		clients = []any{ConfigObject{}}
	}
	client, _ := clients[0].(ConfigObject)
	if client == nil {
		client = ConfigObject{}
	}
	client[configKey] = value
	clients[0] = client
	settings["clients"] = clients
}

func normalizeInboundUserList(settings ConfigObject, protocol string) {
	field := userListField(protocol)
	if field == "users" {
		if _, ok := settings["users"]; !ok {
			if clients, ok := settings["clients"]; ok {
				settings["users"] = clients
			}
		}
		delete(settings, "clients")
		return
	}
	if _, ok := settings["clients"]; !ok {
		if users, ok := settings["users"]; ok {
			settings["clients"] = users
		}
	}
	delete(settings, "users")
}

func userListField(protocol string) string {
	switch protocol {
	case "vless", "vmess", "trojan", "shadowsocks":
		return "clients"
	default:
		return "users"
	}
}

func setJSONIfVar(data ConfigObject, vars ConfigObject, varKey string, configKey string) {
	value, ok := lookupVar(vars, varKey)
	if !ok {
		return
	}
	if text, ok := value.(string); ok {
		var decoded any
		if err := json.Unmarshal([]byte(text), &decoded); err == nil {
			data[configKey] = decoded
			return
		}
	}
	data[configKey] = value
}

func setStringListIfVar(data ConfigObject, vars ConfigObject, varKey string, configKey string) {
	value, ok := lookupVar(vars, varKey)
	if !ok {
		return
	}
	switch v := value.(type) {
	case []any:
		data[configKey] = v
	case []string:
		list := viewutil.NewRowsCollector[any](len(v))
		for _, item := range v {
			if item != "" {
				list.Add(item)
			}
		}
		data[configKey] = list.Rows()
	case string:
		if v == "" {
			return
		}
		data[configKey] = []any{v}
	default:
		data[configKey] = []any{v}
	}
}

func extractSettingsVariables(vars ConfigObject, settings ConfigObject, protocol string, templateType string) {
	if templateType == node.XrayTemplateOutbound {
		copyKey(vars, settings, "out_address", "address")
		copyKey(vars, settings, "out_port", "port")
		copyKey(vars, settings, "out_id", "id")
		copyKey(vars, settings, "out_encryption", "encryption")
		copyKey(vars, settings, "out_security", "security")
		copyKey(vars, settings, "out_flow", "flow")
		copyKey(vars, settings, "out_password", "password")
		copyKey(vars, settings, "out_user", "user")
		copyKey(vars, settings, "out_pass", "pass")
		if protocol == "shadowsocks" {
			copyKey(vars, settings, "ss_method", "method")
			copyKey(vars, settings, "ss_password", "password")
		}
		copyKey(vars, settings, "hysteria_version", "version")
		return
	}
	if protocol == "dokodemo-door" {
		copyKey(vars, settings, "dokodemo_address", "address")
		copyKey(vars, settings, "dokodemo_port", "port")
	} else {
		copyKey(vars, settings, "address", "address")
	}
	if protocol == "shadowsocks" {
		copyKey(vars, settings, "ss_method", "method")
		copyKey(vars, settings, "ss_password", "password")
	} else {
		copyKey(vars, settings, "method", "method")
		copyKey(vars, settings, "password", "password")
	}
	copyKey(vars, settings, "id", "id")
	copyKey(vars, settings, "flow", "flow")
	if protocol == "vless" {
		copyFirstClientKey(vars, settings, "flow", "flow")
	}
	copyKey(vars, settings, "hysteria_version", "version")
}

func copyFirstClientKey(vars ConfigObject, settings ConfigObject, dstKey string, srcKey string) {
	clients, ok := settings["clients"].([]any)
	if !ok || len(clients) == 0 {
		return
	}
	client, ok := clients[0].(ConfigObject)
	if !ok {
		return
	}
	copyKey(vars, client, dstKey, srcKey)
}

func extractDNSVariables(vars ConfigObject, cfg ConfigObject) {
	copyKey(vars, cfg, "dns_servers", "servers")
	copyKey(vars, cfg, "hosts", "hosts")
	copyKey(vars, cfg, "query_strategy", "queryStrategy")
	copyKey(vars, cfg, "disable_cache", "disableCache")
	copyKey(vars, cfg, "serve_stale", "serveStale")
	copyKey(vars, cfg, "serve_expired_ttl", "serveExpiredTTL")
	copyKey(vars, cfg, "disable_fallback", "disableFallback")
	copyKey(vars, cfg, "disable_fallback_if_match", "disableFallbackIfMatch")
	copyKey(vars, cfg, "enable_parallel_query", "enableParallelQuery")
	copyKey(vars, cfg, "use_system_hosts", "useSystemHosts")
	copyKey(vars, cfg, "client_ip", "clientIp")
}

func extractRoutingVariables(vars ConfigObject, cfg ConfigObject) {
	copyKey(vars, cfg, "routing_domain_strategy", "domainStrategy")
	copyKey(vars, cfg, "routing_rules", "rules")
	copyKey(vars, cfg, "routing_balancers", "balancers")
}

func extractGeodataVariables(vars ConfigObject, cfg ConfigObject) {
	copyKey(vars, cfg, "geodata_cron", "cron")
	copyKey(vars, cfg, "geodata_outbound", "outbound")
	copyKey(vars, cfg, "geodata_assets", "assets")
}

func extractStreamVariables(vars ConfigObject, stream ConfigObject, templateType string) {
	copyKey(vars, stream, "network", "network")
	copyKey(vars, stream, "transport", "network")
	copyKey(vars, stream, "security", "security")

	if tls, ok := stream["tlsSettings"].(ConfigObject); ok {
		copyKey(vars, tls, "sni", "serverName")
		copyKey(vars, tls, "server_name", "serverName")
		copyKey(vars, tls, "fingerprint", "fingerprint")
		copyKey(vars, tls, "allow_insecure", "allowInsecure")
		copyKey(vars, tls, "pinned_peer_cert_sha256", "pinnedPeerCertSha256")
		copyKey(vars, tls, "verify_peer_cert_by_name", "verifyPeerCertByName")
	}
	if reality, ok := stream["realitySettings"].(ConfigObject); ok {
		if templateType == node.XrayTemplateInbound {
			copyKey(vars, reality, "reality_target", "target")
			copyKey(vars, reality, "reality_private_key", "privateKey")
			copyKey(vars, reality, "reality_server_names", "serverNames")
			copyKey(vars, reality, "sni", "serverNames")
			copyKey(vars, reality, "server_name", "serverNames")
			copyKey(vars, reality, "reality_short_ids", "shortIds")
			copyKey(vars, reality, "reality_short_id", "shortIds")
		}
		if templateType == node.XrayTemplateOutbound {
			copyKey(vars, reality, "reality_public_key", "password")
			copyKey(vars, reality, "reality_short_id", "shortId")
			copyKey(vars, reality, "sni", "serverName")
			copyKey(vars, reality, "server_name", "serverName")
		}
	}

	switch stringValue(stream, "network") {
	case "ws":
		if ws, ok := stream["wsSettings"].(ConfigObject); ok {
			copyKey(vars, ws, "path", "path")
			if headers, ok := ws["headers"].(ConfigObject); ok {
				copyKey(vars, headers, "host", "Host")
			}
		}
	case "grpc":
		if grpc, ok := stream["grpcSettings"].(ConfigObject); ok {
			copyKey(vars, grpc, "service_name", "serviceName")
			copyKey(vars, grpc, "grpc_service_name", "serviceName")
			copyKey(vars, grpc, "grpc_authority", "authority")
			copyKey(vars, grpc, "grpc_multi_mode", "multiMode")
			copyKey(vars, grpc, "grpc_user_agent", "user_agent")
			copyKey(vars, grpc, "grpc_idle_timeout", "idle_timeout")
			copyKey(vars, grpc, "grpc_health_check_timeout", "health_check_timeout")
			copyKey(vars, grpc, "grpc_permit_without_stream", "permit_without_stream")
			copyKey(vars, grpc, "grpc_initial_windows_size", "initial_windows_size")
		}
	case "xhttp":
		if xhttp, ok := stream["xhttpSettings"].(ConfigObject); ok {
			copyKey(vars, xhttp, "host", "host")
			copyKey(vars, xhttp, "path", "path")
			copyKey(vars, xhttp, "xhttp_mode", "mode")
			copyKey(vars, xhttp, "xhttp_extra", "extra")
		}
	case "hysteria":
		if hysteria, ok := stream["hysteriaSettings"].(ConfigObject); ok {
			copyKey(vars, hysteria, "hysteria_version", "version")
			copyKey(vars, hysteria, "udp_idle_timeout", "udpIdleTimeout")
			copyKey(vars, hysteria, "hysteria_masquerade", "masquerade")
		}
	case "httpupgrade":
		if upgrade, ok := stream["httpupgradeSettings"].(ConfigObject); ok {
			copyKey(vars, upgrade, "host", "host")
			copyKey(vars, upgrade, "path", "path")
		}
	}
}

func applyDNSVariableOverrides(data ConfigObject, vars ConfigObject) {
	setJSONIfVar(data, vars, "dns_servers", "servers")
	setJSONIfVar(data, vars, "servers", "servers")
	setJSONIfVar(data, vars, "hosts", "hosts")
	setIfVar(data, vars, "query_strategy", "queryStrategy")
	setIfVar(data, vars, "disable_cache", "disableCache")
	setIfVar(data, vars, "serve_stale", "serveStale")
	setIfVar(data, vars, "serve_expired_ttl", "serveExpiredTTL")
	setIfVar(data, vars, "disable_fallback", "disableFallback")
	setIfVar(data, vars, "disable_fallback_if_match", "disableFallbackIfMatch")
	setIfVar(data, vars, "enable_parallel_query", "enableParallelQuery")
	setIfVar(data, vars, "use_system_hosts", "useSystemHosts")
	setIfVar(data, vars, "client_ip", "clientIp")
}

func applyRoutingVariableOverrides(data ConfigObject, vars ConfigObject) {
	setIfVar(data, vars, "routing_domain_strategy", "domainStrategy")
	setJSONIfVar(data, vars, "routing_rules", "rules")
	setJSONIfVar(data, vars, "rules", "rules")
	setJSONIfVar(data, vars, "routing_balancers", "balancers")
	setJSONIfVar(data, vars, "balancers", "balancers")
}

func applyGeodataVariableOverrides(data ConfigObject, vars ConfigObject) {
	setIfVar(data, vars, "geodata_cron", "cron")
	setIfVar(data, vars, "geodata_outbound", "outbound")
	setJSONIfVar(data, vars, "geodata_assets", "assets")
	setJSONIfVar(data, vars, "assets", "assets")
}

func copyKey(dst ConfigObject, src ConfigObject, dstKey string, srcKey string) {
	value, ok := src[srcKey]
	if !ok || value == nil || value == "" {
		return
	}
	dst[dstKey] = value
}

func renderString(source string, data any) (string, error) {
	tpl, err := template.New("xray-template").Funcs(sprig.TxtFuncMap()).Parse(source)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func jsonObject(source string) ConfigObject {
	if source == "" {
		return ConfigObject{}
	}
	var out ConfigObject
	if err := json.Unmarshal([]byte(source), &out); err != nil || out == nil {
		return ConfigObject{}
	}
	return out
}

func merge(dst, src ConfigObject) {
	for key, value := range src {
		dst[key] = value
	}
}

func stringValue(data ConfigObject, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func stringOrJSONValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(data)
	}
}

func firstString(data ConfigObject, key, fallback string) string {
	if value := stringValue(data, key); value != "" {
		return value
	}
	return fallback
}

func firstInt(data ConfigObject, keys ...string) int {
	for _, key := range keys {
		if value := intValue(data, key); value != 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstStringFromList(data ConfigObject, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case []any:
		if len(v) == 0 {
			return ""
		}
		if item, ok := v[0].(string); ok {
			return item
		}
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	case string:
		return v
	}
	return ""
}

func boolValue(data ConfigObject, key string) bool {
	value, ok := data[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	default:
		return false
	}
}

func intValue(data ConfigObject, key string) int {
	value, ok := data[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		num, _ := v.Int64()
		return int(num)
	case string:
		num, _ := strconv.Atoi(v)
		return num
	default:
		return 0
	}
}

func uint16Value(data ConfigObject, key string) uint16 {
	value, ok := data[key]
	if !ok || value == nil {
		return 0
	}
	switch v := value.(type) {
	case float64:
		if v < 0 || v > math.MaxUint16 || math.Trunc(v) != v {
			return 0
		}
		return uint16(v)
	case int:
		if v < 0 || v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case int64:
		if v < 0 || v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case json.Number:
		n, err := strconv.ParseUint(string(v), 10, 16)
		if err != nil {
			return 0
		}
		return uint16(n)
	case string:
		n, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return 0
		}
		return uint16(n)
	default:
		return 0
	}
}
