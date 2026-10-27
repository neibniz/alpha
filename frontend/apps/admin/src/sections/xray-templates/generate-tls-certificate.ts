import { z } from "zod";

export type GeneratedTLSCertificate = {
  certificate: string[];
  key: string[];
  pinnedPeerCertSha256: string;
};

export function certificateNames(value: string) {
  const names = [
    ...new Set(
      value
        .split(/[,\n]/)
        .map((item) => item.trim())
        .filter(Boolean)
    ),
  ];
  if (!names.length || names.length > 32)
    throw new Error("请输入 1 至 32 个域名或 IP，多个值用逗号或换行分隔。");
  return names.map((name) => {
    if (z.ipv4().safeParse(name).success || z.ipv6().safeParse(name).success) {
      return { type: "ip" as const, value: name };
    }
    const domain = name.startsWith("*.") ? name.slice(2) : name;
    if (
      domain.length > 253 ||
      !domain
        .split(".")
        .every((label) => /^[a-z\d](?:[a-z\d-]{0,61}[a-z\d])?$/i.test(label))
    ) {
      throw new Error("请输入有效的域名或 IP，不要填写 URL、端口或模板变量。");
    }
    return { type: "dns" as const, value: name.toLowerCase() };
  });
}

export async function generateTLSCertificate(
  namesText: string,
  days = 365
): Promise<GeneratedTLSCertificate> {
  const names = certificateNames(namesText);
  if (!Number.isSafeInteger(days) || days < 1 || days > 3650)
    throw new Error("有效期必须为 1 至 3650 天的整数。");
  if (!globalThis.crypto?.subtle)
    throw new Error(
      "当前环境不支持 Web Crypto，请通过 HTTPS 或 localhost 访问管理端。"
    );
  // Load certificate encoding only when requested; signing and randomness stay in Web Crypto.
  await import("reflect-metadata");
  const x509 = await import("@peculiar/x509");
  const provider = globalThis.crypto;
  const algorithm = { name: "ECDSA", namedCurve: "P-256", hash: "SHA-256" };
  const keys = await provider.subtle.generateKey(algorithm, true, [
    "sign",
    "verify",
  ]);
  const serial = provider.getRandomValues(new Uint8Array(16));
  serial[0] = 1;
  const now = Date.now();
  const cert = await x509.X509CertificateGenerator.createSelfSigned(
    {
      serialNumber: Array.from(serial, (byte) =>
        byte.toString(16).padStart(2, "0")
      ).join(""),
      name: `CN=${names[0]?.value}`,
      notBefore: new Date(now - 5 * 60_000),
      notAfter: new Date(now + days * 86_400_000),
      signingAlgorithm: algorithm,
      keys,
      extensions: [
        new x509.BasicConstraintsExtension(false, undefined, true),
        new x509.KeyUsagesExtension(x509.KeyUsageFlags.digitalSignature, true),
        new x509.ExtendedKeyUsageExtension(["1.3.6.1.5.5.7.3.1"]),
        new x509.SubjectAlternativeNameExtension(names),
      ],
    },
    provider
  );
  const [keyBytes, hash] = await Promise.all([
    provider.subtle.exportKey("pkcs8", keys.privateKey),
    provider.subtle.digest("SHA-256", cert.rawData),
  ]);
  const base64 = btoa(
    Array.from(new Uint8Array(keyBytes), (byte) =>
      String.fromCharCode(byte)
    ).join("")
  );
  const key = [
    "-----BEGIN PRIVATE KEY-----",
    ...(base64.match(/.{1,64}/g) || []),
    "-----END PRIVATE KEY-----",
  ];
  return {
    certificate: cert.toString("pem").trim().split(/\r?\n/),
    key,
    pinnedPeerCertSha256: Array.from(new Uint8Array(hash), (byte) =>
      byte.toString(16).padStart(2, "0")
    ).join(":"),
  };
}
