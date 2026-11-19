import type { QueryClient } from "@tanstack/react-query";

export async function requestLogout(
  request: (url: string, options: RequestInit) => Promise<Response> = fetch
) {
  const response = await request("/api/auth/logout", { method: "POST" });
  if (!response.ok) throw new Error("Sign-out request failed");
}

export function replaceClientSession(
  queryClient: QueryClient,
  destination: "/dashboard" | "/login",
  navigate = (path: string) => window.location.replace(path)
) {
  queryClient.clear();
  // A new document also discards Next.js's router cache from the old session.
  navigate(destination);
}
