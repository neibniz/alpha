import { describe, expect, test } from "vitest";
import {
  mapServerNetworkRateSeries,
  mergeServerNetworkRatePoint,
  serverStatusToNetworkRatePoint,
} from "./network-rate-data";

describe("server network rate data", () => {
  test("replaces points in the same five-second bucket", () => {
    const first = serverStatusToNetworkRatePoint(
      { updated_at: 10_100, net_rx_bps: 10 },
      0
    );
    const replacement = serverStatusToNetworkRatePoint(
      { updated_at: 14_900, net_rx_bps: 20 },
      0
    );

    const points = mergeServerNetworkRatePoint([first], replacement);
    expect(points).toEqual([{ ...replacement, timestamp: 10_000 }]);
  });

  test("keeps only the newest requested number of points", () => {
    const points = [0, 5000, 10_000].map((timestamp) => ({
      timestamp,
      net_rx_bps: timestamp,
      net_tx_bps: 0,
      xray_rx_bps: 0,
      xray_tx_bps: 0,
    }));
    const next = mergeServerNetworkRatePoint(
      points,
      { ...points[0]!, timestamp: 15_000 },
      3
    );
    expect(next.map((point) => point.timestamp)).toEqual([
      5000, 10_000, 15_000,
    ]);
  });

  test("maps batch history by server id", () => {
    const result = mapServerNetworkRateSeries([
      { server_id: 7, points: [] },
      {
        server_id: 8,
        points: [
          {
            timestamp: 5000,
            net_rx_bps: 1,
            net_tx_bps: 2,
            xray_rx_bps: 3,
            xray_tx_bps: 4,
          },
        ],
      },
    ]);
    expect(result[7]).toEqual([]);
    expect(result[8]?.[0]?.xray_tx_bps).toBe(4);
  });
});
