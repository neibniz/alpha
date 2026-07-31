import { describe, expect, it } from "vitest";
import {
  analyseNodes,
  buildNodeSummary,
  getInboundProtocol,
  getNodeReadiness,
  getServerHealth,
  sortNodes,
} from "./node-directory-data";

const NOW = 1_000_000;

function server(id: number, status = "online") {
  return {
    id,
    name: `Server ${id}`,
    country: "",
    city: "",
    address: `10.0.0.${id}`,
    sort: id,
    last_reported_at: NOW - 5000,
    status: {
      cpu: 10,
      mem: 20,
      disk: 30,
      status,
      last_seen: status === "offline" ? NOW - 31_000 : NOW - 5000,
    },
    created_at: NOW,
    updated_at: NOW,
  } satisfies API.Server;
}

function node(id: number, overrides: Partial<API.NodeListItem> = {}) {
  return {
    id,
    name: `Node ${id}`,
    tags: [],
    port: 443,
    address: `node-${id}.example.com`,
    server_id: 1,
    inbound_alias: "vless-reality-443",
    protocol: "vless",
    enabled: true,
    sort: id,
    created_at: NOW,
    updated_at: NOW,
    ...overrides,
  } satisfies API.NodeListItem;
}

describe("node directory data", () => {
  it("normalizes inbound protocol labels", () => {
    expect(getInboundProtocol(node(1))).toBe("VLESS");
    expect(getInboundProtocol(node(2, { inbound_alias: "trojan-tls" }))).toBe(
      "Trojan"
    );
  });

  it("detects stale and missing servers", () => {
    expect(getServerHealth(server(1), NOW)).toBe("online");
    expect(getServerHealth(server(1, "offline"), NOW)).toBe("offline");
    expect(getServerHealth(undefined, NOW)).toBe("missing");
  });

  it("only raises operational issues for enabled nodes", () => {
    const servers = new Map<number, API.Server>([
      [1, server(1)],
      [2, server(2, "offline")],
    ]);
    const nodes = [
      node(1, { address: "shared.example.com" }),
      node(2, { address: "shared.example.com" }),
      node(3, { port: 0 }),
      node(4, { server_id: 2 }),
      node(5, { server_id: 999 }),
      node(6, { enabled: false, port: 0, server_id: 999 }),
    ];
    const analysis = analyseNodes(nodes, servers, NOW);

    expect([...analysis.conflictingNodeIds]).toEqual([1, 2]);
    expect([...analysis.incompleteNodeIds]).toEqual([3]);
    expect([...analysis.offlineNodeIds]).toEqual([4]);
    expect([...analysis.missingServerNodeIds]).toEqual([5]);
    expect([...analysis.attentionNodeIds]).toEqual([1, 2, 3, 4, 5]);
    expect(getNodeReadiness(nodes[5]!, analysis)).toBe("disabled");
  });

  it("summarizes valid server coverage and attention", () => {
    const servers = new Map<number, API.Server>([[1, server(1)]]);
    const nodes = [
      node(1),
      node(2, { enabled: false }),
      node(3, { server_id: 9 }),
    ];
    const analysis = analyseNodes(nodes, servers, NOW);

    expect(buildNodeSummary(nodes, analysis, servers)).toEqual({
      total: 3,
      enabled: 2,
      disabled: 1,
      attention: 1,
      servers: 1,
    });
  });

  it("sorts by explicit order and then id", () => {
    expect(
      sortNodes([node(3, { sort: 2 }), node(1, { sort: 1 })]).map(
        (item) => item.id
      )
    ).toEqual([1, 3]);
  });
});
