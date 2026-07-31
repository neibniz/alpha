import "server-only";

import { cookies, headers as nextHeaders } from "next/headers";
import { cache } from "react";
import {
  type ApiRequester,
  getAnnouncementsWith,
  getCurrentUserWith,
  getDocumentsWith,
  getOAuthMethodsWith,
  getOrders,
  getPaymentMethodsWith,
  getProducts,
  getSiteConfigWith,
  getSubscriptionsWith,
  getSubscriptionScopesWith,
  getTicketsWith,
  getWalletLogsWith,
} from "@/lib/client-api";
import { authCookieName } from "@/lib/runtime";
import { upstreamRequestHeaders } from "@/lib/server/proxy-headers";
import { buildUpstreamUrl } from "@/lib/server/upstream";

type ApiEnvelope<T> = {
  code?: number;
  data?: T;
  message?: string;
  msg?: string;
};

const AUTH_ERROR_CODES = new Set([40_002, 40_003, 40_004]);

export class ServerApiError extends Error {
  code?: number;
  status: number;

  constructor(message: string, status: number, code?: number) {
    super(message);
    this.name = "ServerApiError";
    this.code = code;
    this.status = status;
  }
}

export function isServerAuthError(error: unknown) {
  return (
    error instanceof ServerApiError &&
    (error.status === 401 ||
      (error.code !== undefined && AUTH_ERROR_CODES.has(error.code)))
  );
}

async function requestUpstream<T>(
  path: string,
  init: RequestInit | undefined,
  options: {
    authenticated: boolean;
    revalidate?: number;
    tags?: string[];
  }
) {
  let upstreamHeaders = new Headers(init?.headers);
  upstreamHeaders.set("Accept", "application/json");

  if (options.authenticated) {
    const [cookieStore, requestHeaders] = await Promise.all([
      cookies(),
      nextHeaders(),
    ]);
    const token = cookieStore.get(authCookieName())?.value;
    if (!token) throw new ServerApiError("Authentication required", 401);
    upstreamHeaders = upstreamRequestHeaders(
      requestHeaders,
      upstreamHeaders,
      process.env.ALPHA_TRUST_PROXY_HEADERS === "true"
    );
    upstreamHeaders.set("Authorization", token);
  }

  const response = await fetch(buildUpstreamUrl(path), {
    ...init,
    cache: options.authenticated ? "no-store" : "force-cache",
    headers: upstreamHeaders,
    next: options.authenticated
      ? undefined
      : { revalidate: options.revalidate ?? 60, tags: options.tags },
    signal: init?.signal || AbortSignal.timeout(8000),
  });
  const raw = await response.text();
  let payload: ApiEnvelope<T> = {};
  try {
    payload = raw ? (JSON.parse(raw) as ApiEnvelope<T>) : {};
  } catch {
    throw new ServerApiError("Invalid upstream response", response.status);
  }
  if (
    !response.ok ||
    (payload.code !== undefined && payload.code !== 0 && payload.code !== 200)
  ) {
    throw new ServerApiError(
      payload.message || payload.msg || "Upstream request failed",
      response.status,
      payload.code
    );
  }
  return payload.data;
}

const authenticatedRequest: ApiRequester = <T>(
  path: string,
  init?: RequestInit
) => requestUpstream<T>(path, init, { authenticated: true });

const siteConfigRequest: ApiRequester = <T>(path: string, init?: RequestInit) =>
  requestUpstream<T>(path, init, {
    authenticated: false,
    revalidate: 300,
    tags: ["site-config"],
  });

const publicProductsRequest: ApiRequester = <T>(
  path: string,
  init?: RequestInit
) =>
  requestUpstream<T>(path, init, {
    authenticated: false,
    revalidate: 60,
    tags: ["products"],
  });

export const getServerCurrentUser = cache(() =>
  getCurrentUserWith(authenticatedRequest)
);

export const getServerSubscriptions = cache(() =>
  getSubscriptionsWith(authenticatedRequest)
);

export const getServerSubscriptionScopes = cache(() =>
  getSubscriptionScopesWith(authenticatedRequest)
);

export const getServerSiteConfig = cache(() =>
  getSiteConfigWith(siteConfigRequest)
);

export const getServerProducts = cache((authenticated: boolean) =>
  getProducts(
    { authenticated },
    authenticated ? authenticatedRequest : publicProductsRequest
  )
);

export const getServerOAuthMethods = cache(() =>
  getOAuthMethodsWith(authenticatedRequest)
);

export const getServerWalletLogs = cache(() =>
  getWalletLogsWith(authenticatedRequest)
);

export const getServerPaymentMethods = cache(() =>
  getPaymentMethodsWith(authenticatedRequest)
);

export const getServerInitialOrders = cache(() =>
  getOrders({ page: 1, search: "", size: 20 }, authenticatedRequest)
);

export const getServerTickets = cache(() =>
  getTicketsWith(authenticatedRequest)
);

export const getServerDocuments = cache(() =>
  getDocumentsWith(authenticatedRequest)
);

export const getServerAnnouncements = cache(() =>
  getAnnouncementsWith(authenticatedRequest)
);
