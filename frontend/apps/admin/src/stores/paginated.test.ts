import { describe, expect, it } from "vitest";
import { fetchAllPages } from "./paginated";

type TestResponse = {
  data?: {
    data?: {
      list?: number[];
      total?: number;
    };
  };
};

function pageResponse(list: number[], total?: number): TestResponse {
  return {
    data: {
      data: {
        list,
        total,
      },
    },
  };
}

describe("paginated store helpers", () => {
  it("returns an empty list when the first page has no items", async () => {
    const calls: number[] = [];

    const result = await fetchAllPages(async ({ page }) => {
      calls.push(page);
      return pageResponse([], 0);
    }, 2);

    expect(result).toEqual([]);
    expect(calls).toEqual([1]);
  });

  it("fetches remaining pages concurrently after reading total", async () => {
    const calls: number[] = [];
    let inFlight = 0;
    let maxInFlight = 0;

    const result = await fetchAllPages(
      async ({ page }) => {
        calls.push(page);
        if (page === 1) {
          return pageResponse([1, 2], 5);
        }

        inFlight += 1;
        maxInFlight = Math.max(maxInFlight, inFlight);
        await new Promise((resolve) => setTimeout(resolve, 0));
        inFlight -= 1;

        return page === 2 ? pageResponse([3, 4], 5) : pageResponse([5], 5);
      },
      2,
      2
    );

    expect(result).toEqual([1, 2, 3, 4, 5]);
    expect(calls).toEqual([1, 2, 3]);
    expect(maxInFlight).toBe(2);
  });

  it("preserves page order when later pages resolve out of order", async () => {
    const result = await fetchAllPages(
      async ({ page }) => {
        if (page === 1) {
          return pageResponse([1, 2], 6);
        }
        if (page === 2) {
          await new Promise((resolve) => setTimeout(resolve, 5));
          return pageResponse([3, 4], 6);
        }
        return pageResponse([5, 6], 6);
      },
      2,
      2
    );

    expect(result).toEqual([1, 2, 3, 4, 5, 6]);
  });

  it("batches requests by the configured concurrency", async () => {
    let inFlight = 0;
    let maxInFlight = 0;

    const result = await fetchAllPages(
      async ({ page }) => {
        if (page === 1) {
          return pageResponse([1], 5);
        }

        inFlight += 1;
        maxInFlight = Math.max(maxInFlight, inFlight);
        await new Promise((resolve) => setTimeout(resolve, 0));
        inFlight -= 1;

        return pageResponse([page], 5);
      },
      1,
      2
    );

    expect(result).toEqual([1, 2, 3, 4, 5]);
    expect(maxInFlight).toBe(2);
  });

  it("trims overfilled final pages to the reported total", async () => {
    const result = await fetchAllPages(
      async ({ page }) => {
        if (page === 1) {
          return pageResponse([1, 2], 3);
        }
        return pageResponse([3, 4], 3);
      },
      2,
      2
    );

    expect(result).toEqual([1, 2, 3]);
  });
});
