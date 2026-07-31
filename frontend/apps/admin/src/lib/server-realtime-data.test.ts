import { describe, expect, it } from "vitest";
import {
  mergeRealtimeStatus,
  normalizeRealtimeServerIDs,
  parseRealtimeServerMessage,
} from "./server-realtime-data";

describe("server realtime data", () => {
  it("normalizes subscription IDs without duplicates", () => {
    expect(normalizeRealtimeServerIDs([3, 3, -1, 2, Number.NaN])).toEqual([
      3, 2,
    ]);
  });

  it("parses snapshots and clears omitted realtime counters", () => {
    const updates = parseRealtimeServerMessage(
      JSON.stringify({
        type: "snapshot",
        data: [
          {
            type: "node_status",
            server_id: 7,
            status: "online",
            last_seen: 123,
          },
        ],
      })
    );

    expect(updates).toHaveLength(1);
    expect(updates[0]).toMatchObject({
      serverId: 7,
      status: { status: "online", last_seen: 123, connections: 0 },
    });
  });

  it("ignores malformed messages and invalid server IDs", () => {
    expect(parseRealtimeServerMessage("not-json")).toEqual([]);
    expect(
      parseRealtimeServerMessage({ type: "node_status", server_id: 0 })
    ).toEqual([]);
  });

  it("does not let an older event overwrite a newer status", () => {
    const current = {
      status: "online",
      last_seen: 200,
      updated_at: 200,
    } satisfies Partial<API.ServerStatus>;
    expect(
      mergeRealtimeStatus(current, {
        status: "offline",
        last_seen: 100,
        updated_at: 150,
      })
    ).toBe(current);
    expect(
      mergeRealtimeStatus(current, {
        status: "offline",
        last_seen: 200,
        updated_at: 250,
      })
    ).toMatchObject({ status: "offline", updated_at: 250 });
  });
});
