import type { TrafficPeriod, TrafficPoint, TrafficSeries } from "@/lib/models";

const LIVE_AVERAGE_SECONDS = 15;

export type CurrentTrafficRate = {
  averagingSeconds: number;
  downloadMbps: number;
  observedAt?: number;
  status: "active" | "idle" | "unavailable";
  uploadMbps: number;
};

function livePeriod(period: TrafficPeriod) {
  return period === "5m" || period === "15m";
}

function averagingWindow(intervalSeconds: number) {
  const interval = Math.max(1, intervalSeconds);
  const samples = Math.max(1, Math.round(LIVE_AVERAGE_SECONDS / interval));
  return { samples, seconds: samples * interval };
}

function withoutIncompleteTail(points: TrafficPoint[]) {
  if (points.length > 0 && !points.at(-1)?.observed) {
    return points.slice(0, -1);
  }
  return points;
}

function averageAt(points: TrafficPoint[], index: number, samples: number) {
  const start = Math.max(0, index - samples + 1);
  const count = index - start + 1;
  let downloadMbps = 0;
  let uploadMbps = 0;
  for (let cursor = start; cursor <= index; cursor += 1) {
    downloadMbps += points[cursor]?.downloadMbps || 0;
    uploadMbps += points[cursor]?.uploadMbps || 0;
  }
  return {
    downloadMbps: downloadMbps / count,
    uploadMbps: uploadMbps / count,
  };
}

export function trafficAverageSeconds(intervalSeconds: number) {
  return averagingWindow(intervalSeconds).seconds;
}

export function latestObservedTimestamp(points: TrafficPoint[]) {
  for (let index = points.length - 1; index >= 0; index -= 1) {
    if (points[index]?.observed) return points[index]?.timestamp;
  }
  return undefined;
}

export function smoothTrafficPoints(
  points: TrafficPoint[],
  period: TrafficPeriod,
  intervalSeconds: number
) {
  if (!livePeriod(period)) return points;

  const visiblePoints = withoutIncompleteTail(points);
  const { samples } = averagingWindow(intervalSeconds);
  return visiblePoints.map((point, index) => ({
    ...point,
    ...averageAt(visiblePoints, index, samples),
  }));
}

export function getCurrentTrafficRate(
  series: TrafficSeries | undefined
): CurrentTrafficRate {
  const intervalSeconds = Math.max(1, series?.intervalSeconds || 5);
  const { samples, seconds } = averagingWindow(intervalSeconds);
  const points = series?.points || [];
  if (points.length === 0) {
    return {
      averagingSeconds: seconds,
      downloadMbps: 0,
      status: "unavailable",
      uploadMbps: 0,
    };
  }

  const observedAt = latestObservedTimestamp(points);
  if (observedAt === undefined) {
    return {
      averagingSeconds: seconds,
      downloadMbps: 0,
      status: "idle",
      uploadMbps: 0,
    };
  }

  const newestTimestamp = points.at(-1)?.timestamp || observedAt;
  const staleAfterMs = Math.max(20_000, (seconds + intervalSeconds) * 1000);
  if (newestTimestamp - observedAt > staleAfterMs) {
    return {
      averagingSeconds: seconds,
      downloadMbps: 0,
      observedAt,
      status: "idle",
      uploadMbps: 0,
    };
  }

  const visiblePoints = withoutIncompleteTail(points);
  if (visiblePoints.length === 0) {
    return {
      averagingSeconds: seconds,
      downloadMbps: 0,
      observedAt,
      status: "idle",
      uploadMbps: 0,
    };
  }
  const rate = averageAt(visiblePoints, visiblePoints.length - 1, samples);
  return {
    averagingSeconds: seconds,
    downloadMbps: rate.downloadMbps,
    observedAt,
    status: rate.downloadMbps > 0 || rate.uploadMbps > 0 ? "active" : "idle",
    uploadMbps: rate.uploadMbps,
  };
}
