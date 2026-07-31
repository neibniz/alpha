import {
  filterNodeList,
  queryNodeTag,
} from "@workspace/ui/services/admin/server";
import { create } from "zustand";
import {
  buildNodeIndexes,
  type CompactLookup,
  mergeAvailableNodeTags,
  nodeInboundKey,
} from "./node-data";
import { fetchAllPages } from "./paginated";

const NODE_STORE_PAGE_SIZE = 100;

interface NodeState {
  // Data
  nodes: API.Node[];
  nodeById: Map<number, API.Node>;
  nodesByTag: Map<string, API.Node[]>;
  nodesWithoutTags: API.Node[];
  nodeTags: string[];
  inboundUsage: CompactLookup<string>;
  serverIds: CompactLookup<number>;
  tags: string[];
  availableTags: string[];

  // Loading states
  loading: boolean;
  loadingTags: boolean;
  loaded: boolean;
  loadedTags: boolean;

  // Actions
  fetchNodes: () => Promise<void>;
  fetchTags: () => Promise<void>;

  // Getters
  getNodeById: (nodeId: number) => API.Node | undefined;
  isInboundUsedInNodes: (serverId: number, inboundAlias: string) => boolean;
  isProtocolUsedInNodes: (serverId: number, protocolType: string) => boolean;
  isServerReferencedByNodes: (serverId: number) => boolean;
  getNodesByTag: (tag: string) => API.Node[];
  getNodesWithoutTags: () => API.Node[];
  getNodeTags: () => string[];
  getAllAvailableTags: () => string[];
}

export const useNodeStore = create<NodeState>((set, get) => ({
  // Initial state
  nodes: [],
  nodeById: new Map(),
  nodesByTag: new Map(),
  nodesWithoutTags: [],
  nodeTags: [],
  inboundUsage: new Set(),
  serverIds: new Set(),
  tags: [],
  availableTags: [],
  loading: false,
  loadingTags: false,
  loaded: false,
  loadedTags: false,

  // Actions
  fetchNodes: async () => {
    if (get().loading) return;

    set({ loading: true });
    try {
      const nodes = await fetchAllPages<API.Node>(
        filterNodeList,
        NODE_STORE_PAGE_SIZE
      );
      const indexes = buildNodeIndexes(nodes);
      set((state) => ({
        nodes,
        ...indexes,
        availableTags: mergeAvailableNodeTags(state.tags, indexes.nodeTags),
        loaded: true,
      }));
    } catch (_error) {
      // Handle error silently
      set({ loaded: true });
    } finally {
      set({ loading: false });
    }
  },

  fetchTags: async () => {
    if (get().loadingTags) return;

    set({ loadingTags: true });
    try {
      const { data } = await queryNodeTag();
      const tags = mergeAvailableNodeTags(data?.data?.tags, undefined);
      set((state) => ({
        tags,
        availableTags: mergeAvailableNodeTags(tags, state.nodeTags),
        loadedTags: true,
      }));
    } catch (_error) {
      // Handle error silently
      set({ loadedTags: true });
    } finally {
      set({ loadingTags: false });
    }
  },

  // Getters
  getNodeById: (nodeId: number) => get().nodeById.get(nodeId),

  isInboundUsedInNodes: (serverId: number, inboundAlias: string) =>
    get().inboundUsage.has(nodeInboundKey(serverId, inboundAlias)),

  isProtocolUsedInNodes: (serverId: number, protocolType: string) =>
    get().isInboundUsedInNodes(serverId, protocolType),

  isServerReferencedByNodes: (serverId: number) =>
    get().serverIds.has(serverId),

  getNodesByTag: (tag: string) => get().nodesByTag.get(tag) || [],

  getNodesWithoutTags: () => get().nodesWithoutTags,

  getNodeTags: () => get().nodeTags,

  getAllAvailableTags: () => get().availableTags,
}));

export const useNode = () => {
  const store = useNodeStore();

  // Auto-fetch nodes and tags
  if (!(store.loaded || store.loading)) {
    store.fetchNodes();
  }
  if (!(store.loadedTags || store.loadingTags)) {
    store.fetchTags();
  }

  return {
    nodes: store.nodes,
    tags: store.tags,
    loading: store.loading,
    loadingTags: store.loadingTags,
    loaded: store.loaded,
    loadedTags: store.loadedTags,
    fetchNodes: store.fetchNodes,
    fetchTags: store.fetchTags,
    getNodeById: store.getNodeById,
    isInboundUsedInNodes: store.isInboundUsedInNodes,
    isProtocolUsedInNodes: store.isProtocolUsedInNodes,
    isServerReferencedByNodes: store.isServerReferencedByNodes,
    getNodesByTag: store.getNodesByTag,
    getNodesWithoutTags: store.getNodesWithoutTags,
    getNodeTags: store.getNodeTags,
    getAllAvailableTags: store.getAllAvailableTags,
  };
};

export default useNodeStore;
