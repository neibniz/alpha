import { describe, expect, it } from "bun:test";
import { filterAvailableTagOptions } from "./tag-input";

describe("filterAvailableTagOptions", () => {
  it("filters selected options and matches query case-insensitively", () => {
    const result = filterAvailableTagOptions(
      ["Alpha", "Beta", "Gamma"],
      ["Beta"],
      "al"
    );

    expect(result).toEqual(["Alpha"]);
  });

  it("keeps option order and uses a set for large selected lists", () => {
    const tags = Array.from({ length: 12 }, (_, index) => `tag-${index}`);
    const result = filterAvailableTagOptions(
      ["tag-0", "extra", "tag-11", "Alpha"],
      tags,
      ""
    );

    expect(result).toEqual(["extra", "Alpha"]);
  });

  it("preserves blank-query semantics from the component filter", () => {
    const result = filterAvailableTagOptions(["Alpha", "Beta"], [], "   ");

    expect(result).toEqual(["Alpha", "Beta"]);
  });

  it("keeps raw nonblank query spacing semantics", () => {
    const result = filterAvailableTagOptions(["Alpha", "Beta"], [], " al ");

    expect(result).toEqual([]);
  });
});
