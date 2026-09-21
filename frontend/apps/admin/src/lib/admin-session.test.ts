/** @vitest-environment jsdom */
import { QueryClient } from "@tanstack/react-query";
import request from "@workspace/ui/lib/request";
import { AxiosError, AxiosHeaders, type AxiosResponse } from "axios";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { adminSessionQueryKey, loadAdminSession } from "./admin-bootstrap";
import { useGlobalStore } from "@/stores/global";

const auth = vi.hoisted(() => ({
  getAuthorizationToken: vi.fn(() => "expired-test-token"),
  removeAuthorizationToken: vi.fn(),
  clearLegacyAuthorizationToken: vi.fn(),
}));
vi.mock("@workspace/ui/lib/auth-token", () => auth);
const originalAdapter = request.defaults.adapter;
let client: QueryClient;
beforeEach(() => {
  vi.clearAllMocks();
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  window.logout = vi.fn();
  useGlobalStore.getState().setUser({ id: 1 } as API.User);
});
afterEach(() => {
  request.defaults.adapter = originalAdapter;
  client.clear();
  useGlobalStore.getState().setUser(undefined);
});

test.each([
  40_002, 40_003, 40_004, 40_005,
])("an old credential with business code %i resolves to logged out, not a route error", async (code) => {
  request.defaults.adapter = async (config) => ({
    config,
    data: { code },
    status: 200,
    statusText: "OK",
    headers: new AxiosHeaders(),
  });
  await expect(loadAdminSession(client)).resolves.toBeUndefined();
  expect(auth.removeAuthorizationToken).toHaveBeenCalledOnce();
  expect(auth.clearLegacyAuthorizationToken).toHaveBeenCalledOnce();
  expect(useGlobalStore.getState().user).toBeUndefined();
  expect(client.getQueryState(adminSessionQueryKey)).toBeUndefined();
});

test.each([401, 403])("HTTP %i clears an invalid session", async (status) => {
  request.defaults.adapter = async (config) => {
    const response: AxiosResponse = {
      config,
      status,
      statusText: "Unauthorized",
      data: {},
      headers: new AxiosHeaders(),
    };
    throw new AxiosError(
      "Unauthorized",
      "ERR_BAD_REQUEST",
      config,
      undefined,
      response
    );
  };
  await expect(loadAdminSession(client)).resolves.toBeUndefined();
  expect(auth.removeAuthorizationToken).toHaveBeenCalledOnce();
});

test.each([
  401, 500, 10_001,
])("business error %i is not treated as an expired credential", async (code) => {
  request.defaults.adapter = async (config) => ({
    config,
    data: { code },
    status: 200,
    statusText: "OK",
    headers: new AxiosHeaders(),
  });
  await expect(loadAdminSession(client)).rejects.toMatchObject({
    data: { code },
  });
  expect(auth.removeAuthorizationToken).not.toHaveBeenCalled();
  expect(window.logout).not.toHaveBeenCalled();
});
