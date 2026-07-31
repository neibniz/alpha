BEGIN;

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        REPLACE(
            "subscribe_template",
            $old$  {{- $network := default "tcp" $proxy.Transport -}}$old$,
            $new$  {{- $network := default "tcp" $proxy.Transport -}}
  {{- if and (or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria")) (eq (default "" $proxy.Transport) "") -}}{{- $network = "hysteria" -}}{{- end -}}$new$
        ),
        $old$"hysteriaSettings": { "version": 2{{ if $proxy.ServerKey }}, "auth": {{ $proxy.ServerKey | quote }}{{ end }} }$old$,
        $new$"hysteriaSettings": { "version": 2{{ if $proxy.ServerKey }}, "auth": {{ $proxy.ServerKey | quote }}{{ end }}{{ if $proxy.HysteriaUDPIdleTimeout }}, "udpIdleTimeout": {{ $proxy.HysteriaUDPIdleTimeout }}{{ end }}{{ if $proxy.HysteriaMasquerade }}, "masquerade": {{ $proxy.HysteriaMasquerade }}{{ end }} }$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 14 AND "user_agent" = 'xray-core';

COMMIT;
