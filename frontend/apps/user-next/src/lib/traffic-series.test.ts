import { describe, expect, test } from "bun:test";
import type { TrafficPoint, TrafficSeries } from "@/lib/models";
import {
  getCurrentTrafficRate,
  smoothTrafficPoints,
} from "@/lib/traffic-series";

function point(
  timestamp: number,
  downloadMbps: number,
  observed = true,
  uploadMbps = downloadMbps / 4
): TrafficPoint {
  return { downloadMbps, observed, timestamp, uploadMbps };
}

function series(points: TrafficPoint[]): TrafficSeries {
  return { intervalSeconds: 5, points };
}

describe("traffic series presentation", () => {
  test("uses recent observed buckets instead of an incomplete zero tail", () => {
    const rate = getCurrentTrafficRate(
      series([
        point(0, 10),
        point(5000, 20),
        point(10_000, 30),
        point(15_000, 0, false, 0),
      ])
    );

    expect(rate.status).toBe("active");
    expect(rate.averagingSeconds).toBe(15);
    expect(rate.downloadMbps).toBe(20);
    expect(rate.uploadMbps).toBe(5);
    expect(rate.observedAt).toBe(10_000);
  });

  test("returns an idle zero rate after the last sample becomes stale", () => {
    const rate = getCurrentTrafficRate(
      series([
        point(0, 30),
        point(5000, 0, false, 0),
        point(10_000, 0, false, 0),
        point(15_000, 0, false, 0),
        point(20_000, 0, false, 0),
        point(25_000, 0, false, 0),
      ])
    );

    expect(rate.status).toBe("idle");
    expect(rate.downloadMbps).toBe(0);
    expect(rate.uploadMbps).toBe(0);
  });

  test("spreads an isolated live spike across a short readable plateau", () => {
    const smoothed = smoothTrafficPoints(
      [
        point(0, 0, false, 0),
        point(5000, 0, false, 0),
        point(10_000, 30),
        point(15_000, 0, false, 0),
        point(20_000, 0, false, 0),
        point(25_000, 0, false, 0),
      ],
      "5m",
      5
    );

    expect(smoothed.length).toBe(5);
    expect(smoothed.slice(2).map((item) => item.downloadMbps)).toEqual([
      10, 10, 10,
    ]);
  });

  test("does not smooth historical aggregate periods", () => {
    const points = [point(0, 10), point(60_000, 30)];
    expect(smoothTrafficPoints(points, "1h", 60)).toBe(points);
  });
});
