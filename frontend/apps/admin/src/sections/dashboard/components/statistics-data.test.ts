import { describe, expect, it } from "vitest";
import {
  aggregateNetworkRateSeries,
  appendRealtimeTrafficPoint,
  appendRealtimeValuePoint,
  buildTrafficRankingData,
  mergeRealtimeNetworkPoints,
} from "./statistics-data";

describe("dashboard statistics data helpers", () => {
  it("keeps empty traffic rankings as empty arrays", () => {
    const got = buildTrafficRankingData(undefined);

    expect(got.nodes.today).toEqual([]);
    expect(got.nodes.month).toEqual([]);
    expect(got.users.today).toEqual([]);
    expect(got.users.month).toEqual([]);
  });

  it("maps server and user traffic rankings in one pass per ranking", () => {
    const got = buildTrafficRankingData({
      server_ranking_today: [
        { name: "Node A", server_id: 1, upload: 100, download: 200 },
      ],
      server_ranking_monthly: [
        { name: "Node B", server_id: 2, upload: 300, download: 400 },
      ],
      user_ranking_today: [{ sid: 11, upload: 10, download: 20 }],
      user_ranking_monthly: [{ sid: 12, upload: 30, download: 40 }],
    });

    expect(got.nodes.today).toEqual([{ name: "Node A", traffic: 300 }]);
    expect(got.nodes.month).toEqual([{ name: "Node B", traffic: 700 }]);
    expect(got.users.today).toEqual([{ name: "11", traffic: 30 }]);
    expect(got.users.month).toEqual([{ name: "12", traffic: 70 }]);
  });

  it("supports partial realtime traffic patches", () => {
    const got = buildTrafficRankingData({
      user_ranking_today: [{ sid: 7, upload: 1, download: 2 }],
    });

    expect(got.nodes.today).toEqual([]);
    expect(got.nodes.month).toEqual([]);
    expect(got.users.today).toEqual([{ name: "7", traffic: 3 }]);
    expect(got.users.month).toEqual([]);
  });

  it("appends realtime value points while replacing minute buckets in one pass", () => {
    const got = appendRealtimeValuePoint(
      [valuePoint(1000, 1), valuePoint(61_000, 2), valuePoint(181_000, 4)],
      valuePoint(121_000, 3),
      60_000
    );

    expect(got).toEqual([
      valuePoint(61_000, 2),
      valuePoint(121_000, 3),
      valuePoint(181_000, 4),
    ]);
  });

  it("filters stale realtime traffic points and replaces existing minute buckets", () => {
    const got = appendRealtimeTrafficPoint(
      [trafficPoint(60_000, 10), trafficPoint(120_000, 20)],
      trafficPoint(125_000, 30),
      60_000
    );

    expect(got).toEqual([trafficPoint(60_000, 10), trafficPoint(125_000, 30)]);
    expect(
      appendRealtimeTrafficPoint(
        [trafficPoint(60_000, 10)],
        trafficPoint(1, 1),
        60_000
      )
    ).toEqual([trafficPoint(60_000, 10)]);
  });

  it("merges realtime network points by rate bucket", () => {
    const got = mergeRealtimeNetworkPoints(
      [networkPoint(1000, 1), networkPoint(7000, 2), networkPoint(20_000, 3)],
      [
        networkPoint(8000, 20),
        networkPoint(13_000, 30),
        networkPoint(21_000, 40),
      ],
      5000
    );

    expect(got).toEqual([
      networkPoint(8000, 20),
      networkPoint(13_000, 30),
      networkPoint(21_000, 40),
    ]);
  });

  it("does not let stale incoming points evict retained bucket data", () => {
    const got = mergeRealtimeNetworkPoints(
      [networkPoint(6000, 60)],
      [networkPoint(5000, 50)],
      5500
    );

    expect(got).toEqual([networkPoint(6000, 60)]);
  });

  it("keeps realtime network point empty results as empty arrays", () => {
    expect(mergeRealtimeNetworkPoints(undefined, undefined, 0)).toEqual([]);
    expect(
      mergeRealtimeNetworkPoints([networkPoint(1000, 1)], [], 5000)
    ).toEqual([]);
  });

  it("aggregates five-second network rates into minute averages", () => {
    const got = aggregateNetworkRateSeries([
      networkPoint(5000, 10),
      networkPoint(45_000, 30),
      networkPoint(65_000, 50),
    ]);

    expect(got).toEqual([
      {
        system_rx_bps: 20,
        system_tx_bps: 21,
        timestamp: 45_000,
        xray_rx_bps: 22,
        xray_tx_bps: 23,
      },
      networkPoint(65_000, 50),
    ]);
  });
});

function valuePoint(
  timestamp: number,
  value: number
): API.DashboardRealtimeResponse["online_user_series"]["last_day"][number] {
  return { timestamp, value };
}

function trafficPoint(
  timestamp: number,
  value: number
): API.DashboardRealtimeResponse["traffic"]["today_series"][number] {
  return {
    download: value,
    timestamp,
    total: value * 2,
    upload: value,
  };
}

function networkPoint(
  timestamp: number,
  value: number
): API.DashboardRealtimeResponse["network_series"][number] {
  return {
    system_rx_bps: value,
    system_tx_bps: value + 1,
    timestamp,
    xray_rx_bps: value + 2,
    xray_tx_bps: value + 3,
  };
}
