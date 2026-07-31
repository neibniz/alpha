// @ts-nocheck
import { describe, expect, test } from "bun:test";
import {
  createFilterSnapshot,
  createPageRequest,
  fetchAllPages,
  normalizeListResult,
  normalizePageResult,
} from "./pagination.js";

describe("pagination utilities", () => {
  test("creates one based API page requests", () => {
    expect(createPageRequest(0, 10)).toEqual({ page: 1, size: 10 });
    expect(createPageRequest(4, 25)).toEqual({ page: 5, size: 25 });
  });

  test("converts column filters into request values and a stable signature", () => {
    const snapshot = createFilterSnapshot<{ status?: string; tags?: string[] }>(
      [
        { id: "status", value: "active" },
        { id: "tags", value: ["vip", "trial"] },
      ]
    );

    expect(snapshot.values).toEqual({
      status: "active",
      tags: ["vip", "trial"],
    });
    expect(snapshot.signature).toBe(
      '["status","active"]|["tags",["vip","trial"]]'
    );
  });

  test("reuses the empty filter values object", () => {
    const first = createFilterSnapshot([]);
    const second = createFilterSnapshot([]);

    expect(first).toEqual({ values: {}, signature: "" });
    expect(second.values).toBe(first.values);
  });

  test("normalizes incomplete page responses", () => {
    expect(normalizePageResult<string>(undefined)).toEqual({
      list: [],
      total: 0,
    });
    expect(normalizePageResult<string>({ total: -1 })).toEqual({
      list: [],
      total: 0,
    });
    expect(normalizePageResult({ list: ["a"] })).toEqual({
      list: ["a"],
      total: 1,
    });
    expect(normalizePageResult({ list: ["a"], total: 2 })).toEqual({
      list: ["a"],
      total: 2,
    });
  });

  test("normalizes list only responses", () => {
    expect(normalizeListResult<number>(undefined)).toEqual([]);
    expect(normalizeListResult({ list: [1, 2] })).toEqual([1, 2]);
  });

  test("fetches all pages with bounded concurrency and total truncation", async () => {
    const calls: Array<{ page: number; size: number }> = [];
    const result = await fetchAllPages(
      async ({ page, size }) => {
        calls.push({ page, size });
        const pages: Record<number, number[]> = {
          1: [1, 2],
          2: [3, 4],
          3: [5, 6],
          4: [7],
        };
        return {
          data: {
            data: {
              list: pages[page] || [],
              total: 5,
            },
          },
        };
      },
      2,
      2
    );

    expect(result).toEqual([1, 2, 3, 4, 5]);
    expect(calls).toEqual([
      { page: 1, size: 2 },
      { page: 2, size: 2 },
      { page: 3, size: 2 },
    ]);
  });

  test("fetches all pages with normalized concurrency", async () => {
    const calls: number[] = [];
    const result = await fetchAllPages(
      async ({ page }) => {
        calls.push(page);
        return {
          data: {
            data: {
              list: [page],
              total: 3,
            },
          },
        };
      },
      1,
      0
    );

    expect(result).toEqual([1, 2, 3]);
    expect(calls).toEqual([1, 2, 3]);
  });

  test("fetch all pages stops on empty first page", async () => {
    let calls = 0;
    const result = await fetchAllPages(async () => {
      calls += 1;
      return {
        data: {
          data: {
            list: [],
            total: 100,
          },
        },
      };
    });

    expect(result).toEqual([]);
    expect(calls).toBe(1);
  });
});
