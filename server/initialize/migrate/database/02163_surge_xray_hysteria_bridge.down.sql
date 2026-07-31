-- 02163_surge_xray_hysteria_bridge.down.sql

BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    '{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true "hysteria2" true "hysteria" true -}}',
    '{{- $supportSet := dict "shadowsocks" true "vmess" true "vless" true "trojan" true -}}'
),
    "updated_at" = NOW()
WHERE "id" = 15
  AND "user_agent" = 'surge-xray';

COMMIT;
