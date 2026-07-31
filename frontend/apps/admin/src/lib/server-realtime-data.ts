export const REALTIME_STATUS_DEFAULTS: Partial<API.ServerStatus> = {
  connections: 0,
  system_connections: 0,
  system_inbound_connections: 0,
  system_outbound_connections: 0,
  xray_connections: 0,
  xray_inbound_connections: 0,
  xray_outbound_connections: 0,
  online_users: 0,
  net_rx_bps: 0,
  net_tx_bps: 0,
  net_rx_bytes: 0,
  net_tx_bytes: 0,
  xray_rx_bps: 0,
  xray_tx_bps: 0,
  xray_rx_bytes: 0,
  xray_tx_bytes: 0,
  load1: 0,
  load5: 0,
  load15: 0,
  uptime: 0,
  xray_stats_error: "",
  pending_config_hash: "",
  last_config_error: "",
  last_apply_error: "",
};

export type RealtimeServerStatusUpdate = {
  serverId: number;
  status: Partial<API.ServerStatus>;
};

export function normalizeRealtimeServerIDs(
  serverIDs: readonly number[],
  limit = 256
) {
  const result: number[] = [];
  const seen = new Set<number>();
  for (const rawID of serverIDs) {
    const id = Number(rawID);
    if (!(Number.isSafeInteger(id) && id > 0) || seen.has(id)) continue;
    seen.add(id);
    result.push(id);
    if (result.length === limit) break;
  }
  return result;
}

export function serverStatusVersion(status?: Partial<API.ServerStatus>) {
  if (!status) return 0;
  return Math.max(status.last_seen || 0, status.updated_at || 0);
}

export function normalizeRealtimeStatus(
  status: Partial<API.ServerStatus>
): Partial<API.ServerStatus> {
  return { ...REALTIME_STATUS_DEFAULTS, ...status };
}

export function mergeRealtimeStatus(
  previous: API.ServerStatus,
  incoming: Partial<API.ServerStatus>
): API.ServerStatus;
export function mergeRealtimeStatus(
  previous: Partial<API.ServerStatus> | undefined,
  incoming: Partial<API.ServerStatus>
): Partial<API.ServerStatus>;
export function mergeRealtimeStatus(
  previous: Partial<API.ServerStatus> | undefined,
  incoming: Partial<API.ServerStatus>
): Partial<API.ServerStatus> {
  const next = normalizeRealtimeStatus(incoming);
  const previousVersion = serverStatusVersion(previous);
  const nextVersion = serverStatusVersion(next);
  if (previousVersion > 0 && nextVersion > 0 && nextVersion < previousVersion) {
    return previous || next;
  }
  return { ...(previous || {}), ...next };
}

export function parseRealtimeServerMessage(
  raw: unknown
): RealtimeServerStatusUpdate[] {
  let message: unknown = raw;
  if (typeof raw === "string") {
    try {
      message = JSON.parse(raw);
    } catch {
      return [];
    }
  }
  if (!isRecord(message)) return [];

  const candidates =
    message.type === "snapshot" && Array.isArray(message.data)
      ? message.data
      : message.type === "node_status"
        ? [message]
        : [];
  const updates: RealtimeServerStatusUpdate[] = [];
  for (const candidate of candidates) {
    if (!isRecord(candidate)) continue;
    const { server_id: serverIDValue, type: eventType, ...status } = candidate;
    if (eventType !== undefined && typeof eventType !== "string") continue;
    const serverId = Number(serverIDValue);
    if (!(Number.isSafeInteger(serverId) && serverId > 0)) continue;
    updates.push({
      serverId,
      status: normalizeRealtimeStatus(status as Partial<API.ServerStatus>),
    });
  }
  return updates;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}
