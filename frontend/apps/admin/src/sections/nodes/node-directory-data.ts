import { statusIsOffline } from "../servers/server-fleet-data";

export type NodeFilter = "all" | "enabled" | "disabled" | "attention";
export type ServerHealth = "online" | "offline" | "unknown" | "missing";
export type NodeReadiness = "ready" | "disabled" | "attention";

export type NodeAnalysis = {
  attentionNodeIds: Set<number>;
  conflictingNodeIds: Set<number>;
  incompleteNodeIds: Set<number>;
  offlineNodeIds: Set<number>;
  offlineServerIds: Set<number>;
  missingServerNodeIds: Set<number>;
};

export function getInboundAlias(node: API.NodeListItem) {
  return node.inbound_alias || node.protocol || "";
}

export function getInboundProtocol(node: API.NodeListItem) {
  const protocol = getInboundAlias(node).split(/[-_]/, 1)[0]?.toLowerCase();
  const labels: Record<string, string> = {
    shadowsocks: "Shadowsocks",
    trojan: "Trojan",
    vless: "VLESS",
    vmess: "VMess",
  };
  return (protocol && labels[protocol]) || "Inbound";
}

export function getServerHealth(
  server?: API.Server,
  now = Date.now()
): ServerHealth {
  if (!server) return "missing";
  if (statusIsOffline(server.status || {}, now)) return "offline";
  if (server.status?.status === "online") return "online";
  return "unknown";
}

function endpointKey(node: API.NodeListItem) {
  const address = node.address?.trim().toLowerCase();
  const port = Number(node.port);
  return address && port > 0 ? `${address}:${port}` : "";
}

export function analyseNodes(
  nodes: API.NodeListItem[],
  serverById: Map<number, API.Server>,
  now = Date.now()
): NodeAnalysis {
  const endpointOwner = new Map<string, number>();
  const conflictingNodeIds = new Set<number>();
  const incompleteNodeIds = new Set<number>();
  const offlineNodeIds = new Set<number>();
  const offlineServerIds = new Set<number>();
  const missingServerNodeIds = new Set<number>();

  for (const node of nodes) {
    if (!node.enabled) continue;

    const key = endpointKey(node);
    if (key) {
      const owner = endpointOwner.get(key);
      if (owner !== undefined) {
        conflictingNodeIds.add(owner);
        conflictingNodeIds.add(node.id);
      } else {
        endpointOwner.set(key, node.id);
      }
    }

    if (
      !(node.address?.trim() && Number(node.port) > 0 && getInboundAlias(node))
    ) {
      incompleteNodeIds.add(node.id);
    }

    const server = serverById.get(node.server_id);
    const health = getServerHealth(server, now);
    if (health === "missing") missingServerNodeIds.add(node.id);
    if (health === "offline") {
      offlineNodeIds.add(node.id);
      offlineServerIds.add(node.server_id);
    }
  }

  return {
    attentionNodeIds: new Set([
      ...conflictingNodeIds,
      ...incompleteNodeIds,
      ...offlineNodeIds,
      ...missingServerNodeIds,
    ]),
    conflictingNodeIds,
    incompleteNodeIds,
    offlineNodeIds,
    offlineServerIds,
    missingServerNodeIds,
  };
}

export function getNodeReadiness(
  node: API.NodeListItem,
  analysis: NodeAnalysis
): NodeReadiness {
  if (!node.enabled) return "disabled";
  return analysis.attentionNodeIds.has(node.id) ? "attention" : "ready";
}

export function sortNodes(nodes: API.NodeListItem[]) {
  return nodes.slice().sort((a, b) => {
    const aSort = typeof a.sort === "number" ? a.sort : Number.MAX_SAFE_INTEGER;
    const bSort = typeof b.sort === "number" ? b.sort : Number.MAX_SAFE_INTEGER;
    if (aSort !== bSort) return aSort - bSort;
    return a.id - b.id;
  });
}

export function buildNodeSummary(
  nodes: API.NodeListItem[],
  analysis: NodeAnalysis,
  serverById: Map<number, API.Server>
) {
  return {
    total: nodes.length,
    enabled: nodes.filter((node) => node.enabled).length,
    disabled: nodes.filter((node) => !node.enabled).length,
    attention: analysis.attentionNodeIds.size,
    servers: new Set(
      nodes
        .filter((node) => serverById.has(node.server_id))
        .map((node) => node.server_id)
    ).size,
  };
}
