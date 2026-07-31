import { describe, expect, test } from "bun:test";
import {
  availableTicketImageSlots,
  MAX_TICKET_IMAGES,
  mergeTicketImages,
} from "./ticket-images";

describe("ticket image collections", () => {
  test("calculates bounded available slots", () => {
    expect(availableTicketImageSlots(-1)).toBe(MAX_TICKET_IMAGES);
    expect(availableTicketImageSlots(4)).toBe(2);
    expect(availableTicketImageSlots(10)).toBe(0);
  });

  test("deduplicates images and preserves stable order", () => {
    expect(mergeTicketImages(["a", "b"], ["b", "c"])).toEqual(["a", "b", "c"]);
  });

  test("caps merged galleries at six images", () => {
    expect(mergeTicketImages([], ["1", "2", "3", "4", "5", "6", "7"])).toEqual([
      "1",
      "2",
      "3",
      "4",
      "5",
      "6",
    ]);
  });
});
