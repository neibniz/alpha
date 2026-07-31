import { describe, expect, it } from "vitest";
import {
  buildNodeIndexes,
  mergeAvailableNodeTags,
  newCompactLookupBuilder,
} from "./node-data";

describe("node store data helpers", () => {
  it("builds compact node lookup indexes without per-node tag arrays", () => {
    const nodes = [
      {
        id: 1,
        server_id: 10,
        protocol: "vless",
        inbound_alias: "",
        tags: [" edge ", "", "premium"],
      },
      {
        id: 2,
        server_id: 10,
        protocol: "trojan",
        inbound_alias: "reality",
        tags: [],
      },
      {
        id: 3,
        server_id: 11,
        protocol: "hysteria2",
        inbound_alias: "",
        tags: ["edge"],
      },
    ] as API.Node[];

    const indexes = buildNodeIndexes(nodes);

    expect(indexes.nodeById.get(1)).toBe(nodes[0]);
    expect(indexes.serverIds.has(10)).toBe(true);
    expect(indexes.inboundUsage.has("10:vless")).toBe(true);
    expect(indexes.inboundUsage.has("10:reality")).toBe(true);
    expect(indexes.nodeTags).toEqual(["edge", "premium"]);
    expect(indexes.nodesByTag.has(" edge ")).toBe(false);
    expect(indexes.nodesByTag.get("edge")).toEqual([nodes[0], nodes[2]]);
    expect(indexes.nodesWithoutTags).toEqual([nodes[1]]);
  });

  it("deduplicates repeated tags within a single node before grouping", () => {
    const nodes = [
      {
        id: 1,
        server_id: 10,
        protocol: "vless",
        inbound_alias: "",
        tags: [" edge ", "edge", "premium", " premium "],
      },
    ] as API.Node[];

    const indexes = buildNodeIndexes(nodes);

    expect(indexes.nodeTags).toEqual(["edge", "premium"]);
    expect(indexes.nodesByTag.get("edge")).toEqual([nodes[0]]);
    expect(indexes.nodesByTag.get("premium")).toEqual([nodes[0]]);
    expect(indexes.nodesWithoutTags).toEqual([]);
  });

  it("uses compact lookup objects for small inbound and server indexes", () => {
    const nodes = [
      {
        id: 1,
        server_id: 10,
        protocol: "vless",
        inbound_alias: "",
      },
      {
        id: 2,
        server_id: 10,
        protocol: "trojan",
        inbound_alias: "reality",
      },
    ] as API.Node[];

    const indexes = buildNodeIndexes(nodes);

    expect(indexes.serverIds).not.toBeInstanceOf(Set);
    expect(indexes.inboundUsage).not.toBeInstanceOf(Set);
    expect(indexes.serverIds.has(10)).toBe(true);
    expect(indexes.serverIds.has(11)).toBe(false);
    expect(indexes.inboundUsage.has("10:vless")).toBe(true);
    expect(indexes.inboundUsage.has("10:missing")).toBe(false);
  });

  it("keeps compact lookup behavior after upgrading large inputs", () => {
    const builder = newCompactLookupBuilder<number>();
    for (let index = 1; index <= 18; index += 1) {
      builder.add(index);
    }
    builder.add(1);

    const lookup = builder.lookup();
    expect(lookup).toBeInstanceOf(Set);
    expect(lookup.has(1)).toBe(true);
    expect(lookup.has(18)).toBe(true);
    expect(lookup.has(19)).toBe(false);
  });

  it("merges API and node tags once in API order", () => {
    const manyNodeTags = Array.from(
      { length: 18 },
      (_, index) => `node-${index}`
    );

    expect(
      mergeAvailableNodeTags(
        ["api", "", "edge", "api"],
        [" edge ", "premium", ...manyNodeTags, "node-3"]
      )
    ).toEqual(["api", "edge", "premium", ...manyNodeTags]);
  });
});
