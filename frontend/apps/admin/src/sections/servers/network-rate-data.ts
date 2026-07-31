export const SERVER_NETWORK_RATE_BUCKET_MS = 5000;
export const SERVER_NETWORK_RATE_SPARKLINE_POINTS = 60;

export function serverStatusToNetworkRatePoint(
  status: Partial<API.ServerStatus>,
  now = Date.now()
): API.ServerNetworkRatePoint {
  const sourceTimestamp = status.updated_at || status.last_seen || now;
  return {
    timestamp:
      Math.floor(sourceTimestamp / SERVER_NETWORK_RATE_BUCKET_MS) *
      SERVER_NETWORK_RATE_BUCKET_MS,
    net_rx_bps: status.net_rx_bps || 0,
    net_tx_bps: status.net_tx_bps || 0,
    xray_rx_bps: status.xray_rx_bps || 0,
    xray_tx_bps: status.xray_tx_bps || 0,
  };
}

export function mergeServerNetworkRatePoint(
  points: API.ServerNetworkRatePoint[],
  point: API.ServerNetworkRatePoint,
  limit = SERVER_NETWORK_RATE_SPARKLINE_POINTS
) {
  const next = points.slice();
  const last = next.at(-1);
  if (last?.timestamp === point.timestamp) {
    next[next.length - 1] = point;
  } else if (!last || last.timestamp < point.timestamp) {
    next.push(point);
  } else {
    const index = next.findIndex((item) => item.timestamp >= point.timestamp);
    if (index >= 0 && next[index]?.timestamp === point.timestamp) {
      next[index] = point;
    } else if (index >= 0) {
      next.splice(index, 0, point);
    }
  }
  return next.slice(-limit);
}

export function mapServerNetworkRateSeries(
  series: API.ServerNetworkRateSeries[],
  limit = SERVER_NETWORK_RATE_SPARKLINE_POINTS
) {
  const result: Record<number, API.ServerNetworkRatePoint[]> = {};
  for (const item of series) {
    result[item.server_id] = item.points.slice(-limit);
  }
  return result;
}
