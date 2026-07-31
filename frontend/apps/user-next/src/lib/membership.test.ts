import { describe, expect, test } from "bun:test";
import {
  membershipDurationLabel,
  membershipUsagePercent,
  safeMembershipReturnPath,
} from "./membership";

describe("membership helpers", () => {
  test("formats supported membership durations", () => {
    expect(membershipDurationLabel(1, "Year")).toBe("1 年");
    expect(membershipDurationLabel(3, "Month")).toBe("3 个月");
  });

  test("clamps usage progress", () => {
    expect(membershipUsagePercent(25, 100)).toBe(25);
    expect(membershipUsagePercent(120, 100)).toBe(100);
    expect(membershipUsagePercent(10, 0)).toBe(0);
  });

  test("allows only same-origin relative return paths", () => {
    expect(safeMembershipReturnPath("/products?duration=12")).toBe(
      "/products?duration=12"
    );
    expect(safeMembershipReturnPath("//attacker.example/path")).toBe(
      "/dashboard"
    );
    expect(safeMembershipReturnPath("https://attacker.example")).toBe(
      "/dashboard"
    );
  });
});
