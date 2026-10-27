import { X509Certificate, createPrivateKey } from "node:crypto";
import { expect, test } from "vitest";
import {
  certificateNames,
  generateTLSCertificate,
} from "./generate-tls-certificate";

test("generates a signed server certificate, matching private key and exact DER fingerprint", async () => {
  const result = await generateTLSCertificate(
    "example.com, 127.0.0.1, ::1",
    30
  );
  const certificate = new X509Certificate(result.certificate.join("\n"));
  expect(certificate.verify(certificate.publicKey)).toBe(true);
  expect(
    certificate.checkPrivateKey(createPrivateKey(result.key.join("\n")))
  ).toBe(true);
  expect(certificate.checkHost("example.com")).toBe("example.com");
  expect(certificate.checkIP("127.0.0.1")).toBe("127.0.0.1");
  expect(certificate.checkIP("::1")).toBe("::1");
  expect(result.pinnedPeerCertSha256.toUpperCase()).toBe(
    certificate.fingerprint256
  );
  expect(certificate.ca).toBe(false);
  expect(Date.parse(certificate.validTo)).toBeGreaterThan(
    Date.now() + 29 * 86_400_000
  );
  expect(Date.parse(certificate.validFrom)).toBeLessThan(Date.now());
});

test("uses fresh keys and certificate fingerprints on each generation", async () => {
  const first = await generateTLSCertificate("localhost");
  const second = await generateTLSCertificate("localhost");
  expect(first.key).not.toEqual(second.key);
  expect(first.pinnedPeerCertSha256).not.toBe(second.pinnedPeerCertSha256);
});

test.each([
  "",
  "https://example.com",
  "example.com:443",
  "{{ .Vars.domain }}",
  "example.com,CN=attacker",
  "bad domain",
])("rejects invalid SAN input: %s", (value) => {
  expect(() => certificateNames(value)).toThrow();
});

test.each([
  0,
  -1,
  0.5,
  3651,
  Number.NaN,
])("rejects invalid certificate lifetime: %s", async (days) => {
  await expect(generateTLSCertificate("example.com", days)).rejects.toThrow();
});
