package adapter

import (
	"encoding/json"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func surgeSupportsProxy(proxy ProxyTemplateData) bool {
	if proxyString(proxy, "Type") == "vless" {
		return surgeVLESSOptions(proxy) != ""
	}
	return clientSupportsProxy("surge", proxy)
}

// surgeVLESSOptions returns no options for a combination Surge cannot express.
func surgeVLESSOptions(proxy ProxyTemplateData) string {
	security := proxyString(proxy, "Security")
	if tls, _ := proxy["TLS"].(bool); security == "" && tls {
		security = "tls"
	}
	tls, reality := security == "tls", security == "reality"
	if !tls && !reality && security != "" && security != "none" {
		return ""
	}
	if reality && (proxyString(proxy, "RealityPublicKey") == "" || proxyString(proxy, "SNI") == "") {
		return ""
	}

	encryption, valid := surgeVLESSEncryption(proxy)
	if !valid {
		return ""
	}
	flow := proxyString(proxy, "Flow")
	vision := flow != "" && flow != "none"
	if vision && flow != "xtls-rprx-vision" && flow != "xtls-rprx-vision-udp443" {
		return ""
	}
	opts := surgeOptions{"tfo=true"}
	transport := proxyString(proxy, "Transport")
	switch transport {
	case "", "tcp", "raw":
		if header := proxyString(proxy, "HeaderType"); header != "" && header != "none" {
			return ""
		}
		if vision && !tls && !reality && encryption == "" {
			return ""
		}
	case "ws", "websocket", "httpupgrade":
		if reality || (vision && encryption == "") {
			return ""
		}
		opts.add("ws", "true")
		if transport == "httpupgrade" {
			opts.add("ws-http-upgrade", "true")
		}
		opts.add("ws-path", proxyString(proxy, "Path"))
		if host := proxyString(proxy, "Host"); host != "" {
			opts.add("ws-headers", "Host:"+host)
		}
	case "grpc":
		if (!tls && !reality) || vision {
			return ""
		}
		opts.add("grpc", "true")
		opts.add("grpc-service-name", proxyString(proxy, "ServiceName"))
		opts.add("grpc-authority", proxyString(proxy, "GrpcAuthority"))
		if multi, _ := proxy["GrpcMultiMode"].(bool); multi {
			opts.add("grpc-mode", "multi")
		}
	case "xhttp", "splithttp":
		if (vision && encryption == "") || !opts.addXHTTP(proxy) {
			return ""
		}
	default:
		return ""
	}

	if vision {
		opts.add("flow", flow)
	}
	opts.add("encryption", encryption)
	if tls || reality {
		opts.add("sni", proxyString(proxy, "SNI"))
		// Xray's uTLS profile names are not native Surge profile names.
		switch fp := proxyString(proxy, "Fingerprint"); fp {
		case "apple-26", "chromium-155":
			opts.add("client-fingerprint", fp)
		}
	}
	if reality {
		opts.add("reality-public-key", proxyString(proxy, "RealityPublicKey"))
		opts.add("reality-short-id", proxyString(proxy, "RealityShortId"))
	}
	if tls {
		opts.add("tls", "true")
		if alpn, ok := proxy["ALPN"].([]string); ok {
			opts.add("alpn", strings.Join(alpn, ","))
		}
		if pins := proxyString(proxy, "PinnedPeerCertSha256"); pins != "" {
			pin, _, _ := strings.Cut(pins, ",")
			opts.add("server-cert-fingerprint-sha256", pin)
		} else if insecure, _ := proxy["AllowInsecure"].(bool); insecure {
			opts.add("skip-cert-verify", "true")
		} else {
			for name := range strings.SplitSeq(proxyString(proxy, "VerifyPeerCertByName"), ",") {
				if name = strings.TrimSpace(name); name != "" {
					opts.add("server-cert-verify-name", name)
					break
				}
			}
		}
	}
	return ", " + strings.Join(opts, ", ")
}

func surgeVLESSEncryption(proxy ProxyTemplateData) (string, bool) {
	value := proxyString(proxy, "Encryption")
	if value == "" || value == "none" {
		required, _ := proxy["EncryptionRequired"].(bool)
		return "", !required
	}
	if value == "mlkem768x25519plus" {
		mode := defaultString(proxyString(proxy, "EncryptionMode"), proxyString(proxy, "Encryption_Mode"))
		rtt := proxyString(proxy, "EncryptionRtt")
		key := proxyString(proxy, "EncryptionPassword")
		if key == "" {
			return "", false
		}
		parts := []string{value, mode, rtt}
		if padding := proxyString(proxy, "EncryptionClientPadding"); padding != "" {
			parts = append(parts, padding)
		}
		value = strings.Join(append(parts, key), ".")
	}
	parts := strings.Split(value, ".")
	if len(parts) < 4 || parts[0] != "mlkem768x25519plus" || slices.Contains(parts, "") {
		return "", false
	}
	validMode := parts[1] == "native" || parts[1] == "xorpub" || parts[1] == "random"
	validRTT := parts[2] == "0rtt" || parts[2] == "1rtt"
	return value, validMode && validRTT
}

type surgeOptions []string

func (o *surgeOptions) add(key, value string) {
	if value == "" {
		return
	}
	if strings.ContainsAny(value, ",\"\\\r\n") || strings.TrimSpace(value) != value {
		value = strconv.Quote(value)
	}
	*o = append(*o, key+"="+value)
}

func (o *surgeOptions) addXHTTP(proxy ProxyTemplateData) bool {
	raw := strings.TrimSpace(proxyString(proxy, "XhttpExtra"))
	extra := parseJSONObject(raw)
	separateDownload, _ := proxy["XhttpDownloadSettings"].(bool)
	if separateDownload || extra["downloadSettings"] != nil || (raw != "" && extra == nil && raw != "null") {
		return false
	}
	mode := proxyString(proxy, "XhttpMode")
	switch mode {
	case "", "auto", "packet-up", "stream-up", "stream-one":
	default:
		return false
	}
	method, _ := extra["uplinkHTTPMethod"].(string)
	method = strings.ToUpper(method)
	placement, _ := extra["uplinkDataPlacement"].(string)
	if mode != "packet-up" && (method == "GET" || placement == "header" || placement == "cookie") {
		return false
	}
	o.add("xhttp", "true")
	o.add("xhttp-path", proxyString(proxy, "Path"))
	o.add("xhttp-host", proxyString(proxy, "Host"))
	o.add("xhttp-mode", mode)
	if headers, ok := asStringMap(extra["headers"]); ok {
		var values []string
		for _, key := range slices.Sorted(maps.Keys(headers)) {
			if strings.EqualFold(key, "Host") {
				continue
			}
			value, ok := headers[key].(string)
			if !ok || strings.ContainsAny(key, ":|\r\n") || strings.ContainsAny(value, "|\r\n") {
				return false
			}
			values = append(values, key+":"+value)
		}
		o.add("xhttp-headers", strings.Join(values, "|"))
	}
	o.add("xhttp-uplink-method", method)
	o.addXHTTPFields(extra, []surgeMapping{
		{"xPaddingBytes", "padding-bytes"},
		{"xPaddingObfsMode", "padding-obfs"},
		{"xPaddingKey", "padding-key"},
		{"xPaddingHeader", "padding-header"},
		{"xPaddingPlacement", "padding-placement"},
		{"xPaddingMethod", "padding-method"},
		{"sessionIDPlacement", "session-placement"},
		{"sessionIDKey", "session-key"},
		{"sessionIDTable", "session-table"},
		{"sessionIDLength", "session-length"},
		{"seqPlacement", "seq-placement"},
		{"seqKey", "seq-key"},
		{"uplinkDataPlacement", "uplink-data-placement"},
		{"uplinkDataKey", "uplink-data-key"},
		{"uplinkChunkSize", "uplink-chunk-size"},
		{"noGRPCHeader", "no-grpc-header"},
		{"scMaxEachPostBytes", "max-each-post-bytes"},
		{"scMinPostsIntervalMs", "min-posts-interval"},
	})
	if xmux, ok := asStringMap(extra["xmux"]); ok {
		for _, mapping := range []surgeMapping{
			{"maxConcurrency", "max-concurrency"},
			{"maxConnections", "max-connections"},
			{"cMaxReuseTimes", "max-reuse-times"},
			{"hMaxRequestTimes", "max-request-times"},
			{"hMaxReusableSecs", "max-reusable-secs"},
			{"hKeepAlivePeriod", "keep-alive"},
		} {
			value := surgeXMUXValue(xmux[mapping.xray], mapping.xray == "hKeepAlivePeriod")
			o.add("xhttp-xmux-"+mapping.surge, value)
		}
	}
	return true
}

type surgeMapping struct{ xray, surge string }

var (
	surgeXMUXRange     = regexp.MustCompile("^[0-9]+(-[0-9]+)?$")
	surgeXMUXKeepAlive = regexp.MustCompile("^-?[0-9]+$")
)

func surgeXMUXValue(value any, keepAlive bool) string {
	var text string
	switch value := value.(type) {
	case string:
		text = strings.TrimSpace(value)
	case json.Number:
		text = value.String()
	default:
		return ""
	}
	pattern := surgeXMUXRange
	if keepAlive {
		pattern = surgeXMUXKeepAlive
	}
	// A zero-only XMUX block in Xray uses defaults; don't disable Surge's defaults with placeholders.
	if !pattern.MatchString(text) || strings.Trim(text, "0-") == "" {
		return ""
	}
	return text
}

func (o *surgeOptions) addXHTTPFields(fields map[string]any, mappings []surgeMapping) {
	for _, item := range mappings {
		key := "xhttp-" + item.surge
		switch value := fields[item.xray].(type) {
		case string:
			o.add(key, value)
		case json.Number:
			o.add(key, value.String())
		case bool:
			o.add(key, strconv.FormatBool(value))
		}
	}
}
