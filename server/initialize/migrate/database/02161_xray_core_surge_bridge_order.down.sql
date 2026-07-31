-- 02161_xray_core_surge_bridge_order.down.sql

BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '{{- $sortFields := list "Sort" "Port" "Name" -}}
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
',
    ''
),
    "updated_at" = NOW()
WHERE "id" = 14;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template", '{{- range $proxy := $supportedProxies -}}', '{{- range $proxy := .Proxies -}}'),
    "updated_at" = NOW()
WHERE "id" = 14;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '  {{- $keyParts = append $keyParts (printf "%08d" $idx) -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}',
    '  {{- $_ := set $byKey (join "|" $keyParts) $p -}}'
),
    "updated_at" = NOW()
WHERE "id" IN (14, 15);

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template", '{{- range $idx, $p := .Proxies -}}', '{{- range $p := .Proxies -}}'),
    "updated_at" = NOW()
WHERE "id" IN (14, 15);

COMMIT;
