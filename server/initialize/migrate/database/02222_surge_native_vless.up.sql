-- Refresh recognized Surge blocks without replacing customized profiles.
WITH patch (old_filter, new_filter, old_vless, new_vless, old_group, new_group) AS (VALUES (
    $old_filter$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge" $proxy) -}}$old_filter$,
    $new_filter$  {{- if and (hasKey $supportSet $proxy.Type) (surgeSupportsProxy $proxy) -}}$new_filter$,
    $old_vless$  {{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = vless, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{- if or (eq $proxy.Transport "ws") (eq $proxy.Transport "websocket") }}, ws=true{{- if ne (default "" $proxy.Path) "" }}, ws-path={{ $proxy.Path }}{{- end }}{{- if ne (default "" $proxy.Host) "" }}, ws-headers="Host:{{ $proxy.Host }}"{{- end }}{{- else if eq $proxy.Transport "grpc" }}, grpc=true{{- if ne (default "" $proxy.ServiceName) "" }}, grpc-service-name={{ $proxy.ServiceName }}{{- end }}{{- end }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if and (ne $proxy.Security "reality") $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if and (ne (default "" $proxy.Flow) "") (ne $proxy.Flow "none") }}, flow={{ $proxy.Flow }}{{- end }}, {{ $common }}$old_vless$,
    $new_vless$  {{- else if eq $proxy.Type "vless" }}
{{ $proxy.Name }} = vless, {{ $server }}, {{ $proxy.Port }}, username={{ $password }}{{ surgeVLESSOptions $proxy }}$new_vless$,
    $old_group$🇺🇳 Nodes = select, {{ $proxyNames }}, hidden=1$old_group$,
    $new_group$🇺🇳 Nodes = select, {{ default "REJECT" $proxyNames }}, hidden=1$new_group$
))
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE(REPLACE(REPLACE("subscribe_template",
    patch.old_filter, patch.new_filter),
    patch.old_vless, patch.new_vless),
    patch.old_group, patch.new_group),
    "updated_at" = NOW()
FROM patch
WHERE "output_format" = 'conf'
  AND strpos("subscribe_template", patch.old_filter) > 0
  AND strpos("subscribe_template", patch.old_vless) > 0
  AND strpos("subscribe_template", patch.old_group) > 0;
