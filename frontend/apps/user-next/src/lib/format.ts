const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB"];

export function formatBytes(value: number, fractionDigits = 0) {
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const exponent = Math.min(
    Math.floor(Math.log(value) / Math.log(1024)),
    BYTE_UNITS.length - 1
  );
  return `${(value / 1024 ** exponent).toFixed(fractionDigits)} ${BYTE_UNITS[exponent]}`;
}

export function formatDate(timestamp: number) {
  return new Intl.DateTimeFormat("zh-CN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  })
    .format(timestamp)
    .replaceAll("/", "-");
}

export function formatCurrency(value: number, symbol = "¥") {
  return `${symbol}${value.toFixed(2)}`;
}

export function bitsPerSecondToMbps(value: number) {
  if (!Number.isFinite(value) || value <= 0) return 0;
  return value / 1024 ** 2;
}

export function formatSpeedLimitMbps(value: number) {
  if (!Number.isFinite(value) || value <= 0) return "不限";
  return `${new Intl.NumberFormat("zh-CN", {
    maximumFractionDigits: 2,
  }).format(value)} Mbps`;
}

export function percentOf(used: number, total: number) {
  if (!total) return 0;
  return Math.min(100, Math.max(0, (used / total) * 100));
}

export function normalizeTimestamp(value: number) {
  return value < 1_000_000_000_000 ? value * 1000 : value;
}
