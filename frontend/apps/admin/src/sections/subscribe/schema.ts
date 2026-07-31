export const subscribeSchema = {
  SiteName: { type: "string", description: "Site name" },
  SubscribeName: { type: "string", description: "Subscribe name" },
  OutputFormat: {
    type: "string",
    description: "Rendered subscription output format",
  },
  Params: {
    type: "object",
    description: "Subscription request query parameters",
    additionalProperties: { type: "string" },
  },
  Proxies: {
    type: "array",
    description: "Array of proxy nodes",
    items: {
      type: "object",
      properties: {
        Name: { type: "string", description: "Node name" },
        Server: { type: "string", description: "Server host" },
        Port: { type: "number", description: "Server port" },
        Type: { type: "string", description: "Proxy type" },
        Tags: {
          type: "array",
          description: "Node tags",
          items: { type: "string" },
        },
        Sort: { type: "number", description: "Node sort order" },
        // Security Options
        Security: {
          type: "string",
          description: "Security protocol",
        },
        SNI: {
          type: "string",
          description: "Server Name Indication for TLS",
        },
        AllowInsecure: {
          type: "boolean",
          description: "Deprecated Xray-core insecure TLS verification flag",
        },
        Fingerprint: {
          type: "string",
          description: "Client fingerprint for TLS connections",
        },
        PinnedPeerCertSha256: {
          type: "string",
          description: "Pinned TLS peer certificate SHA-256 for Xray-core",
        },
        VerifyPeerCertByName: {
          type: "string",
          description: "TLS certificate name override for Xray-core",
        },
        RealityServerAddr: {
          type: "string",
          description: "Reality server address",
        },
        RealityServerPort: {
          type: "number",
          description: "Reality server port",
        },
        RealityPrivateKey: {
          type: "string",
          description: "Reality private key for authentication",
        },
        RealityPublicKey: {
          type: "string",
          description: "Reality public key for authentication",
        },
        RealityShortId: {
          type: "string",
          description: "Reality short ID for authentication",
        },
        // Transport Options
        Transport: {
          type: "string",
          description: "Transport protocol (e.g., ws, http, grpc)",
        },
        Network: {
          type: "string",
          description: "Compatibility alias of Transport",
        },
        Host: {
          type: "string",
          description: "For WebSocket/HTTP/HTTPS",
        },
        Path: { type: "string", description: "For HTTP/HTTPS" },
        ServiceName: {
          type: "string",
          description: "For gRPC",
        },
        GrpcAuthority: {
          type: "string",
          description: "gRPC authority or Host override",
        },
        GrpcMultiMode: {
          type: "boolean",
          description: "Whether gRPC multi mode is enabled",
        },
        GrpcUserAgent: {
          type: "string",
          description: "gRPC outbound User-Agent",
        },
        GrpcIdleTimeout: {
          type: "number",
          description: "gRPC idle timeout in seconds",
        },
        GrpcHealthCheckTimeout: {
          type: "number",
          description: "gRPC health-check timeout in seconds",
        },
        GrpcPermitWithoutStream: {
          type: "boolean",
          description: "Allow gRPC health checks without active streams",
        },
        GrpcInitialWindowsSize: {
          type: "number",
          description: "Initial gRPC HTTP/2 stream window size",
        },
        // Shadowsocks Options
        Method: { type: "string", description: "Encryption method" },
        ServerKey: {
          type: "string",
          description: "For Shadowsocks 2022",
        },
        Username: {
          type: "string",
          description: "Protocol account username when available",
        },
        HeaderType: {
          type: "string",
          description: "Transport header type, usually from raw/kcp settings",
        },
        TLS: {
          type: "boolean",
          description: "Whether TLS-like security is enabled",
        },
        EchEnable: {
          type: "boolean",
          description: "Whether ECH is enabled in TLS settings",
        },
        EchServerName: {
          type: "string",
          description: "ECH query server name",
        },
        // Vmess/Vless/Trojan Options
        Flow: {
          type: "string",
          description: "Flow for Vmess/Vless/Trojan",
        },
        // Hysteria2 Options
        HopPorts: {
          type: "string",
          description: "Comma-separated list of hop ports",
        },
        HopInterval: {
          type: "number",
          description: "Interval for hop ports in seconds",
        },
        ObfsPassword: {
          type: "string",
          description: "Obfuscation password for Hysteria2",
        },
        // Tuic Options
        DisableSNI: {
          type: "boolean",
          description: "Disable SNI",
        },
        ReduceRtt: {
          type: "boolean",
          description: "Reduce RTT",
        },
        UDPRelayMode: {
          type: "string",
          description: 'UDP relay mode (e.g., "full", "partial")',
        },
        CongestionController: {
          type: "string",
          description: 'Congestion controller (e.g., "cubic", "bbr")',
        },
        // Hysteria2 additional options
        UpMbps: {
          type: "number",
          description: "Upload bandwidth in Mbps",
        },
        DownMbps: {
          type: "number",
          description: "Download bandwidth in Mbps",
        },
        HysteriaUDPIdleTimeout: {
          type: "number",
          description: "Xray-core Hysteria UDP idle timeout in seconds",
        },
        HysteriaMasquerade: {
          type: "string",
          description: "Xray-core Hysteria masquerade JSON",
        },
        // VLESS encryption options
        Encryption: {
          type: "string",
          description: "Encryption type for VLESS",
        },
        EncryptionMode: {
          type: "string",
          description: 'Encryption mode (e.g., "native", "xorpub", "random")',
        },
        Encryption_Mode: {
          type: "string",
          description: "Compatibility alias of EncryptionMode",
        },
        EncryptionRtt: {
          type: "string",
          description: 'Encryption RTT (e.g., "0rtt", "1rtt")',
        },
        EncryptionTicket: {
          type: "string",
          description: "Encryption ticket",
        },
        EncryptionServerPadding: {
          type: "string",
          description: "Server padding for encryption",
        },
        EncryptionClientPadding: {
          type: "string",
          description: "Client padding for encryption",
        },
        EncryptionPassword: {
          type: "string",
          description: "Encryption password",
        },
        EncryptionPrivateKey: {
          type: "string",
          description: "Private key for encryption",
        },
        // XHTTP options
        XhttpMode: {
          type: "string",
          description:
            'XHTTP mode (e.g., "auto", "packet-up", "stream-up", "stream-one")',
        },
        XhttpExtra: {
          type: "string",
          description: "XHTTP extra parameters",
        },
        // Shadowsocks obfs options (combined with Hysteria2 obfs)
        ObfsHost: {
          type: "string",
          description: "Obfuscation host",
        },
        Obfs: {
          type: "string",
          description: "Obfuscation type",
        },
        ObfsPath: {
          type: "string",
          description: "Obfuscation path",
        },
        // Shadowsocks cipher
        Cipher: {
          type: "string",
          description: "Shadowsocks cipher method",
        },
        // AnyTLS options
        PaddingScheme: {
          type: "string",
          description: "Padding scheme for AnyTLS",
        },
        // Mieru options
        Multiplex: {
          type: "string",
          description:
            'Multiplex level (e.g., "none", "low", "middle", "high")',
        },
        Ratio: {
          type: "number",
          description: "Traffic multiplier for this proxy",
        },
        CertMode: {
          type: "string",
          description: "Certificate mode (none, http, dns, or self)",
        },
        CertDNSProvider: {
          type: "string",
          description: "DNS provider used for certificate issuance",
        },
        CertDNSEnv: {
          type: "string",
          description: "DNS provider environment configuration",
        },
      },
    },
  },
  UserInfo: {
    type: "object",
    description: "User information",
    properties: {
      Password: { type: "string", description: "User password" },
      ExpiredAt: { type: "string", description: "Expiration date" },
      Download: { type: "number", description: "Downloaded bytes" },
      Upload: { type: "number", description: "Uploaded bytes" },
      Traffic: { type: "number", description: "Total traffic bytes" },
      SubscribeURL: {
        type: "string",
        description: "Subscription URL",
      },
    },
  },
};
