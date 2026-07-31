package adapter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
)

type Proxy struct {
	Sort   int
	Name   string
	Server string
	Port   uint16
	Type   string
	Tags   []string

	// Security Options
	Security             string
	SNI                  string // Server Name Indication for TLS
	AllowInsecure        bool   // Allow insecure connections (skip certificate verification)
	Fingerprint          string // Client fingerprint for TLS connections
	PinnedPeerCertSha256 string // TLS peer certificate SHA-256 pin for Xray-core
	VerifyPeerCertByName string // TLS certificate name verification override for Xray-core
	RealityServerAddr    string // Reality server address
	RealityServerPort    int    // Reality server port
	RealityPrivateKey    string // Reality private key for authentication
	RealityPublicKey     string // Reality public key for authentication
	RealityShortId       string // Reality short ID for authentication
	// Transport Options
	Transport               string // Transport protocol (e.g., ws, http, grpc)
	Host                    string // For WebSocket/HTTP/HTTPS
	Path                    string // For HTTP/HTTPS
	ServiceName             string // For gRPC
	GrpcAuthority           string // For gRPC authority/Host override in Xray-core
	GrpcMultiMode           bool   // For Xray-core gRPC multiMode
	GrpcUserAgent           string // For gRPC User-Agent when supported by the client
	GrpcIdleTimeout         int    // For gRPC idle timeout in seconds
	GrpcHealthCheckTimeout  int    // For gRPC health check / ping timeout in seconds
	GrpcPermitWithoutStream bool   // For gRPC keepalive without active streams
	GrpcInitialWindowsSize  int    // For Xray-core gRPC initial window size
	// Shadowsocks Options
	Method     string
	ServerKey  string // For Shadowsocks 2022
	Username   string
	HeaderType string

	Obfs     string
	ObfsHost string
	ObfsPath string

	TLS           bool
	EchEnable     bool
	EchServerName string

	// Vmess/Vless/Trojan Options
	Flow string // Flow for Vmess/Vless/Trojan
	// Hysteria2 Options
	HopPorts               string // Comma-separated list of hop ports
	HopInterval            int    // Interval for hop ports in seconds
	ObfsPassword           string // Obfuscation password for Hysteria2
	UpMbps                 int    // Upload speed in Mbps
	DownMbps               int    // Download speed in Mbps
	HysteriaUDPIdleTimeout int    // Xray-core Hysteria UDP idle timeout in seconds
	HysteriaMasquerade     string // Xray-core Hysteria masquerade JSON

	// Tuic Options
	DisableSNI           bool   // Disable SNI
	ReduceRtt            bool   // Reduce RTT
	UDPRelayMode         string // UDP relay mode (e.g., "full", "partial")
	CongestionController string // Congestion controller (e.g., "cubic", "bbr")

	// AnyTLS
	PaddingScheme string

	// Mieru
	Multiplex string

	// Obfs
	//Obfs     string // obfs, 'none', 'http', 'tls'
	//ObfsHost string // obfs host
	//ObfsPath string // obfs path

	// Vless
	XhttpMode  string // xhttp mode
	XhttpExtra string // xhttp extra JSON

	// encryption
	Encryption              string // encryption，'none', 'mlkem768x25519plus'
	EncryptionMode          string // encryption mode，'native', 'xorpub', 'random'
	Encryption_Mode         string // Compatibility alias used by upstream subscription templates.
	EncryptionRtt           string // encryption rtt，'0rtt', '1rtt'
	EncryptionTicket        string // encryption ticket
	EncryptionServerPadding string // encryption server padding
	EncryptionPrivateKey    string // encryption private key
	EncryptionClientPadding string // encryption client padding
	EncryptionPassword      string // encryption password

	Ratio           float64 // Traffic ratio, default is 1
	CertMode        string  // Certificate mode, `none`｜`http`｜`dns`｜`self`
	CertDNSProvider string  // DNS provider for certificate
	CertDNSEnv      string  // Environment for DNS provider
}

type User struct {
	Password     string
	ExpiredAt    time.Time
	Download     int64
	Upload       int64
	Traffic      int64
	SubscribeURL string
}

type Client struct {
	SiteName       string            // Name of the site
	SubscribeName  string            // Name of the subscription
	ClientTemplate string            // Template for the entire client configuration
	OutputFormat   string            // json, yaml, etc.
	Proxies        []Proxy           // List of proxy configurations
	UserInfo       User              // User information
	Params         map[string]string // Additional parameters
}

type ProxyTemplateData map[string]any

func (c *Client) Build() ([]byte, error) {
	var buf bytes.Buffer
	funcs := sprig.TxtFuncMap()
	funcs["clientSupportsProxy"] = clientSupportsProxy
	funcs["xhttpMihomoOpts"] = xhttpMihomoOpts
	funcs["xhttpMihomoInlineOpts"] = xhttpMihomoInlineOpts
	tmpl, err := template.New("client").Funcs(funcs).Parse(c.ClientTemplate)
	if err != nil {
		return nil, err
	}

	proxies := make([]ProxyTemplateData, len(c.Proxies))
	for i, proxy := range c.Proxies {
		proxies[i] = proxyTemplateData(proxy)
	}

	err = tmpl.Execute(&buf, map[string]any{
		"SiteName":      c.SiteName,
		"SubscribeName": c.SubscribeName,
		"OutputFormat":  c.OutputFormat,
		"Proxies":       proxies,
		"UserInfo":      c.UserInfo,
		"Params":        c.Params,
	})
	if err != nil {
		return nil, err
	}

	result := buf.String()
	if c.OutputFormat == "base64" {
		encoded := base64.StdEncoding.EncodeToString([]byte(result))
		return []byte(encoded), nil
	}

	return buf.Bytes(), nil
}

func proxyTemplateData(proxy Proxy) ProxyTemplateData {
	return ProxyTemplateData{
		"Sort":                    proxy.Sort,
		"Name":                    proxy.Name,
		"Server":                  proxy.Server,
		"Port":                    proxy.Port,
		"Type":                    proxy.Type,
		"Tags":                    proxy.Tags,
		"Security":                proxy.Security,
		"SNI":                     proxy.SNI,
		"AllowInsecure":           proxy.AllowInsecure,
		"Fingerprint":             proxy.Fingerprint,
		"PinnedPeerCertSha256":    proxy.PinnedPeerCertSha256,
		"VerifyPeerCertByName":    proxy.VerifyPeerCertByName,
		"RealityServerAddr":       proxy.RealityServerAddr,
		"RealityServerPort":       proxy.RealityServerPort,
		"RealityPrivateKey":       proxy.RealityPrivateKey,
		"RealityPublicKey":        proxy.RealityPublicKey,
		"RealityShortId":          proxy.RealityShortId,
		"Transport":               proxy.Transport,
		"Network":                 proxy.Transport,
		"Host":                    proxy.Host,
		"Path":                    proxy.Path,
		"ServiceName":             proxy.ServiceName,
		"GrpcAuthority":           proxy.GrpcAuthority,
		"GrpcMultiMode":           proxy.GrpcMultiMode,
		"GrpcUserAgent":           proxy.GrpcUserAgent,
		"GrpcIdleTimeout":         proxy.GrpcIdleTimeout,
		"GrpcHealthCheckTimeout":  proxy.GrpcHealthCheckTimeout,
		"GrpcPermitWithoutStream": proxy.GrpcPermitWithoutStream,
		"GrpcInitialWindowsSize":  proxy.GrpcInitialWindowsSize,
		"Method":                  proxy.Method,
		"ServerKey":               proxy.ServerKey,
		"Username":                proxy.Username,
		"HeaderType":              proxy.HeaderType,
		"Obfs":                    proxy.Obfs,
		"ObfsHost":                proxy.ObfsHost,
		"ObfsPath":                proxy.ObfsPath,
		"TLS":                     proxy.TLS,
		"EchEnable":               proxy.EchEnable,
		"EchServerName":           proxy.EchServerName,
		"Flow":                    proxy.Flow,
		"HopPorts":                proxy.HopPorts,
		"HopInterval":             proxy.HopInterval,
		"ObfsPassword":            proxy.ObfsPassword,
		"UpMbps":                  proxy.UpMbps,
		"DownMbps":                proxy.DownMbps,
		"HysteriaUDPIdleTimeout":  proxy.HysteriaUDPIdleTimeout,
		"HysteriaMasquerade":      proxy.HysteriaMasquerade,
		"DisableSNI":              proxy.DisableSNI,
		"ReduceRtt":               proxy.ReduceRtt,
		"UDPRelayMode":            proxy.UDPRelayMode,
		"CongestionController":    proxy.CongestionController,
		"PaddingScheme":           proxy.PaddingScheme,
		"Multiplex":               proxy.Multiplex,
		"XhttpMode":               proxy.XhttpMode,
		"XhttpExtra":              proxy.XhttpExtra,
		"Encryption":              proxy.Encryption,
		"EncryptionMode":          proxy.EncryptionMode,
		"Encryption_Mode":         proxy.Encryption_Mode,
		"EncryptionRtt":           proxy.EncryptionRtt,
		"EncryptionTicket":        proxy.EncryptionTicket,
		"EncryptionServerPadding": proxy.EncryptionServerPadding,
		"EncryptionPrivateKey":    proxy.EncryptionPrivateKey,
		"EncryptionClientPadding": proxy.EncryptionClientPadding,
		"EncryptionPassword":      proxy.EncryptionPassword,
		"Ratio":                   proxy.Ratio,
		"CertMode":                proxy.CertMode,
		"CertDNSProvider":         proxy.CertDNSProvider,
		"CertDNSEnv":              proxy.CertDNSEnv,
	}
}

func clientSupportsProxy(client string, proxy ProxyTemplateData) bool {
	if !isXHTTPProxy(proxy) {
		return true
	}

	proxyType := strings.ToLower(proxyString(proxy, "Type"))
	if proxyType != "vless" && client != "xray-core" {
		return false
	}

	switch strings.ToLower(client) {
	case "default", "shadowrocket", "clash", "stash", "xray-core", "surge-xray":
		return true
	default:
		return false
	}
}

func isXHTTPProxy(proxy ProxyTemplateData) bool {
	transport := strings.ToLower(proxyString(proxy, "Transport"))
	return transport == "xhttp" || transport == "splithttp"
}

type xhttpOption struct {
	key         string
	value       any
	children    []xhttpOption
	forceString bool
}

func xhttpMihomoOpts(proxy ProxyTemplateData, indent int) string {
	return renderXHTTPOptionsYAML(buildMihomoXHTTPOptions(proxy), indent)
}

func xhttpMihomoInlineOpts(proxy ProxyTemplateData) string {
	return renderXHTTPOptionsInline(buildMihomoXHTTPOptions(proxy))
}

func buildMihomoXHTTPOptions(proxy ProxyTemplateData) []xhttpOption {
	opts := []xhttpOption{{
		key:   "path",
		value: defaultString(proxyString(proxy, "Path"), "/"),
	}}
	if host := proxyString(proxy, "Host"); host != "" {
		opts = append(opts, xhttpOption{key: "host", value: host})
	}
	if mode := proxyString(proxy, "XhttpMode"); mode != "" {
		opts = append(opts, xhttpOption{key: "mode", value: mode})
	}

	extra := parseJSONObject(proxyString(proxy, "XhttpExtra"))
	if len(extra) == 0 {
		return opts
	}

	scalarMappings := []struct {
		from        string
		to          string
		forceString bool
	}{
		{"headers", "headers", false},
		{"noGRPCHeader", "no-grpc-header", false},
		{"xPaddingBytes", "x-padding-bytes", true},
		{"xPaddingObfsMode", "x-padding-obfs-mode", false},
		{"xPaddingKey", "x-padding-key", false},
		{"xPaddingHeader", "x-padding-header", false},
		{"xPaddingPlacement", "x-padding-placement", false},
		{"xPaddingMethod", "x-padding-method", false},
		{"uplinkHTTPMethod", "uplink-http-method", false},
		{"sessionPlacement", "session-placement", false},
		{"sessionKey", "session-key", false},
		{"seqPlacement", "seq-placement", false},
		{"seqKey", "seq-key", false},
		{"uplinkDataPlacement", "uplink-data-placement", false},
		{"uplinkDataKey", "uplink-data-key", false},
		{"uplinkChunkSize", "uplink-chunk-size", true},
		{"scMaxEachPostBytes", "sc-max-each-post-bytes", true},
		{"scMinPostsIntervalMs", "sc-min-posts-interval-ms", true},
	}
	for _, item := range scalarMappings {
		value, ok := extra[item.from]
		if !ok || isEmptyXHTTPValue(value) {
			continue
		}
		if childMap, ok := asStringMap(value); ok {
			children := orderedMapOptions(childMap, false)
			if len(children) > 0 {
				opts = append(opts, xhttpOption{key: item.to, children: children})
			}
			continue
		}
		opts = append(opts, xhttpOption{key: item.to, value: value, forceString: item.forceString})
	}

	if xmux, ok := asStringMap(extra["xmux"]); ok {
		reuseMappings := []struct {
			from string
			to   string
		}{
			{"maxConcurrency", "max-concurrency"},
			{"maxConnections", "max-connections"},
			{"cMaxReuseTimes", "c-max-reuse-times"},
			{"hMaxRequestTimes", "h-max-request-times"},
			{"hMaxReusableSecs", "h-max-reusable-secs"},
			{"hKeepAlivePeriod", "h-keep-alive-period"},
		}
		children := make([]xhttpOption, 0, len(reuseMappings))
		for _, item := range reuseMappings {
			value, ok := xmux[item.from]
			if !ok || isZeroXHTTPNumber(value) || isEmptyXHTTPValue(value) {
				continue
			}
			children = append(children, xhttpOption{key: item.to, value: value, forceString: true})
		}
		if len(children) > 0 {
			opts = append(opts, xhttpOption{key: "reuse-settings", children: children})
		}
	}

	return opts
}

func renderXHTTPOptionsYAML(opts []xhttpOption, indent int) string {
	if len(opts) == 0 {
		return ""
	}
	pad := strings.Repeat(" ", indent)
	var b strings.Builder
	for _, opt := range opts {
		if len(opt.children) > 0 {
			b.WriteString(pad)
			b.WriteString(opt.key)
			b.WriteString(":\n")
			b.WriteString(renderXHTTPOptionsYAML(opt.children, indent+2))
			continue
		}
		b.WriteString(pad)
		b.WriteString(opt.key)
		b.WriteString(": ")
		b.WriteString(formatYAMLScalar(opt.value, opt.forceString))
		b.WriteByte('\n')
	}
	return b.String()
}

func renderXHTTPOptionsInline(opts []xhttpOption) string {
	if len(opts) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(opts))
	for _, opt := range opts {
		if len(opt.children) > 0 {
			parts = append(parts, fmt.Sprintf("%s: %s", opt.key, renderXHTTPOptionsInline(opt.children)))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", opt.key, formatYAMLScalar(opt.value, opt.forceString)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func orderedMapOptions(data map[string]any, forceString bool) []xhttpOption {
	keys := make([]string, 0, len(data))
	for key, value := range data {
		if isEmptyXHTTPValue(value) {
			continue
		}
		keys = append(keys, key)
	}
	sortStrings(keys)
	opts := make([]xhttpOption, 0, len(keys))
	for _, key := range keys {
		value := data[key]
		if childMap, ok := asStringMap(value); ok {
			children := orderedMapOptions(childMap, forceString)
			if len(children) > 0 {
				opts = append(opts, xhttpOption{key: key, children: children})
			}
			continue
		}
		opts = append(opts, xhttpOption{key: key, value: value, forceString: forceString})
	}
	return opts
}

func parseJSONObject(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var data map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&data); err != nil {
		return nil
	}
	return data
}

func asStringMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	default:
		return nil, false
	}
}

func formatYAMLScalar(value any, forceString bool) string {
	if forceString {
		return quoteYAML(fmt.Sprint(value))
	}
	switch typed := value.(type) {
	case string:
		return quoteYAML(typed)
	case json.Number:
		return typed.String()
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case nil:
		return "null"
	default:
		return quoteYAML(fmt.Sprint(typed))
	}
}

func quoteYAML(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

func isEmptyXHTTPValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case json.Number:
		return typed.String() == ""
	case bool:
		return !typed
	default:
		return false
	}
}

func isZeroXHTTPNumber(value any) bool {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		return err == nil && number == 0
	case float64:
		return typed == 0
	case int:
		return typed == 0
	case int64:
		return typed == 0
	case string:
		return typed == "0"
	default:
		return false
	}
}

func proxyString(proxy ProxyTemplateData, key string) string {
	value, ok := proxy[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
