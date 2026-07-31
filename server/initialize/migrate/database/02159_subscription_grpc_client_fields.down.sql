BEGIN;

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $new$"grpcSettings": { "serviceName": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcAuthority }}, "authority": {{ $proxy.GrpcAuthority | quote }}{{ end }}{{ if $proxy.GrpcMultiMode }}, "multiMode": true{{ end }}{{ if $proxy.GrpcUserAgent }}, "user_agent": {{ $proxy.GrpcUserAgent | quote }}{{ end }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ $proxy.GrpcIdleTimeout }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "health_check_timeout": {{ $proxy.GrpcHealthCheckTimeout }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}{{ if $proxy.GrpcInitialWindowsSize }}, "initial_windows_size": {{ $proxy.GrpcInitialWindowsSize }}{{ end }} }$new$,
        $old$"grpcSettings": { "serviceName": {{ default "grpc" $proxy.ServiceName | quote }} }$old$
    ),
    "updated_at" = NOW()
WHERE "id" = 14 AND "user_agent" = 'xray-core';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $new${{- $transportOpts = printf "\"transport\": {\"type\": \"grpc\", \"service_name\": \"%s\"" $grpcService -}}
  {{- if $proxy.GrpcIdleTimeout -}}{{- $transportOpts = printf "%s, \"idle_timeout\": \"%ds\"" $transportOpts $proxy.GrpcIdleTimeout -}}{{- end -}}
  {{- if $proxy.GrpcHealthCheckTimeout -}}{{- $transportOpts = printf "%s, \"ping_timeout\": \"%ds\"" $transportOpts $proxy.GrpcHealthCheckTimeout -}}{{- end -}}
  {{- if $proxy.GrpcPermitWithoutStream -}}{{- $transportOpts = printf "%s, \"permit_without_stream\": true" $transportOpts -}}{{- end -}}
  {{- $transportOpts = printf "%s}" $transportOpts -}}$new$,
        $old${{- $transportOpts = printf "\"transport\": {\"type\": \"grpc\", \"service_name\": \"%s\"}" $grpcService -}}$old$
    ),
    "updated_at" = NOW()
WHERE "id" IN (4, 13) AND "user_agent" = 'sing-box';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $new$"transport": {"type": "grpc", "service_name": {{ default "grpc" $proxy.ServiceName | quote }}{{ if $proxy.GrpcIdleTimeout }}, "idle_timeout": {{ printf "%ds" $proxy.GrpcIdleTimeout | quote }}{{ end }}{{ if $proxy.GrpcHealthCheckTimeout }}, "ping_timeout": {{ printf "%ds" $proxy.GrpcHealthCheckTimeout | quote }}{{ end }}{{ if $proxy.GrpcPermitWithoutStream }}, "permit_without_stream": true{{ end }}}$new$,
        $old$"transport": {"type": "grpc", "service_name": {{ $proxy.ServiceName | quote }}}$old$
    ),
    "updated_at" = NOW()
WHERE "id" = 12 AND "user_agent" = 'Hiddify';

COMMIT;
