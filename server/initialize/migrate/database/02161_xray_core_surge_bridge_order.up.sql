-- 02161_xray_core_surge_bridge_order.up.sql
-- Keep Xray Core local SOCKS inbounds aligned with the Surge Xray bridge proxy order.

BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '{{- $basePort := int (default "10800" .Params.xray_inbound_start_port) -}}
{{- define "streamSettings" -}}',
    '{{- $basePort := int (default "10800" .Params.xray_inbound_start_port) -}}
{{- $sortFields := list "Sort" "Port" "Name" -}}
{{- $sortConfig := dict "Sort" "asc" "Port" "asc" "Name" "asc" -}}
{{- $byKey := dict -}}
{{- range $idx, $p := .Proxies -}}
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
  {{- $keyParts = append $keyParts (printf "%08d" $idx) -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}
{{- end -}}
{{- $sorted := list -}}
{{- range $k := sortAlpha (keys $byKey) -}}
  {{- $sorted = append $sorted (index $byKey $k) -}}
{{- end -}}
{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true -}}
{{- $supportedProxies := list -}}
{{- range $proxy := $sorted -}}
  {{- if hasKey $supportSet $proxy.Type -}}
    {{- $supportedProxies = append $supportedProxies $proxy -}}
  {{- end -}}
{{- end -}}
{{- define "streamSettings" -}}'
),
    "updated_at" = NOW()
WHERE "id" = 14
  AND "subscribe_template" NOT LIKE '%$supportedProxies := list%';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template", '{{- range $proxy := .Proxies -}}', '{{- range $proxy := $supportedProxies -}}'),
    "updated_at" = NOW()
WHERE "id" = 14;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template", '{{- range $p := .Proxies -}}', '{{- range $idx, $p := .Proxies -}}'),
    "updated_at" = NOW()
WHERE "id" IN (14, 15);

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '  {{- $_ := set $byKey (join "|" $keyParts) $p -}}',
    '  {{- $keyParts = append $keyParts (printf "%08d" $idx) -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}'
),
    "updated_at" = NOW()
WHERE "id" IN (14, 15)
  AND "subscribe_template" NOT LIKE '%printf "%08d" $idx%';

COMMIT;
