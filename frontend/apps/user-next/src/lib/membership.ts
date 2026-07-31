export function membershipDurationLabel(value: number, unit: string) {
  const normalized = Math.max(1, Math.trunc(value || 1));
  const labels: Record<string, string> = {
    day: "天",
    hour: "小时",
    minute: "分钟",
    month: "个月",
    year: "年",
  };
  return `${normalized} ${labels[unit.toLowerCase()] || unit}`;
}

export function membershipUsagePercent(current: number, limit: number) {
  if (!(Number.isFinite(current) && Number.isFinite(limit)) || limit <= 0) {
    return 0;
  }
  return Math.min(100, Math.max(0, (current / limit) * 100));
}

export function safeMembershipReturnPath(value: string | null) {
  if (!value?.startsWith("/") || value.startsWith("//")) {
    return "/dashboard";
  }
  try {
    const parsed = new URL(value, "https://alpha.invalid");
    if (parsed.origin !== "https://alpha.invalid") return "/dashboard";
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return "/dashboard";
  }
}
