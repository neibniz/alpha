import { expect, test } from "bun:test";
import { QueryClient } from "@tanstack/react-query";
import { replaceClientSession, requestLogout } from "./client-session";

test("throws on a failed logout response", async () => {
  const failed = await requestLogout(async (_url, options) => {
    expect(options.method).toBe("POST");
    return new Response(null, { status: 403 });
  }).then(
    () => false,
    () => true
  );
  expect(failed).toBe(true);
});

test("clears private queries before navigating into a new session", () => {
  const queryClient = new QueryClient();
  queryClient.setQueryData(["alpha-orders"], ["previous-account"]);
  queryClient.setQueryData(["alpha-subscriptions"], ["previous-account"]);
  let destination = "";
  replaceClientSession(queryClient, "/dashboard", (path) => {
    expect(queryClient.getQueryCache().getAll()).toEqual([]);
    destination = path;
  });
  expect(destination).toBe("/dashboard");
});
