export const SERVER_STATUS_TTL_MS = 30_000;

export type HealthState = "ok" | "warning" | "error" | "unknown";
export type FleetFilter = "all" | "online" | "attention" | "pending";
export type OverallHealth = "healthy" | "attention" | "offline";

export function statusIsOffline(
  status: Partial<API.ServerStatus>,
  now = Date.now()
) {
  if (status.status === "offline") return true;
  return Boolean(
    status.last_seen && now - status.last_seen > SERVER_STATUS_TTL_MS
  );
}

export function configHealth(status: Partial<API.ServerStatus>): HealthState {
  const failed =
    status.config_apply_status === "failed" ||
    status.config_sync_status === "failed" ||
    Boolean(status.last_apply_error || status.last_config_error);
  if (failed) return "error";
  if (
    status.config_apply_status === "pending" ||
    status.config_sync_status === "pending" ||
    status.pending_config_hash
  ) {
    return "warning";
  }
  return "ok";
}

export function nodeHealth(
  status: Partial<API.ServerStatus>,
  now = Date.now()
): HealthState {
  return statusIsOffline(status, now) ? "error" : "ok";
}

export function xrayHealth(
  status: Partial<API.ServerStatus>,
  now = Date.now()
): HealthState {
  if (statusIsOffline(status, now)) return "unknown";
  return status.xray_running ? "ok" : "error";
}

export function resourceHealth(status: Partial<API.ServerStatus>): HealthState {
  const peak = Math.max(
    Number(status.cpu) || 0,
    Number(status.mem) || 0,
    Number(status.disk) || 0
  );
  if (peak >= 95) return "error";
  if (peak >= 90) return "warning";
  return "ok";
}

export function serverNeedsAttention(
  status: Partial<API.ServerStatus>,
  now = Date.now()
) {
  return (
    nodeHealth(status, now) === "error" ||
    xrayHealth(status, now) === "error" ||
    ["warning", "error"].includes(configHealth(status)) ||
    ["warning", "error"].includes(resourceHealth(status))
  );
}

export function getOverallHealth(
  status: Partial<API.ServerStatus>,
  now = Date.now()
): OverallHealth {
  if (statusIsOffline(status, now)) return "offline";
  return serverNeedsAttention(status, now) ? "attention" : "healthy";
}

export function buildFleetSummary(
  statuses: Partial<API.ServerStatus>[],
  matchingTotal = statuses.length,
  now = Date.now()
) {
  return statuses.reduce(
    (summary, status) => {
      if (!statusIsOffline(status, now)) summary.online += 1;
      if (serverNeedsAttention(status, now)) summary.attention += 1;
      if (configHealth(status) === "warning") summary.pending += 1;
      return summary;
    },
    { total: matchingTotal, online: 0, attention: 0, pending: 0 }
  );
}

export function buildAttentionBreakdown(
  statuses: Partial<API.ServerStatus>[],
  now = Date.now()
) {
  return statuses.reduce(
    (result, status) => {
      if (statusIsOffline(status, now)) result.offline += 1;
      else if (xrayHealth(status, now) === "error") result.xray += 1;
      if (["warning", "error"].includes(configHealth(status))) {
        result.config += 1;
      }
      if (["warning", "error"].includes(resourceHealth(status))) {
        result.resources += 1;
      }
      return result;
    },
    { offline: 0, xray: 0, config: 0, resources: 0 }
  );
}
