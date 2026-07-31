const TEMPLATE_EXPRESSION = /{{[\s\S]*?}}/;

export function parseOptionalInteger(value: unknown) {
  if (value === undefined || value === null || value === "") return undefined;
  if (typeof value === "string") {
    const normalized = value.trim();
    if (!normalized) return undefined;
    if (!/^-?\d+$/.test(normalized)) return value;
    return Number(normalized);
  }
  return value;
}

export function isValidPort(value: unknown) {
  const parsed = parseOptionalInteger(value);
  return (
    typeof parsed === "number" &&
    Number.isInteger(parsed) &&
    parsed >= 1 &&
    parsed <= 65_535
  );
}

export function isFiveFieldCron(value?: string) {
  const normalized = value?.trim();
  if (!normalized || TEMPLATE_EXPRESSION.test(normalized)) return true;
  return normalized.split(/\s+/).length === 5;
}

export function isXrayVersion(value?: string) {
  const normalized = value?.trim();
  if (!normalized || TEMPLATE_EXPRESSION.test(normalized)) return true;
  return /^\d+\.\d+\.\d+$/.test(normalized);
}

export function isRealityTarget(value?: string) {
  const normalized = value?.trim();
  if (!normalized || TEMPLATE_EXPRESSION.test(normalized)) return true;

  const ipv6Match = normalized.match(/^\[([^\]]+)]:(\d+)$/);
  if (ipv6Match) return isValidPort(ipv6Match[2]);

  const hostMatch = normalized.match(/^([^\s:/]+):(\d+)$/);
  return !!hostMatch && isValidPort(hostMatch[2]);
}

export function isRealityShortId(value: unknown) {
  return (
    typeof value === "string" &&
    (value === "" ||
      (/^[0-9a-f]+$/i.test(value) &&
        value.length % 2 === 0 &&
        value.length <= 16))
  );
}

export function isRealityShortIdList(value?: string) {
  if (!value?.trim() || TEMPLATE_EXPRESSION.test(value)) return true;
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .every(isRealityShortId);
}

export function isVlessFlowAllowed(network: unknown, flow: unknown) {
  const normalizedFlow = typeof flow === "string" ? flow.trim() : "";
  if (!normalizedFlow || normalizedFlow === "none") return true;
  const normalizedNetwork =
    typeof network === "string" ? network.trim().toLowerCase() : "";
  return normalizedNetwork === "raw" || normalizedNetwork === "tcp";
}
