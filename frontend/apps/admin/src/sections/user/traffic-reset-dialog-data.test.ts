import { describe, expect, test } from "vitest";
import {
  canSubmitTrafficReset,
  RESET_ALL_TRAFFIC_CONFIRMATION,
} from "./traffic-reset-dialog-data";

describe("traffic reset confirmation", () => {
  test("requires the exact confirmation phrase for all subscriptions", () => {
    expect(canSubmitTrafficReset("all", "", [])).toBe(false);
    expect(canSubmitTrafficReset("all", "reset all traffic", [])).toBe(false);
    expect(
      canSubmitTrafficReset("all", ` ${RESET_ALL_TRAFFIC_CONFIRMATION} `, [])
    ).toBe(true);
  });

  test("requires at least one valid selected user", () => {
    expect(canSubmitTrafficReset("users", "", [])).toBe(false);
    expect(canSubmitTrafficReset("users", "", [0, -1])).toBe(false);
    expect(canSubmitTrafficReset("users", "", [42])).toBe(true);
  });
});
