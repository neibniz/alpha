-- 02169_subscription_vless_flow_guards.up.sql
-- Normalize VLESS flow rendering across built-in client templates. Flow should
-- only be emitted when it is explicitly configured and not "none".

BEGIN;

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    $old${{- if ne (default "" $proxy.Flow) "none" -}}$old$,
    $new${{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") -}}$new$
  ),
  "updated_at" = NOW()
WHERE "id" = 2
  AND "subscribe_template" LIKE '%{{- if ne (default "" $proxy.Flow) "none" -}}%';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    $old${{- if ne (default "" $proxy.Flow) "" }}$old$,
    $new${{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}$new$
  ),
  "updated_at" = NOW()
WHERE "id" = 3
  AND "subscribe_template" LIKE '%{{- if ne (default "" $proxy.Flow) "" }}%';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    $old${{- if $proxy.Flow -}}$old$,
    $new${{- if and $proxy.Flow (ne $proxy.Flow "none") -}}$new$
  ),
  "updated_at" = NOW()
WHERE "id" = 4
  AND "subscribe_template" LIKE '%{{- if $proxy.Flow -}}%';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    $old${{- if ne (default "" $proxy.Flow) "none" }}, flow={{ $proxy.Flow }}{{- end }}$old$,
    $new${{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow={{ $proxy.Flow }}{{- end }}$new$
  ),
  "updated_at" = NOW()
WHERE "id" = 5
  AND "subscribe_template" LIKE '%{{- if ne (default "" $proxy.Flow) "none" }}, flow={{ $proxy.Flow }}{{- end }}%';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    $old${{- if ne (default "" $proxy.Flow) "none" }}, flow: {{ $proxy.Flow }}{{- end }}$old$,
    $new${{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow: {{ $proxy.Flow }}{{- end }}$new$
  ),
  "updated_at" = NOW()
WHERE "id" = 6
  AND "subscribe_template" LIKE '%{{- if ne (default "" $proxy.Flow) "none" }}, flow: {{ $proxy.Flow }}{{- end }}%';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    $old${{- if $proxy.Flow }},flow={{ $proxy.Flow }}{{ end }}$old$,
    $new${{- if and $proxy.Flow (ne $proxy.Flow "none") }},flow={{ $proxy.Flow }}{{ end }}$new$
  ),
  "updated_at" = NOW()
WHERE "id" = 8
  AND "subscribe_template" LIKE '%{{- if $proxy.Flow }},flow={{ $proxy.Flow }}{{ end }}%';

UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(
    "subscribe_template",
    $old${{- else if eq $proxy.Type "vless" -}}
{ "type": "vless", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $pwd }}"{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}{{ if $tlsOpts }}, {{ $tlsOpts }}{{ end }} }$old$,
    $new${{- else if eq $proxy.Type "vless" -}}
{{- $flowOpts := "" -}}
{{- if and $proxy.Flow (ne $proxy.Flow "none") -}}
  {{- $flowOpts = printf ", \"flow\": \"%s\"" ($proxy.Flow) -}}
{{- end -}}
{ "type": "vless", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $pwd }}"{{ $flowOpts }}{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}{{ if $tlsOpts }}, {{ $tlsOpts }}{{ end }} }$new$
  ),
  "updated_at" = NOW()
WHERE "id" = 13
  AND "subscribe_template" LIKE '%{ "type": "vless", "tag": "{{ $name }}", "server": "{{ $server }}", "server_port": {{ $port }}, "uuid": "{{ $pwd }}"{{ if $transportOpts }}, {{ $transportOpts }}{{ end }}{{ if $tlsOpts }}, {{ $tlsOpts }}{{ end }} }%';

COMMIT;
