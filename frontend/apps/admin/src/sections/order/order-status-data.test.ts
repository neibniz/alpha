import { describe, expect, it } from "vitest";
import {
  adminOrderStatusTargets,
  toAdminOrderListRequest,
} from "./order-status-data";

describe("adminOrderStatusTargets", () => {
  it("only exposes transitions that preserve the activation lifecycle", () => {
    expect(adminOrderStatusTargets(1)).toEqual([2, 3, 4]);
    expect(adminOrderStatusTargets(3)).toEqual([1]);
    expect(adminOrderStatusTargets(4)).toEqual([1]);
  });

  it("keeps paid, finished, and unknown statuses immutable", () => {
    expect(adminOrderStatusTargets(2)).toEqual([]);
    expect(adminOrderStatusTargets(5)).toEqual([]);
    expect(adminOrderStatusTargets(0)).toEqual([]);
    expect(adminOrderStatusTargets(9)).toEqual([]);
  });
});

describe("toAdminOrderListRequest", () => {
  it("normalizes table filter strings into API numbers", () => {
    expect(
      toAdminOrderListRequest(
        { page: 2, size: 25 },
        {
          search: "  QA-100 ",
          status: "2",
          subscribe_id: "11",
          user_id: "42",
        }
      )
    ).toEqual({
      page: 2,
      size: 25,
      search: "QA-100",
      status: 2,
      subscribe_id: 11,
      user_id: 42,
    });
  });

  it("drops malformed numeric filters", () => {
    expect(
      toAdminOrderListRequest(
        { page: 1, size: 10 },
        { status: "paid", subscribe_id: "0", user_id: "4.2" }
      )
    ).toEqual({
      page: 1,
      size: 10,
      search: undefined,
      status: undefined,
      subscribe_id: undefined,
      user_id: undefined,
    });
  });
});
