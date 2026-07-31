-- 02133_add_xray_core_subscription_templates.up.sql
-- Add Xray Core and Surge Xray bridge subscription templates.

BEGIN;

INSERT INTO "subscribe_application" ("id", "name", "icon", "description", "scheme", "user_agent", "is_default", "subscribe_template", "output_format", "download_link", "created_at", "updated_at")
VALUES (14, 'Xray Core', '', 'Xray-core client config with one local SOCKS inbound per node and one-to-one routing.', '', 'xray-core', FALSE, $alpha_subscription_template${{- $host := default "127.0.0.1" .Params.xray_inbound_host -}}
{{- $basePort := int (default "10800" .Params.xray_inbound_start_port) -}}
{{- define "streamSettings" -}}
  {{- $proxy := . -}}
  {{- $network := default "tcp" $proxy.Transport -}}
  {{- if eq $network "raw" -}}{{- $network = "tcp" -}}{{- end -}}
  {{- if eq $network "websocket" -}}{{- $network = "ws" -}}{{- end -}}
  {{- $security := default "none" $proxy.Security -}}
  {{- $sni := default $proxy.Server $proxy.Host -}}
  {{- $sni = default $sni $proxy.SNI -}}
  {{- if or (ne $network "tcp") (eq $security "tls") (eq $security "reality") -}}
, "streamSettings": {
    "network": {{ $network | quote }}
    {{- if eq $security "tls" }},
    "security": "tls",
    "tlsSettings": {
      "serverName": {{ $sni | quote }}{{ if $proxy.AllowInsecure }}, "allowInsecure": true{{ end }}{{ if $proxy.Fingerprint }}, "fingerprint": {{ $proxy.Fingerprint | quote }}{{ end }}
    }
    {{- else if eq $security "reality" }},
    "security": "reality",
    "realitySettings": {
      "serverName": {{ $sni | quote }}{{ if $proxy.Fingerprint }}, "fingerprint": {{ $proxy.Fingerprint | quote }}{{ end }}{{ if $proxy.RealityPublicKey }}, "password": {{ $proxy.RealityPublicKey | quote }}{{ end }}{{ if $proxy.RealityShortId }}, "shortId": {{ $proxy.RealityShortId | quote }}{{ end }}{{ if $proxy.Path }}, "spiderX": {{ $proxy.Path | quote }}{{ end }}
    }
    {{- end }}
    {{- if or (eq $network "ws") (eq $network "websocket") }},
    "wsSettings": { "path": {{ default "/" $proxy.Path | quote }}{{ if $proxy.Host }}, "headers": { "Host": {{ $proxy.Host | quote }} }{{ end }} }
    {{- else if eq $network "grpc" }},
    "grpcSettings": { "serviceName": {{ default "grpc" $proxy.ServiceName | quote }} }
    {{- else if eq $network "xhttp" }},
    "xhttpSettings": { "path": {{ default "/" $proxy.Path | quote }}{{ if $proxy.Host }}, "host": {{ $proxy.Host | quote }}{{ end }}{{ if $proxy.XhttpMode }}, "mode": {{ $proxy.XhttpMode | quote }}{{ end }} }
    {{- else if eq $network "httpupgrade" }},
    "httpupgradeSettings": { "path": {{ default "/" $proxy.Path | quote }}{{ if $proxy.Host }}, "host": {{ $proxy.Host | quote }}{{ end }} }
    {{- end }}
  }
  {{- end -}}
{{- end -}}
{{- define "outbound" -}}
  {{- $proxy := index . "proxy" -}}
  {{- $i := index . "index" -}}
  {{- $password := $.UserInfo.Password -}}
  {{- if eq $proxy.Type "shadowsocks" -}}
    {{- $method := default "aes-128-gcm" $proxy.Method -}}
    {{- if and $proxy.ServerKey (or (hasPrefix "2022-blake3-" $method) (eq $method "2022-blake3-aes-128-gcm") (eq $method "2022-blake3-aes-256-gcm")) -}}
      {{- $userKeyLen := ternary 16 32 (hasSuffix "128-gcm" $method) -}}
      {{- $pwdStr := printf "%s" $password -}}
      {{- $userKey := ternary $pwdStr (trunc $userKeyLen $pwdStr) (le (len $pwdStr) $userKeyLen) -}}
      {{- $password = printf "%s:%s" (b64enc $proxy.ServerKey) (b64enc $userKey) -}}
    {{- end -}}
  {{- end -}}
{
  "tag": "out-{{ $i }}",
  "protocol": {{ $proxy.Type | quote }},
  "settings": {
    {{- if eq $proxy.Type "vless" }}
    "vnext": [{ "address": {{ $proxy.Server | quote }}, "port": {{ $proxy.Port }}, "users": [{ "id": {{ $password | quote }}, "encryption": "none"{{ if and $proxy.Flow (ne $proxy.Flow "none") }}, "flow": {{ $proxy.Flow | quote }}{{ end }} }] }]
    {{- else if eq $proxy.Type "vmess" }}
    "vnext": [{ "address": {{ $proxy.Server | quote }}, "port": {{ $proxy.Port }}, "users": [{ "id": {{ $password | quote }}, "alterId": 0, "security": "auto" }] }]
    {{- else if eq $proxy.Type "trojan" }}
    "servers": [{ "address": {{ $proxy.Server | quote }}, "port": {{ $proxy.Port }}, "password": {{ $password | quote }} }]
    {{- else if eq $proxy.Type "shadowsocks" }}
    "servers": [{ "address": {{ $proxy.Server | quote }}, "port": {{ $proxy.Port }}, "method": {{ default "aes-128-gcm" $proxy.Method | quote }}, "password": {{ $password | quote }} }]
    {{- end }}
  }{{ template "streamSettings" $proxy }}
}
{{- end -}}
{
  "log": { "loglevel": "warning" },
  "inbounds": [
    {{- $state := dict "i" 0 "first" true -}}
    {{- range $proxy := .Proxies -}}
      {{- if or (eq $proxy.Type "vless") (eq $proxy.Type "vmess") (eq $proxy.Type "trojan") (eq $proxy.Type "shadowsocks") -}}
        {{- $i := int (get $state "i") -}}
        {{- if not (get $state "first") }},{{ end }}
        { "tag": "in-{{ $i }}", "listen": {{ $host | quote }}, "port": {{ add $basePort $i }}, "protocol": "socks", "settings": { "auth": "noauth", "udp": true } }
        {{- $_ := set $state "i" (add $i 1) -}}{{- $_ := set $state "first" false -}}
      {{- end -}}
    {{- end }}
  ],
  "outbounds": [
    {{- $state := dict "i" 0 "first" true -}}
    {{- range $proxy := .Proxies -}}
      {{- if or (eq $proxy.Type "vless") (eq $proxy.Type "vmess") (eq $proxy.Type "trojan") (eq $proxy.Type "shadowsocks") -}}
        {{- $i := int (get $state "i") -}}
        {{- if not (get $state "first") }},{{ end }}
        {{ template "outbound" (dict "proxy" $proxy "index" $i "UserInfo" $.UserInfo) }}
        {{- $_ := set $state "i" (add $i 1) -}}{{- $_ := set $state "first" false -}}
      {{- end -}}
    {{- end }}
  ],
  "routing": {
    "domainStrategy": "AsIs",
    "rules": [
      {{- $state := dict "i" 0 "first" true -}}
      {{- range $proxy := .Proxies -}}
        {{- if or (eq $proxy.Type "vless") (eq $proxy.Type "vmess") (eq $proxy.Type "trojan") (eq $proxy.Type "shadowsocks") -}}
          {{- $i := int (get $state "i") -}}
          {{- if not (get $state "first") }},{{ end }}
          { "type": "field", "inboundTag": ["in-{{ $i }}"], "outboundTag": "out-{{ $i }}" }
          {{- $_ := set $state "i" (add $i 1) -}}{{- $_ := set $state "first" false -}}
        {{- end -}}
      {{- end }}
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
VALUES (15, 'Surge Xray', '', 'Surge config that connects to local xray-core SOCKS inbounds while reusing the built-in Surge rules and policy groups.', 'surge:///install-config?url=${encodeURIComponent(url)}', 'surge-xray', FALSE, $alpha_subscription_template${{- $GiB := 1073741824.0 -}}
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

{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if hasKey $supportSet $proxy.Type -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}

{{- $proxyNames := "" -}}
{{- range $proxy := $supportedProxies -}}
  {{- $name := replace "," " " $proxy.Name -}}
  {{- if eq $proxyNames "" -}}
    {{- $proxyNames = $name -}}
  {{- else -}}
    {{- $proxyNames = printf "%s, %s" $proxyNames $name -}}
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

{{ $host := default "127.0.0.1" .Params.xray_inbound_host -}}
{{- $basePort := int (default "10800" .Params.xray_inbound_start_port) -}}

[Proxy]
{{- $state := dict "i" 0 -}}
{{- range $proxy := $supportedProxies }}
{{ replace "," " " $proxy.Name }} = socks5, {{ $host }}, {{ add $basePort (int (get $state "i")) }}, udp-relay=true, tfo=true
  {{- $_ := set $state "i" (add (int (get $state "i")) 1) -}}
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

SELECT setval(pg_get_serial_sequence('"subscribe_application"', 'id'), GREATEST(COALESCE((SELECT MAX("id") FROM "subscribe_application"), 0), 1), TRUE);
COMMIT;