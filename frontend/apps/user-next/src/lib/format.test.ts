import { describe, expect, test } from "bun:test";
import {
  bitsPerSecondToMbps,
  formatDate,
  formatSpeedLimitMbps,
} from "@/lib/format";

describe("speed limit formatting", () => {
  test("converts the API bit-rate unit to Mbps", () => {
    expect(bitsPerSecondToMbps(100 * 1024 ** 2)).toBe(100);
    expect(bitsPerSecondToMbps(1.5 * 1024 ** 2)).toBe(1.5);
  });

  test("handles unlimited and fractional limits", () => {
    expect(formatSpeedLimitMbps(0)).toBe("不限");
    expect(formatSpeedLimitMbps(Number.NaN)).toBe("不限");
    expect(formatSpeedLimitMbps(12.5)).toBe("12.5 Mbps");
  });
});

describe("date formatting", () => {
  test("uses a deterministic timezone for server and client rendering", () => {
    expect(formatDate(Date.UTC(2026, 0, 1, 0, 30))).toBe("2026-01-01");
  });
});
