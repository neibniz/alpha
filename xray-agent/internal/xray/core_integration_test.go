package xray

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/neibniz/alpha/xray-agent/internal/config"
	"github.com/neibniz/alpha/xray-agent/internal/panel"
)

type coreObject = map[string]interface{}

// The real core is required: configuration parsing alone cannot detect failed TLS handshakes.
func TestCoreTransportRoundTrip(t *testing.T) {
	serverBin := os.Getenv("ALPHA_TEST_XRAY_BIN")
	if serverBin == "" {
		t.Skip("ALPHA_TEST_XRAY_BIN is not set")
	}
	clientBin := os.Getenv("ALPHA_TEST_XRAY_CLIENT_BIN")
	if clientBin == "" {
		clientBin = serverBin
	}
	for _, mode := range []string{"grpc", "grpc-multi", "hysteria"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "alpha-transport-ok") }))
			defer echo.Close()
			_, echoPort, _ := net.SplitHostPort(echo.Listener.Addr().String())
			serverPort, apiPort := coreTestPort(t, mode == "hysteria"), coreTestPort(t, false)
			serverStream, clientStream := coreTestStreams(t, mode)
			protocolName := "vless"
			settings := coreObject{"decryption": "none", "clients": []interface{}{coreObject{"id": "template"}}}
			if mode == "hysteria" {
				protocolName = "hysteria"
				settings = coreObject{"version": 2, "users": []interface{}{coreObject{"auth": "template-password"}}}
			}
			base := coreObject{
				"inbounds": []interface{}{coreObject{"tag": "proxy", "listen": "127.0.0.1", "port": serverPort, "protocol": protocolName, "settings": settings, "streamSettings": serverStream}},
				"outbounds": []interface{}{coreObject{"tag": "direct", "protocol": "freedom", "settings": coreObject{
					// Only this test endpoint is exempted from the core's private-address protection.
					"finalRules": []interface{}{coreObject{"action": "allow", "ip": []string{"127.0.0.1/32"}, "port": echoPort}},
				}}},
			}
			users := []panel.XrayUser{{UserSubscribeID: 41, Email: "u_41", UUID: "b1710a33-a698-4b38-81dc-1d6b493442c0", Enabled: true, Inbounds: []string{"proxy"}}}
			// Exercise the server response envelopes, authentication headers, and user injection together.
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Alpha-Server-ID") != "7" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				var data interface{}
				switch r.URL.Path {
				case "/v2/server/7/xray-config":
					data = base
				case "/v2/server/7/xray-users":
					data = panel.XrayUsersResponse{Version: "v1", Full: true, Users: users}
				default:
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(coreObject{"code": 200, "data": data})
			}))
			defer upstream.Close()
			client := panel.NewClient(config.PanelConfig{BaseURL: upstream.URL, ServerID: 7, Token: "test-token"})
			_, raw, err := client.FetchXrayConfig(ctx)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.FetchXrayUsers(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			rendered, err := InjectUsers(raw, response.Users)
			if err != nil {
				t.Fatal(err)
			}
			runtimeConfig, err := BuildRuntimeConfig(rendered, apiPort, true)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateRuntimeConfig(runtimeConfig); err != nil {
				t.Fatal(err)
			}
			coreTestStart(t, ctx, serverBin, runtimeConfig, apiPort)
			apiAddress := net.JoinHostPort("127.0.0.1", strconv.Itoa(apiPort))
			policy := NewIPLimitPolicy(apiAddress)
			if err := policy.SetBaseConfig(runtimeConfig); err != nil {
				t.Fatal(err)
			}
			if err := policy.Reapply(ctx, true); err != nil {
				t.Fatalf("apply routing through core API: %v", err)
			}
			desired, err := BuildInboundUsers(raw, response.Users)
			if err != nil {
				t.Fatal(err)
			}
			reloader := NewUserHotReloader(apiAddress)
			reloader.SetSnapshot(desired)
			check := func(uuid string) {
				t.Helper()
				port := coreTestPort(t, false)
				outSettings := coreObject{"vnext": []interface{}{coreObject{"address": "127.0.0.1", "port": serverPort, "users": []interface{}{coreObject{"id": uuid, "encryption": "none"}}}}}
				if mode == "hysteria" {
					outSettings = coreObject{"address": "127.0.0.1", "port": serverPort, "version": 2}
					clientStream["hysteriaSettings"].(coreObject)["auth"] = uuid
				}
				payload, _ := json.Marshal(coreObject{
					"inbounds":  []interface{}{coreObject{"listen": "127.0.0.1", "port": port, "protocol": "socks", "settings": coreObject{"auth": "noauth"}}},
					"outbounds": []interface{}{coreObject{"protocol": protocolName, "settings": outSettings, "streamSettings": clientStream}},
				})
				stop := coreTestStart(t, ctx, clientBin, payload, port)
				defer stop()
				if err := coreTestRequest(port, echo.Listener.Addr().String()); err != nil {
					t.Fatalf("actual proxy request failed: %v", err)
				}
			}
			check(users[0].UUID)
			response.Users[0].UUID = "419ad4aa-bd8b-4ccf-91c6-174121d686b3"
			desired, err = BuildInboundUsers(raw, response.Users)
			if err != nil {
				t.Fatal(err)
			}
			if err := reloader.Sync(ctx, desired); err != nil {
				t.Fatalf("hot reload users through core API: %v", err)
			}
			check(response.Users[0].UUID)
		})
	}
}

func coreTestStreams(t *testing.T, mode string) (coreObject, coreObject) {
	t.Helper()
	if mode == "hysteria" {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		certificate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "edge.example.com"}, DNSNames: []string{"edge.example.com"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
		der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		private, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		pin := sha256.Sum256(der)
		return coreObject{"network": "hysteria", "security": "tls", "hysteriaSettings": coreObject{"version": 2}, "tlsSettings": coreObject{"alpn": []string{"h3"}, "certificates": []interface{}{coreObject{
			"certificate": strings.Split(strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))), "\n"),
			"key":         strings.Split(strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}))), "\n"),
		}}}}, coreObject{"network": "hysteria", "security": "tls", "hysteriaSettings": coreObject{"version": 2}, "tlsSettings": coreObject{"serverName": "edge.example.com", "alpn": []string{"h3"}, "pinnedPeerCertSha256": hex.EncodeToString(pin[:])}}
	}
	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "target") }))
	target.EnableHTTP2 = true
	target.Config.ErrorLog = log.New(io.Discard, "", 0)
	target.TLS = &tls.Config{MinVersion: tls.VersionTLS13, CurvePreferences: []tls.CurveID{tls.X25519}}
	target.StartTLS()
	t.Cleanup(target.Close)
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return coreObject{"network": "grpc", "security": "reality", "grpcSettings": coreObject{"serviceName": "alpha"}, "realitySettings": coreObject{
		"target": target.Listener.Addr().String(), "privateKey": base64.RawURLEncoding.EncodeToString(key.Bytes()), "serverNames": []string{"edge.example.com"}, "shortIds": []string{"1234567890abcdef"},
	}}, coreObject{"network": "grpc", "security": "reality", "grpcSettings": coreObject{"serviceName": "alpha", "multiMode": mode == "grpc-multi"}, "realitySettings": coreObject{
		"serverName": "edge.example.com", "fingerprint": "chrome", "publicKey": base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()), "shortId": "1234567890abcdef",
	}}
}

func coreTestPort(t *testing.T, udp bool) int {
	t.Helper()
	if udp {
		listener, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		return listener.LocalAddr().(*net.UDPAddr).Port
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func coreTestStart(t *testing.T, ctx context.Context, binaryPath string, payload []byte, port int) func() {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := os.OpenFile(filepath.Join(dir, "core.log"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, binaryPath, "run", "-config", path)
	cmd.Stdout, cmd.Stderr = output, output
	if err := cmd.Start(); err != nil {
		output.Close()
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if t.Failed() {
			_, _ = output.Seek(0, io.SeekStart)
			data, _ := io.ReadAll(io.LimitReader(output, 16<<10))
			t.Logf("core output: %s", data)
		}
		_ = output.Close()
	}
	t.Cleanup(stop)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return stop
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("core did not open its local listener on port %d", port)
	return stop
}

func coreTestRequest(port int, target string) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
		return err
	}
	var auth [2]byte
	if _, err := io.ReadFull(conn, auth[:]); err != nil {
		return err
	}
	if auth != [2]byte{5, 0} {
		return fmt.Errorf("unexpected SOCKS authentication reply")
	}
	host, value, err := net.SplitHostPort(target)
	if err != nil {
		return err
	}
	targetPort, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	request := append([]byte{5, 1, 0, 1}, net.ParseIP(host).To4()...)
	request = binary.BigEndian.AppendUint16(request, uint16(targetPort))
	if _, err := conn.Write(request); err != nil {
		return err
	}
	var reply [10]byte
	if _, err := io.ReadFull(conn, reply[:]); err != nil {
		return err
	}
	if reply[1] != 0 {
		return fmt.Errorf("SOCKS connection rejected: %d", reply[1])
	}
	if _, err := fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: test\r\nConnection: close\r\n\r\n"); err != nil {
		return err
	}
	body, err := io.ReadAll(conn)
	if err != nil {
		return err
	}
	if !bytes.Contains(body, []byte("alpha-transport-ok")) {
		return fmt.Errorf("proxy did not return the test response")
	}
	return nil
}
