import { expect, test } from "vitest";
import { buildStreamSettings, configToFormValues } from "./config";
import { isTLSCertificatesJson, isPinnedPeerCertSha256 } from "./tls-fields";

test("round trips certificate files, PEM arrays and advanced TLS values", () => {
  const certificates = [
    {
      certificateFile: "/certs/fullchain.pem",
      keyFile: "/certs/privkey.pem",
      usage: "encipherment",
      oneTimeLoading: false,
      ocspStapling: 3600,
    },
    {
      certificate: [
        "-----BEGIN CERTIFICATE-----",
        "test-cert",
        "-----END CERTIFICATE-----",
      ],
      key: [
        "-----BEGIN PRIVATE KEY-----",
        "test-key",
        "-----END PRIVATE KEY-----",
      ],
      usage: "issue",
      buildChain: true,
      futureField: "preserved",
    },
  ];
  const tls = {
    certificates,
    alpn: ["h2"],
    minVersion: "1.2",
    rejectUnknownSni: true,
    pinnedPeerCertSha256: "aa".repeat(32),
  };
  const values = configToFormValues("inbound", {
    protocol: "vless",
    streamSettings: { network: "raw", security: "tls", tlsSettings: tls },
  });
  expect(JSON.parse(String(values.tls_certificates_json))).toEqual(
    certificates
  );
  expect(
    buildStreamSettings({ ...values, type: "inbound" })?.tlsSettings
  ).toMatchObject(tls);
});

test("allows deleting certificates without resurrecting them from extra JSON", () => {
  const values = configToFormValues("inbound", {
    streamSettings: {
      security: "tls",
      tlsSettings: {
        certificates: [{ certificateFile: "/old.pem", keyFile: "/old.key" }],
        alpn: ["h2"],
      },
    },
  });
  const stream = buildStreamSettings({
    ...values,
    type: "inbound",
    tls_certificates_json: "[]",
  });
  expect(stream?.tlsSettings).toMatchObject({ alpn: ["h2"] });
  expect(stream?.tlsSettings).not.toHaveProperty("certificates");
});

test("preserves explicitly supplied legacy TLS extras when no certificate form field exists", () => {
  const certificates = [{ certificateFile: "/cert.pem", keyFile: "/key.pem" }];
  const stream = buildStreamSettings({
    security: "tls",
    stream_extra_json: JSON.stringify({ tlsSettings: { certificates } }),
  });
  expect(stream?.tlsSettings).toMatchObject({ certificates });
});

test.each([
  ["[]", true],
  [
    JSON.stringify([
      { certificateFile: "{{ .Vars.cert }}", keyFile: "{{ .Vars.key }}" },
    ]),
    true,
  ],
  [JSON.stringify([{ certificate: ["CA PEM"], usage: "verify" }]), true],
  [JSON.stringify([{ certificateFile: "/cert.pem" }]), false],
  [JSON.stringify([{ keyFile: "/key.pem" }]), false],
  [JSON.stringify([{ certificate: "not-an-array", key: [] }]), false],
  ["{}", false],
  ["[", false],
])("validates certificate sources and key requirements: %s", (value, valid) => {
  expect(isTLSCertificatesJson(value)).toBe(valid);
});

test("validates pinned hashes in Xray hex, OpenSSL colon and comma-separated formats", () => {
  expect(isPinnedPeerCertSha256("ab".repeat(32))).toBe(true);
  expect(isPinnedPeerCertSha256(new Array(32).fill("AB").join(":"))).toBe(true);
  expect(isPinnedPeerCertSha256(`${"ab".repeat(32)}, ${"cd".repeat(32)}`)).toBe(
    true
  );
  expect(isPinnedPeerCertSha256("{{ .Vars.cert_hash }}")).toBe(true);
  expect(isPinnedPeerCertSha256("AA:BB")).toBe(false);
  expect(isPinnedPeerCertSha256("zz".repeat(32))).toBe(false);
});
