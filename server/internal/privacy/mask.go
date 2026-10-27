package privacy

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const redactedValue = "***"

// NormalizeIPNetwork returns the address scope used for concurrent-device
// accounting: an exact IPv4 address and a /64 network for IPv6 privacy.
// Already-normalized scopes are accepted without widening their range.
func NormalizeIPNetwork(value string) (string, bool) {
	value = strings.TrimSpace(value)
	addr, err := netip.ParseAddr(value)
	if err != nil {
		prefix, prefixErr := netip.ParsePrefix(value)
		if prefixErr != nil {
			return "", false
		}
		addr = prefix.Addr()
		bits := prefix.Bits()
		if addr.Is4In6() {
			bits -= 96
		}
		wantBits := 64
		if addr.Unmap().Is4() {
			wantBits = 32
		}
		if bits != wantBits {
			return "", false
		}
	}
	addr = addr.Unmap()
	prefixBits := 64
	if addr.Is4() {
		prefixBits = 32
	}
	return netip.PrefixFrom(addr, prefixBits).Masked().String(), true
}

func MaskIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if value == redactedValue || strings.Contains(value, "*") {
		return value
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return MaskIdentifier(value)
	}
	if v4 := ip.To4(); v4 != nil {
		return maskIPv4(v4)
	}
	v6 := ip.To16()
	if v6 == nil {
		return MaskIdentifier(value)
	}
	return maskIPv6(v6)
}

func maskIPv4(v4 net.IP) string {
	var buf [15]byte
	out := buf[:0]
	out = strconv.AppendInt(out, int64(v4[0]), 10)
	out = append(out, '.')
	out = strconv.AppendInt(out, int64(v4[1]), 10)
	out = append(out, '.')
	out = strconv.AppendInt(out, int64(v4[2]), 10)
	out = append(out, '.', '*')
	return string(out)
}

func maskIPv6(v6 net.IP) string {
	var buf [39]byte
	out := buf[:0]
	for i := 0; i < 4; i++ {
		if i > 0 {
			out = append(out, ':')
		}
		group := uint16(v6[i*2])<<8 | uint16(v6[i*2+1])
		out = strconv.AppendUint(out, uint64(group), 16)
	}
	out = append(out, ":****:****:****:****"...)
	return string(out)
}

func MaskUserAgent(value string) string {
	first, next, ok := nextUserAgentField(value, 0)
	if !ok {
		return ""
	}
	if strings.HasPrefix(first, "(") {
		return redactedValue
	}
	maskedFirst := maskUserAgentToken(first)
	if strings.Contains(first, "/") {
		return maskedFirst
	}
	second, _, ok := nextUserAgentField(value, next)
	if !ok || strings.HasPrefix(second, "(") {
		return maskedFirst
	}
	return maskedFirst + " " + maskUserAgentToken(second)
}

func nextUserAgentField(value string, start int) (string, int, bool) {
	for start < len(value) {
		r, size := utf8.DecodeRuneInString(value[start:])
		if r == utf8.RuneError && size == 0 {
			return "", start, false
		}
		if !unicode.IsSpace(r) {
			break
		}
		start += size
	}
	if start >= len(value) {
		return "", start, false
	}
	end := start
	for end < len(value) {
		r, size := utf8.DecodeRuneInString(value[end:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		if unicode.IsSpace(r) {
			break
		}
		end += size
	}
	return value[start:end], end, true
}

func MaskToken(value string) string {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return ""
	case len(value) <= 8:
		return strings.Repeat("*", len(value))
	default:
		return value[:4] + "..." + value[len(value)-4:]
	}
}

func MaskTokenPrefix(value string) string {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return ""
	case len(value) <= 4:
		return strings.Repeat("*", len(value))
	default:
		return value[:4] + strings.Repeat("*", len(value)-4)
	}
}

func MaskIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "@") {
		return maskEmail(value)
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	if len(value) <= 8 {
		return value[:1] + strings.Repeat("*", len(value)-2) + value[len(value)-1:]
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func MaskJSON(data []byte, extraFields ...string) []byte {
	original := data
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return data
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return original
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return original
	}

	masked, changed := maskJSONValue(value, extraFields)
	if !changed {
		return original
	}
	out, err := json.Marshal(masked)
	if err != nil {
		return original
	}
	return out
}

func MaskMessageContentJSON(data []byte) []byte {
	masked := MaskJSON(data, "code", "verify_code", "verification_code", "otp", "pin")
	trimmed := bytes.TrimSpace(masked)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		return []byte(`"***"`)
	}
	return masked
}

func SanitizeURL(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return sanitizePath(raw)
	}
	parsed.Path = sanitizePath(parsed.Path)
	parsed.RawPath = ""
	parsed.RawQuery = SanitizeQuery(parsed.RawQuery)
	return parsed.String()
}

func SanitizeRequestTarget(value string) string {
	fields := strings.Fields(strings.TrimSpace(value))
	if len(fields) >= 2 && isHTTPMethod(fields[0]) {
		fields[1] = SanitizeURL(fields[1])
		return strings.Join(fields, " ")
	}
	return SanitizeURL(value)
}

func SanitizeHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse("//" + value)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil {
		return MaskIdentifier(value)
	}
	hostname := parsed.Hostname()
	if net.ParseIP(hostname) != nil {
		hostname = MaskIP(hostname)
	} else {
		labels := strings.Split(hostname, ".")
		for index, label := range labels {
			if isLikelySecretPathSegment(label) {
				labels[index] = MaskToken(label)
			}
		}
		hostname = strings.Join(labels, ".")
	}
	if port := parsed.Port(); port != "" {
		return net.JoinHostPort(hostname, port)
	}
	if strings.Contains(hostname, ":") {
		return "[" + hostname + "]"
	}
	return hostname
}

func SanitizeQuery(raw string) string {
	if raw == "" {
		return ""
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return MaskIdentifier(raw)
	}
	changed := false
	for key, items := range values {
		if !isSensitiveKey(key) {
			continue
		}
		changed = true
		for i := range items {
			items[i] = redactedValue
		}
		values[key] = items
	}
	if !changed {
		return raw
	}
	return values.Encode()
}

func MaskLogFieldJSON(key string, raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return json.RawMessage(MaskJSON(raw))
	}
	var value string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return raw
	}
	if normalizeKey(key) == "payload" || normalizeKey(key) == "body" || normalizeKey(key) == "data" {
		masked := MaskJSON([]byte(value))
		if !bytes.Equal(masked, []byte(value)) {
			return marshalLogString(string(masked))
		}
	}
	masked := maskLogStringField(key, value)
	if masked == value {
		return raw
	}
	return marshalLogString(masked)
}

func maskJSONValue(value any, extraFields []string) (any, bool) {
	switch data := value.(type) {
	case map[string]any:
		changed := false
		for key, item := range data {
			if isSensitiveKey(key) || isExtraSensitiveKey(key, extraFields) {
				data[key] = redactedValue
				changed = true
				continue
			}
			next, ok := maskJSONValue(item, extraFields)
			if ok {
				data[key] = next
				changed = true
			}
		}
		return data, changed
	case []any:
		changed := false
		for i, item := range data {
			next, ok := maskJSONValue(item, extraFields)
			if ok {
				data[i] = next
				changed = true
			}
		}
		return data, changed
	default:
		return value, false
	}
}

func maskLogStringField(key, value string) string {
	switch normalizeKey(key) {
	case "ip", "clientip", "loginip", "registerip":
		return MaskIP(value)
	case "useragent":
		return MaskUserAgent(value)
	case "email", "identifier", "authidentifier", "deviceid", "mobile", "phone", "telephone", "recipient", "to":
		return MaskIdentifier(value)
	case "token", "subscribetoken", "session", "uuid", "accesstoken", "refreshtoken", "authorization", "apikey", "secret", "secretkey", "clientsecret", "privatekey", "webhooksecret", "verifycode", "verificationcode", "otp", "pin":
		return redactedValue
	case "query":
		return SanitizeQuery(value)
	case "request", "url", "path":
		return SanitizeRequestTarget(value)
	default:
		return value
	}
}

func marshalLogString(value string) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`""`)
	}
	return json.RawMessage(data)
}

func isHTTPMethod(value string) bool {
	switch strings.ToUpper(value) {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return true
	default:
		return false
	}
}

func isExtraSensitiveKey(key string, extraFields []string) bool {
	normalized := normalizeKey(key)
	for _, field := range extraFields {
		if normalized == normalizeKey(field) {
			return true
		}
	}
	return false
}

func maskUserAgentToken(value string) string {
	value = strings.Trim(value, ";,")
	if value == "" {
		return redactedValue
	}
	if idx := strings.Index(value, "/"); idx > 0 {
		return value[:idx] + "/***"
	}
	if len(value) > 32 {
		return MaskIdentifier(value)
	}
	return value
}

func maskEmail(value string) string {
	at := strings.Index(value, "@")
	if at <= 0 || at == len(value)-1 {
		return MaskIdentifier(strings.ReplaceAll(value, "@", ""))
	}
	local := value[:at]
	domain := value[at+1:]
	if len(local) <= 2 {
		return strings.Repeat("*", len(local)) + "@" + domain
	}
	return local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:] + "@" + domain
}

func sanitizePath(path string) string {
	if path == "" {
		return path
	}
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if isLikelySecretPathSegment(part) {
			parts[i] = MaskToken(part)
		}
	}
	return strings.Join(parts, "/")
}

func isSensitiveKey(key string) bool {
	switch normalizeKey(key) {
	case "token", "accesstoken", "refreshtoken", "authorization", "apikey", "secret", "secretkey",
		"clientsecret", "password", "oldpassword", "newpassword", "cf_token", "cftoken",
		"email", "mobile", "phone", "telephone", "identifier", "privatekey", "webhooksecret",
		"authidentifier", "authidentifierdisplay", "deviceid", "recipient", "to", "subscribetoken", "session", "uuid",
		"verifycode", "verificationcode", "otp", "pin",
		"ip", "clientip", "loginip", "logoutip", "registerip", "sourceip", "destinationip", "useragent":
		return true
	default:
		return false
	}
}

func isLikelySecretPathSegment(value string) bool {
	if len(value) < 16 {
		return false
	}
	hasLetter := false
	hasDigit := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case r == '-' || r == '_' || r == '.' || r == '~':
		default:
			return false
		}
	}
	return hasLetter && hasDigit
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	for idx, r := range key {
		if r == '-' || r == '_' || r == ' ' {
			var builder strings.Builder
			builder.Grow(len(key))
			builder.WriteString(key[:idx])
			for _, next := range key[idx:] {
				if next == '-' || next == '_' || next == ' ' {
					continue
				}
				builder.WriteRune(unicode.ToLower(next))
			}
			return builder.String()
		}
		lower := unicode.ToLower(r)
		if lower != r {
			var builder strings.Builder
			builder.Grow(len(key))
			builder.WriteString(key[:idx])
			builder.WriteRune(lower)
			for offset, next := range key[idx:] {
				if offset == 0 {
					continue
				}
				if next == '-' || next == '_' || next == ' ' {
					continue
				}
				builder.WriteRune(unicode.ToLower(next))
			}
			return builder.String()
		}
	}
	return key
}
