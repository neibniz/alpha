BEGIN;

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "default" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 1 AND "user_agent" = 'default';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "shadowrocket" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 2 AND "user_agent" = 'Shadowrocket';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        REPLACE(
            "subscribe_template",
            $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "clash" $proxy) -}}$old$,
            $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
        ),
        $old$  xhttp-opts:
{{ xhttpMihomoOpts $proxy 4 -}}$old$,
        $new$  xhttp-opts:
    path: {{ default "/" $proxy.Path }}
    {{- if ne (default "" $proxy.Host) "" }}
    host: {{ $proxy.Host }}
    {{- end }}
    {{- if ne (default "" $proxy.XhttpMode) "" }}
    mode: {{ $proxy.XhttpMode }}
    {{- end }}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 3 AND "user_agent" = 'Clash';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 5 AND "user_agent" = 'Surge';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        REPLACE(
            "subscribe_template",
            $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "stash" $proxy) -}}$old$,
            $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
        ),
        $old$network: xhttp, xhttp-opts: {{ xhttpMihomoInlineOpts $proxy }}$old$,
        $new$network: xhttp, xhttp-opts: { path: {{ default "/" $proxy.Path }}{{- if ne (default "" $proxy.Host) "" }}, host: {{ $proxy.Host }}{{- end }}{{- if ne (default "" $proxy.XhttpMode) "" }}, mode: {{ $proxy.XhttpMode }}{{- end }} }$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 6 AND "user_agent" = 'Stash';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "loon" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 8 AND "user_agent" = 'Loon';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "quantumult" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 9 AND "user_agent" = 'Quantumult';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxyType) (clientSupportsProxy "quantumult-x" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxyType -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 10 AND "user_agent" = 'Quantumult%20X';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "egern" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 11 AND "user_agent" = 'Egern';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "hiddify" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 12 AND "user_agent" = 'Hiddify';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        REPLACE(
            "subscribe_template",
            $old$len $supportedProxies$old$,
            $new$len .Proxies$new$
        ),
        $old$range $i, $proxy := $supportedProxies$old$,
        $new$range $i, $proxy := .Proxies$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 13 AND "user_agent" = 'sing-box';

UPDATE "subscribe_application"
SET
    "subscribe_template" = REPLACE(
        "subscribe_template",
        $old$  {{- if and (hasKey $supportSet $proxy.Type) (clientSupportsProxy "surge-xray" $proxy) -}}$old$,
        $new$  {{- if hasKey $supportSet $proxy.Type -}}$new$
    ),
    "updated_at" = NOW()
WHERE "id" = 15 AND "user_agent" = 'surge-xray';

COMMIT;
