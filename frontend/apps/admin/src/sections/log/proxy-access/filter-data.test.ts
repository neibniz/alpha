import { describe, expect, it } from "vitest";
import {
  proxyAccessFilterError,
  proxyAccessFilterParams,
  type ProxyAccessFilters,
} from "./filter-data";

const valid: ProxyAccessFilters = {
  start: "2026-07-01T00:00",
  end: "2026-07-02T00:00",
  serverId: "",
  userId: "",
  subscribeId: "",
  sourceIP: "",
  destination: "",
  status: "all",
};

describe("proxy access filters", () => {
  it("rejects invalid time ranges before querying", () => {
    expect(proxyAccessFilterError({ ...valid, start: "" })).toBe("invalidTime");
    expect(
      proxyAccessFilterError({ ...valid, start: "2026-07-03T00:00" })
    ).toBe("invalidTimeOrder");
    expect(proxyAccessFilterError({ ...valid, end: "2026-08-03T00:01" })).toBe(
      "timeRangeTooLarge"
    );
  });

  it("rejects non-positive and non-integer identifiers", () => {
    expect(proxyAccessFilterError({ ...valid, serverId: "abc" })).toBe(
      "invalidServerId"
    );
    expect(proxyAccessFilterError({ ...valid, userId: "0" })).toBe(
      "invalidUserId"
    );
    expect(proxyAccessFilterError({ ...valid, subscribeId: "1.5" })).toBe(
      "invalidSubscribeId"
    );
  });

  it("normalizes exact-match filters into API parameters", () => {
    const result = proxyAccessFilterParams(
      {
        ...valid,
        serverId: " 7 ",
        userId: "8",
        subscribeId: "9",
        sourceIP: " 203.0.113.7 ",
        destination: " api.example.com ",
        status: "accepted",
      },
      50,
      "cursor"
    );
    expect(result).toMatchObject({
      server_id: 7,
      user_id: 8,
      user_subscribe_id: 9,
      source_ip: "203.0.113.7",
      destination: "api.example.com",
      status: "accepted",
      size: 50,
      cursor: "cursor",
    });
  });
});
