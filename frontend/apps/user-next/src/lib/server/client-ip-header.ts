export const EDGE_CLIENT_IP_HEADER = "X-Alpha-Client-IP";

type ClientIPSource = "auto" | "cloudflare" | "forwarded" | "vercel";
type HeaderReader = Pick<Headers, "get">;

function normalizeSource(
  value: string | undefined
): ClientIPSource | undefined {
  const source = value?.trim().toLowerCase() || "auto";
  if (source === "none") return undefined;
  if (
    source === "auto" ||
    source === "cloudflare" ||
    source === "forwarded" ||
    source === "vercel"
  ) {
    return source;
  }
  throw new Error(
    "ALPHA_CLIENT_IP_SOURCE must be auto, cloudflare, forwarded, vercel, or none"
  );
}

function normalizeIPAddress(raw: string | null) {
  let value = raw?.trim() || "";
  if (!value || value.length > 128 || value.includes(",")) return undefined;
  if (value.startsWith("[")) {
    const closing = value.indexOf("]");
    if (
      closing < 0 ||
      (value.length > closing + 1 && value[closing + 1] !== ":")
    ) {
      return undefined;
    }
    value = value.slice(1, closing);
  } else if ((value.match(/:/g) || []).length === 1 && value.includes(".")) {
    value = value.split(":", 1)[0] || "";
  }
  if (/^\d{1,3}(?:\.\d{1,3}){3}$/.test(value)) {
    const octets = value.split(".").map(Number);
    if (
      octets.every((part) => Number.isInteger(part) && part >= 0 && part <= 255)
    ) {
      return octets.join(".");
    }
    return undefined;
  }
  if (!value.includes(":") || !/^[0-9A-Fa-f:.]+$/.test(value)) {
    return undefined;
  }
  try {
    return new URL(`http://[${value}]/`).hostname
      .replace(/^\[|\]$/g, "")
      .toLowerCase();
  } catch {
    return undefined;
  }
}

function firstForwardedAddress(value: string | null) {
  if (!value || value.length > 4096) return undefined;
  const parts = value.split(",");
  if (parts.length > 32) return undefined;
  return normalizeIPAddress(parts[0] || null);
}

export function resolveEdgeClientIP(
  headers: HeaderReader,
  source: ClientIPSource
) {
  const cloudflare = () => normalizeIPAddress(headers.get("cf-connecting-ip"));
  const forwarded = () =>
    normalizeIPAddress(headers.get("x-real-ip")) ||
    firstForwardedAddress(headers.get("x-forwarded-for"));
  switch (source) {
    case "auto":
      return cloudflare() || forwarded();
    case "cloudflare":
      return cloudflare();
    case "forwarded":
      return forwarded();
    case "vercel":
      return firstForwardedAddress(headers.get("x-forwarded-for"));
  }
}

export function attachEdgeClientIPHeader(
  destination: Headers,
  sourceHeaders: HeaderReader
) {
  destination.delete(EDGE_CLIENT_IP_HEADER);
  const source = normalizeSource(process.env.ALPHA_CLIENT_IP_SOURCE);
  if (!source) return;
  const clientIP = resolveEdgeClientIP(sourceHeaders, source);
  if (clientIP) destination.set(EDGE_CLIENT_IP_HEADER, clientIP);
}
