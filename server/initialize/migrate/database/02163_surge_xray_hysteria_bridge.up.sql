-- 02163_surge_xray_hysteria_bridge.up.sql
-- Existing databases may still have the older Surge Xray bridge template that
-- filters out Hysteria2, while Xray Core already creates the local outbound.

BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true -}}',
    '{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true -}}'
),
    "updated_at" = NOW()
WHERE "id" = 15
  AND "user_agent" = 'surge-xray';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template", '{{- range $p := .Proxies -}}', '{{- range $idx, $p := .Proxies -}}'),
    "updated_at" = NOW()
WHERE "id" = 15
  AND "user_agent" = 'surge-xray';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '  {{- $_ := set $byKey (join "|" $keyParts) $p -}}',
    '  {{- $keyParts = append $keyParts (printf "%08d" $idx) -}}
  {{- $_ := set $byKey (join "|" $keyParts) $p -}}'
),
    "updated_at" = NOW()
WHERE "id" = 15
  AND "user_agent" = 'surge-xray'
  AND "subscribe_template" NOT LIKE '%printf "%08d" $idx%';

COMMIT;
