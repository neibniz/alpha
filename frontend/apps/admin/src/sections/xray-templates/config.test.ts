import { describe, expect, it } from "vitest";
import {
  buildInboundConfig,
  buildStreamSettings,
  configToFormValues,
  normalizeInboundProtocol,
  normalizeTransportNetwork,
} from "./config";

describe("Xray template config compatibility", () => {
  it("normalizes legacy protocol and transport aliases", () => {
    expect(normalizeInboundProtocol("dokodemo-door")).toBe("tunnel");
    expect(normalizeInboundProtocol("vless")).toBe("vless");
    expect(normalizeTransportNetwork("tcp")).toBe("raw");
    expect(normalizeTransportNetwork("ws")).toBe("websocket");
    expect(normalizeTransportNetwork("kcp")).toBe("mkcp");
  });

  it("emits current WebSocket and TLS fields while preserving advanced values", () => {
    const stream = buildStreamSettings({
      type: "outbound",
      network: "ws",
      security: "tls",
      host: "edge.example.com",
      path: "/ray",
      sni: "edge.example.com",
      allow_insecure: true,
      stream_extra_json: JSON.stringify({
        wsSettings: {
          acceptProxyProtocol: true,
          heartbeatPeriod: 15,
          headers: { "X-Trace": "alpha" },
        },
        tlsSettings: { alpn: ["h2", "http/1.1"] },
      }),
      raw_settings_json: "{}",
      sockopt_json: "{}",
      finalmask_json: "{}",
    });

    expect(stream).toMatchObject({
      network: "websocket",
      security: "tls",
      wsSettings: {
        acceptProxyProtocol: true,
        heartbeatPeriod: 15,
        headers: { "X-Trace": "alpha" },
        host: "edge.example.com",
        path: "/ray",
      },
      tlsSettings: {
        allowInsecure: true,
        alpn: ["h2", "http/1.1"],
        serverName: "edge.example.com",
      },
    });
  });

  it("normalizes the tunnel protocol without changing pinned-core fields", () => {
    const values = configToFormValues("inbound", {
      protocol: "dokodemo-door",
      settings: {
        address: "8.8.8.8",
        port: 53,
        network: "udp",
        followRedirect: true,
      },
    });
    const config = buildInboundConfig({
      ...values,
      tag: "dns-tunnel",
      listen: "0.0.0.0",
      port: 1053,
      settings_json: "{}",
      advanced_json: "{}",
      sniffing: false,
    });

    expect(config).toMatchObject({
      protocol: "tunnel",
      settings: {
        address: "8.8.8.8",
        followRedirect: true,
        network: "udp",
        port: 53,
      },
    });
  });

  it("uses pinned-core client arrays and exact TUN field casing", () => {
    const trojan = buildInboundConfig({
      protocol: "trojan",
      tag: "trojan-in",
      port: 443,
      trojan_clients_json: '[{"password":"secret"}]',
      fallbacks_json: "[]",
      settings_json: "{}",
      network: "raw",
      security: "none",
      raw_settings_json: "{}",
      stream_extra_json: "{}",
      sockopt_json: "{}",
      finalmask_json: "{}",
      sniffing: false,
      advanced_json: "{}",
    });
    const hysteria = buildInboundConfig({
      protocol: "hysteria",
      tag: "hy2-in",
      port: 8443,
      hysteria_version: 2,
      hysteria_users_json: '[{"auth":"secret"}]',
      settings_json: "{}",
      network: "hysteria",
      security: "none",
      hysteria_settings_json: "{}",
      hysteria_masquerade_json: "{}",
      stream_extra_json: "{}",
      sockopt_json: "{}",
      finalmask_json: "{}",
      sniffing: false,
      advanced_json: "{}",
    });
    const tun = buildInboundConfig({
      protocol: "tun",
      tag: "tun-in",
      tun_name: "xray0",
      tun_mtu: 1500,
      user_level: 0,
      settings_json: "{}",
      sniffing: false,
      advanced_json: "{}",
    });

    expect(trojan.settings).toEqual({ clients: [{ password: "secret" }] });
    expect(hysteria.settings).toEqual({
      users: [{ auth: "secret" }],
      version: 2,
    });
    expect(tun.settings).toEqual({
      name: "xray0",
      MTU: 1500,
      userLevel: 0,
    });
    expect(hysteria.streamSettings).toMatchObject({
      network: "hysteria",
      security: "tls",
    });
  });

  it("loads current users aliases without duplicating them on rebuild", () => {
    const values = configToFormValues("inbound", {
      protocol: "vless",
      port: 443,
      settings: {
        users: [{ id: "{{ .Vars.uuid }}", email: "template" }],
        decryption: "none",
      },
      streamSettings: { network: "raw", security: "reality" },
    });

    expect(JSON.parse(String(values.vless_clients_json))).toEqual([
      { id: "{{ .Vars.uuid }}", email: "template" },
    ]);
    expect(JSON.parse(String(values.hysteria_users_json))).toEqual([]);
    expect(JSON.parse(String(values.trojan_clients_json))).toEqual([]);
    expect(JSON.parse(String(values.settings_json))).toEqual({});

    const rebuilt = buildInboundConfig({
      ...values,
      tag: "vless-in",
      advanced_json: "{}",
      sockopt_json: "{}",
      finalmask_json: "{}",
      stream_extra_json: "{}",
      raw_settings_json: "{}",
      sniffing: false,
    });

    expect(rebuilt.settings).toEqual({
      clients: [{ id: "{{ .Vars.uuid }}", email: "template" }],
      decryption: "none",
    });
  });

  it("migrates legacy Hysteria clients to the official users field", () => {
    const values = configToFormValues("inbound", {
      protocol: "hysteria",
      port: 443,
      settings: {
        version: 2,
        clients: [{ auth: "legacy", email: "template", level: 0 }],
      },
      streamSettings: {
        network: "hysteria",
        security: "tls",
        hysteriaSettings: { version: 2 },
      },
    });
    const rebuilt = buildInboundConfig({
      ...values,
      tag: "hy2-in",
      advanced_json: "{}",
      settings_json: "{}",
      stream_extra_json: "{}",
      hysteria_settings_json: "{}",
      hysteria_masquerade_json: "{}",
      sockopt_json: "{}",
      finalmask_json: "{}",
      sniffing: false,
    });

    expect(rebuilt.settings).toEqual({
      version: 2,
      users: [{ auth: "legacy", email: "template", level: 0 }],
    });
  });

  it("round-trips REALITY ML-DSA-65 keys", () => {
    const values = configToFormValues("inbound", {
      protocol: "vless",
      streamSettings: {
        network: "raw",
        security: "reality",
        realitySettings: {
          mldsa65Seed: "server-seed",
          target: "example.com:443",
          serverNames: ["example.com"],
          privateKey: "private-key",
          shortIds: ["a1b2c3d4"],
        },
      },
    });

    expect(values.reality_mldsa65_seed).toBe("server-seed");
    expect(
      buildStreamSettings({ ...values, type: "inbound" })?.realitySettings
    ).toMatchObject({ mldsa65Seed: "server-seed" });
  });

  it("round-trips supported outbound gRPC settings", () => {
    const values = configToFormValues("outbound", {
      protocol: "vless",
      settings: { address: "edge.example.com", port: 443, id: "uuid" },
      streamSettings: {
        network: "grpc",
        grpcSettings: {
          serviceName: "old",
          user_agent: "alpha-client",
          permit_without_stream: true,
          initial_windows_size: 65_536,
        },
      },
    });
    expect(values.grpc_user_agent).toBe("alpha-client");
    expect(values.grpc_permit_without_stream).toBe(true);
    expect(values.grpc_initial_windows_size).toBe(65_536);
    const stream = buildStreamSettings({
      ...values,
      type: "outbound",
      service_name: "new",
    });

    expect(stream?.grpcSettings).toEqual({
      serviceName: "new",
      user_agent: "alpha-client",
      permit_without_stream: true,
      initial_windows_size: 65_536,
    });
  });

  it("does not emit outbound-only gRPC fields for an inbound", () => {
    const stream = buildStreamSettings({
      type: "inbound",
      network: "grpc",
      security: "none",
      service_name: "edge",
      grpc_authority: "edge.example.com",
      grpc_multi_mode: true,
      grpc_user_agent: "alpha-client",
      grpc_idle_timeout: 60,
      grpc_health_check_timeout: 20,
      grpc_permit_without_stream: true,
      grpc_initial_windows_size: 65_536,
      stream_extra_json: "{}",
      sockopt_json: "{}",
      finalmask_json: "{}",
    });

    expect(stream?.grpcSettings).toEqual({
      authority: "edge.example.com",
      serviceName: "edge",
    });
  });
});
