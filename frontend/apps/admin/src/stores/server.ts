import { filterServerList } from "@workspace/ui/services/admin/server";
import { create } from "zustand";
import { buildByNumericId } from "./list-index";
import { fetchAllPages } from "./paginated";

const SERVER_STORE_PAGE_SIZE = 100;

interface ServerState {
  // Data
  servers: API.Server[];
  serverById: Map<number, API.Server>;

  // Loading states
  loading: boolean;
  loaded: boolean;

  // Actions
  fetchServers: () => Promise<void>;

  // Getters
  getServerById: (serverId: number) => API.Server | undefined;
  getServerName: (serverId?: number) => string;
  getServerAddress: (serverId?: number) => string;
}

export const useServerStore = create<ServerState>((set, get) => ({
  // Initial state
  servers: [],
  serverById: new Map(),
  loading: false,
  loaded: false,

  // Actions
  fetchServers: async () => {
    if (get().loading) return;

    set({ loading: true });
    try {
      const servers = await fetchAllPages<API.Server>(
        filterServerList,
        SERVER_STORE_PAGE_SIZE
      );
      set({
        servers,
        serverById: buildByNumericId(servers),
        loaded: true,
      });
    } catch (_error) {
      // Handle error silently
      set({ loaded: true });
    } finally {
      set({ loading: false });
    }
  },

  // Getters
  getServerById: (serverId: number) => get().serverById.get(serverId),

  getServerName: (serverId?: number) => {
    if (!serverId) return "—";
    const server = get().serverById.get(serverId);
    return server?.name ?? `#${serverId}`;
  },

  getServerAddress: (serverId?: number) => {
    if (!serverId) return "—";
    const server = get().serverById.get(serverId);
    return server?.address ?? "—";
  },
}));

export const useServer = () => {
  const store = useServerStore();

  // Auto-fetch servers
  if (!(store.loaded || store.loading)) {
    store.fetchServers();
  }

  return {
    servers: store.servers,
    loading: store.loading,
    loaded: store.loaded,
    fetchServers: store.fetchServers,
    getServerById: store.getServerById,
    getServerName: store.getServerName,
    getServerAddress: store.getServerAddress,
  };
};

export default useServerStore;
