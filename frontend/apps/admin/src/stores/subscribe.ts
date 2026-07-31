import { getSubscribeList } from "@workspace/ui/services/admin/subscribe";
import { create } from "zustand";
import { buildNumericIdListIndex, patchListItemById } from "./list-index";
import { fetchAllPages } from "./paginated";

const SUBSCRIBE_STORE_PAGE_SIZE = 100;

interface SubscribeState {
  // Data
  subscribes: API.SubscribeItem[];
  subscribeById: Map<number, API.SubscribeItem>;
  subscribeIndexById: Map<number, number>;

  // Loading states
  loading: boolean;
  loaded: boolean;

  // Actions
  fetchSubscribes: () => Promise<void>;
  patchSubscribe: (subscribe: API.SubscribeItem) => void;

  // Getters
  getSubscribeName: (subscribeId?: number) => string;
  getSubscribeById: (subscribeId: number) => API.SubscribeItem | undefined;
}

export const useSubscribeStore = create<SubscribeState>((set, get) => ({
  // Initial state
  subscribes: [],
  subscribeById: new Map(),
  subscribeIndexById: new Map(),
  loading: false,
  loaded: false,

  // Actions
  fetchSubscribes: async () => {
    if (get().loading) return;

    set({ loading: true });
    try {
      const subscribes = await fetchAllPages<API.SubscribeItem>(
        getSubscribeList,
        SUBSCRIBE_STORE_PAGE_SIZE
      );
      const subscribeIndex = buildNumericIdListIndex(subscribes);
      set({
        subscribes,
        subscribeById: subscribeIndex.byId,
        subscribeIndexById: subscribeIndex.indexById,
        loaded: true,
      });
    } catch (_error) {
      // Handle error silently
      set({ loaded: true });
    } finally {
      set({ loading: false });
    }
  },

  patchSubscribe: (subscribe) => {
    if (!subscribe.id) return;
    set((state) => {
      const result = patchListItemById(
        state.subscribes,
        state.subscribeById,
        subscribe,
        state.subscribeIndexById
      );
      if (!result) {
        return {};
      }
      return {
        subscribes: result.items,
        subscribeById: result.byId,
        subscribeIndexById: result.indexById ?? state.subscribeIndexById,
      };
    });
  },

  // Getters
  getSubscribeName: (subscribeId?: number) => {
    if (!subscribeId) return "--";
    const subscribe = get().subscribeById.get(subscribeId);
    return subscribe?.name ?? `Subscribe ${subscribeId}`;
  },

  getSubscribeById: (subscribeId: number) =>
    get().subscribeById.get(subscribeId),
}));

export const useSubscribe = () => {
  const store = useSubscribeStore();

  // Auto-fetch subscribes
  if (!(store.loaded || store.loading)) {
    store.fetchSubscribes();
  }

  return {
    subscribes: store.subscribes,
    loading: store.loading,
    loaded: store.loaded,
    fetchSubscribes: store.fetchSubscribes,
    patchSubscribe: store.patchSubscribe,
    getSubscribeName: store.getSubscribeName,
    getSubscribeById: store.getSubscribeById,
  };
};

export default useSubscribeStore;
