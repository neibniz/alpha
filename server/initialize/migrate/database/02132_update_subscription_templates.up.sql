-- 02132_update_subscription_templates.up.sql
-- Refresh built-in subscription templates after protocol field audit.

BEGIN;

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (1, 'Default', '', 'Default URI subscription template for generic clients.', '', 'default', TRUE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "anytls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "default" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

REMARKS={{ .SiteName }}-{{ .SubscribeName }}
STATUS=Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05\n" }}

{{- range $proxy := $supportedProxies }}
  {{- $common := "udp=1&tfo=1" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- /* 公共传输层配置函数 */ -}}
  {{- $buildTransportParams := dict -}}
  {{- $transport := default "tcp" $proxy.Transport -}}
  {{- if ne $transport "" -}}
    {{- $_ := set $buildTransportParams "type" (ternary "ws" $transport (eq $transport "websocket")) -}}
  {{- end -}}
  {{- /* TCP 传输类型配置 */ -}}
  {{- if eq $transport "tcp" -}}
    {{- $headerType := default "none" $proxy.HeaderType -}}
    {{- if ne $headerType "none" -}}
      {{- $_ := set $buildTransportParams "headerType" $headerType -}}
    {{- end -}}
    {{- if and (eq $headerType "http") (ne (default "" $proxy.Host) "") -}}
      {{- $_ := set $buildTransportParams "host" $proxy.Host -}}
    {{- end -}}
    {{- if and (eq $headerType "http") (ne (default "" $proxy.Path) "") -}}
      {{- $_ := set $buildTransportParams "path" ($proxy.Path | urlquery) -}}
    {{- end -}}
  {{- end -}}
  {{- /* WebSocket/xhttp/httpupgrade 传输类型配置 */ -}}
  {{- if and (or (eq $transport "ws") (eq $transport "websocket") (eq $transport "xhttp") (eq $transport "httpupgrade")) (ne (default "" $proxy.Host) "") -}}
    {{- $_ := set $buildTransportParams "host" $proxy.Host -}}
  {{- end -}}
  {{- if and (or (eq $transport "ws") (eq $transport "websocket") (eq $transport "xhttp") (eq $transport "httpupgrade")) (ne (default "" $proxy.Path) "") -}}
    {{- $_ := set $buildTransportParams "path" ($proxy.Path | urlquery) -}}
  {{- end -}}
  {{- /* gRPC 传输类型配置 */ -}}
  {{- if and (eq $transport "grpc") (ne (default "" $proxy.ServiceName) "") -}}
    {{- $_ := set $buildTransportParams "serviceName" $proxy.ServiceName -}}
  {{- end -}}
  {{- /* xhttp 特有配置 */ -}}
  {{- if and (eq $transport "xhttp") (ne (default "" $proxy.XhttpMode) "") -}}
    {{- $_ := set $buildTransportParams "mode" $proxy.XhttpMode -}}
  {{- end -}}
  {{- if and (eq $transport "xhttp") (ne (default "" $proxy.XhttpExtra) "") -}}
    {{- $_ := set $buildTransportParams "extra" (urlquery $proxy.XhttpExtra) -}}
  {{- end -}}

  {{- /* 公共安全层配置 */ -}}
  {{- $buildSecurityParams := dict -}}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") -}}
    {{- $_ := set $buildSecurityParams "security" $proxy.Security -}}
  {{- end -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $_ := set $buildSecurityParams "sni" $proxy.SNI -}}
  {{- end -}}
  {{- if $SkipVerify -}}
    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") -}}
    {{- $_ := set $buildSecurityParams "pbk" $proxy.RealityPublicKey -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityShortId) "") -}}
    {{- $_ := set $buildSecurityParams "sid" $proxy.RealityShortId -}}
  {{- end -}}
  {{- if $proxy.EchEnable -}}
    {{- $_ := set $buildSecurityParams "ech" (printf "%s+udp://1.1.1.1" (default "" $proxy.EchServerName) | urlquery) -}}
  {{- end -}}

  {{- if eq $proxy.Type "shadowsocks" }}
  {{- $params := list -}}
  {{- /* Shadowsocks 特有的 obfs 插件参数 */ -}}
  {{- if ne (default "" $proxy.Obfs) "" -}}
    {{- $params = append $params (printf "obfs=%s" $proxy.Obfs) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsHost) "" -}}
    {{- $params = append $params (printf "obfs-host=%s" $proxy.ObfsHost) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPath) "" -}}
    {{- $params = append $params (printf "obfs-uri=%s" ($proxy.ObfsPath | urlquery)) -}}
  {{- end -}}
  {{- /* 使用公共传输层配置 */ -}}
  {{- range $key, $val := $buildTransportParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- /* 使用公共安全层配置 */ -}}
  {{- range $key, $val := $buildSecurityParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- /* 添加公共参数 */ -}}
  {{- $params = append $params $common }}
ss://{{ printf "%s:%s" (default "aes-128-gcm" $proxy.Method) $password | b64enc }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if eq $proxy.Type "vmess" }}
  {{- $vmessDict := dict "v" "2" "ps" $proxy.Name "add" $proxy.Server "port" (printf "%d" $proxy.Port) "id" $password "aid" "0" "net" "tcp" "type" "none" -}}
  {{- if hasKey $buildTransportParams "type" -}}
    {{- $_ := set $vmessDict "net" (index $buildTransportParams "type") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "host" -}}
    {{- $_ := set $vmessDict "host" (index $buildTransportParams "host") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "path" -}}
    {{- $_ := set $vmessDict "path" (index $buildTransportParams "path") -}}
  {{- end -}}
  {{- if and (eq $transport "grpc") (hasKey $buildTransportParams "serviceName") -}}
    {{- $_ := set $vmessDict "path" (index $buildTransportParams "serviceName") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "mode" -}}
    {{- $_ := set $vmessDict "xhttpMode" (index $buildTransportParams "mode") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "extra" -}}
    {{- $_ := set $vmessDict "xhttpExtra" (index $buildTransportParams "extra") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "security" -}}
    {{- $_ := set $vmessDict "tls" (index $buildSecurityParams "security") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "sni" -}}
    {{- $_ := set $vmessDict "sni" (index $buildSecurityParams "sni") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "fp" -}}
    {{- $_ := set $vmessDict "fp" (index $buildSecurityParams "fp") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "allowInsecure" -}}
    {{- $_ := set $vmessDict "skip-cert-verify" true -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "ech" -}}
    {{- $_ := set $vmessDict "ech" (index $buildSecurityParams "ech") -}}
  {{- end }}
vmess://{{ $vmessDict | toJson | b64enc }}
  {{- else if eq $proxy.Type "vless" }}
  {{- $params := list -}}
  {{- /* 1. Encryption 加密参数 */ -}}
  {{- $encryption := default "none" $proxy.Encryption -}}
  {{- if eq $encryption "none" -}}
    {{- $params = append $params "encryption=none" -}}
  {{- else -}}
    {{- $encParts := list -}}
    {{- $encParts = append $encParts $encryption -}}
    {{- if ne (default "" $proxy.EncryptionMode) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionMode -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionRtt) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionRtt -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionClientPadding) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionClientPadding -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionPassword) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionPassword -}}
    {{- end -}}
    {{- $params = append $params (printf "encryption=%s" (join "." $encParts)) -}}
  {{- end -}}
  {{- /* 2. Flow 流控参数 */ -}}
  {{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") -}}
    {{- $params = append $params (printf "flow=%s" $proxy.Flow) -}}
  {{- end -}}
  {{- /* 3. Security 安全参数 */ -}}
  {{- if hasKey $buildSecurityParams "security" -}}
    {{- $params = append $params (printf "security=%s" (index $buildSecurityParams "security")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "sni" -}}
    {{- $params = append $params (printf "sni=%s" (index $buildSecurityParams "sni")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "fp" -}}
    {{- $params = append $params (printf "fp=%s" (index $buildSecurityParams "fp")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "allowInsecure" -}}
    {{- $params = append $params "allowInsecure=1" -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "pbk" -}}
    {{- $params = append $params (printf "pbk=%s" (index $buildSecurityParams "pbk")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "sid" -}}
    {{- $params = append $params (printf "sid=%s" (index $buildSecurityParams "sid")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "ech" -}}
    {{- $params = append $params (printf "ech=%s" (index $buildSecurityParams "ech")) -}}
  {{- end -}}
  {{- /* 4. Transport 传输层参数 */ -}}
  {{- if hasKey $buildTransportParams "type" -}}
    {{- $params = append $params (printf "type=%s" (index $buildTransportParams "type")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "host" -}}
    {{- $params = append $params (printf "host=%s" (index $buildTransportParams "host")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "path" -}}
    {{- $params = append $params (printf "path=%s" (index $buildTransportParams "path")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "serviceName" -}}
    {{- $params = append $params (printf "serviceName=%s" (index $buildTransportParams "serviceName")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "mode" -}}
    {{- $params = append $params (printf "mode=%s" (index $buildTransportParams "mode")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "extra" -}}
    {{- $params = append $params (printf "extra=%s" (index $buildTransportParams "extra")) -}}
  {{- end -}}
  {{- /* 5. Common 通用参数 */ -}}
  {{- $params = append $params $common }}
vless://{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if eq $proxy.Type "trojan" }}
  {{- $params := list -}}
  {{- range $key, $val := $buildTransportParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- range $key, $val := $buildSecurityParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- $params = append $params $common }}
trojan://{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  {{- else if eq $proxy.Type "tuic" }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.CongestionController) "" -}}
    {{- $params = append $params (printf "congestion_controller=%s" $proxy.CongestionController) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.UDPRelayMode) "" -}}
    {{- $params = append $params (printf "udp_relay_mode=%s" $proxy.UDPRelayMode) -}}
  {{- end -}}
  {{- if $proxy.ReduceRtt -}}
    {{- $params = append $params "reduce_rtt=1" -}}
  {{- end -}}
  {{- if $proxy.DisableSNI -}}
    {{- $params = append $params "disable_sni=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "allow_insecure=1" -}}
  {{- end -}}
  {{- $params = append $params $common }}
tuic://{{ default "" $proxy.ServerKey }}:{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if eq $proxy.Type "anytls" }}
  {{- $params := list -}}
  {{- /* 使用公共传输层配置 */ -}}
  {{- range $key, $val := $buildTransportParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- /* 使用公共安全层配置 */ -}}
  {{- range $key, $val := $buildSecurityParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- $params = append $params $common }}
anytls://{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "http") (eq $proxy.Type "https") }}
  {{- $user := default $password $proxy.Username }}
http{{- if eq $proxy.Type "https" -}}s{{- end -}}://{{- if or (ne (default "" $user) "") (ne (default "" $password) "") -}}{{ $user }}:{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}#{{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "socks") (eq $proxy.Type "socks5") (eq $proxy.Type "socks5-tls") }}
  {{- $user := default $password $proxy.Username }}
socks5://{{- if or (ne (default "" $user) "") (ne (default "" $password) "") -}}{{ $user }}:{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}{{- if eq $proxy.Type "socks5-tls" }}?tls=1{{- end }}#{{ $proxy.Name }}
  {{- end }}
{{- end }}
$alpha_subscription_template$, 'base64', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (2, 'Shadowrocket', '', 'Shadowrocket URI subscription template.', 'shadowrocket://add/sub://${window.btoa(url)}?remark=${encodeURIComponent(name)}', 'Shadowrocket', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "anytls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "shadowrocket" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

REMARKS={{ .SiteName }}-{{ .SubscribeName }}
STATUS=Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05\n" }}

{{- range $proxy := $supportedProxies }}
  {{- $common := "udp=1&tfo=1" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- /* 公共传输层配置函数 */ -}}
  {{- $buildTransportParams := dict -}}
  {{- $transport := default "tcp" $proxy.Transport -}}
  {{- if ne $transport "" -}}
    {{- $_ := set $buildTransportParams "type" (ternary "ws" $transport (eq $transport "websocket")) -}}
  {{- end -}}
  {{- /* TCP 传输类型配置 */ -}}
  {{- if eq $transport "tcp" -}}
    {{- $headerType := default "none" $proxy.HeaderType -}}
    {{- if ne $headerType "none" -}}
      {{- $_ := set $buildTransportParams "headerType" $headerType -}}
    {{- end -}}
    {{- if and (eq $headerType "http") (ne (default "" $proxy.Host) "") -}}
      {{- $_ := set $buildTransportParams "host" $proxy.Host -}}
    {{- end -}}
    {{- if and (eq $headerType "http") (ne (default "" $proxy.Path) "") -}}
      {{- $_ := set $buildTransportParams "path" ($proxy.Path | urlquery) -}}
    {{- end -}}
  {{- end -}}
  {{- /* WebSocket/xhttp/httpupgrade 传输类型配置 */ -}}
  {{- if and (or (eq $transport "ws") (eq $transport "websocket") (eq $transport "xhttp") (eq $transport "httpupgrade")) (ne (default "" $proxy.Host) "") -}}
    {{- $_ := set $buildTransportParams "host" $proxy.Host -}}
  {{- end -}}
  {{- if and (or (eq $transport "ws") (eq $transport "websocket") (eq $transport "xhttp") (eq $transport "httpupgrade")) (ne (default "" $proxy.Path) "") -}}
    {{- $_ := set $buildTransportParams "path" ($proxy.Path | urlquery) -}}
  {{- end -}}
  {{- /* gRPC 传输类型配置 */ -}}
  {{- if and (eq $transport "grpc") (ne (default "" $proxy.ServiceName) "") -}}
    {{- $_ := set $buildTransportParams "serviceName" $proxy.ServiceName -}}
  {{- end -}}
  {{- /* xhttp 特有配置 */ -}}
  {{- if and (eq $transport "xhttp") (ne (default "" $proxy.XhttpMode) "") -}}
    {{- $_ := set $buildTransportParams "mode" $proxy.XhttpMode -}}
  {{- end -}}
  {{- if and (eq $transport "xhttp") (ne (default "" $proxy.XhttpExtra) "") -}}
    {{- $_ := set $buildTransportParams "extra" (urlquery $proxy.XhttpExtra) -}}
  {{- end -}}

  {{- /* 公共安全层配置 */ -}}
  {{- $buildSecurityParams := dict -}}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") -}}
    {{- $_ := set $buildSecurityParams "security" $proxy.Security -}}
  {{- end -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $_ := set $buildSecurityParams "sni" $proxy.SNI -}}
  {{- end -}}
  {{- if $SkipVerify -}}
    {{- $_ := set $buildSecurityParams "allowInsecure" "1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.Fingerprint) "" -}}
    {{- $_ := set $buildSecurityParams "fp" $proxy.Fingerprint -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") -}}
    {{- $_ := set $buildSecurityParams "pbk" $proxy.RealityPublicKey -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityShortId) "") -}}
    {{- $_ := set $buildSecurityParams "sid" $proxy.RealityShortId -}}
  {{- end -}}

  {{- if eq $proxy.Type "shadowsocks" }}
  {{- $params := list -}}
  {{- /* Shadowsocks 特有的 obfs 插件参数 */ -}}
  {{- if ne (default "" $proxy.Obfs) "" -}}
    {{- $params = append $params (printf "obfs=%s" $proxy.Obfs) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsHost) "" -}}
    {{- $params = append $params (printf "obfs-host=%s" $proxy.ObfsHost) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPath) "" -}}
    {{- $params = append $params (printf "obfs-uri=%s" ($proxy.ObfsPath | urlquery)) -}}
  {{- end -}}
  {{- /* 使用公共传输层配置 */ -}}
  {{- range $key, $val := $buildTransportParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- /* 使用公共安全层配置 */ -}}
  {{- range $key, $val := $buildSecurityParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- /* 添加公共参数 */ -}}
  {{- $params = append $params $common }}
ss://{{ printf "%s:%s" (default "aes-128-gcm" $proxy.Method) $password | b64enc }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if eq $proxy.Type "vmess" }}
  {{- $vmessDict := dict "v" "2" "ps" $proxy.Name "add" $proxy.Server "port" (printf "%d" $proxy.Port) "id" $password "aid" "0" "net" "tcp" "type" "none" -}}
  {{- if hasKey $buildTransportParams "type" -}}
    {{- $_ := set $vmessDict "net" (index $buildTransportParams "type") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "host" -}}
    {{- $_ := set $vmessDict "host" (index $buildTransportParams "host") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "path" -}}
    {{- $_ := set $vmessDict "path" (index $buildTransportParams "path") -}}
  {{- end -}}
  {{- if and (eq $transport "grpc") (hasKey $buildTransportParams "serviceName") -}}
    {{- $_ := set $vmessDict "path" (index $buildTransportParams "serviceName") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "mode" -}}
    {{- $_ := set $vmessDict "xhttpMode" (index $buildTransportParams "mode") -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "extra" -}}
    {{- $_ := set $vmessDict "xhttpExtra" (index $buildTransportParams "extra") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "security" -}}
    {{- $_ := set $vmessDict "tls" (index $buildSecurityParams "security") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "sni" -}}
    {{- $_ := set $vmessDict "sni" (index $buildSecurityParams "sni") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "fp" -}}
    {{- $_ := set $vmessDict "fp" (index $buildSecurityParams "fp") -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "allowInsecure" -}}
    {{- $_ := set $vmessDict "skip-cert-verify" true -}}
  {{- end }}
vmess://{{ $vmessDict | toJson | b64enc }}
  {{- else if eq $proxy.Type "vless" }}
  {{- $params := list -}}
  {{- /* 1. Encryption 加密参数 */ -}}
  {{- $encryption := default "none" $proxy.Encryption -}}
  {{- if eq $encryption "none" -}}
    {{- $params = append $params "encryption=none" -}}
  {{- else -}}
    {{- $encParts := list -}}
    {{- $encParts = append $encParts $encryption -}}
    {{- if ne (default "" $proxy.EncryptionMode) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionMode -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionRtt) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionRtt -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionClientPadding) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionClientPadding -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionPassword) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionPassword -}}
    {{- end -}}
    {{- $params = append $params (printf "encryption=%s" (join "." $encParts)) -}}
  {{- end -}}
  {{- /* 2. Flow 流控参数 */ -}}
  {{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") -}}
    {{- $params = append $params (printf "flow=%s" $proxy.Flow) -}}
  {{- end -}}
  {{- /* 3. Security 安全参数 */ -}}
  {{- if hasKey $buildSecurityParams "security" -}}
    {{- $params = append $params (printf "security=%s" (index $buildSecurityParams "security")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "sni" -}}
    {{- $params = append $params (printf "sni=%s" (index $buildSecurityParams "sni")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "fp" -}}
    {{- $params = append $params (printf "fp=%s" (index $buildSecurityParams "fp")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "allowInsecure" -}}
    {{- $params = append $params "allowInsecure=1" -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "pbk" -}}
    {{- $params = append $params (printf "pbk=%s" (index $buildSecurityParams "pbk")) -}}
  {{- end -}}
  {{- if hasKey $buildSecurityParams "sid" -}}
    {{- $params = append $params (printf "sid=%s" (index $buildSecurityParams "sid")) -}}
  {{- end -}}
  {{- /* 4. Transport 传输层参数 */ -}}
  {{- if hasKey $buildTransportParams "type" -}}
    {{- $params = append $params (printf "type=%s" (index $buildTransportParams "type")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "host" -}}
    {{- $params = append $params (printf "host=%s" (index $buildTransportParams "host")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "path" -}}
    {{- $params = append $params (printf "path=%s" (index $buildTransportParams "path")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "serviceName" -}}
    {{- $params = append $params (printf "serviceName=%s" (index $buildTransportParams "serviceName")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "mode" -}}
    {{- $params = append $params (printf "mode=%s" (index $buildTransportParams "mode")) -}}
  {{- end -}}
  {{- if hasKey $buildTransportParams "extra" -}}
    {{- $params = append $params (printf "extra=%s" (index $buildTransportParams "extra")) -}}
  {{- end -}}
  {{- /* 5. Common 通用参数 */ -}}
  {{- $params = append $params $common }}
vless://{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if eq $proxy.Type "trojan" }}
  {{- $params := list -}}
  {{- range $key, $val := $buildTransportParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- range $key, $val := $buildSecurityParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- $params = append $params $common }}
trojan://{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  {{- else if eq $proxy.Type "tuic" }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.CongestionController) "" -}}
    {{- $params = append $params (printf "congestion_controller=%s" $proxy.CongestionController) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.UDPRelayMode) "" -}}
    {{- $params = append $params (printf "udp_relay_mode=%s" $proxy.UDPRelayMode) -}}
  {{- end -}}
  {{- if $proxy.ReduceRtt -}}
    {{- $params = append $params "reduce_rtt=1" -}}
  {{- end -}}
  {{- if $proxy.DisableSNI -}}
    {{- $params = append $params "disable_sni=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "allow_insecure=1" -}}
  {{- end -}}
  {{- $params = append $params $common }}
tuic://{{ default "" $proxy.ServerKey }}:{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if eq $proxy.Type "anytls" }}
  {{- $params := list -}}
  {{- /* 使用公共传输层配置 */ -}}
  {{- range $key, $val := $buildTransportParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- /* 使用公共安全层配置 */ -}}
  {{- range $key, $val := $buildSecurityParams -}}
    {{- $params = append $params (printf "%s=%s" $key $val) -}}
  {{- end -}}
  {{- $params = append $params $common }}
anytls://{{ $password }}@{{ $server }}:{{ $proxy.Port }}?{{ join "&" $params }}#{{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "http") (eq $proxy.Type "https") }}
  {{- $user := default $password $proxy.Username }}
http{{- if eq $proxy.Type "https" -}}s{{- end -}}://{{- if or (ne (default "" $user) "") (ne (default "" $password) "") -}}{{ $user }}:{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}#{{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "socks") (eq $proxy.Type "socks5") (eq $proxy.Type "socks5-tls") }}
  {{- $user := default $password $proxy.Username }}
socks5://{{- if or (ne (default "" $user) "") (ne (default "" $password) "") -}}{{ $user }}:{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}{{- if eq $proxy.Type "socks5-tls" }}?tls=1{{- end }}#{{ $proxy.Name }}
  {{- end }}
{{- end }}
$alpha_subscription_template$, 'base64', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (3, 'Clash', '', 'Clash and Mihomo YAML subscription template.', 'clash://install-config?url=${encodeURIComponent(url)}&name=${encodeURIComponent(name)}', 'Clash', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "anytls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "clash" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = printf "%q" $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %q" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

mode: rule
ipv6: true
allow-lan: true
bind-address: '*'
mixed-port: 10808
log-level: info
unified-delay: true
tcp-concurrent: true
external-controller: '0.0.0.0:9090'
global-client-fingerprint: chrome
tun:
  enable: true
  stack: system
  auto-route: true
dns:
  enable: true
  cache-algorithm: arc
  listen: '0.0.0.0:1053'
  ipv6: true
  use-hosts: true
  use-system-hosts: true
  respect-rules: false
  enhanced-mode: fake-ip
  fake-ip-range: 198.18.0.1/16
  fake-ip-filter:
    - '*.lan'
    - 'localhost'
    - 'lens.l.google.com'
    - '*.srv.nintendo.net'
    - '*.stun.playstation.net'
    - 'xbox.*.*.microsoft.com'
    - '*.xboxlive.com'
    - '*.msftncsi.com'
    - '*.msftconnecttest.com'
    - 'time.*.com'
  default-nameserver:
    - 223.5.5.5
    - 119.29.29.29
  nameserver:
    - https://cloudflare-dns.com/dns-query
    - https://dns.google/dns-query
  fallback:
    - tls://1.1.1.1
    - tls://8.8.8.8
  proxy-server-nameserver:
    - https://dns.alidns.com/dns-query
    - https://doh.pub/dns-query
  direct-nameserver:
    - system
    - https://dns.alidns.com/dns-query
    - https://doh.pub/dns-query
  direct-nameserver-follow-policy: false
  fallback-filter:
    geoip: true
    geoip-code: CN
    geosite:
      - gfw
      - youtube
    domain:
      - '+.google.com'
      - '+.facebook.com'
      - '+.twitter.com'
      - '+.telegram.org'

proxies:
{{- range $proxy := $supportedProxies }}
  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

{{- if eq $proxy.Type "shadowsocks" }}
- name: {{ $proxy.Name | quote }}
  type: ss
  server: {{ $server }}
  port: {{ $proxy.Port }}
  cipher: {{ default "aes-128-gcm" $proxy.Method }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.Obfs) "" }}
  plugin: obfs
  plugin-opts:
    mode: {{ $proxy.Obfs }}
    host: {{ default "" $proxy.ObfsHost }}
  {{- end }}
{{- else if eq $proxy.Type "vmess" }}
- name: {{ $proxy.Name | quote }}
  type: vmess
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ $password }}
  alterId: 0
  cipher: auto
  udp: true
  tfo: true
  {{- if or (eq $proxy.Transport "websocket") (eq $proxy.Transport "ws") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "vless" }}
  {{- $encryptionStr := "" -}}
  {{- $encryption := default "none" $proxy.Encryption -}}
  {{- if eq $encryption "none" -}}
    {{- $encryptionStr = "none" -}}
  {{- else -}}
    {{- $encParts := list -}}
    {{- $encParts = append $encParts $encryption -}}
    {{- if ne (default "" $proxy.Encryption_Mode) "" -}}
      {{- $encParts = append $encParts $proxy.Encryption_Mode -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionRtt) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionRtt -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionClientPadding) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionClientPadding -}}
    {{- end -}}
    {{- if ne (default "" $proxy.EncryptionPassword) "" -}}
      {{- $encParts = append $encParts $proxy.EncryptionPassword -}}
    {{- end -}}
    {{- $encryptionStr = join "." $encParts -}}
  {{- end }}
- name: {{ $proxy.Name | quote }}
  type: vless
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ $password }}
  udp: true
  tfo: true
  encryption: {{ $encryptionStr }}
  {{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}
  flow: {{ $proxy.Flow }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "httpupgrade" }}
  network: httpupgrade
  httpupgrade-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "xhttp" }}
  network: xhttp
  xhttp-opts:
{{ xhttpMihomoOpts $proxy 4 | trimSuffix "\n" }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  servername: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "trojan" }}
- name: {{ $proxy.Name | quote }}
  type: trojan
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}
  tls: true
  {{- end }}
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $SkipVerify }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}
  reality-opts:
    public-key: {{ $proxy.RealityPublicKey }}
    {{- if ne (default "" $proxy.RealityShortId) "" }}
    short-id: {{ $proxy.RealityShortId }}
    {{- end }}
  {{- end }}
  {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}
  network: ws
  ws-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: {{ $proxy.Host }}
    {{- end }}
  {{- else if eq $proxy.Transport "http" }}
  network: http
  http-opts:
    method: GET
    path: [{{ default "/" $proxy.Path | quote }}]
    {{- if ne (default "" $proxy.Host) "" }}
    headers:
      Host: [{{ $proxy.Host | quote }}]
    {{- end }}
  {{- else if eq $proxy.Transport "grpc" }}
  network: grpc
  grpc-opts:
    grpc-service-name: {{ default "grpc" $proxy.ServiceName }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
- name: {{ $proxy.Name | quote }}
  type: hysteria2
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.ObfsPassword) "" }}
  obfs: salamander
  obfs-password: {{ $proxy.ObfsPassword }}
  {{- end }}
  {{- if ne (default "" $proxy.HopPorts) "" }}
  ports: {{ $proxy.HopPorts }}
  {{- end }}
  {{- if ne (default 0 $proxy.HopInterval) 0 }}
  hop-interval: {{ $proxy.HopInterval }}
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.UpMbps)) "" }}
  up: "{{ $proxy.UpMbps }} Mbps"
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.DownMbps)) "" }}
  down: "{{ $proxy.DownMbps }} Mbps"
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "tuic" }}
- name: {{ $proxy.Name | quote }}
  type: tuic
  server: {{ $server }}
  port: {{ $proxy.Port }}
  uuid: {{ default "" $proxy.ServerKey }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if $proxy.DisableSNI }}
  disable-sni: true
  {{- end }}
  {{- if $proxy.ReduceRtt }}
  reduce-rtt: true
  {{- end }}
  {{- if ne (default "" $proxy.UDPRelayMode) "" }}
  udp-relay-mode: {{ $proxy.UDPRelayMode }}
  {{- end }}
  {{- if ne (default "" $proxy.CongestionController) "" }}
  congestion-controller: {{ $proxy.CongestionController }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else if eq $proxy.Type "wireguard" }}
- name: {{ $proxy.Name | quote }}
  type: wireguard
  server: {{ $server }}
  port: {{ $proxy.Port }}
  private-key: {{ default "" $proxy.ServerKey }}
  public-key: {{ default "" $proxy.RealityPublicKey }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.Path) "" }}
  preshared-key: {{ $proxy.Path }}
  {{- end }}
  {{- if ne (default "" $proxy.RealityServerAddr) "" }}
  ip: {{ $proxy.RealityServerAddr }}
  {{- end }}
  {{- if ne (default 0 $proxy.RealityServerPort) 0 }}
  ipv6: {{ $proxy.RealityServerPort }}
  {{- end }}
{{- else if eq $proxy.Type "anytls" }}
- name: {{ $proxy.Name | quote }}
  type: anytls
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.Fingerprint) "" }}
  client-fingerprint: {{ $proxy.Fingerprint }}
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
{{- else }}
- name: {{ $proxy.Name | quote }}
  type: {{ $proxy.Type }}
  server: {{ $server }}
  port: {{ $proxy.Port }}
  udp: true
  tfo: true
{{- end }}
{{- end }}

proxy-groups:
  - { name: 🚀 Proxy, type: select, proxies: [🌏 Auto, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🍎 Apple, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🔍 Google, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🪟 Microsoft, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 📺 GlobalMedia, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 📟 Telegram, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🤖 AI, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🪙 Crypto, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🎮 Game, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🇨🇳 China, type: select, proxies: [🎯 Direct, 🚀 Proxy, {{ $proxyNames }}] }
  - { name: 🎯 Direct, type: select, proxies: [DIRECT], hidden: true }
  - { name: 🐠 Final, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🌏 Auto, type: url-test, proxies: [{{ $proxyNames }}] }

rules:
  - RULE-SET, Apple, 🍎 Apple
  - RULE-SET, Google, 🔍 Google
  - RULE-SET, Microsoft, 🪟 Microsoft
  - RULE-SET, Github, 🪟 Microsoft
  - RULE-SET, HBO, 📺 GlobalMedia
  - RULE-SET, Disney, 📺 GlobalMedia
  - RULE-SET, TikTok, 📺 GlobalMedia
  - RULE-SET, Netflix, 📺 GlobalMedia
  - RULE-SET, GlobalMedia, 📺 GlobalMedia
  - RULE-SET, Telegram, 📟 Telegram
  - RULE-SET, OpenAI, 🤖 AI
  - RULE-SET, Gemini, 🤖 AI
  - RULE-SET, Copilot, 🤖 AI
  - RULE-SET, Claude, 🤖 AI
  - RULE-SET, Crypto, 🪙 Crypto
  - RULE-SET, Cryptocurrency, 🪙 Crypto
  - RULE-SET, Game, 🎮 Game
  - RULE-SET, Global, 🚀 Proxy
  - RULE-SET, ChinaMax, 🇨🇳 China
  - RULE-SET, Lan, 🎯 Direct
  - GEOIP, CN, 🇨🇳 China
  - MATCH, 🐠 Final

rule-providers:
  Apple:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Apple/Apple_Classical_No_Resolve.yaml
    interval: 86400
  Google:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Google/Google_No_Resolve.yaml
    interval: 86400
  Microsoft:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Microsoft/Microsoft.yaml
    interval: 86400
  Github:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/GitHub/GitHub.yaml
    interval: 86400
  HBO:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/HBO/HBO.yaml
    interval: 86400
  Disney:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Disney/Disney.yaml
    interval: 86400
  TikTok:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/TikTok/TikTok.yaml
    interval: 86400
  Netflix:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Netflix/Netflix.yaml
    interval: 86400
  GlobalMedia:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/GlobalMedia/GlobalMedia_Classical_No_Resolve.yaml
    interval: 86400
  Telegram:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Telegram/Telegram_No_Resolve.yaml
    interval: 86400
  OpenAI:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/OpenAI/OpenAI.yaml
    interval: 86400
  Gemini:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Gemini/Gemini.yaml
    interval: 86400
  Copilot:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Copilot/Copilot.yaml
    interval: 86400
  Claude:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Claude/Claude.yaml
    interval: 86400
  Crypto:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Crypto/Crypto.yaml
    interval: 86400
  Cryptocurrency:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Cryptocurrency/Cryptocurrency.yaml
    interval: 86400
  Game:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Game/Game.yaml
    interval: 86400
  Global:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Global/Global_Classical_No_Resolve.yaml
    interval: 86400
  ChinaMax:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/ChinaMax/ChinaMax_Classical_No_Resolve.yaml
    interval: 86400
  Lan:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Lan/Lan.yaml
    interval: 86400

url-rewrite:
  - ^https?:\/\/(www.)?g\.cn https://www.google.com 302
  - ^https?:\/\/(www.)?google\.cn https://www.google.com 302
$alpha_subscription_template$, 'yaml', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (4, 'SingBox', '', 'sing-box 1.12 JSON subscription template.', 'sing-box://import-remote-profile?url=${encodeURIComponent(url)}#${encodeURIComponent(name)}', 'sing-box', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- $isSupported := false -}}
  {{- if or (eq $proxy.Type "shadowsocks") (eq $proxy.Type "vmess") (eq $proxy.Type "trojan") (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") (eq $proxy.Type "tuic") (eq $proxy.Type "anytls") -}}
    {{- $isSupported = true -}}
  {{- else if eq $proxy.Type "vless" -}}
    {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") (eq $proxy.Transport "grpc") (eq $proxy.Transport "tcp") (not $proxy.Transport) -}}
      {{- $isSupported = true -}}
    {{- end -}}
  {{- end -}}
  {{- if $isSupported -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- define "AllNodeNames" -}}
{{- $sortConfig := dict "Sort" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field, $order := $sortConfig -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- $isSupported := false -}}
  {{- if or (eq .Type "shadowsocks") (eq .Type "vmess") (eq .Type "trojan") (eq .Type "hysteria2") (eq .Type "hy2") (eq .Type "tuic") (eq .Type "anytls") -}}
    {{- $isSupported = true -}}
  {{- else if eq .Type "vless" -}}
    {{- if or (eq .Transport "ws") (eq .Transport "websocket") (eq .Transport "grpc") (eq .Transport "tcp") (not .Transport) -}}
      {{- $isSupported = true -}}
    {{- end -}}
  {{- end -}}
  {{- if $isSupported -}}
    {{- $supportedProxies = append $supportedProxies . -}}
  {{- end -}}
{{- end -}}
{{- $first := true -}}
{{- range $supportedProxies -}}
  {{- if $first -}}
    "{{ .Name }}"
    {{- $first = false -}}
  {{- else -}}
    , "{{ .Name }}"
  {{- end -}}
{{- end -}}
{{- end -}}

{{- define "NodeOutbound" -}}
{{- $proxy := .proxy -}}
{{- $server := $proxy.Server -}}
{{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
  {{- $server = printf "[%s]" $server -}}
{{- end -}}
{{- $port := $proxy.Port -}}
{{- $name := $proxy.Name -}}
{{- $pwd := $.UserInfo.Password -}}
{{- $svc := $proxy.ServiceName }}
{{- $tlsRequired := or (eq $proxy.Type "trojan") (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") (eq $proxy.Type "tuic") (eq $proxy.Type "anytls") -}}
{{- $tlsEnabled := or $tlsRequired $proxy.TLS (eq $proxy.Security "tls") (eq $proxy.Security "reality") -}}
{{- $sni := $proxy.SNI -}}
{{- if and (eq (default "" $sni) "") $tlsEnabled -}}
  {{- $sni = $server -}}
{{- end -}}

{{- $tlsOpts := "" -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if $proxy.Fingerprint -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

{{- $transportOpts := "" -}}
{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") -}}
  {{- $wsPath := default "/" $proxy.Path -}}
  {{- $transportOpts = printf "\"transport\": {\"type\": \"ws\", \"path\": \"%s\"" $wsPath -}}
  {{- if $proxy.Host -}}
    {{- $transportOpts = printf "%s, \"headers\": {\"Host\": \"%s\"}" $transportOpts ($proxy.Host) -}}
  {{- end -}}
  {{- $transportOpts = printf "%s}" $transportOpts -}}
{{- else if eq $proxy.Transport "grpc" -}}
  {{- $grpcService := default "grpc" $svc -}}
  {{- $transportOpts = printf "\"transport\": {\"type\": \"grpc\", \"service_name\": \"%s\"" $grpcService -}}
  {{- if $proxy.GrpcIdleTimeout -}}{{- $transportOpts = printf "%s, \"idle_timeout\": \"%ds\"" $transportOpts $proxy.GrpcIdleTimeout -}}{{- end -}}
  {{- if $proxy.GrpcHealthCheckTimeout -}}{{- $transportOpts = printf "%s, \"ping_timeout\": \"%ds\"" $transportOpts $proxy.GrpcHealthCheckTimeout -}}{{- end -}}
  {{- if $proxy.GrpcPermitWithoutStream -}}{{- $transportOpts = printf "%s, \"permit_without_stream\": true" $transportOpts -}}{{- end -}}
  {{- $transportOpts = printf "%s}" $transportOpts -}}
{{- end -}}

{{- if eq $proxy.Type "shadowsocks" -}}
  {{- $method := default "aes-128-gcm" $proxy.Method -}}
  {{- $password := $pwd -}}
  {{- if $proxy.ServerKey -}}
    {{- $needBytes := ternary 16 32 (eq $proxy.Method "2022-blake3-aes-128-gcm") -}}
    {{- $cutLen := min $needBytes (len $pwd) | int -}}
    {{- $userCut := $pwd | trunc $cutLen -}}
    {{- $serverB64 := b64enc $proxy.ServerKey -}}
    {{- $userB64 := b64enc $userCut -}}
    {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
  {{- end -}}
{ "type": "shadowsocks", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "method": "{{ $method }}", "password": "{{ $password }}" }

{{- else if eq $proxy.Type "trojan" -}}
{ "type": "trojan", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "password": "{{ $pwd }}"{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}, {{ $tlsOpts }} }

{{- else if eq $proxy.Type "vless" -}}
{{- $flowOpts := "" -}}
{{- if and $proxy.Flow (ne $proxy.Flow "none") -}}
  {{- $flowOpts = printf ", \"flow\": \"%s\"" ($proxy.Flow) -}}
{{- end -}}
{ "type": "vless", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $pwd }}"{{ $flowOpts }}{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}{{ if $tlsOpts }}, {{ $tlsOpts }}{{ end }} }

{{- else if eq $proxy.Type "vmess" -}}
{{- $vmessTLS := "" -}}
{{- if and $tlsOpts (ne $proxy.Transport "tcp") -}}
  {{- $vmessTLS = $tlsOpts -}}
{{- end -}}
{ "type": "vmess", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $pwd }}", "security": "auto"{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}{{ if $vmessTLS }}, {{ $vmessTLS }}{{ end }} }

{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- $obfsOpts := "" -}}
{{- if $proxy.ObfsPassword -}}
  {{- $obfsOpts = printf "\"obfs\": { \"type\": \"salamander\", \"password\": \"%s\" }" ($proxy.ObfsPassword) -}}
{{- end -}}
{{- $hopPortsOpts := "" -}}
{{- if $proxy.HopPorts -}}
  {{- $hopPortsOpts = printf ", \"ports\": \"%s\"" ($proxy.HopPorts) -}}
{{- end -}}
{{- $hopIntervalOpts := "" -}}
{{- if $proxy.HopInterval -}}
  {{- $hopIntervalOpts = printf ", \"hop_interval\": %v" $proxy.HopInterval -}}
{{- end -}}
{ "type": "hysteria2", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "password": "{{ $pwd }}"{{ if $obfsOpts }}, {{ $obfsOpts }}{{ end }}{{ $hopPortsOpts }}{{ $hopIntervalOpts }}, {{ $tlsOpts }} }

{{- else if eq $proxy.Type "tuic" -}}
{{- $tuicServerKey := $proxy.ServerKey -}}
{{- $tuicOpts := "" -}}
{{- if $proxy.DisableSNI -}}
  {{- $tuicOpts = printf "%s, \"disable_sni\": %v" $tuicOpts $proxy.DisableSNI -}}
{{- end -}}
{{- if $proxy.ReduceRtt -}}
  {{- $tuicOpts = printf "%s, \"reduce_rtt\": %v" $tuicOpts $proxy.ReduceRtt -}}
{{- end -}}
{{- if $proxy.UDPRelayMode -}}
  {{- $tuicOpts = printf "%s, \"udp_relay_mode\": \"%s\"" $tuicOpts ($proxy.UDPRelayMode) -}}
{{- end -}}
{{- if $proxy.CongestionController -}}
  {{- $tuicOpts = printf "%s, \"congestion_control\": \"%s\"" $tuicOpts ($proxy.CongestionController) -}}
{{- end -}}
{ "type": "tuic", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $tuicServerKey }}", "password": "{{ $pwd }}"{{ $tuicOpts }}, "alpn": ["h3"], {{ $tlsOpts }} }

{{- else if eq $proxy.Type "anytls" -}}
{{- $anytlsOpts := "" -}}
{{- if $proxy.Method -}}
  {{- $anytlsOpts = printf "%s, \"method\": \"%s\"" $anytlsOpts ($proxy.Method) -}}
{{- end -}}
{{- if $proxy.ObfsPassword -}}
  {{- $anytlsOpts = printf "%s, \"obfs\": \"%s\"" $anytlsOpts ($proxy.ObfsPassword) -}}
{{- end -}}
{{- if $proxy.Path -}}
  {{- $anytlsOpts = printf "%s, \"path\": \"%s\"" $anytlsOpts ($proxy.Path) -}}
{{- end -}}
{{- if $proxy.Host -}}
  {{- $anytlsOpts = printf "%s, \"host\": \"%s\"" $anytlsOpts ($proxy.Host) -}}
{{- end -}}
{ "type": "anytls", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "password": "{{ $pwd }}"{{ $anytlsOpts }}{{ if $tlsOpts }}, {{ $tlsOpts }}{{ end }} }

{{- else if eq $proxy.Type "wireguard" -}}
{{- $wgPrivateKey := $proxy.ServerKey -}}
{{- $wgPublicKey := $proxy.RealityPublicKey -}}
{{- $wgPreSharedOpts := "" -}}
{{- if $proxy.Path -}}
  {{- $wgPreSharedOpts = printf ", \"pre_shared_key\": \"%s\"" ($proxy.Path) -}}
{{- end -}}
{{- $wgLocalAddressOpts := "" -}}
{{- if $proxy.RealityServerAddr -}}
  {{- $wgLocalAddressOpts = printf ", \"local_address\": [\"%s\"]" ($proxy.RealityServerAddr) -}}
{{- end -}}
{ "type": "wireguard", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "private_key": "{{ $wgPrivateKey }}", "peer_public_key": "{{ $wgPublicKey }}"{{ $wgPreSharedOpts }}{{ $wgLocalAddressOpts }} }

{{- else if or (eq $proxy.Type "http") (eq $proxy.Type "https") -}}
{{- $httpsTLSOpts := "" -}}
{{- if and (eq $proxy.Type "https") $tlsOpts -}}
  {{- $httpsTLSOpts = printf ", %s" $tlsOpts -}}
{{- end -}}
{ "type": "http", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "username": "{{ $pwd }}", "password": "{{ $pwd }}"{{ $httpsTLSOpts }} }

{{- else if or (eq $proxy.Type "socks") (eq $proxy.Type "socks5") -}}
{ "type": "socks", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "version": "5", "username": "{{ $pwd }}", "password": "{{ $pwd }}" }

{{- else -}}
{ "type": "direct", "tag": "{{ $name }}" }
{{- end -}}
{{- end -}}

{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "experimental": {
    "cache_file": {
      "enabled": true,
      "store_fakeip": true,
      "store_rdrc": true
    },
    "clash_api": {
      "external_controller": "127.0.0.1:9090",
      "access_control_allow_origin": [
        "http://127.0.0.1",
        "https://yacd.metacubex.one",
        "https://metacubex.github.io",
        "https://metacubexd.pages.dev",
        "https://board.zash.run.place"
      ]
    }
  },
  "dns": {
    "independent_cache": true,
    "servers": [
      {
        "tag": "google",
        "type": "https",
        "server": "8.8.8.8",
        "detour": "节点选择"
      },
      {
        "tag": "ali",
        "type": "https",
        "server": "223.5.5.5"
      },
      {
        "tag": "fakeip",
        "type": "fakeip",
        "inet4_range": "198.18.0.0/15",
        "inet6_range": "fc00::/18"
      }
    ],
    "rules": [
      {
        "clash_mode": "Direct",
        "server": "ali"
      },
      {
        "clash_mode": "Global",
        "server": "google"
      },
      {
        "query_type": [
          "A",
          "AAAA"
        ],
        "server": "fakeip"
      },
      {
        "rule_set": "geosite-cn",
        "server": "ali"
      }
    ]
  },
  "inbounds": [
    {
      "type": "tun",
      "address": [
        "172.18.0.1/30",
        "fdfe:dcba:9876::1/126"
      ],
      "auto_route": true,
      "strict_route": true
    },
    {
      "type": "mixed",
      "listen": "::",
      "listen_port": 7890
    }
  ],
  "outbounds": [
    {
      "tag": "节点选择",
      "type": "selector",
      "outbounds": [{{ template "AllNodeNames" . }}, "直连"]
    },
    {
      "tag": "Github",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连",
        {{ template "AllNodeNames" . }}
      ]
    },
    {
      "tag": "Google",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连",
        {{ template "AllNodeNames" . }}
      ]
    },
    {
      "tag": "Microsoft",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连",
        {{ template "AllNodeNames" . }}
      ]
    },
    {
      "tag": "OpenAI",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连",
        {{ template "AllNodeNames" . }}
      ]
    },
    {
      "tag": "Telegram",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连",
        {{ template "AllNodeNames" . }}
      ]
    },
    {
      "tag": "Twitter",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连",
        {{ template "AllNodeNames" . }}
      ]
    },
    {
      "tag": "Youtube",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连",
        {{ template "AllNodeNames" . }}
      ]
    },
    {
      "tag": "国内",
      "type": "selector",
      "outbounds": [
        "直连",
        "节点选择",
        {{ template "AllNodeNames" . }}
      ]
    },
    {{- range $i, $proxy := $supportedProxies }}
    {{ if $i }},{{ end }}
    {{ template "NodeOutbound" (dict "proxy" $proxy "UserInfo" $.UserInfo) }}
    {{- end }}
    {{- if gt (len $supportedProxies) 0 }},{{ end }}
    {
      "tag": "直连",
      "type": "direct"
    }
  ],
  "route": {
    "default_domain_resolver": {
      "server": "ali"
    },
    "auto_detect_interface": true,
    "rules": [
      {
        "action": "sniff"
      },
      {
        "protocol": "dns",
        "action": "hijack-dns"
      },
      {
        "ip_is_private": true,
        "outbound": "直连"
      },
      {
        "rule_set": "anti-ad",
        "clash_mode": "Rule",
        "action": "reject"
      },
      {
        "clash_mode": "Direct",
        "outbound": "直连"
      },
      {
        "clash_mode": "Global",
        "outbound": "节点选择"
      },
      {
        "rule_set": "geosite-github",
        "outbound": "Github"
      },
      {
        "rule_set": [
          "geoip-google",
          "geosite-google"
        ],
        "outbound": "Google"
      },
      {
        "rule_set": "geosite-microsoft",
        "outbound": "Microsoft"
      },
      {
        "rule_set": "geosite-openai",
        "outbound": "OpenAI"
      },
      {
        "rule_set": [
          "geoip-telegram",
          "geosite-telegram"
        ],
        "outbound": "Telegram"
      },
      {
        "rule_set": [
          "geoip-twitter",
          "geosite-twitter"
        ],
        "outbound": "Twitter"
      },
      {
        "rule_set": "geosite-youtube",
        "outbound": "Youtube"
      },
      {
        "rule_set": [
          "geoip-cn",
          "geosite-cn"
        ],
        "outbound": "国内"
      }
    ],
    "rule_set": [
      {
        "tag": "anti-ad",
        "type": "remote",
        "format": "binary",
        "url": "https://anti-ad.net/anti-ad-sing-box.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-github",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/github.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-google",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/google.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-google",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/google.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-microsoft",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/microsoft.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-openai",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/openai.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-telegram",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/telegram.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-telegram",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/telegram.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-twitter",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/twitter.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-twitter",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/twitter.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-youtube",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/youtube.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-cn",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/cn.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-cn",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/cn.srs",
        "download_detour": "直连"
      }
    ]
  }
}
$alpha_subscription_template$, 'json', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (5, 'Surge', '', 'Surge managed configuration template.', 'surge:///install-config?url=${encodeURIComponent(url)}', 'Surge', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "wireguard" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %s" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

#!MANAGED-CONFIG {{ .UserInfo.SubscribeURL }} interval=86400 strict=true

[General]
# 日志级别
loglevel = notify

# 外部控制器访问
external-controller-access = perlnk@0.0.0.0:6170

# 网络设置
exclude-simple-hostnames = true
show-error-page-for-reject = true
udp-priority = true
udp-policy-not-supported-behaviour = reject
ipv6 = true
ipv6-vif = auto

# 连接测试
proxy-test-url = http://www.gstatic.com/generate_204
internet-test-url = http://www.gstatic.com/generate_204
test-timeout = 5

# DNS 设置
dns-server = system, 119.29.29.29, 223.5.5.5
encrypted-dns-server = https://dns.alidns.com/dns-query
hijack-dns = 8.8.8.8:53, 8.8.4.4:53, 1.1.1.1:53, 1.0.0.1:53

# 跳过代理
skip-proxy = 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 127.0.0.0/8, localhost, *.local

# 真实 IP
always-real-ip = *.lan, lens.l.google.com, *.srv.nintendo.net, *.stun.playstation.net, *.xboxlive.com, xbox.*.*.microsoft.com, *.msftncsi.com, *.msftconnecttest.com

# Surge Mac 参数
http-listen = 0.0.0.0:6088
socks5-listen = 0.0.0.0:6089

# Surge iOS 参数（WiFi 共享）
allow-wifi-access = true
allow-hotspot-access = true
wifi-access-http-port = 6088
wifi-access-socks5-port = 6089

[Panel]
SubscribeInfo = title={{ .SiteName }} - {{ .SubscribeName }}, content=已用流量: {{ $used }} GiB/{{ $total }} GiB \n到期时间: {{ $ExpiredAt}}, style=info

[Proxy]
{{- range $proxy := $supportedProxies }}
  {{- $common := "udp-relay=true, tfo=true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- if eq $proxy.Type "shadowsocks" }}
{{ $proxy.Name }} = ss, {{ $server }}, {{ $proxy.Port }}, encrypt-method={{ default "aes-128-gcm" $proxy.Method }}, password={{ $password }}{{- if ne (default "" $proxy.Obfs) "" }}, obfs={{ $proxy.Obfs }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls=true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = vless, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow={{ $proxy.Flow }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint={{ $proxy.Fingerprint }}{{- end }}, {{ $common }}
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = hysteria2, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs=salamander, obfs-password={{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports={{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval={{ $proxy.HopInterval }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = tuic, {{ $server }}, {{ $proxy.Port }}, uuid={{ default "" $proxy.ServerKey }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni=true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt=true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode={{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller={{ $proxy.CongestionController }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = wireguard, {{ $server }}, {{ $proxy.Port }}, private-key={{ default "" $proxy.ServerKey }}, public-key={{ default "" $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip={{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6={{ $proxy.RealityServerPort }}{{- end }}, {{ $common }}
  {{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}, {{ $common }}
  {{- else }}
{{ $proxy.Name }} = {{ $proxy.Type }}, {{ $server }}, {{ $proxy.Port }}, {{ $common }}
  {{- end }}
{{- end }}

[Proxy Group]
# 主要策略组
🚀 Proxy = select, 🌏 Auto, 🎯 Direct, include-other-group=🇺🇳 Nodes
🍎 Apple = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
🔍 Google = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
🪟 Microsoft = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
📺 GlobalMedia = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
🤖 AI = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
🪙 Crypto = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
🎮 Game = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
📟 Telegram = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes
🇨🇳 China = select, 🎯 Direct, 🚀 Proxy, include-other-group=🇺🇳 Nodes
🐠 Final = select, 🚀 Proxy, 🎯 Direct, include-other-group=🇺🇳 Nodes

# 智能选择和节点组
🌏 Auto = smart, include-other-group=🇺🇳 Nodes
🎯 Direct = select, DIRECT, hidden=1
🇺🇳 Nodes = select, {{ $proxyNames }}, hidden=1

[Rule]
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Apple/Apple_All.list, 🍎 Apple
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Google/Google.list, 🔍 Google
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/GitHub/GitHub.list, 🪟 Microsoft
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Microsoft/Microsoft.list, 🪟 Microsoft
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/HBO/HBO.list, 📺 GlobalMedia
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Disney/Disney.list, 📺 GlobalMedia
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/TikTok/TikTok.list, 📺 GlobalMedia
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Netflix/Netflix.list, 📺 GlobalMedia
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/GlobalMedia/GlobalMedia_All_No_Resolve.list, 📺 GlobalMedia
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Telegram/Telegram.list, 📟 Telegram
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/OpenAI/OpenAI.list, 🤖 AI
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Gemini/Gemini.list, 🤖 AI
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Copilot/Copilot.list, 🤖 AI
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Claude/Claude.list, 🤖 AI
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Crypto/Crypto.list, 🪙 Crypto
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Cryptocurrency/Cryptocurrency.list, 🪙 Crypto
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Game/Game.list, 🎮 Game
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Global/Global_All_No_Resolve.list, 🚀 Proxy
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/ChinaMax/ChinaMax_All_No_Resolve.list, 🇨🇳 China
RULE-SET, https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Lan/Lan.list, 🎯 Direct

GEOIP, CN, 🇨🇳 China
FINAL, 🐠 Final, dns-failed

[URL Rewrite]
^https?:\/\/(www.)?g\.cn https://www.google.com 302
^https?:\/\/(www.)?google\.cn https://www.google.com 302
$alpha_subscription_template$, 'conf', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (6, 'Stash', '', 'Stash YAML subscription template.', 'stash://install-config?url=${encodeURIComponent(url)}&name=${encodeURIComponent(name)}', 'Stash', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "wireguard" true "anytls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "stash" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %s" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

mode: rule
log-level: info
dns:
  default-nameserver:
    - 223.5.5.5
    - 119.29.29.29
  nameserver:
    - system
    - https://dns.alidns.com/dns-query

proxies:
{{- range $proxy := $supportedProxies }}
  {{- $common := "udp: true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- if eq $proxy.Type "shadowsocks" }}
  - { name: {{ $proxy.Name | quote }}, type: ss, server: {{ $server }}, port: {{ $proxy.Port }}, cipher: {{ default "aes-128-gcm" $proxy.Method }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.Obfs) "" }}, plugin: obfs, plugin-opts: { mode: {{ $proxy.Obfs }}, host: {{ default "" $proxy.ObfsHost }} }{{- end }} }
  {{- else if eq $proxy.Type "vmess" }}
  - { name: {{ $proxy.Name | quote }}, type: vmess, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, alterId: 0, cipher: auto, {{ $common }}{{- if or (eq $proxy.Transport "websocket") (eq $proxy.Transport "ws") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls: true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else if eq $proxy.Type "vless" }}
  - { name: {{ $proxy.Name | quote }}, type: vless, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, {{ $common }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "xhttp" }}, network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}, reality-opts: { public-key: {{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, short-id: {{ $proxy.RealityShortId }}{{- end }} }{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow: {{ $proxy.Flow }}{{- end }} }
  {{- else if eq $proxy.Type "trojan" }}
  - { name: {{ $proxy.Name | quote }}, type: trojan, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }} }
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  {{- else if eq $proxy.Type "tuic" }}
  - { name: {{ $proxy.Name | quote }}, type: tuic, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ default "" $proxy.ServerKey }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni: true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt: true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode: {{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller: {{ $proxy.CongestionController }}{{- end }} }
  {{- else if eq $proxy.Type "wireguard" }}
  - { name: {{ $proxy.Name | quote }}, type: wireguard, server: {{ $server }}, port: {{ $proxy.Port }}, private-key: {{ default "" $proxy.ServerKey }}, public-key: {{ default "" $proxy.RealityPublicKey }}, {{ $common }}{{- if ne (default "" $proxy.Path) "" }}, preshared-key: {{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.RealityServerAddr) "" }}, ip: {{ $proxy.RealityServerAddr }}{{- end }}{{- if ne (default 0 $proxy.RealityServerPort) 0 }}, ipv6: {{ $proxy.RealityServerPort }}{{- end }} }
  {{- else if eq $proxy.Type "anytls" }}
  - { name: {{ $proxy.Name | quote }}, type: anytls, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else }}
  - { name: {{ $proxy.Name | quote }}, type: {{ $proxy.Type }}, server: {{ $server }}, port: {{ $proxy.Port }}, {{ $common }} }
  {{- end }}
{{- end }}

proxy-groups:
  - { name: 🚀 Proxy, type: select, proxies: [🌏 Auto, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🍎 Apple, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🔍 Google, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🪟 Microsoft, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 📺 GlobalMedia, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 📟 Telegram, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🤖 AI, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🪙 Crypto, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🎮 Game, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🇨🇳 China, type: select, proxies: [🎯 Direct, 🚀 Proxy, {{ $proxyNames }}] }
  - { name: 🎯 Direct, type: select, proxies: [DIRECT], hidden: true }
  - { name: 🐠 Final, type: select, proxies: [🚀 Proxy, 🎯 Direct, {{ $proxyNames }}] }
  - { name: 🌏 Auto, type: url-test, proxies: [{{ $proxyNames }}], url: 'http://www.gstatic.com/generate_204', interval: 300 }

rules:
  - RULE-SET, Apple, 🍎 Apple
  - RULE-SET, Google, 🔍 Google
  - RULE-SET, Microsoft, 🪟 Microsoft
  - RULE-SET, Github, 🪟 Microsoft
  - RULE-SET, HBO, 📺 GlobalMedia
  - RULE-SET, Disney, 📺 GlobalMedia
  - RULE-SET, TikTok, 📺 GlobalMedia
  - RULE-SET, Netflix, 📺 GlobalMedia
  - RULE-SET, GlobalMedia, 📺 GlobalMedia
  - RULE-SET, Telegram, 📟 Telegram
  - RULE-SET, OpenAI, 🤖 AI
  - RULE-SET, Gemini, 🤖 AI
  - RULE-SET, Copilot, 🤖 AI
  - RULE-SET, Claude, 🤖 AI
  - RULE-SET, Crypto, 🪙 Crypto
  - RULE-SET, Cryptocurrency, 🪙 Crypto
  - RULE-SET, Game, 🎮 Game
  - RULE-SET, Global, 🚀 Proxy
  - RULE-SET, ChinaMax, 🇨🇳 China
  - RULE-SET, Lan, 🎯 Direct
  - GEOIP, CN, 🇨🇳 China
  - MATCH, 🐠 Final

rule-providers:
  Apple:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Apple/Apple_Classical_No_Resolve.yaml
    interval: 86400
  Google:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Google/Google_No_Resolve.yaml
    interval: 86400
  Microsoft:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Microsoft/Microsoft.yaml
    interval: 86400
  Github:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/GitHub/GitHub.yaml
    interval: 86400
  HBO:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/HBO/HBO.yaml
    interval: 86400
  Disney:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Disney/Disney.yaml
    interval: 86400
  TikTok:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/TikTok/TikTok.yaml
    interval: 86400
  Netflix:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Netflix/Netflix.yaml
    interval: 86400
  GlobalMedia:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/GlobalMedia/GlobalMedia_Classical_No_Resolve.yaml
    interval: 86400
  Telegram:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Telegram/Telegram_No_Resolve.yaml
    interval: 86400
  OpenAI:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/OpenAI/OpenAI.yaml
    interval: 86400
  Gemini:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Gemini/Gemini.yaml
    interval: 86400
  Copilot:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Copilot/Copilot.yaml
    interval: 86400
  Claude:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Claude/Claude.yaml
    interval: 86400
  Crypto:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Crypto/Crypto.yaml
    interval: 86400
  Cryptocurrency:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Cryptocurrency/Cryptocurrency.yaml
    interval: 86400
  Game:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Game/Game.yaml
    interval: 86400
  Global:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Global/Global_Classical_No_Resolve.yaml
    interval: 86400
  ChinaMax:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/ChinaMax/ChinaMax_Classical_No_Resolve.yaml
    interval: 86400
  Lan:
    type: http
    behavior: classical
    format: yaml
    url: https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Lan/Lan.yaml
    interval: 86400

url-rewrite:
  - ^https?:\/\/(www.)?g\.cn https://www.google.com 302
  - ^https?:\/\/(www.)?google\.cn https://www.google.com 302
$alpha_subscription_template$, 'yaml', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (7, 'Surfboard', '', 'Surfboard configuration template.', 'surfboard:///install-config?url=${encodeURIComponent(url)}', 'Surfboard', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if or (eq $proxy.Type "shadowsocks") (eq $proxy.Type "vmess") (eq $proxy.Type "trojan") (eq $proxy.Type "http") (eq $proxy.Type "https") (eq $proxy.Type "socks5") -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- define "SurfboardProxy" -}}
{{- $proxy := .proxy -}}
{{- $server := $proxy.Server -}}
{{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
  {{- $server = printf "[%s]" $server -}}
{{- end -}}
{{- $port := $proxy.Port -}}
{{- $name := $proxy.Name -}}
{{- $pwd := $.UserInfo.Password -}}
{{- $sni := or $proxy.SNI $server -}}

{{- if eq $proxy.Type "shadowsocks" -}}
{{- $method := default "aes-128-gcm" $proxy.Method -}}
{{- $password := $pwd -}}
{{- if $proxy.ServerKey -}}
  {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
    {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
    {{- $pwdStr := printf "%s" $pwd -}}
    {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
    {{- $serverB64 := b64enc $proxy.ServerKey -}}
    {{- $userB64 := b64enc $userKey -}}
    {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
  {{- end -}}
{{- end -}}
{{- if ne (default "" $proxy.Obfs) "" -}}
{{ $name }} = ss, {{ $server }}, {{ $port }}, encrypt-method={{ $method }}, password={{ $password }}, obfs={{ $proxy.Obfs }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- if ne (default "" $proxy.ObfsPath) "" }}, obfs-uri={{ $proxy.ObfsPath }}{{- end }}, udp-relay=true
{{- else -}}
{{ $name }} = ss, {{ $server }}, {{ $port }}, encrypt-method={{ $method }}, password={{ $password }}, udp-relay=true
{{- end -}}
{{- else if eq $proxy.Type "vmess" -}}
{{- $wsPath := default "/" $proxy.Path -}}
{{- $wsHeaders := "" -}}
{{- if $proxy.Host -}}
  {{- $wsHeaders = printf ", ws-headers=Host:%s" $proxy.Host -}}
{{- end -}}
{{- $tlsOpts := "" -}}
{{- if $proxy.TLS -}}
  {{- $tlsOpts = ", tls=true" -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
  {{- end -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
  {{- end -}}
{{- end -}}
{{ $name }} = vmess, {{ $server }}, {{ $port }}, username={{ $pwd }}, udp-relay=true, ws=true, ws-path={{ $wsPath }}{{ $wsHeaders }}{{ $tlsOpts }}, vmess-aead=true
{{- else if eq $proxy.Type "trojan" -}}
{{- $wsOpts := "" -}}
{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") -}}
  {{- $wsPath := default "/" $proxy.Path -}}
  {{- $wsOpts = printf ", ws=true, ws-path=%s" $wsPath -}}
  {{- if $proxy.Host -}}
    {{- $wsOpts = printf "%s, ws-headers=Host:%s" $wsOpts $proxy.Host -}}
  {{- end -}}
{{- end -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}
{{- if $sni -}}
  {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
{{- end -}}
{{ $name }} = trojan, {{ $server }}, {{ $port }}, password={{ $pwd }}, udp-relay=true{{ $wsOpts }}{{ $tlsOpts }}
{{- else if eq $proxy.Type "http" -}}
{{ $name }} = http, {{ $server }}, {{ $port }}, {{ $pwd }}, {{ $pwd }}
{{- else if eq $proxy.Type "https" -}}
{{- $tlsOpts := ", tls=true" -}}
{{- if $proxy.AllowInsecure -}}
  {{- $tlsOpts = printf "%s, skip-cert-verify=true" $tlsOpts -}}
{{- end -}}
{{- if $sni -}}
  {{- $tlsOpts = printf "%s, sni=%s" $tlsOpts $sni -}}
{{- end -}}
{{ $name }} = https, {{ $server }}, {{ $port }}, {{ $pwd }}, {{ $pwd }}{{ $tlsOpts }}
{{- else if eq $proxy.Type "socks5" -}}
{{ $name }} = socks5, {{ $server }}, {{ $port }}, {{ $pwd }}, {{ $pwd }}, udp-relay=true
{{- end -}}
{{- end -}}

{{- define "AllProxyNames" -}}
{{- $sortConfig := dict "Sort" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- if or (eq $p.Type "shadowsocks") (eq $p.Type "vmess") (eq $p.Type "trojan") (eq $p.Type "http") (eq $p.Type "https") (eq $p.Type "socks5") -}}
    {{- $keyParts := list -}}
    {{- range $field, $order := $sortConfig -}}
      {{- $val := default "" (printf "%v" (index $p $field)) -}}
      {{- if or (eq $field "Sort") (eq $field "Port") -}}
        {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
      {{- end -}}
      {{- if eq $order "desc" -}}
        {{- $val = printf "~%s" $val -}}
      {{- end -}}
      {{- $keyParts = append $keyParts $val -}}
    {{- end -}}
    {{- $_ := set $byKey (join "|" $keyParts) $p -}}
  {{- end -}}
{{- end -}}
{{- $first := true -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $proxy := index $byKey $k -}}
  {{- if $first -}}
    {{ $proxy.Name }}
    {{- $first = false -}}
  {{- else -}}
    , {{ $proxy.Name }}
  {{- end -}}
{{- end -}}
{{- end -}}

#!MANAGED-CONFIG {{ .UserInfo.SubscribeURL }} interval=60 strict=true
# 订阅链接: {{ .UserInfo.SubscribeURL }}
# 流量用量: {{ $used }}GB / {{ $total }}GB
# 到期时间: {{ $ExpiredAt }}
# 更新时间: {{ now | date "2006-01-02 15:04:05" }}

[General]
# DNS服务器配置
dns-server = 114.114.114.114, 223.5.5.5, 8.8.8.8, 8.8.4.4, 9.9.9.9:9953, system

# DoH服务器配置
doh-server = https://doh.pub/dns-query, https://dns.alidns.com/dns-query, https://9.9.9.9/dns-query

# 跳过代理的地址范围
skip-proxy = 127.0.0.1, 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 100.64.0.0/10, 17.0.0.0/8, localhost, *.crashlytics.com, *.local, captive.apple.com, www.baidu.com

# 代理测试URL
proxy-test-url = http://www.gstatic.com/generate_204

# 直连测试URL
internet-test-url = http://www.gstatic.cn/generate_204

# 连接测试超时
test-timeout = 30

# 真实IP域名
always-real-ip = *.lan, *.localdomain, *.example, *.invalid, *.localhost, *.test, *.local, *.home.arpa, time.*.com, time.*.gov, time.*.edu.cn, time.*.apple.com, time1.*.com, time2.*.com, time3.*.com, time4.*.com, time5.*.com, time6.*.com, time7.*.com, ntp.*.com, ntp1.*.com, ntp2.*.com, ntp3.*.com, ntp4.*.com, ntp5.*.com, ntp6.*.com, ntp7.*.com, *.time.edu.cn, *.ntp.org.cn, +.pool.ntp.org, time1.cloud.tencent.com, music.163.com, *.music.163.com, *.126.net, musicapi.taihe.com, music.taihe.com, songsearch.kugou.com, trackercdn.kugou.com, *.kuwo.cn, api-jooxtt.sanook.com, api.joox.com, joox.com, y.qq.com, *.y.qq.com, streamoc.music.tc.qq.com, mobileoc.music.tc.qq.com, isure.stream.qqmusic.qq.com, dl.stream.qqmusic.qq.com, aqqmusic.tc.qq.com, amobile.music.tc.qq.com, *.xiami.com, *.music.migu.cn, music.migu.cn, *.msftconnecttest.com, *.msftncsi.com, msftconnecttest.com, msftncsi.com, localhost.ptlogin2.qq.com, localhost.sec.qq.com, +.srv.nintendo.net, +.stun.playstation.net, xbox.*.microsoft.com, *.*.xboxlive.com, +.battlenet.com.cn, +.wotgame.cn, +.wggames.cn, +.wowsgame.cn, +.wargaming.net, proxy.golang.org, stun.*.*, stun.*.*.*, +.stun.*.*, +.stun.*.*.*, +.stun.*.*.*.*, heartbeat.belkin.com, *.linksys.com, *.linksyssmartwifi.com, *.router.asus.com, mesu.apple.com, swscan.apple.com, swquery.apple.com, swdownload.apple.com, swcdn.apple.com, swdist.apple.com, lens.l.google.com, stun.l.google.com, +.nflxvideo.net, *.square-enix.com, *.finalfantasyxiv.com, *.ffxiv.com, *.mcdn.bilivideo.cn

# HTTP代理监听端口
http-listen = 0.0.0.0:1234

# SOCKS5代理监听端口
socks5-listen = 127.0.0.1:1235

# UDP策略
udp-policy-not-supported-behaviour = DIRECT

[Host]
localhost = 127.0.0.1

[Proxy]
# 内置策略
DIRECT = direct
REJECT = reject

{{- range $proxy := $supportedProxies }}
{{ template "SurfboardProxy" (dict "proxy" $proxy "UserInfo" $.UserInfo) }}
{{- end }}

[Proxy Group]
# 主要策略组
{{- if gt (len $supportedProxies) 0 }}
🔰节点选择 = select, {{ template "AllProxyNames" . }}, DIRECT

⚓️其他流量 = select, 🔰节点选择, 🚀直接连接, {{ template "AllProxyNames" . }}
{{- else }}
🔰节点选择 = select, DIRECT

⚓️其他流量 = select, 🔰节点选择, 🚀直接连接
{{- end }}

# 应用分组
{{- if gt (len $supportedProxies) 0 }}
✈️Telegram = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🎙Discord = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

📘Facebook = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

📕Reddit = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🤖OpenAI = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🤖Claude = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🤖Gemini = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接
{{- else }}
✈️Telegram = select, 🔰节点选择, 🚀直接连接

🎙Discord = select, 🔰节点选择, 🚀直接连接

📘Facebook = select, 🔰节点选择, 🚀直接连接

📕Reddit = select, 🔰节点选择, 🚀直接连接

🤖OpenAI = select, 🔰节点选择, 🚀直接连接

🤖Claude = select, 🔰节点选择, 🚀直接连接

🤖Gemini = select, 🔰节点选择, 🚀直接连接
{{- end }}

{{- if gt (len $supportedProxies) 0 }}
🎬Youtube = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🎬TikTok = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🎬Netflix = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🎬DisneyPlus = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🎬哔哩哔哩 = select, 🚀直接连接, 🔰节点选择, {{ template "AllProxyNames" . }}
{{- else }}
🎬Youtube = select, 🔰节点选择, 🚀直接连接

🎬TikTok = select, 🔰节点选择, 🚀直接连接

🎬Netflix = select, 🔰节点选择, 🚀直接连接

🎬DisneyPlus = select, 🔰节点选择, 🚀直接连接

🎬哔哩哔哩 = select, 🚀直接连接, 🔰节点选择
{{- end }}

{{- if gt (len $supportedProxies) 0 }}
🎬国外媒体 = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🎧Spotify = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🎮Steam = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

💻Microsoft = select, 🚀直接连接, 🔰节点选择, {{ template "AllProxyNames" . }}

☁OneDrive = select, 🚀直接连接, 🔰节点选择, {{ template "AllProxyNames" . }}

📧OutLook = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🤖Copilot = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🧧Paypal = select, 🔰节点选择, {{ template "AllProxyNames" . }}, 🚀直接连接

🚚Amazon = select, 🚀直接连接, 🔰节点选择, {{ template "AllProxyNames" . }}

📡Speedtest = select, 🚀直接连接, 🔰节点选择, {{ template "AllProxyNames" . }}

🍎苹果服务 = select, 🚀直接连接, 🔰节点选择, {{ template "AllProxyNames" . }}
{{- else }}
🎬国外媒体 = select, 🔰节点选择, 🚀直接连接

🎧Spotify = select, 🔰节点选择, 🚀直接连接

🎮Steam = select, 🔰节点选择, 🚀直接连接

💻Microsoft = select, 🚀直接连接, 🔰节点选择

☁OneDrive = select, 🚀直接连接, 🔰节点选择

📧OutLook = select, 🔰节点选择, 🚀直接连接

🤖Copilot = select, 🔰节点选择, 🚀直接连接

🧧Paypal = select, 🔰节点选择, 🚀直接连接

🚚Amazon = select, 🚀直接连接, 🔰节点选择

📡Speedtest = select, 🚀直接连接, 🔰节点选择

🍎苹果服务 = select, 🚀直接连接, 🔰节点选择
{{- end }}

🚀直接连接 = select, DIRECT



[Rule]
# 本地网络直连
DOMAIN-SUFFIX,smtp,DIRECT
DOMAIN-KEYWORD,aria2,DIRECT
DOMAIN,clash.razord.top,DIRECT
DOMAIN-SUFFIX,lancache.steamcontent.com,DIRECT

# 管理面板
DOMAIN,yacd.haishan.me,🔰节点选择
DOMAIN-SUFFIX,appinn.com,🔰节点选择

# 广告拦截
RULE-SET,https://anti-ad.net/surge2.txt,REJECT

# AI服务规则
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/OpenAI/OpenAI.list,🤖OpenAI,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Claude/Claude.list,🤖Claude,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Gemini/Gemini.list,🤖Gemini,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Copilot/Copilot.list,🤖Copilot,enhanced-mode

# 苹果服务
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Apple/Apple_All.list,🍎苹果服务,enhanced-mode

# 流媒体规则
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Netflix/Netflix.list,🎬Netflix,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Disney/Disney.list,🎬DisneyPlus,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/TikTok/TikTok.list,🎬TikTok,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/YouTube/YouTube.list,🎬Youtube,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Spotify/Spotify.list,🎧Spotify,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/BiliBili/BiliBili.list,🎬哔哩哔哩,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/GlobalMedia/GlobalMedia_All_No_Resolve.list,🎬国外媒体,enhanced-mode

# 社交媒体规则
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Telegram/Telegram.list,✈️Telegram,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Discord/Discord.list,�Discord,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Facebook/Facebook.list,📘Facebook,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Reddit/Reddit.list,📕Reddit,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Twitter/Twitter.list,🔰节点选择,enhanced-mode

# 各大平台规则
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/GitHub/GitHub.list,🔰节点选择,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Google/Google.list,�节点选择,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Microsoft/Microsoft.list,�Microsoft,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/OneDrive/OneDrive.list,☁OneDrive,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Amazon/Amazon.list,�Amazon,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/PayPal/PayPal.list,🧧Paypal,enhanced-mode

# 游戏规则
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Game/Game.list,�Steam,enhanced-mode

# 代理和直连规则
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Global/Global_All_No_Resolve.list,🔰节点选择,enhanced-mode
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/ChinaMax/ChinaMax_All_No_Resolve.list,🚀直接连接
RULE-SET,https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Surge/Lan/Lan.list,DIRECT

# 本地域名直连
DOMAIN-SUFFIX,live.cn,🚀直接连接

# 地理位置规则
GEOIP,CN,DIRECT

# 最终规则
FINAL,⚓️其他流量

[Panel]
PanelA = title="订阅信息", content="流量用量: {{ $used }}GB / {{ $total }}GB\n到期时间: {{ $ExpiredAt }}\n更新时间: {{ now | date "2006-01-02 15:04:05" }}", style=info$alpha_subscription_template$, 'conf', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (8, 'Loon', '', 'Loon configuration template.', 'loon://import?sub=${encodeURIComponent(url)}', 'Loon', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "wireguard" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "loon" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %s" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

[General]
ip-mode = v4-only
dns-server = system, 223.5.5.5
doh-server = https://dns.alidns.com/dns-query
allow-wifi-access = true
wifi-access-http-port = 6088
wifi-access-socks5-port = 6089
proxy-test-url = http://www.gstatic.com/generate_204
internet-test-url = http://connectivitycheck.platform.hicloud.com/generate_204
test-timeout = 5
interface-mode = auto
bypass-tun = 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 127.0.0.0/8, localhost, *.local
skip-proxy = 192.168.0.0/16, 10.0.0.0/8, 172.16.0.0/12, 127.0.0.0/8, localhost, *.local
real-ip = *.lan, lens.l.google.com, *.srv.nintendo.net, *.stun.playstation.net, *.xboxlive.com, xbox.*.*.microsoft.com, *.msftncsi.com, *.msftconnecttest.com
hijack-dns = 8.8.8.8:53, 1.1.1.1:53

[Proxy]
{{- range $proxy := $supportedProxies }}
  {{- $common := "udp=true,fast-open=true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

{{- if eq $proxy.Type "shadowsocks" }}
{{ $proxy.Name }} = Shadowsocks,{{ $server }},{{ $proxy.Port }},{{ $proxy.Method }},{{ $password }}{{- if $proxy.Transport }},obfs={{ $proxy.Transport }},obfs-host={{ $proxy.SNI }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vmess" }}
{{ $proxy.Name }} = vmess,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if or $proxy.SNI $proxy.Fingerprint $proxy.AllowInsecure }},over-tls=true{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-fingerprint={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = VLESS,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if and $proxy.Flow (ne $proxy.Flow "none") }},flow={{ $proxy.Flow }}{{ end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.RealityPublicKey }},reality-public-key={{ $proxy.RealityPublicKey }}{{- if $proxy.RealityShortId }},reality-short-id={{ $proxy.RealityShortId }}{{ end }}{{ end }},{{ $common }}
{{- else if eq $proxy.Type "trojan" }}
{{ $proxy.Name }} = trojan,{{ $server }},{{ $proxy.Port }},{{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }},transport=ws{{- if $proxy.Path }},path={{ $proxy.Path }}{{ end }}{{- if $proxy.Host }},host={{ $proxy.Host }}{{ end }}{{- else if eq $proxy.Transport "grpc" }},transport=grpc,servicename={{ $proxy.ServiceName | default "grpc" }}{{ end }}{{- if $proxy.SNI }},tls-name={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.Fingerprint }},tls-fingerprint={{ $proxy.Fingerprint }}{{ end }},{{ $common }}
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = Hysteria2,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.ObfsPassword }},obfs=salamander,obfs-password={{ $proxy.ObfsPassword }}{{ end }}{{- if $proxy.HopPorts }},port-hopping={{ $proxy.HopPorts }}{{ end }}{{- if $proxy.HopInterval }},hop-interval={{ $proxy.HopInterval }}{{ end }}
{{- else if eq $proxy.Type "tuic" }}
{{ $proxy.Name }} = TUIC,{{ $proxy.Server }},{{ $proxy.Port }},{{ $proxy.ServerKey }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.DisableSNI }},disable-sni={{ $proxy.DisableSNI }}{{ end }}{{- if $proxy.ReduceRtt }},reduce-rtt={{ $proxy.ReduceRtt }}{{ end }}{{- if $proxy.UDPRelayMode }},udp-relay-mode={{ $proxy.UDPRelayMode }}{{ end }}{{- if $proxy.CongestionController }},congestion-control={{ $proxy.CongestionController }}{{ end }}
{{- else if eq $proxy.Type "wireguard" }}
{{ $proxy.Name }} = WireGuard,interface-ip={{ $proxy.RealityServerAddr | default "10.0.0.2" }},interface-ipv6={{ $proxy.RealityServerPort | default "::" }},private-key={{ $proxy.ServerKey }},mtu=1280,dns=8.8.8.8,keepalive=25,peers=[{public-key={{ $proxy.RealityPublicKey }},allowed-ips="0.0.0.0/0,::/0",endpoint={{ $proxy.Host }}:{{ $proxy.Port }}{{- if $proxy.Path }},preshared-key={{ $proxy.Path }}{{ end }}}]
{{- else if eq $proxy.Type "anytls" }}
{{ $proxy.Name }} = anytls,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}
{{- end }}
{{- end }}

[Remote Filter]
🇺🇳 Nodes = NameRegex, FilterKey = ".*"

[Proxy Group]
🚀 Proxy = select, 🌏 Auto, 🎯 Direct, 🇺🇳 Nodes
🍎 Apple = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
🔍 Google = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
🪟 Microsoft = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
📺 GlobalMedia = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
🤖 AI = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
🪙 Crypto = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
🎮 Game = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
📟 Telegram = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
🇨🇳 China = select, 🎯 Direct, 🚀 Proxy, 🇺🇳 Nodes
🐠 Final = select, 🚀 Proxy, 🎯 Direct, 🇺🇳 Nodes
🌏 Auto = url-test, 🇺🇳 Nodes
🎯 Direct = select, DIRECT

[Rewrite]
^https?:\/\/(www.)?g\.cn 302 https://www.google.com
^https?:\/\/(www.)?google\.cn 302 https://www.google.com

[Remote Rule]
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Apple/Apple.list, policy = 🍎 Apple, tag = Apple, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Google/Google.list, policy = 🔍 Google, tag = Google, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/GitHub/GitHub.list, policy = 🪟 Microsoft, tag = GitHub, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Microsoft/Microsoft.list, policy = 🪟 Microsoft, tag = Microsoft, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/HBO/HBO.list, policy = 📺 GlobalMedia, tag = HBO, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Disney/Disney.list, policy = 📺 GlobalMedia, tag = Disney, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/TikTok/TikTok.list, policy = 📺 GlobalMedia, tag = TikTok, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Netflix/Netflix.list, policy = 📺 GlobalMedia, tag = Netflix, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/GlobalMedia/GlobalMedia.list, policy = 📺 GlobalMedia, tag = GlobalMedia, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/GlobalMedia/GlobalMedia_Domain.list, policy = 📺 GlobalMedia, tag = GlobalMedia, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Telegram/Telegram.list, policy = 📟 Telegram, tag = Telegram, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/OpenAI/OpenAI.list, policy = 🤖 AI, tag = OpenAI, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Gemini/Gemini.list, policy = 🤖 AI, tag = Gemini, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Copilot/Copilot.list, policy = 🤖 AI, tag = Copilot, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Claude/Claude.list, policy = 🤖 AI, tag = Claude, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Crypto/Crypto.list, policy = 🪙 Crypto, tag = Crypto, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Cryptocurrency/Cryptocurrency.list, policy = 🪙 Crypto, tag = Cryptocurrency, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Game/Game.list, policy = 🎮 Game, tag = Game, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Global/Global.list, policy = 🚀 Proxy, tag = Global, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Global/Global_Domain.list, policy = 🚀 Proxy, tag = Global, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/ChinaMax/ChinaMax.list, policy = 🇨🇳 China, tag = ChinaMax, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/ChinaMax/ChinaMax_Domain.list, policy = 🇨🇳 China, tag = ChinaMax, enabled = true
https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Loon/Lan/Lan.list, policy = 🎯 Direct, tag = Lan, enabled = true

[Rule]
GEOIP, CN, 🇨🇳 China
FINAL, 🐠 Final
$alpha_subscription_template$, 'conf', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (9, 'Quantumult', '', 'Quantumult configuration template.', 'quantumult:///add-resource?remote-resource=${encodeURIComponent(url)}', 'Quantumult', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "trojan" true "anytls" true "http" true "https" true "socks5" true "socks5-tls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "quantumult" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

{{- range $proxy := $supportedProxies }}
  {{- $common := "fast-open=false, udp-relay=true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- if eq $proxy.Type "shadowsocks" }}
    {{- $method := default "aes-128-gcm" $proxy.Method }}
shadowsocks={{ $server }}:{{ $proxy.Port }}, method={{ $method }}, password={{ $password }}{{- if ne (default "" $proxy.Obfs) "" }}, obfs={{ $proxy.Obfs }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- if ne (default "" $proxy.ObfsPath) "" }}, obfs-uri={{ $proxy.ObfsPath }}{{- end }}{{- end }}, {{ $common }}, tag={{ $proxy.Name }}
  {{- else if eq $proxy.Type "vmess" }}
vmess={{ $server }}:{{ $proxy.Port }}, method=none, password={{ $password }}{{- if or (eq (default "" $proxy.Security) "tls") (eq $proxy.Transport "over-tls") }}, obfs=over-tls{{- end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, obfs=ws{{- if ne (default "" $proxy.Path) "" }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- end }}{{- if eq $proxy.Transport "wss" }}, obfs=wss{{- if ne (default "" $proxy.Path) "" }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}, udp-relay=false, tag={{ $proxy.Name }}
  {{- else if eq $proxy.Type "trojan" }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}, udp-relay=false, tag={{ $proxy.Name }}
  {{- else if eq $proxy.Type "anytls" }}
anytls={{ $server }}:{{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}, udp-relay=false, tag={{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "http") (eq $proxy.Type "https") }}
http={{ $server }}:{{ $proxy.Port }}{{- if or (ne (default "" $proxy.Username) "") (ne $password "") }}, username={{ default $password $proxy.Username }}, password={{ $password }}{{- end }}{{- if eq $proxy.Type "https" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- end }}, udp-relay=false, tag={{ $proxy.Name }}
  {{- else if or (eq $proxy.Type "socks5") (eq $proxy.Type "socks5-tls") }}
socks5={{ $server }}:{{ $proxy.Port }}{{- if or (ne (default "" $proxy.Username) "") (ne $password "") }}, username={{ default $password $proxy.Username }}, password={{ $password }}{{- end }}{{- if eq $proxy.Type "socks5-tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- end }}, udp-relay=false, tag={{ $proxy.Name }}
  {{- end }}
{{- end }}$alpha_subscription_template$, 'conf', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (10, 'Quantumult X', '', 'Quantumult X configuration template.', 'quantumult-x:///add-resource?remote-resource=${encodeURIComponent(url)}', 'Quantumult%20X', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- /* $supportSet derived from constants.ts protocols + Quantumult X platform support */ -}}
{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "anytls" true "http" true "socks" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- $proxyType := lower (default "" $proxy.Type) -}}
  {{- if and (hasKey $supportSet $proxyType) (clientSupportsProxy "quantumult-x" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

{{- range $proxy := $supportedProxies -}}
  {{- $proxyType := lower (default "" $proxy.Type) -}}
  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}
  {{- /* Password: use global UserInfo.Password; SS 2022 ciphers derive a combined key */ -}}
  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxyType "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}
  {{- $SkipVerify := $proxy.AllowInsecure -}}
  {{- $hasReality := and (ne (default "" $proxy.Security) "") (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") -}}
  {{- /* ========== shadowsocks ========== */ -}}
  {{- if eq $proxyType "shadowsocks" }}
    {{- $method := default "aes-128-gcm" $proxy.Method }}
    {{- /* obfs enum from constants.ts SECURITY.shadowsocks: 'none','http','tls' */ -}}
    {{- $ssObfs := default "none" $proxy.Obfs }}
shadowsocks={{ $server }}:{{ $proxy.Port }}, method={{ $method }}, password={{ $password }}{{- if ne $ssObfs "none" }}, obfs={{ $ssObfs }}{{- end }}{{- if ne (default "" $proxy.ObfsHost) "" }}, obfs-host={{ $proxy.ObfsHost }}{{- end }}{{- if ne (default "" $proxy.ObfsPath) "" }}, obfs-uri={{ $proxy.ObfsPath }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vmess ========== */ -}}
  {{- else if eq $proxyType "vmess" }}
    {{- $vmMethod := default "none" $proxy.Method }}
    {{- /* Transport enum from constants.ts TRANSPORTS.vmess: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum from constants.ts SECURITY.vmess: 'none','tls' */ -}}
    {{- $vmTransport := default "tcp" $proxy.Transport }}
    {{- $vmSecurity := default "none" $proxy.Security }}
    {{- $vmObfs := "" -}}
    {{- if eq $vmTransport "websocket" -}}
      {{- if eq $vmSecurity "tls" -}}
        {{- $vmObfs = "wss" -}}
      {{- else -}}
        {{- $vmObfs = "ws" -}}
      {{- end -}}
    {{- else if eq $vmSecurity "tls" -}}
      {{- $vmObfs = "over-tls" -}}
    {{- end }}
vmess={{ $server }}:{{ $proxy.Port }}, method={{ $vmMethod }}, password={{ $password }}{{- if ne $vmObfs "" }}, obfs={{ $vmObfs }}{{- end }}{{- if and (or (eq $vmObfs "ws") (eq $vmObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else if ne $vmObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== vless ========== */ -}}
  {{- else if eq $proxyType "vless" }}
    {{- /* Transport enum: 'tcp','websocket','grpc','mkcp','httpupgrade','xhttp' */ -}}
    {{- /* Security enum: 'none','tls','reality' */ -}}
    {{- $vlTransport := default "tcp" $proxy.Transport }}
    {{- $vlSecurity := default "none" $proxy.Security }}
    {{- $vlObfs := "" -}}
    {{- if eq $vlTransport "websocket" -}}
      {{- if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
        {{- $vlObfs = "wss" -}}
      {{- else -}}
        {{- $vlObfs = "ws" -}}
      {{- end -}}
    {{- else if or (eq $vlSecurity "tls") (eq $vlSecurity "reality") -}}
      {{- $vlObfs = "over-tls" -}}
    {{- end }}
vless={{ $server }}:{{ $proxy.Port }}, method=none, password={{ $password }}{{- if ne $vlObfs "" }}, obfs={{ $vlObfs }}{{- end }}{{- if and (or (eq $vlObfs "ws") (eq $vlObfs "wss")) (ne (default "" $proxy.Path) "") }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else if ne $vlObfs "" }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, vless-flow={{ $proxy.Flow }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== trojan ========== */ -}}
  {{- else if eq $proxyType "trojan" }}
    {{- /* Transport enum: 'tcp','websocket','grpc' */ -}}
    {{- /* Security enum: 'tls' (always tls for trojan) */ -}}
    {{- $trTransport := default "tcp" $proxy.Transport }}
    {{- if eq $trTransport "websocket" }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, obfs=wss{{- if ne (default "" $proxy.Path) "" }}, obfs-uri={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, obfs-host={{ $proxy.Host }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- else }}
trojan={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
    {{- end }}
  {{- /* ========== anytls ========== */ -}}
  {{- else if eq $proxyType "anytls" }}
    {{- /* Security enum: 'tls' (always tls for anytls) */ -}}
anytls={{ $server }}:{{ $proxy.Port }}, password={{ $password }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}, fast-open=false, udp-relay=true, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== http (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "http" }}
    {{- /* Security enum: 'none','tls' */ -}}
    {{- $httpSecurity := default "none" $proxy.Security }}
http={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $httpSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{- /* ========== socks (supports TLS via over-tls) ========== */ -}}
  {{- else if eq $proxyType "socks" }}
    {{- /* socks has no security/transport in schema; TLS detected via Security field */ -}}
    {{- $socksSecurity := default "none" $proxy.Security }}
socks5={{ $server }}:{{ $proxy.Port }}{{- if ne $password "" }}, password={{ $password }}{{- end }}{{- if eq $socksSecurity "tls" }}, over-tls=true{{- if ne (default "" $proxy.SNI) "" }}, tls-host={{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, tls-verification=false{{- else }}, tls-verification=true{{- end }}{{- if $hasReality }}, reality-base64-pubkey={{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, reality-hex-shortid={{ $proxy.RealityShortId }}{{- end }}{{- end }}{{- end }}, fast-open=false, udp-relay=false, tag={{ $proxy.Name }}{{- "\n" -}}
  {{ end }}
{{ end }}
$alpha_subscription_template$, 'conf', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (11, 'Egern', '', 'Egern YAML subscription template.', 'egern://subscribe?url=${encodeURIComponent(url)}', 'Egern', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "anytls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "egern" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = printf "%q" $proxy.Name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %q" $proxyNames $proxy.Name -}}
  {{- end -}}
{{- end -}}

# {{ .SiteName }}-{{ .SubscribeName }}
# Traffic: {{ $used }} GiB/{{ $total }} GiB | Expires: {{ $ExpiredAt }}
# Generated at: {{ now | date "2006-01-02 15:04:05" }}

ipv6: true
http_port: 6088
socks_port: 6089
allow_external_connections: true
hijack_dns:
- 8.8.8.8:53
- 1.1.1.1:53
dns:
  bootstrap:
    - system
    - 223.5.5.5
  upstreams:
    proxy:
      - 8.8.8.8
      - 1.1.1.1
  forward:
  - proxy_rule_set:
      match: https://github.com/ACL4SSR/ACL4SSR/raw/master/Clash/ChinaDomain.list
      value: system
  - wildcard:
      match: '*'
      value: proxy

proxies:
{{- range $proxy := $supportedProxies }}
  {{- $common := "udp: true, tfo: true" -}}

  {{- $server := $proxy.Server -}}
  {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
    {{- $server = printf "[%s]" $server -}}
  {{- end -}}

  {{- $password := $.UserInfo.Password -}}
  {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
    {{- $method := $proxy.Method -}}
    {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $serverB64 := b64enc $proxy.ServerKey -}}
      {{- $userB64 := b64enc $userKey -}}
      {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
    {{- end -}}
  {{- end -}}

  {{- $SkipVerify := $proxy.AllowInsecure -}}

  {{- if eq $proxy.Type "shadowsocks" }}
  - { name: {{ $proxy.Name | quote }}, type: ss, server: {{ $server }}, port: {{ $proxy.Port }}, cipher: {{ default "aes-128-gcm" $proxy.Method }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.Obfs) "" }}, plugin: obfs, plugin-opts: { mode: {{ $proxy.Obfs }}, host: {{ default "" $proxy.ObfsHost }} }{{- end }} }
  {{- else if eq $proxy.Type "vmess" }}
  - { name: {{ $proxy.Name | quote }}, type: vmess, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, alterId: 0, cipher: auto, {{ $common }}{{- if or (eq $proxy.Transport "websocket") (eq $proxy.Transport "ws") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if or (eq $proxy.Security "tls") (eq $proxy.Security "reality") }}, tls: true{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else if eq $proxy.Type "vless" }}
  - { name: {{ $proxy.Name | quote }}, type: vless, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ $password }}, {{ $common }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "httpupgrade" }}, network: httpupgrade, httpupgrade-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, servername: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, client-fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}, tls: true, reality-opts: { public-key: {{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, short-id: {{ $proxy.RealityShortId }}{{- end }} }{{- end }}{{- if ne (default "none" $proxy.Flow) "none" }}, flow: {{ $proxy.Flow }}{{- end }} }
  {{- else if eq $proxy.Type "trojan" }}
  - { name: {{ $proxy.Name | quote }}, type: trojan, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint: {{ $proxy.Fingerprint }}{{- end }}{{- if and (eq $proxy.Security "reality") (ne (default "" $proxy.RealityPublicKey) "") }}, reality-opts: { public-key: {{ $proxy.RealityPublicKey }}{{- if ne (default "" $proxy.RealityShortId) "" }}, short-id: {{ $proxy.RealityShortId }}{{- end }} }{{- end }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, network: ws, ws-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: {{ $proxy.Host }} }{{- end }} }{{- else if eq $proxy.Transport "http" }}, network: http, http-opts: { method: GET, path: [{{ default "/" $proxy.Path | quote }}]{{- if ne (default "" $proxy.Host) "" }}, headers: { Host: [{{ $proxy.Host | quote }}] }{{- end }} }{{- else if eq $proxy.Transport "grpc" }}, network: grpc, grpc-opts: { grpc-service-name: {{ default "grpc" $proxy.ServiceName }} }{{- end }} }
  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  {{- else if eq $proxy.Type "tuic" }}
  - { name: {{ $proxy.Name | quote }}, type: tuic, server: {{ $server }}, port: {{ $proxy.Port }}, uuid: {{ default $.UserInfo.Password $proxy.ServerKey }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if $proxy.DisableSNI }}, disable-sni: true{{- end }}{{- if $proxy.ReduceRtt }}, reduce-rtt: true{{- end }}{{- if ne (default "" $proxy.UDPRelayMode) "" }}, udp-relay-mode: {{ $proxy.UDPRelayMode }}{{- end }}{{- if ne (default "" $proxy.CongestionController) "" }}, congestion-controller: {{ $proxy.CongestionController }}{{- end }} }
  {{- else if eq $proxy.Type "anytls" }}
  - { name: {{ $proxy.Name | quote }}, type: anytls, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.Fingerprint) "" }}, fingerprint: {{ $proxy.Fingerprint }}{{- end }} }
  {{- else }}
  - { name: {{ $proxy.Name | quote }}, type: {{ $proxy.Type }}, server: {{ $server }}, port: {{ $proxy.Port }}, {{ $common }} }
  {{- end }}
{{- end }}

policy_groups:
  - select: { name: Proxy, policies: [{{ $proxyNames }}] }

rules:
- geoip:
    match: CN
    policy: DIRECT
- default:
    policy: Proxy

mitm:
  enabled: true
$alpha_subscription_template$, 'yaml', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (12, 'Hiddify', '', 'Hiddify JSON subscription template.', 'hiddify://import/${encodeURIComponent(url)}', 'Hiddify', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true "tuic" true "wireguard" true "anytls" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "hiddify" $proxy) -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- if gt (len $supportedProxies) 0 -}}
  {{- range $proxy := $supportedProxies -}}
    {{- if eq $proxyNames "" -}}
      {{- $proxyNames = printf "\"%s\"" $proxy.Name -}}
    {{- else -}}
      {{- $proxyNames = printf "%s, \"%s\"" $proxyNames $proxy.Name -}}
    {{- end -}}
  {{- end -}}
  {{- $proxyNames = printf ", %s" $proxyNames -}}
{{- end -}}

{
  "log": {"level": "info", "timestamp": true},
  "experimental": {
    "cache_file": {"enabled": true, "path": "hiddify_cache.db", "cache_id": "{{ .SiteName | default "hiddify" }}", "store_fakeip": true},
    "clash_api": {"external_controller": "127.0.0.1:9090", "external_ui": "ui", "external_ui_download_url": "https://github.com/MetaCubeX/metacubexd/archive/gh-pages.zip", "external_ui_download_detour": "direct", "default_mode": "rule"}
  },
  "dns": {
    "servers": [
      {"tag": "dns_proxy","address": "https://8.8.8.8/dns-query","detour": "Proxy"},
      {"tag": "dns_direct","address": "https://223.5.5.5/dns-query","detour": "direct"},
      {"tag": "dns_local","address": "local","detour": "direct"}
    ],
    "rules": [
      {"outbound": "any", "server": "dns_local"},
      {"rule_set": "geosite-cn", "server": "dns_direct"},
      {"clash_mode": "direct", "server": "dns_direct"},
      {"clash_mode": "global", "server": "dns_proxy"},
      {"rule_set": "geosite-geolocation-!cn", "server": "dns_proxy"}
    ],
    "final": "dns_direct", "strategy": "ipv4_only"
  },
  "inbounds": [
    {"tag": "tun-in", "type": "tun", "inet4_address": "172.19.0.1/30", "auto_route": true, "strict_route": true, "stack": "system", "sniff": true,
      "platform": {"http_proxy": {"enabled": true, "server": "127.0.0.1", "server_port": 7890}}},
    {"tag": "mixed-in", "type": "mixed", "listen": "127.0.0.1", "listen_port": 7890, "sniff": true}
  ],
  "outbounds": [
    {"tag": "Proxy", "type": "selector", "outbounds": ["Auto - UrlTest", "direct"{{ $proxyNames }}]},
    {"tag": "Domestic", "type": "selector", "outbounds": ["direct", "Proxy"{{ $proxyNames }}]},
    {"tag": "Others", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "AI Suite", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "Netflix", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "Disney Plus", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "YouTube", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "Spotify", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "Apple", "type": "selector", "outbounds": ["direct", "Proxy"{{ $proxyNames }}]},
    {"tag": "Telegram", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "Microsoft", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "TikTok", "type": "selector", "outbounds": ["Proxy", "direct"{{ $proxyNames }}]},
    {"tag": "AdBlock", "type": "selector", "outbounds": ["block", "direct", "Proxy"]},
    {{- if gt (len $supportedProxies) 0 }}
    {"tag": "Auto - UrlTest", "type": "urltest", "outbounds": [{{ $proxyNames | trimPrefix ", " }}], "url": "http://cp.cloudflare.com/", "interval": "10m", "tolerance": 50}
    {{- range $proxy := $supportedProxies }},

      {{- $common := `"tcp_fast_open": false, "udp_over_tcp": true` -}}

      {{- $server := $proxy.Server -}}
      {{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
        {{- $server = printf "[%s]" $server -}}
      {{- end -}}

      {{- $password := $.UserInfo.Password -}}
      {{- if and (eq $proxy.Type "shadowsocks") (ne (default "" $proxy.ServerKey) "") -}}
        {{- $method := $proxy.Method -}}
        {{- if or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm") -}}
          {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
          {{- $pwdStr := printf "%s" $password -}}
          {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
          {{- $serverB64 := b64enc $proxy.ServerKey -}}
          {{- $userB64 := b64enc $userKey -}}
          {{- $password = printf "%s:%s" $serverB64 $userB64 -}}
        {{- end -}}
      {{- end -}}

      {{- $SkipVerify := $proxy.AllowInsecure -}}

{{- if eq $proxy.Type "shadowsocks" -}}
    { "type": "shadowsocks", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "method": {{ $proxy.Method | quote }}, "password": {{ $password | quote }}, {{ $common }} }
{{- else if eq $proxy.Type "vmess" -}}
    { "type": "vmess", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "security": "auto", {{ $common }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "vless" -}}
    { "type": "vless", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, {{ $common }}{{ if and $proxy.Flow (ne $proxy.Flow "none") }}, "flow": {{ $proxy.Flow | quote }}{{ end }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}{{ if or $proxy.RealityPublicKey $proxy.SNI $SkipVerify $proxy.Fingerprint }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}{{ if $proxy.RealityPublicKey }}, "reality": {"enabled": true, "public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.RealityShortId }}, "short_id": {{ $proxy.RealityShortId | quote }}{{ end }}}{{ end }}}{{ end }} }
{{- else if eq $proxy.Type "trojan" -}}
    { "type": "trojan", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, "transport": {"type": "ws", "path": {{ $proxy.Path | default "/" | quote }}{{- if $proxy.Host }}, "headers": {"Host": {{ $proxy.Host | quote }} }{{- end }}}{{ else if eq $proxy.Transport "grpc" }}, "transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") -}}
    { "type": "hysteria2", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if $proxy.ObfsPassword }}, "obfs": { "type": "salamander", "password": {{ $proxy.ObfsPassword | quote }} }{{ end }}{{ if $proxy.HopPorts }}, "ports": {{ $proxy.HopPorts | quote }}{{ end }}{{ if $proxy.HopInterval }}, "hop_interval": {{ $proxy.HopInterval }}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if eq $proxy.Type "tuic" -}}
    { "type": "tuic", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "uuid": {{ $.UserInfo.Password | quote }}, "password": {{ $password | quote }}{{ if $proxy.DisableSNI }}, "disable_sni": {{ $proxy.DisableSNI }}{{ end }}{{ if $proxy.ReduceRtt }}, "reduce_rtt": {{ $proxy.ReduceRtt }}{{ end }}{{ if $proxy.UDPRelayMode }}, "udp_relay_mode": {{ $proxy.UDPRelayMode | quote }}{{ end }}{{ if $proxy.CongestionController }}, "congestion_control": {{ $proxy.CongestionController | quote }}{{ end }}, {{ $common }}, "alpn": ["h3"], "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else if eq $proxy.Type "wireguard" -}}
    { "type": "wireguard", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "private_key": {{ $proxy.ServerKey | quote }}, "peer_public_key": {{ $proxy.RealityPublicKey | quote }}{{ if $proxy.Path }}, "pre_shared_key": {{ $proxy.Path | quote }}{{ end }}{{ if $proxy.RealityServerAddr }}, "local_address": [{{ $proxy.RealityServerAddr | quote }}]{{ end }}, {{ $common }} }
{{- else if eq $proxy.Type "anytls" -}}
    { "type": "anytls", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
{{- else -}}
    { "type": "direct", "tag": {{ $proxy.Name | quote }}, {{ $common }} }
{{- end }}
    {{- end }},
    {{- end }}
    {"type": "direct", "tag": "direct"},
    {"type": "block", "tag": "block"}
  ],
  "route": {
    "auto_detect_interface": true, "final": "Proxy",
    "rules": [
      {"type": "logical", "mode": "or", "rules": [{"port": 53},{"protocol": "dns"}], "action": "hijack-dns"},
      {"rule_set": "geosite-category-ads-all", "outbound": "AdBlock"},
      {"clash_mode": "direct", "outbound": "direct"},
      {"clash_mode": "global", "outbound": "Proxy"},
      {"domain": ["clash.razord.top","yacd.metacubex.one","yacd.haishan.me","d.metacubex.one"], "outbound": "direct"},
      {"ip_is_private": true, "outbound": "direct"},
      {"rule_set": ["geoip-netflix","geosite-netflix"], "outbound": "Netflix"},
      {"rule_set": "geosite-disney", "outbound": "Disney Plus"},
      {"rule_set": "geosite-youtube", "outbound": "YouTube"},
      {"rule_set": "geosite-spotify", "outbound": "Spotify"},
      {"rule_set": ["geoip-apple","geosite-apple"], "outbound": "Apple"},
      {"rule_set": ["geoip-telegram","geosite-telegram"], "outbound": "Telegram"},
      {"rule_set": "geosite-openai", "outbound": "AI Suite"},
      {"rule_set": "geosite-microsoft", "outbound": "Microsoft"},
      {"rule_set": "geosite-tiktok", "outbound": "TikTok"},
      {"rule_set": "geosite-private", "outbound": "direct"},
      {"rule_set": ["geoip-cn","geosite-cn"], "outbound": "Domestic"},
      {"rule_set": "geosite-geolocation-!cn", "outbound": "Others"}
    ],
    "rule_set": [
      {"tag": "geoip-cn","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/cn.srs","download_detour": "direct"},
      {"tag": "geosite-cn","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/cn.srs","download_detour": "direct"},
      {"tag": "geosite-private","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/private.srs","download_detour": "direct"},
      {"tag": "geosite-geolocation-!cn","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/geolocation-!cn.srs","download_detour": "direct"},
      {"tag": "geosite-category-ads-all","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/category-ads-all.srs","download_detour": "direct"},
      {"tag": "geoip-netflix","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/netflix.srs","download_detour": "direct"},
      {"tag": "geosite-netflix","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/netflix.srs","download_detour": "direct"},
      {"tag": "geosite-disney","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/disney.srs","download_detour": "direct"},
      {"tag": "geosite-youtube","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/youtube.srs","download_detour": "direct"},
      {"tag": "geosite-spotify","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/spotify.srs","download_detour": "direct"},
      {"tag": "geoip-apple","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo-lite/geoip/apple.srs","download_detour": "direct"},
      {"tag": "geosite-apple","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/apple.srs","download_detour": "direct"},
      {"tag": "geoip-telegram","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/telegram.srs","download_detour": "direct"},
      {"tag": "geosite-telegram","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/telegram.srs","download_detour": "direct"},
      {"tag": "geosite-openai","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/openai.srs","download_detour": "direct"},
      {"tag": "geosite-microsoft","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/microsoft.srs","download_detour": "direct"},
      {"tag": "geosite-tiktok","type": "remote","format": "binary","url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/tiktok.srs","download_detour": "direct"}
    ]
  }
}
$alpha_subscription_template$, 'json', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (13, 'SingBox 1.11', '', 'sing-box 1.11 JSON subscription template.', 'sing-box://import-remote-profile?url=${encodeURIComponent(url)}#${encodeURIComponent(name)}', 'sing-box', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
{{- $used := printf "%.2f" (divf (add (.UserInfo.Download | default 0 | float64) (.UserInfo.Upload | default 0 | float64)) $GiB) -}}
{{- $traffic := (.UserInfo.Traffic | default 0 | float64) -}}
{{- $total := printf "%.2f" (divf $traffic $GiB) -}}

{{- $ExpiredAt := "" -}}
{{- $expStr := printf "%v" .UserInfo.ExpiredAt -}}
{{- if regexMatch `^[0-9]+$` $expStr -}}
  {{- $ts := $expStr | float64 -}}
  {{- $sec := ternary (divf $ts 1000.0) $ts (ge (len $expStr) 13) -}}
  {{- $ExpiredAt = (date "2006-01-02 15:04:05" (unixEpoch ($sec | int64))) -}}
{{- else -}}
  {{- $ExpiredAt = $expStr -}}
{{- end -}}

{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field := $sortFields -}}
    {{- $order := index $sortConfig $field -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}

{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- $isSupported := false -}}
  {{- if or (eq $proxy.Type "shadowsocks") (eq $proxy.Type "vmess") (eq $proxy.Type "trojan") (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") (eq $proxy.Type "tuic") -}}
    {{- $isSupported = true -}}
  {{- else if eq $proxy.Type "vless" -}}
    {{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") (eq $proxy.Transport "httpupgrade") (eq $proxy.Transport "grpc") (eq $proxy.Transport "tcp") (not $proxy.Transport) -}}
      {{- $isSupported = true -}}
    {{- end -}}
  {{- end -}}
  {{- if $isSupported -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- define "AllNodeNames" -}}
{{- $sortConfig := dict "Sort" "asc" -}}
{{- $byKey := dict -}}
{{- range $p := .Proxies -}}
  {{- $keyParts := list -}}
  {{- range $field, $order := $sortConfig -}}
    {{- $val := default "" (printf "%v" (index $p $field)) -}}
    {{- if or (eq $field "Sort") (eq $field "Port") -}}
      {{- $val = printf "%08d" (int (default 0 (index $p $field))) -}}
    {{- end -}}
    {{- if eq $order "desc" -}}
      {{- $val = printf "~%s" $val -}}
    {{- end -}}
    {{- $keyParts = append $keyParts $val -}}
  {{- end -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- $isSupported := false -}}
  {{- if or (eq .Type "shadowsocks") (eq .Type "vmess") (eq .Type "trojan") (eq .Type "hysteria2") (eq .Type "hy2") (eq .Type "tuic") -}}
    {{- $isSupported = true -}}
  {{- else if eq .Type "vless" -}}
    {{- if or (eq .Transport "ws") (eq .Transport "websocket") (eq .Transport "httpupgrade") (eq .Transport "grpc") (eq .Transport "tcp") (not .Transport) -}}
      {{- $isSupported = true -}}
    {{- end -}}
  {{- end -}}
  {{- if $isSupported -}}
    {{- $supportedProxies = append $supportedProxies . -}}
  {{- end -}}
{{- end -}}
{{- $first := true -}}
{{- range $supportedProxies -}}
  {{- if $first -}}
    "{{ .Name }}"
    {{- $first = false -}}
  {{- else -}}
    , "{{ .Name }}"
  {{- end -}}
{{- end -}}
{{- end -}}

{{- define "NodeOutbound" -}}
{{- $proxy := .proxy -}}
{{- $server := $proxy.Server -}}
{{- if and (contains $server ":") (not (hasPrefix "[" $server)) -}}
  {{- $server = printf "[%s]" $server -}}
{{- end -}}
{{- $port := $proxy.Port -}}
{{- $name := $proxy.Name -}}
{{- $pwd := $.UserInfo.Password -}}
{{- $svc := $proxy.ServiceName }}
{{- $tlsRequired := or (eq $proxy.Type "trojan") (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") (eq $proxy.Type "tuic") (eq $proxy.Type "anytls") -}}
{{- $tlsEnabled := or $tlsRequired $proxy.TLS (eq $proxy.Security "tls") (eq $proxy.Security "reality") -}}
{{- $sni := $proxy.SNI -}}
{{- if and (eq (default "" $sni) "") $tlsEnabled -}}
  {{- $sni = $server -}}
{{- end -}}

{{- $tlsOpts := "" -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if $proxy.Fingerprint -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

{{- $transportOpts := "" -}}
{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") -}}
  {{- $wsPath := default "/" $proxy.Path -}}
  {{- $transportOpts = printf "\"transport\": {\"type\": \"ws\", \"path\": \"%s\"" $wsPath -}}
  {{- if $proxy.Host -}}
    {{- $transportOpts = printf "%s, \"headers\": {\"Host\": \"%s\"}" $transportOpts ($proxy.Host) -}}
  {{- end -}}
  {{- $transportOpts = printf "%s}" $transportOpts -}}
{{- else if eq $proxy.Transport "httpupgrade" -}}
  {{- $wsPath := default "/" $proxy.Path -}}
  {{- $transportOpts = printf "\"transport\": {\"type\": \"httpupgrade\", \"path\": \"%s\"" $wsPath -}}
  {{- if $proxy.Host -}}
    {{- $transportOpts = printf "%s, \"headers\": {\"Host\": \"%s\"}" $transportOpts ($proxy.Host) -}}
  {{- end -}}
  {{- $transportOpts = printf "%s}" $transportOpts -}}
{{- else if eq $proxy.Transport "grpc" -}}
  {{- $grpcService := default "grpc" $svc -}}
  {{- $transportOpts = printf "\"transport\": {\"type\": \"grpc\", \"service_name\": \"%s\"" $grpcService -}}
  {{- if $proxy.GrpcIdleTimeout -}}{{- $transportOpts = printf "%s, \"idle_timeout\": \"%ds\"" $transportOpts $proxy.GrpcIdleTimeout -}}{{- end -}}
  {{- if $proxy.GrpcHealthCheckTimeout -}}{{- $transportOpts = printf "%s, \"ping_timeout\": \"%ds\"" $transportOpts $proxy.GrpcHealthCheckTimeout -}}{{- end -}}
  {{- if $proxy.GrpcPermitWithoutStream -}}{{- $transportOpts = printf "%s, \"permit_without_stream\": true" $transportOpts -}}{{- end -}}
  {{- $transportOpts = printf "%s}" $transportOpts -}}
{{- end -}}

{{- if eq $proxy.Type "shadowsocks" -}}
{ "type": "shadowsocks", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "method": "{{ default "aes-128-gcm" $proxy.Method }}", "password": "{{ $pwd }}" }

{{- else if eq $proxy.Type "trojan" -}}
{ "type": "trojan", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "password": "{{ $pwd }}"{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}, {{ $tlsOpts }} }

{{- else if eq $proxy.Type "vless" -}}
{{- $flowOpts := "" -}}
{{- if and $proxy.Flow (ne $proxy.Flow "none") -}}
  {{- $flowOpts = printf ", \"flow\": \"%s\"" ($proxy.Flow) -}}
{{- end -}}
{ "type": "vless", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $pwd }}"{{ $flowOpts }}{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}{{ if $tlsOpts }}, {{ $tlsOpts }}{{ end }} }

{{- else if eq $proxy.Type "vmess" -}}
{ "type": "vmess", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $pwd }}", "security": "auto"{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}{{ if $tlsOpts }}, {{ $tlsOpts }}{{ end }} }

{{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{ "type": "hysteria2", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "password": "{{ $pwd }}", {{ $tlsOpts }} }

{{- else if eq $proxy.Type "tuic" -}}
{ "type": "tuic", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $proxy.ServerKey }}", "password": "{{ $pwd }}", "alpn": ["h3"], {{ $tlsOpts }} }

{{- else if eq $proxy.Type "wireguard" -}}
{ "type": "wireguard", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "private_key": "{{ $proxy.ServerKey }}", "peer_public_key": "{{ $proxy.RealityPublicKey }}" }

{{- else if or (eq $proxy.Type "http") (eq $proxy.Type "https") -}}
{ "type": "http", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "username": "{{ $pwd }}", "password": "{{ $pwd }}"{{ if eq $proxy.Type "https" }}, {{ $tlsOpts }}{{ end }} }

{{- else if or (eq $proxy.Type "socks") (eq $proxy.Type "socks5") -}}
{ "type": "socks", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "version": "5", "username": "{{ $pwd }}", "password": "{{ $pwd }}" }

{{- else -}}
{ "type": "direct", "tag": "{{ $name }}" }
{{- end -}}
{{- end -}}

{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "experimental": {
    "cache_file": {
      "enabled": true,
      "store_fakeip": true,
      "store_rdrc": true
    },
    "clash_api": {
      "external_controller": "127.0.0.1:9090",
      "access_control_allow_origin": [
        "http://127.0.0.1",
        "https://yacd.metacubex.one",
        "https://metacubex.github.io",
        "https://metacubexd.pages.dev",
        "https://board.zash.run.place"
      ]
    }
  },
  "dns": {
    "independent_cache": true,
    "servers": [
      {
        "tag": "google",
        "address": "https://8.8.8.8/dns-query",
        "detour": "节点选择"
      },
      {
        "tag": "ali",
        "address": "https://223.5.5.5/dns-query",
        "detour": "直连"
      },
      {
        "tag": "fakeip",
        "address": "fakeip"
      }
    ],
    "rules": [
      {
        "outbound": "any",
        "server": "ali"
      },
      {
        "clash_mode": "Direct",
        "server": "ali"
      },
      {
        "clash_mode": "Global",
        "server": "google"
      },
      {
        "rule_set": "geosite-cn",
        "server": "ali"
      },
      {
        "query_type": [
          "A",
          "AAAA"
        ],
        "server": "fakeip"
      }
    ],
    "fakeip": {
      "enabled": true,
      "inet4_range": "198.18.0.0/15",
      "inet6_range": "fc00::/18"
    }
  },
  "inbounds": [
    {
      "type": "tun",
      "address": [
        "172.18.0.1/30",
        "fdfe:dcba:9876::1/126"
      ],
      "auto_route": true,
      "strict_route": true
    },
    {
      "type": "mixed",
      "listen": "::",
      "listen_port": 7890
    }
  ],
  "outbounds": [
    {
      "tag": "节点选择",
      "type": "selector",
      "outbounds": [{{ if gt (len $supportedProxies) 0 }}{{ template "AllNodeNames" . }}{{ end }}, "直连"]
    },
    {
      "tag": "Github",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {
      "tag": "Google",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {
      "tag": "Microsoft",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {
      "tag": "OpenAI",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {
      "tag": "Telegram",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {
      "tag": "Twitter",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {
      "tag": "Youtube",
      "type": "selector",
      "outbounds": [
        "节点选择",
        "直连"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {
      "tag": "国内",
      "type": "selector",
      "outbounds": [
        "直连",
        "节点选择"
        {{- if gt (len $supportedProxies) 0 -}}
        , {{ template "AllNodeNames" . }}
        {{- end -}}
      ]
    },
    {{- range $i, $proxy := $supportedProxies }}
    {{ if $i }},{{ end }}
    {{ template "NodeOutbound" (dict "proxy" $proxy "UserInfo" $.UserInfo) }}
    {{- end }}
    {{- if gt (len $supportedProxies) 0 }},{{ end }}
    {
      "tag": "直连",
      "type": "direct"
    }
  ],
  "route": {
    "auto_detect_interface": true,
    "rules": [
      {
        "action": "sniff"
      },
      {
        "protocol": "dns",
        "action": "hijack-dns"
      },
      {
        "ip_is_private": true,
        "outbound": "直连"
      },
      {
        "rule_set": "anti-ad",
        "clash_mode": "Rule",
        "action": "reject"
      },
      {
        "clash_mode": "Direct",
        "outbound": "直连"
      },
      {
        "clash_mode": "Global",
        "outbound": "节点选择"
      },
      {
        "rule_set": "geosite-github",
        "outbound": "Github"
      },
      {
        "rule_set": [
          "geoip-google",
          "geosite-google"
        ],
        "outbound": "Google"
      },
      {
        "rule_set": "geosite-microsoft",
        "outbound": "Microsoft"
      },
      {
        "rule_set": "geosite-openai",
        "outbound": "OpenAI"
      },
      {
        "rule_set": [
          "geoip-telegram",
          "geosite-telegram"
        ],
        "outbound": "Telegram"
      },
      {
        "rule_set": [
          "geoip-twitter",
          "geosite-twitter"
        ],
        "outbound": "Twitter"
      },
      {
        "rule_set": "geosite-youtube",
        "outbound": "Youtube"
      },
      {
        "rule_set": [
          "geoip-cn",
          "geosite-cn"
        ],
        "outbound": "国内"
      }
    ],
    "rule_set": [
      {
        "tag": "anti-ad",
        "type": "remote",
        "format": "binary",
        "url": "https://anti-ad.net/anti-ad-sing-box.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-github",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/github.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-google",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/google.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-google",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/google.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-microsoft",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/microsoft.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-openai",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/openai.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-telegram",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/telegram.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-telegram",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/telegram.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-twitter",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/twitter.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-twitter",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/twitter.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-youtube",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/youtube.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geosite-cn",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geosite/cn.srs",
        "download_detour": "直连"
      },
      {
        "tag": "geoip-cn",
        "type": "remote",
        "format": "binary",
        "url": "https://cdn.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@sing/geo/geoip/cn.srs",
        "download_detour": "直连"
      }
    ]
  }
}$alpha_subscription_template$, 'json', '{}', NOW(), NOW())
ON CONFLICT ("id") DO UPDATE SET
  "name" = EXCLUDED."name",
  "description" = EXCLUDED."description",
  "scheme" = EXCLUDED."scheme",
  "user_agent" = EXCLUDED."user_agent",
  "subscribe_template" = EXCLUDED."subscribe_template",
  "output_format" = EXCLUDED."output_format",
  "updated_at" = NOW();

SELECT setval(pg_get_serial_sequence('"subscribe_application"', 'id'), GREATEST(COALESCE((SELECT MAX("id") FROM "subscribe_application"), 0), 1), TRUE);
COMMIT;
