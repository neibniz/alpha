import { describe, expect, it } from "vitest";
import {
  buildAttentionBreakdown,
  buildFleetSummary,
  configHealth,
  getOverallHealth,
  resourceHealth,
  statusIsOffline,
} from "./server-fleet-data";

const NOW = 1_000_000;

function status(overrides: Partial<API.ServerStatus> = {}) {
  return {
    cpu: 20,
    mem: 30,
    disk: 40,
    status: "online",
    xray_running: true,
    config_sync_status: "synced",
    config_apply_status: "applied",
    last_seen: NOW - 5000,
    ...overrides,
  } satisfies Partial<API.ServerStatus>;
}

describe("server fleet health", () => {
  it("marks stale heartbeats offline", () => {
    expect(statusIsOffline(status(), NOW)).toBe(false);
    expect(
      statusIsOffline(status({ status: "", last_seen: NOW - 31_000 }), NOW)
    ).toBe(true);
  });

  it("trusts an explicit server status over a stale browser cache", () => {
    expect(statusIsOffline(status({ last_seen: NOW - 60_000 }), NOW)).toBe(
      false
    );
    expect(
      statusIsOffline(status({ status: "offline", last_seen: NOW - 1000 }), NOW)
    ).toBe(true);
  });

  it("prioritizes failed and pending configuration states", () => {
    expect(configHealth(status({ config_sync_status: "failed" }))).toBe(
      "error"
    );
    expect(configHealth(status({ pending_config_hash: "pending" }))).toBe(
      "warning"
    );
  });

  it("flags sustained resource pressure", () => {
    expect(resourceHealth(status({ cpu: 89 }))).toBe("ok");
    expect(resourceHealth(status({ mem: 92 }))).toBe("warning");
    expect(resourceHealth(status({ disk: 97 }))).toBe("error");
  });

  it("builds fleet summaries without double-counting categories", () => {
    const statuses = [
      status(),
      status({ status: "offline" }),
      status({ xray_running: false }),
      status({ config_sync_status: "pending" }),
      status({ cpu: 93 }),
    ];

    expect(buildFleetSummary(statuses, 8, NOW)).toEqual({
      total: 8,
      online: 4,
      attention: 4,
      pending: 1,
    });
    expect(buildAttentionBreakdown(statuses, NOW)).toEqual({
      offline: 1,
      xray: 1,
      config: 1,
      resources: 1,
    });
  });

  it("returns a single overall state for scan-first rendering", () => {
    expect(getOverallHealth(status(), NOW)).toBe("healthy");
    expect(getOverallHealth(status({ cpu: 91 }), NOW)).toBe("attention");
    expect(getOverallHealth(status({ status: "offline" }), NOW)).toBe(
      "offline"
    );
  });
});
