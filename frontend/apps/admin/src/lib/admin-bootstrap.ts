import type { QueryClient } from "@tanstack/react-query";
import { queryOptions } from "@tanstack/react-query";
import {
  clearLegacyAuthorizationToken,
  getAuthorizationToken,
  removeAuthorizationToken,
} from "@workspace/ui/lib/auth-token";
import { currentUser } from "@workspace/ui/services/admin/user";
import { getGlobalConfig } from "@workspace/ui/services/common/common";
import { useGlobalStore } from "@/stores/global";

export const adminSessionQueryKey = ["admin", "session"] as const;

export const siteConfigQueryOptions = queryOptions({
  queryKey: ["admin", "bootstrap", "site-config"] as const,
  queryFn: async () => {
    const response = await getGlobalConfig({ skipErrorHandler: true });
    const config = response.data.data;
    if (!config) {
      throw new Error("Site configuration is unavailable");
    }
    return config;
  },
  gcTime: 30 * 60_000,
  retry: 1,
  staleTime: 5 * 60_000,
});

export const adminSessionQueryOptions = queryOptions({
  queryKey: adminSessionQueryKey,
  queryFn: async () => {
    const response = await currentUser({ skipErrorHandler: true });
    const user = response.data.data;
    if (!user) {
      throw new Error("Administrator session is unavailable");
    }
    return user;
  },
  gcTime: 5 * 60_000,
  retry: false,
  staleTime: 30_000,
});

function numericField(value: unknown, key: string): number | undefined {
  if (!value || typeof value !== "object" || !(key in value)) return;
  const field = (value as Record<string, unknown>)[key];
  return typeof field === "number" ? field : undefined;
}

export function isUnauthorizedSessionError(error: unknown) {
  const directStatus = numericField(error, "status");
  const directCode = numericField(
    error && typeof error === "object"
      ? (error as Record<string, unknown>).data
      : undefined,
    "code"
  );
  const response =
    error && typeof error === "object"
      ? (error as Record<string, unknown>).response
      : undefined;
  const responseStatus = numericField(response, "status");
  const responseCode = numericField(
    response && typeof response === "object"
      ? (response as Record<string, unknown>).data
      : undefined,
    "code"
  );

  // HTTP statuses and application codes use different namespaces: code 401 is rate limiting.
  return (
    [directStatus, responseStatus].some(
      (value) => value === 401 || value === 403
    ) ||
    [directCode, responseCode].some(
      (value) =>
        value !== undefined && [40_002, 40_003, 40_004, 40_005].includes(value)
    )
  );
}

export function clearAdminSession(queryClient: QueryClient) {
  removeAuthorizationToken();
  clearLegacyAuthorizationToken();
  queryClient.removeQueries({ queryKey: adminSessionQueryKey });
  useGlobalStore.getState().setUser(undefined);
}

export async function loadAdminSession(queryClient: QueryClient) {
  if (!getAuthorizationToken()) return;

  try {
    const user = await queryClient.ensureQueryData(adminSessionQueryOptions);
    useGlobalStore.getState().setUser(user);
    return user;
  } catch (error) {
    if (!isUnauthorizedSessionError(error)) throw error;
    clearAdminSession(queryClient);
    return;
  }
}

export async function refreshAdminSession(queryClient: QueryClient) {
  queryClient.removeQueries({ queryKey: adminSessionQueryKey });
  const user = await queryClient.fetchQuery(adminSessionQueryOptions);
  useGlobalStore.getState().setUser(user);
  return user;
}
