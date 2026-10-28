-- Patch only recognized HY2/TLS blocks, including matching custom template copies.
-- Modified blocks are left untouched. Public certificate extraction requires this server release.
BEGIN;

-- 01-default.gotmpl.txt
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  $hy2_old$,
$hy2_new$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if $proxy.PinnedPeerCertSha256 -}}
    {{- $params = append $params (printf "pinSHA256=%s" (first (splitList "," $proxy.PinnedPeerCertSha256) | urlquery)) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if or $proxy.AllowInsecure $proxy.PinnedPeerCertSha256 -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  $hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'base64'
  AND strpos("subscribe_template", $hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  $hy2_old$) > 0;

-- 02-shadowrocket.gotmpl.txt
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  $hy2_old$,
$hy2_new$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if $proxy.PinnedPeerCertSha256 -}}
    {{- $params = append $params (printf "pinSHA256=%s" (first (splitList "," $proxy.PinnedPeerCertSha256) | urlquery)) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if or $proxy.AllowInsecure $proxy.PinnedPeerCertSha256 -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  $hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'base64'
  AND strpos("subscribe_template", $hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  {{- $params := list -}}
  {{- if ne (default "" $proxy.SNI) "" -}}
    {{- $params = append $params (printf "sni=%s" $proxy.SNI) -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $params = append $params "insecure=1" -}}
  {{- end -}}
  {{- if ne (default "" $proxy.ObfsPassword) "" -}}
    {{- $params = append $params (printf "obfs=salamander&obfs-password=%s" $proxy.ObfsPassword) -}}
  {{- end -}}
  {{- if ne (default "" $proxy.HopPorts) "" -}}
    {{- $params = append $params (printf "mport=%s" $proxy.HopPorts) -}}
  {{- end }}
hysteria2://{{- if ne $password "" -}}{{ $password }}@{{- end -}}{{ $server }}:{{ $proxy.Port }}?{{ join "&" (append $params $common) }}#{{ $proxy.Name | urlquery }}
  $hy2_old$) > 0;

-- 03-clash.gotmpl.yaml
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
- name: {{ $proxy.Name | quote }}
  type: hysteria2
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.ObfsPassword) "" }}
  obfs: salamander
  obfs-password: {{ $proxy.ObfsPassword }}
  {{- end }}
  {{- if ne (default "" $proxy.HopPorts) "" }}
  ports: {{ $proxy.HopPorts }}
  {{- end }}
  {{- if ne (default 0 $proxy.HopInterval) 0 }}
  hop-interval: {{ $proxy.HopInterval }}
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.UpMbps)) "" }}
  up: "{{ $proxy.UpMbps }} Mbps"
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.DownMbps)) "" }}
  down: "{{ $proxy.DownMbps }} Mbps"
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
$hy2_old$,
$hy2_new${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
- name: {{ $proxy.Name | quote }}
  type: hysteria2
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  {{- if $proxy.PinnedPeerCertSha256 }}
  fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}
  {{- end }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if and $proxy.AllowInsecure (not $proxy.PinnedPeerCertSha256) }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.ObfsPassword) "" }}
  obfs: salamander
  obfs-password: {{ $proxy.ObfsPassword }}
  {{- end }}
  {{- if ne (default "" $proxy.HopPorts) "" }}
  ports: {{ $proxy.HopPorts }}
  {{- end }}
  {{- if ne (default 0 $proxy.HopInterval) 0 }}
  hop-interval: {{ $proxy.HopInterval }}
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.UpMbps)) "" }}
  up: "{{ $proxy.UpMbps }} Mbps"
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.DownMbps)) "" }}
  down: "{{ $proxy.DownMbps }} Mbps"
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
$hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'yaml'
  AND strpos("subscribe_template", $hy2_old${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
- name: {{ $proxy.Name | quote }}
  type: hysteria2
  server: {{ $server }}
  port: {{ $proxy.Port }}
  password: {{ $password }}
  udp: true
  tfo: true
  {{- if ne (default "" $proxy.SNI) "" }}
  sni: {{ $proxy.SNI }}
  {{- end }}
  {{- if $proxy.AllowInsecure }}
  skip-cert-verify: true
  {{- end }}
  {{- if ne (default "" $proxy.ObfsPassword) "" }}
  obfs: salamander
  obfs-password: {{ $proxy.ObfsPassword }}
  {{- end }}
  {{- if ne (default "" $proxy.HopPorts) "" }}
  ports: {{ $proxy.HopPorts }}
  {{- end }}
  {{- if ne (default 0 $proxy.HopInterval) 0 }}
  hop-interval: {{ $proxy.HopInterval }}
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.UpMbps)) "" }}
  up: "{{ $proxy.UpMbps }} Mbps"
  {{- end }}
  {{- if ne (default "" (printf "%v" $proxy.DownMbps)) "" }}
  down: "{{ $proxy.DownMbps }} Mbps"
  {{- end }}
  {{- if $proxy.EchEnable }}
  ech-opts:
    enable: true
    {{- if ne (default "" $proxy.EchServerName) "" }}
    query-server-name: {{ $proxy.EchServerName }}
    {{- end }}
  {{- end }}
$hy2_old$) > 0;

-- 04-singbox.gotmpl.json
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old${{- $tlsOpts := "" -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if $proxy.Fingerprint -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

$hy2_old$,
$hy2_new${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- if and $hy2Trust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for sing-box 1.12" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $hy2Trust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

$hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'json'
  AND strpos("subscribe_template", $hy2_old${{- $tlsOpts := "" -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if $proxy.Fingerprint -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

$hy2_old$) > 0;

-- 05-surge.gotmpl.conf
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = hysteria2, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs=salamander, obfs-password={{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports={{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval={{ $proxy.HopInterval }}{{- end }}, {{ $common }}
  $hy2_old$,
$hy2_new$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = hysteria2, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs=salamander, obfs-password={{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports={{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval={{ $proxy.HopInterval }}{{- end }}, {{ $common }}
  $hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'conf'
  AND strpos("subscribe_template", $hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = hysteria2, {{ $server }}, {{ $proxy.Port }}, password={{ $password }}{{- if ne (default "" $proxy.SNI) "" }}, sni={{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify=true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs=salamander, obfs-password={{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports={{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval={{ $proxy.HopInterval }}{{- end }}, {{ $common }}
  $hy2_old$) > 0;

-- 06-stash.gotmpl.yaml
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  $hy2_old$,
$hy2_new$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, auth: {{ $password | quote }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, server-cert-fingerprint: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  $hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'yaml'
  AND strpos("subscribe_template", $hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $SkipVerify }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  $hy2_old$) > 0;

-- 08-loon.gotmpl.conf
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = Hysteria2,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.ObfsPassword }},obfs=salamander,obfs-password={{ $proxy.ObfsPassword }}{{ end }}{{- if $proxy.HopPorts }},port-hopping={{ $proxy.HopPorts }}{{ end }}{{- if $proxy.HopInterval }},hop-interval={{ $proxy.HopInterval }}{{ end }}
$hy2_old$,
$hy2_new${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = Hysteria2,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.PinnedPeerCertSha256 }},tls-cert-sha256={{ first (splitList "," $proxy.PinnedPeerCertSha256) }}{{- else if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.ObfsPassword }},obfs=salamander,obfs-password={{ $proxy.ObfsPassword }}{{ end }}{{- if $proxy.HopPorts }},port-hopping={{ $proxy.HopPorts }}{{ end }}{{- if $proxy.HopInterval }},hop-interval={{ $proxy.HopInterval }}{{ end }}
$hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'conf'
  AND strpos("subscribe_template", $hy2_old${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
{{ $proxy.Name }} = Hysteria2,{{ $proxy.Server }},{{ $proxy.Port }},{{ $.UserInfo.Password }}{{- if $proxy.SNI }},sni={{ $proxy.SNI }}{{ end }}{{- if $proxy.AllowInsecure }},skip-cert-verify={{ $proxy.AllowInsecure }}{{ end }}{{- if $proxy.ObfsPassword }},obfs=salamander,obfs-password={{ $proxy.ObfsPassword }}{{ end }}{{- if $proxy.HopPorts }},port-hopping={{ $proxy.HopPorts }}{{ end }}{{- if $proxy.HopInterval }},hop-interval={{ $proxy.HopInterval }}{{ end }}
$hy2_old$) > 0;

-- 11-egern.gotmpl.yaml
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  $hy2_old$,
$hy2_new$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - hysteria2: { name: {{ $proxy.Name | quote }}, server: {{ $proxy.Server | quote }}, port: {{ $proxy.Port }}, auth: {{ $password | quote }}{{- if $proxy.SNI }}, sni: {{ $proxy.SNI | quote }}{{- end }}{{- if $proxy.PinnedPeerCertSha256 }}, fingerprint_sha256: {{ first (splitList "," $proxy.PinnedPeerCertSha256) | quote }}{{- else if $proxy.AllowInsecure }}, skip_tls_verify: true{{- end }}{{- if $proxy.ObfsPassword }}, obfs: salamander, obfs_password: {{ $proxy.ObfsPassword | quote }}{{- end }}{{- if $proxy.HopPorts }}, port_hopping: {{ $proxy.HopPorts | quote }}{{- end }} }
  $hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'yaml'
  AND strpos("subscribe_template", $hy2_old$  {{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") }}
  - { name: {{ $proxy.Name | quote }}, type: hysteria2, server: {{ $server }}, port: {{ $proxy.Port }}, password: {{ $password }}, {{ $common }}{{- if ne (default "" $proxy.SNI) "" }}, sni: {{ $proxy.SNI }}{{- end }}{{- if $proxy.AllowInsecure }}, skip-cert-verify: true{{- end }}{{- if ne (default "" $proxy.ObfsPassword) "" }}, obfs: salamander, obfs-password: {{ $proxy.ObfsPassword }}{{- end }}{{- if ne (default "" $proxy.HopPorts) "" }}, ports: {{ $proxy.HopPorts }}{{- end }}{{- if ne (default 0 $proxy.HopInterval) 0 }}, hop-interval: {{ $proxy.HopInterval }}{{- end }} }
  $hy2_old$) > 0;

-- 12-hiddify.gotmpl.json
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") -}}
    { "type": "hysteria2", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if $proxy.ObfsPassword }}, "obfs": { "type": "salamander", "password": {{ $proxy.ObfsPassword | quote }} }{{ end }}{{ if $proxy.HopPorts }}, "ports": {{ $proxy.HopPorts | quote }}{{ end }}{{ if $proxy.HopInterval }}, "hop_interval": {{ $proxy.HopInterval }}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
$hy2_old$,
$hy2_new${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") -}}
    {{- if and $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
      {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for Hiddify" -}}
    {{- end -}}
    { "type": "hysteria2", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if $proxy.ObfsPassword }}, "obfs": { "type": "salamander", "password": {{ $proxy.ObfsPassword | quote }} }{{ end }}{{ if $proxy.HopPorts }}, "ports": {{ $proxy.HopPorts | quote }}{{ end }}{{ if $proxy.HopInterval }}, "hop_interval": {{ $proxy.HopInterval }}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $proxy.TLSCertificate }}, "certificate": {{ toJson $proxy.TLSCertificate }}{{ else if $SkipVerify }}, "insecure": true{{ end }}} }
$hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'json'
  AND strpos("subscribe_template", $hy2_old${{- else if or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hysteria") -}}
    { "type": "hysteria2", "tag": {{ $proxy.Name | quote }}, "server": {{ $server | quote }}, "server_port": {{ $proxy.Port }}, "password": {{ $password | quote }}{{ if $proxy.ObfsPassword }}, "obfs": { "type": "salamander", "password": {{ $proxy.ObfsPassword | quote }} }{{ end }}{{ if $proxy.HopPorts }}, "ports": {{ $proxy.HopPorts | quote }}{{ end }}{{ if $proxy.HopInterval }}, "hop_interval": {{ $proxy.HopInterval }}{{ end }}, {{ $common }}, "tls": {"enabled": true{{ if $proxy.SNI }}, "server_name": {{ $proxy.SNI | quote }}{{ end }}{{ if $SkipVerify }}, "insecure": true{{ end }}{{ if $proxy.Fingerprint }}, "utls": {"enabled": true, "fingerprint": {{ $proxy.Fingerprint | quote }} }{{ end }}} }
$hy2_old$) > 0;

-- 13-singbox-1-11.gotmpl.json
UPDATE "subscribe_application"
SET "subscribe_template" = REPLACE("subscribe_template",
$hy2_old${{- $tlsOpts := "" -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if $proxy.Fingerprint -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

$hy2_old$,
$hy2_new${{- $tlsOpts := "" -}}
{{- $hy2Trust := or (eq $proxy.Type "hysteria2") (eq $proxy.Type "hy2") -}}
{{- if and $hy2Trust $proxy.PinnedPeerCertSha256 (not $proxy.TLSCertificate) -}}
  {{- fail "HY2 certificate pin requires a matching inline public PEM certificate for sing-box 1.11" -}}
{{- end -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if and $hy2Trust $proxy.TLSCertificate -}}
    {{- $tlsOpts = printf "%s, \"certificate\": %s" $tlsOpts (toJson $proxy.TLSCertificate) -}}
  {{- else if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if and $proxy.Fingerprint (not $hy2Trust) -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

$hy2_new$),
"updated_at" = NOW()
WHERE "output_format" = 'json'
  AND strpos("subscribe_template", $hy2_old${{- $tlsOpts := "" -}}
{{- if or $tlsEnabled $proxy.AllowInsecure $proxy.Fingerprint -}}
  {{- $tlsOpts = "\"tls\": {\"enabled\": true" -}}
  {{- if $sni -}}
    {{- $tlsOpts = printf "%s, \"server_name\": \"%s\"" $tlsOpts $sni -}}
  {{- end -}}
  {{- if $proxy.AllowInsecure -}}
    {{- $tlsOpts = printf "%s, \"insecure\": true" $tlsOpts -}}
  {{- end -}}
  {{- if $proxy.Fingerprint -}}
    {{- $tlsOpts = printf "%s, \"utls\": {\"enabled\": true, \"fingerprint\": \"%s\"}" $tlsOpts ($proxy.Fingerprint) -}}
  {{- end -}}
  {{- if and (eq $proxy.Security "reality") $proxy.RealityPublicKey -}}
    {{- $tlsOpts = printf "%s, \"reality\": {\"enabled\": true, \"public_key\": \"%s\"" $tlsOpts ($proxy.RealityPublicKey) -}}
    {{- if $proxy.RealityShortId -}}
      {{- $tlsOpts = printf "%s, \"short_id\": \"%s\"" $tlsOpts ($proxy.RealityShortId) -}}
    {{- end -}}
    {{- $tlsOpts = printf "%s}" $tlsOpts -}}
  {{- end -}}
  {{- $tlsOpts = printf "%s}" $tlsOpts -}}
{{- end -}}

$hy2_old$) > 0;

COMMIT;

