import { describe, expect, it } from "vitest";
import {
  buildByNumericId,
  buildNumericIdListIndex,
  patchListItemById,
} from "./list-index";

type TestItem = {
  id?: number;
  name: string;
  sort?: number;
};

describe("store list indexes", () => {
  it("builds numeric id maps without temporary tuple arrays", () => {
    const first: TestItem = { id: 1, name: "one" };
    const updated: TestItem = { id: 1, name: "updated" };
    const result = buildByNumericId([
      first,
      { id: 0, name: "zero" },
      { id: -1, name: "negative" },
      { id: 1.5, name: "decimal" },
      { name: "missing" },
      updated,
      { id: 2, name: "two" },
    ]);

    expect([...result.entries()]).toEqual([
      [1, updated],
      [2, { id: 2, name: "two" }],
    ]);
  });

  it("builds numeric id and position indexes in one pass", () => {
    const first: TestItem = { id: 1, name: "one" };
    const updated: TestItem = { id: 1, name: "updated" };
    const result = buildNumericIdListIndex([
      first,
      { id: 0, name: "zero" },
      { id: -1, name: "negative" },
      { id: 1.5, name: "decimal" },
      { name: "missing" },
      updated,
      { id: 2, name: "two" },
    ]);

    expect([...result.byId.entries()]).toEqual([
      [1, updated],
      [2, { id: 2, name: "two" }],
    ]);
    expect([...result.indexById.entries()]).toEqual([
      [1, 5],
      [2, 6],
    ]);
  });

  it("patches one indexed item without rebuilding the whole index", () => {
    const items: TestItem[] = [
      { id: 1, name: "Basic", sort: 1 },
      { id: 2, name: "Premium", sort: 2 },
    ];
    const byId = buildByNumericId(items);
    const result = patchListItemById(items, byId, {
      id: 2,
      name: "Premium Plus",
    });

    expect(result?.items).toEqual([
      { id: 1, name: "Basic", sort: 1 },
      { id: 2, name: "Premium Plus", sort: 2 },
    ]);
    expect(result?.items[0]).toBe(items[0]);
    expect(result?.byId.get(1)).toBe(byId.get(1));
    expect(result?.byId.get(2)).toEqual({
      id: 2,
      name: "Premium Plus",
      sort: 2,
    });
  });

  it("patches by position index when available", () => {
    const items: TestItem[] = [
      { id: 1, name: "Basic", sort: 1 },
      { id: 2, name: "Premium", sort: 2 },
      { id: 3, name: "Enterprise", sort: 3 },
    ];
    const index = buildNumericIdListIndex(items);
    const result = patchListItemById(
      items,
      index.byId,
      {
        id: 3,
        name: "Enterprise Plus",
      },
      index.indexById
    );

    expect(result?.items).toEqual([
      { id: 1, name: "Basic", sort: 1 },
      { id: 2, name: "Premium", sort: 2 },
      { id: 3, name: "Enterprise Plus", sort: 3 },
    ]);
    expect(result?.items[0]).toBe(items[0]);
    expect(result?.items[1]).toBe(items[1]);
    expect(result?.indexById).toBe(index.indexById);
    expect(result?.indexById?.get(3)).toBe(2);
  });

  it("falls back safely when a provided position index is stale", () => {
    const items: TestItem[] = [
      { id: 1, name: "Basic", sort: 1 },
      { id: 2, name: "Premium", sort: 2 },
    ];
    const byId = buildByNumericId(items);
    const staleIndex = new Map([
      [1, 1],
      [2, 0],
    ]);

    const result = patchListItemById(
      items,
      byId,
      {
        id: 2,
        name: "Premium Plus",
      },
      staleIndex
    );

    expect(result?.items).toEqual([
      { id: 1, name: "Basic", sort: 1 },
      { id: 2, name: "Premium Plus", sort: 2 },
    ]);
    expect(result?.indexById?.get(2)).toBe(1);
  });

  it("returns undefined for invalid or missing patches", () => {
    const items: TestItem[] = [{ id: 1, name: "Basic" }];
    const byId = buildByNumericId(items);

    expect(patchListItemById(items, byId, { name: "No id" })).toBeUndefined();
    expect(
      patchListItemById(items, byId, { id: -1, name: "Negative" })
    ).toBeUndefined();
    expect(
      patchListItemById(items, byId, { id: 1.5, name: "Decimal" })
    ).toBeUndefined();
    expect(
      patchListItemById(items, byId, { id: 99, name: "Missing" })
    ).toBeUndefined();
  });

  it("skips no-op patches without copying list indexes", () => {
    const items: TestItem[] = [
      { id: 1, name: "Basic", sort: 1 },
      { id: 2, name: "Premium", sort: 2 },
    ];
    const index = buildNumericIdListIndex(items);

    expect(
      patchListItemById(
        items,
        index.byId,
        { id: 2, name: "Premium", sort: 2 },
        index.indexById
      )
    ).toBeUndefined();
  });
});
