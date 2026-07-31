export type ProxyAccessFilters = {
  start: string;
  end: string;
  serverId: string;
  userId: string;
  subscribeId: string;
  sourceIP: string;
  destination: string;
  status: string;
};

export type ProxyAccessFilterError =
  | "invalidTime"
  | "invalidTimeOrder"
  | "timeRangeTooLarge"
  | "invalidServerId"
  | "invalidUserId"
  | "invalidSubscribeId";

const maxRangeMilliseconds = 31 * 24 * 60 * 60 * 1000;

function timestamp(value: string) {
  const parsed = new Date(value).getTime();
  return Number.isFinite(parsed) ? parsed : undefined;
}

function positiveInteger(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return undefined;
  const parsed = Number(trimmed);
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null;
}

export function proxyAccessFilterError(
  filters: ProxyAccessFilters
): ProxyAccessFilterError | undefined {
  const start = timestamp(filters.start);
  const end = timestamp(filters.end);
  if (start === undefined || end === undefined) return "invalidTime";
  if (start >= end) return "invalidTimeOrder";
  if (end - start > maxRangeMilliseconds) return "timeRangeTooLarge";
  if (positiveInteger(filters.serverId) === null) return "invalidServerId";
  if (positiveInteger(filters.userId) === null) return "invalidUserId";
  if (positiveInteger(filters.subscribeId) === null)
    return "invalidSubscribeId";
  return undefined;
}

export function proxyAccessFilterParams(
  filters: ProxyAccessFilters,
  size: number,
  cursor?: string
): API.FilterProxyAccessLogParams {
  return {
    start_time: timestamp(filters.start),
    end_time: timestamp(filters.end),
    server_id: positiveInteger(filters.serverId) ?? undefined,
    user_id: positiveInteger(filters.userId) ?? undefined,
    user_subscribe_id: positiveInteger(filters.subscribeId) ?? undefined,
    source_ip: filters.sourceIP.trim() || undefined,
    destination: filters.destination.trim() || undefined,
    status: filters.status === "all" ? undefined : filters.status,
    size,
    cursor: cursor || undefined,
  };
}
