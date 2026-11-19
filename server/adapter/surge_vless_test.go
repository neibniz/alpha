package adapter

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

type surgePatch struct{ before, after string }

func surgeMigrationPatches(t *testing.T, direction string) []surgePatch {
	t.Helper()
	sql, err := os.ReadFile("../initialize/migrate/database/02222_surge_native_vless." + direction + ".sql")
	if err != nil {
		t.Fatal(err)
	}
	var patches []surgePatch
	for _, name := range []string{"filter", "vless", "group"} {
		pattern := regexp.MustCompile("(?s)\\$old_" + name + "\\$(.*?)\\$old_" + name + "\\$,\\s*\\$new_" + name + "\\$(.*?)\\$new_" + name + "\\$")
		match := pattern.FindSubmatch(sql)
		if match == nil {
			t.Fatalf("missing Surge migration patch %s", name)
		}
		patches = append(patches, surgePatch{string(match[1]), string(match[2])})
	}
	return patches
}

func applySurgePatches(profile string, patches []surgePatch) string {
	for _, patch := range patches {
		if !strings.Contains(profile, patch.before) {
			return profile
		}
	}
	for _, patch := range patches {
		profile = strings.ReplaceAll(profile, patch.before, patch.after)
	}
	return profile
}

func TestSurgeMigrationPreservesCustomContent(t *testing.T) {
	up := surgeMigrationPatches(t, "up")
	down := surgeMigrationPatches(t, "down")
	template := findMigrationTemplate(t, 5).templateText
	old := applySurgePatches(template, down)
	if old == template || applySurgePatches(old, up) != template {
		t.Fatal("Surge migration must round-trip")
	}
	customRule := "\nDOMAIN,example.test,DIRECT\n"
	if got := applySurgePatches(old+customRule, up); got != template+customRule {
		t.Fatal("Surge migration removed unrelated rules")
	}
	customBlock := strings.Replace(old, up[1].before, "# Custom VLESS block", 1)
	if got := applySurgePatches(customBlock, up); got != customBlock {
		t.Fatal("Surge migration partially patched a customized template")
	}
	if got := applySurgePatches(template, up); got != template {
		t.Fatal("Surge migration is not idempotent")
	}
}

func surgeProxyLine(profile, name string) string {
	for line := range strings.SplitSeq(profile, "\n") {
		if strings.HasPrefix(line, name+" = vless, ") {
			return line
		}
	}
	return ""
}

func TestSurgeNativeVLESS(t *testing.T) {
	pin := strings.Repeat("ab", 32)
	template := findMigrationTemplate(t, 5)
	tests := []struct {
		name    string
		proxy   Proxy
		want    []string
		notWant []string
	}{
		{
			name: "RAW REALITY Vision",
			proxy: Proxy{Transport: "raw", Security: "reality", SNI: "sni.example.test",
				RealityPublicKey: "public-key", RealityShortId: "01ab", Flow: "xtls-rprx-vision",
				Fingerprint: "chrome140", AllowInsecure: true, PinnedPeerCertSha256: pin},
			want:    []string{"sni=sni.example.test", "reality-public-key=public-key", "reality-short-id=01ab", "flow=xtls-rprx-vision"},
			notWant: []string{"tls=", "skip-cert-verify", "server-cert-fingerprint", "fingerprint=", "udp-relay"},
		},
		{
			name: "gRPC REALITY",
			proxy: Proxy{Transport: "grpc", Security: "reality", SNI: "sni.example.test", RealityPublicKey: "public-key",
				ServiceName: "/custom/path/tun|tunmulti", GrpcAuthority: "authority.example.test", GrpcMultiMode: true, Flow: "none"},
			want:    []string{"grpc=true", "grpc-service-name=/custom/path/tun|tunmulti", "grpc-authority=authority.example.test", "grpc-mode=multi", "reality-public-key=public-key"},
			notWant: []string{"flow=", "reality-short-id=", "tls="},
		},
		{
			name: "WebSocket TLS",
			proxy: Proxy{Transport: "websocket", Security: "tls", Host: "cdn.example.test",
				Path: "/ws?arg=1,2", PinnedPeerCertSha256: pin + "," + strings.Repeat("cd", 32), AllowInsecure: true},
			want:    []string{"tls=true", "ws=true", "ws-path=\"/ws?arg=1,2\"", "ws-headers=Host:cdn.example.test", "server-cert-fingerprint-sha256=" + pin},
			notWant: []string{"skip-cert-verify", strings.Repeat("cd", 32), "ws-http-upgrade"},
		},
		{
			name: "HTTPUpgrade TLS",
			proxy: Proxy{Transport: "httpupgrade", Security: "tls", Path: "/upgrade", SNI: "cdn.example.test",
				VerifyPeerCertByName: " , cert.example.test, other.example.test", ALPN: []string{"http/1.1"}},
			want:    []string{"ws=true", "ws-http-upgrade=true", "ws-path=/upgrade", "tls=true", "server-cert-verify-name=cert.example.test", "alpn=http/1.1"},
			notWant: []string{"other.example.test"},
		},
		{
			name: "gRPC TLS",
			proxy: Proxy{Transport: "grpc", Security: "tls", SNI: "sni.example.test", ServiceName: "shop",
				ALPN: []string{"h2", "http/1.1"}, AllowInsecure: true},
			want:    []string{"grpc=true", "grpc-service-name=shop", "tls=true", "alpn=\"h2,http/1.1\"", "skip-cert-verify=true"},
			notWant: []string{"reality-", "flow=", "grpc-mode="},
		},
		{
			name: "XHTTP TLS",
			proxy: Proxy{Transport: "splithttp", Security: "tls", Host: "cdn.example.test", XhttpMode: "packet-up",
				XhttpExtra: "{\"headers\":{\"User-Agent\":\"chrome\",\"Host\":\"ignored.test\",\"X-Token\":\"a,b\"},\"sessionIDPlacement\":\"query\",\"scMinPostsIntervalMs\":30,\"uplinkHTTPMethod\":\"get\",\"xmux\":{\"maxConnections\":\"1-2\",\"cMaxReuseTimes\":18,\"hMaxRequestTimes\":\"600-900\",\"hMaxReusableSecs\":3600,\"hKeepAlivePeriod\":-1},\"serverMaxHeaderBytes\":8192}"},
			want:    []string{"xhttp=true", "xhttp-host=cdn.example.test", "xhttp-mode=packet-up", "tls=true", "xhttp-headers=\"User-Agent:chrome|X-Token:a,b\"", "xhttp-session-placement=query", "xhttp-min-posts-interval=30", "xhttp-uplink-method=GET", "xhttp-xmux-max-connections=1-2", "xhttp-xmux-max-reuse-times=18", "xhttp-xmux-max-request-times=600-900", "xhttp-xmux-max-reusable-secs=3600", "xhttp-xmux-keep-alive=-1"},
			notWant: []string{"serverMaxHeaderBytes", "ignored.test", "ws=true", "grpc=true"},
		},
		{
			name: "XHTTP concurrency and typed placeholders",
			proxy: Proxy{Transport: "xhttp", Security: "tls",
				XhttpExtra: "{\"noGRPCHeader\":false,\"xmux\":{\"maxConcurrency\":\"16-32\",\"maxConnections\":\"0-0\",\"hKeepAlivePeriod\":0,\"cMaxReuseTimes\":null,\"hMaxRequestTimes\":{},\"hMaxReusableSecs\":true}}"},
			want:    []string{"xhttp-xmux-max-concurrency=16-32", "xhttp-no-grpc-header=false"},
			notWant: []string{"xhttp-xmux-max-connections", "xhttp-xmux-keep-alive", "xhttp-xmux-max-reuse-times", "xhttp-xmux-max-request-times", "xhttp-xmux-max-reusable-secs"},
		},
		{
			name: "RAW encrypted Vision",
			proxy: Proxy{Transport: "tcp", Security: "none", Flow: "xtls-rprx-vision-udp443", Encryption: "mlkem768x25519plus",
				EncryptionMode: "native", EncryptionRtt: "0rtt", EncryptionClientPadding: "100-111-1111", EncryptionPassword: "public-key", EncryptionPrivateKey: "private-key"},
			want:    []string{"flow=xtls-rprx-vision-udp443", "encryption=mlkem768x25519plus.native.0rtt.100-111-1111.public-key"},
			notWant: []string{"tls=", "private-key"},
		},
		{
			name:  "WebSocket encrypted Vision",
			proxy: Proxy{Transport: "ws", Flow: "xtls-rprx-vision", Encryption: "mlkem768x25519plus.native.1rtt.public-key"},
			want:  []string{"ws=true", "flow=xtls-rprx-vision", "encryption=mlkem768x25519plus.native.1rtt.public-key"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proxy := tc.proxy
			proxy.Name, proxy.Type, proxy.Server, proxy.Port = "Node", "vless", "node.example.test", 443
			profile := renderTemplateString(t, template, []Proxy{proxy}, User{Password: "test-user"}, nil)
			line := surgeProxyLine(profile, proxy.Name)
			if !strings.HasPrefix(line, "Node = vless, node.example.test, 443, username=test-user, ") {
				t.Fatalf("missing VLESS declaration: %q", line)
			}
			for _, want := range tc.want {
				if !strings.Contains(line, ", "+want) {
					t.Errorf("missing %q: %s", want, line)
				}
			}
			for _, notWant := range tc.notWant {
				if strings.Contains(line, notWant) {
					t.Errorf("unexpected %q: %s", notWant, line)
				}
			}
		})
	}
}

func TestSurgeNativeVLESSOmitsUnsupportedNodes(t *testing.T) {
	template := findMigrationTemplate(t, 5)
	for _, proxy := range []Proxy{
		{Transport: "kcp"},
		{Transport: "tcp", HeaderType: "http"},
		{Transport: "grpc", Security: "none"},
		{Transport: "grpc", Security: "tls", Flow: "xtls-rprx-vision"},
		{Transport: "ws", Security: "reality", SNI: "sni.example.test", RealityPublicKey: "key"},
		{Transport: "ws", Security: "tls", Flow: "xtls-rprx-vision"},
		{Transport: "xhttp", Security: "tls", XhttpExtra: "{\"downloadSettings\":{}}"},
		{Transport: "splithttp", Security: "tls", XhttpDownloadSettings: true},
		{Transport: "xhttp", Security: "tls", XhttpMode: "stream-up", XhttpExtra: "{\"uplinkHTTPMethod\":\"get\"}"},
		{Transport: "xhttp", Security: "tls", XhttpMode: "stream-one", XhttpExtra: "{\"uplinkDataPlacement\":\"cookie\"}"},
		{Transport: "raw", Security: "reality", SNI: "sni.example.test"},
		{Transport: "raw", Encryption: "mlkem768x25519plus", EncryptionMode: "native", EncryptionRtt: "0rtt"},
		{Transport: "raw", EncryptionRequired: true},
	} {
		proxy.Type, proxy.Name, proxy.Server, proxy.Port = "vless", "UnsupportedNode", "node.example.test", 443
		profile := renderTemplateString(t, template, []Proxy{proxy}, User{Password: "test-user"}, nil)
		if strings.Contains(profile, proxy.Name) || !strings.Contains(profile, "Nodes = select, REJECT, hidden=1") {
			t.Errorf("unsupported node not filtered from declarations and groups: %+v", proxy)
		}
	}
}
