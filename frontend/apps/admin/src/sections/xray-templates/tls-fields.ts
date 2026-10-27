import { z } from "zod";
import type { JsonArrayColumn } from "./json-form-controls";

export const TLS_CERTIFICATE_COLUMNS: JsonArrayColumn[] = [
  {
    key: "usage",
    label: "用途 (usage)",
    type: "select",
    options: ["encipherment", "verify", "issue"],
  },
  {
    key: "certificateFile",
    label: "证书路径 (certificateFile)",
    placeholder: "/etc/ssl/fullchain.pem",
  },
  {
    key: "keyFile",
    label: "私钥路径 (keyFile)",
    placeholder: "/etc/ssl/privkey.pem",
  },
  {
    key: "certificate",
    label: "证书 PEM (certificate)",
    type: "csv",
    multiline: true,
    splitMode: "line",
    span: "full",
    placeholder: "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  },
  {
    key: "key",
    label: "私钥 PEM (key)",
    type: "csv",
    multiline: true,
    splitMode: "line",
    span: "full",
    placeholder: "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
  },
  {
    key: "oneTimeLoading",
    label: "仅加载一次 (oneTimeLoading)",
    type: "boolean",
  },
  { key: "buildChain", label: "构建证书链 (buildChain)", type: "boolean" },
  { key: "ocspStapling", label: "OCSP 更新间隔（秒）", type: "integer" },
];

const certificateSchema = z
  .object({
    certificateFile: z.string().optional(),
    certificate: z.array(z.string()).optional(),
    keyFile: z.string().optional(),
    key: z.array(z.string()).optional(),
    usage: z.enum(["encipherment", "verify", "issue"]).optional(),
    oneTimeLoading: z.boolean().optional(),
    buildChain: z.boolean().optional(),
    ocspStapling: z.number().int().nonnegative().optional(),
  })
  .passthrough()
  .refine((value) => {
    const hasCertificate = Boolean(
      value.certificateFile?.trim() ||
        value.certificate?.some((line) => line.trim())
    );
    const hasKey = Boolean(
      value.keyFile?.trim() || value.key?.some((line) => line.trim())
    );
    return hasCertificate && (value.usage === "verify" || hasKey);
  });

export function isTLSCertificatesJson(value?: string) {
  if (!value?.trim()) return true;
  try {
    return z.array(certificateSchema).safeParse(JSON.parse(value)).success;
  } catch {
    return false;
  }
}

export function isPinnedPeerCertSha256(value?: string) {
  if (!value?.trim() || /{{[\s\S]*?}}/.test(value)) return true;
  return value
    .split(",")
    .every((item) => /^[a-f\d]{64}$/i.test(item.trim().replaceAll(":", "")));
}
