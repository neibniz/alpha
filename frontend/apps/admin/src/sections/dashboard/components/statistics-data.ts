export type TrafficRankDatum = {
  name: string;
  traffic: number;
};

export type TrafficRankingData = {
  nodes: {
    today: TrafficRankDatum[];
    month: TrafficRankDatum[];
  };
  users: {
    today: TrafficRankDatum[];
    month: TrafficRankDatum[];
  };
};

type DashboardTraffic = API.DashboardRealtimeResponse["traffic"];
type DashboardNetworkSeries = API.DashboardRealtimeResponse["network_series"];
type DashboardNetworkPoint = DashboardNetworkSeries[number];
type DashboardValuePoint =
  API.DashboardRealtimeResponse["online_user_series"]["last_day"][number];
type DashboardValueSeries = DashboardValuePoint[];
type DashboardTrafficPoint = DashboardTraffic["today_series"][number];
type DashboardTrafficSeries = DashboardTrafficPoint[];
type ServerTrafficRank = DashboardTraffic["server_ranking_today"];
type UserTrafficRank = DashboardTraffic["user_ranking_today"];

const MINUTE_BUCKET_MS = 60 * 1000;
const NETWORK_RATE_BUCKET_MS = 5 * 1000;

export function aggregateNetworkRateSeries(
  points: DashboardNetworkSeries | undefined,
  bucketMs = MINUTE_BUCKET_MS
): DashboardNetworkSeries {
  if (!points?.length || bucketMs <= NETWORK_RATE_BUCKET_MS) {
    return points ? [...points] : [];
  }

  const buckets = new Map<number, DashboardNetworkPoint & { count: number }>();
  for (const point of points) {
    const bucket = Math.floor(point.timestamp / bucketMs);
    const current = buckets.get(bucket);
    if (!current) {
      buckets.set(bucket, { ...point, count: 1 });
      continue;
    }
    current.count += 1;
    current.timestamp = Math.max(current.timestamp, point.timestamp);
    current.system_rx_bps += point.system_rx_bps;
    current.system_tx_bps += point.system_tx_bps;
    current.xray_rx_bps += point.xray_rx_bps;
    current.xray_tx_bps += point.xray_tx_bps;
  }

  return Array.from(buckets.values())
    .map(({ count, ...point }) => ({
      ...point,
      system_rx_bps: point.system_rx_bps / count,
      system_tx_bps: point.system_tx_bps / count,
      xray_rx_bps: point.xray_rx_bps / count,
      xray_tx_bps: point.xray_tx_bps / count,
    }))
    .sort((a, b) => a.timestamp - b.timestamp);
}

export function buildTrafficRankingData(
  traffic: Partial<DashboardTraffic> | undefined
): TrafficRankingData {
  return {
    nodes: {
      today: serverTrafficRankRows(traffic?.server_ranking_today),
      month: serverTrafficRankRows(traffic?.server_ranking_monthly),
    },
    users: {
      today: userTrafficRankRows(traffic?.user_ranking_today),
      month: userTrafficRankRows(traffic?.user_ranking_monthly),
    },
  };
}

function serverTrafficRankRows(
  rows: ServerTrafficRank | undefined
): TrafficRankDatum[] {
  if (!rows?.length) {
    return [];
  }
  const result = new Array<TrafficRankDatum>(rows.length);
  for (let index = 0; index < rows.length; index += 1) {
    const item = rows[index]!;
    result[index] = {
      name: String(item.name),
      traffic: item.download + item.upload,
    };
  }
  return result;
}

function userTrafficRankRows(
  rows: UserTrafficRank | undefined
): TrafficRankDatum[] {
  if (!rows?.length) {
    return [];
  }
  const result = new Array<TrafficRankDatum>(rows.length);
  for (let index = 0; index < rows.length; index += 1) {
    const item = rows[index]!;
    result[index] = {
      name: String(item.sid),
      traffic: item.download + item.upload,
    };
  }
  return result;
}

export function appendRealtimeValuePoint(
  points: DashboardValueSeries | undefined,
  point: DashboardValuePoint,
  cutoff: number
): DashboardValueSeries {
  return appendMinuteBucketPoint(points, point, cutoff);
}

export function appendRealtimeTrafficPoint(
  points: DashboardTrafficSeries | undefined,
  point: DashboardTrafficPoint,
  cutoff: number
): DashboardTrafficSeries {
  return appendMinuteBucketPoint(points, point, cutoff);
}

function appendMinuteBucketPoint<T extends { timestamp: number }>(
  points: T[] | undefined,
  point: T,
  cutoff: number
): T[] {
  const keepPoint = point.timestamp >= cutoff;
  if (!points?.length) {
    return keepPoint ? [point] : [];
  }

  const pointBucket = minuteBucket(point.timestamp);
  const next: T[] = [];
  let inserted = false;
  let replaced = false;
  for (const existing of points) {
    if (existing.timestamp < cutoff) {
      continue;
    }
    if (minuteBucket(existing.timestamp) === pointBucket) {
      if (keepPoint && !replaced) {
        next.push(point);
      }
      replaced = true;
      continue;
    }
    if (
      keepPoint &&
      !inserted &&
      !replaced &&
      point.timestamp < existing.timestamp
    ) {
      next.push(point);
      inserted = true;
    }
    next.push(existing);
  }
  if (keepPoint && !inserted && !replaced) {
    next.push(point);
  }
  return next;
}

function minuteBucket(timestamp: number) {
  return Math.floor(timestamp / MINUTE_BUCKET_MS);
}

export function mergeRealtimeNetworkPoints(
  points: DashboardNetworkSeries | undefined,
  incoming: DashboardNetworkSeries | undefined,
  cutoff: number
): DashboardNetworkSeries {
  const next = retainNetworkPoints(points, cutoff);
  if (incoming?.length === 1) {
    replaceOrAppendNetworkPoint(next, incoming[0]!, cutoff);
  } else if (incoming?.length) {
    const bucketIndex = new Map<number, number>();
    for (let index = 0; index < next.length; index += 1) {
      const bucket = networkRateBucket(next[index]!);
      if (!bucketIndex.has(bucket)) {
        bucketIndex.set(bucket, index);
      }
    }
    for (const point of incoming) {
      if (point.timestamp < cutoff) {
        continue;
      }
      const bucket = networkRateBucket(point);
      const index = bucketIndex.get(bucket);
      if (index === undefined) {
        bucketIndex.set(bucket, next.length);
        next.push(point);
      } else {
        next[index] = point;
      }
    }
  }

  return next.sort((a, b) => a.timestamp - b.timestamp);
}

function retainNetworkPoints(
  points: DashboardNetworkSeries | undefined,
  cutoff: number
): DashboardNetworkPoint[] {
  if (!points?.length) {
    return [];
  }
  const next: DashboardNetworkPoint[] = [];
  for (const point of points) {
    if (point.timestamp >= cutoff) {
      next.push(point);
    }
  }
  return next;
}

function replaceOrAppendNetworkPoint(
  points: DashboardNetworkPoint[],
  point: DashboardNetworkPoint,
  cutoff: number
) {
  if (point.timestamp < cutoff) {
    return;
  }
  const bucket = networkRateBucket(point);
  for (let index = 0; index < points.length; index += 1) {
    if (networkRateBucket(points[index]!) === bucket) {
      points[index] = point;
      return;
    }
  }
  points.push(point);
}

function networkRateBucket(point: DashboardNetworkPoint) {
  return Math.floor(point.timestamp / NETWORK_RATE_BUCKET_MS);
}
