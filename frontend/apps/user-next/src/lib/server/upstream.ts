import "server-only";

import type { NextRequest } from "next/server";
import { authCookieName, authCookieSecure } from "@/lib/runtime";
import { upstreamRequestHeaders } from "@/lib/server/proxy-headers";
import { buildUpstreamPath } from "@/lib/upstream-url";

const MUTATING_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);
const MAX_PUBLIC_PROXY_BODY_BYTES = 10 * 1024 * 1024;

function trustProxyHeaders() {
  return process.env.ALPHA_TRUST_PROXY_HEADERS === "true";
}

function upstreamBase() {
  const configured = process.env.ALPHA_UPSTREAM_URL?.trim();
  const value = configured || "http://127.0.0.1:8080";
  const url = new URL(value);
  if (url.protocol !== "http:" && url.protocol !== "https:") {
    throw new Error("ALPHA_UPSTREAM_URL must use http or https");
  }
  return url;
}

export function buildUpstreamUrl(path: string, search = "") {
  const prefix = process.env.ALPHA_UPSTREAM_API_PREFIX || "";
  const url = new URL(buildUpstreamPath(path, prefix), upstreamBase());
  url.search = search;
  return url;
}

export async function proxySubscriptionRequest(
  request: NextRequest,
  upstreamPath = "/v1/subscribe/config"
) {
  const headers = upstreamRequestHeaders(
    request.headers,
    { Accept: request.headers.get("accept") || "*/*" },
    trustProxyHeaders()
  );

  const upstreamResponse = await fetch(
    buildUpstreamUrl(upstreamPath, request.nextUrl.search),
    {
      cache: "no-store",
      headers,
      method: "GET",
      redirect: "manual",
    }
  );
  const responseHeaders = new Headers({
    "Cache-Control": "private, no-store",
    "Content-Type":
      upstreamResponse.headers.get("content-type") ||
      "text/plain; charset=utf-8",
    "X-Content-Type-Options": "nosniff",
  });
  for (const name of [
    "content-disposition",
    "etag",
    "profile-update-interval",
    "subscription-userinfo",
  ]) {
    const value = upstreamResponse.headers.get(name);
    if (value) responseHeaders.set(name, value);
  }
  return new Response(await upstreamResponse.arrayBuffer(), {
    headers: responseHeaders,
    status: upstreamResponse.status,
  });
}

export function publicOrigin(request: NextRequest) {
  const configured = process.env.ALPHA_PUBLIC_ORIGIN?.trim();
  if (configured) {
    const url = new URL(configured);
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      throw new Error("ALPHA_PUBLIC_ORIGIN must use http or https");
    }
    return url.origin;
  }
  const forwardedHost = request.headers
    .get("x-forwarded-host")
    ?.split(",")[0]
    ?.trim();
  const forwardedProto = request.headers
    .get("x-forwarded-proto")
    ?.split(",")[0]
    ?.trim();
  if (forwardedHost && trustProxyHeaders()) {
    return `${forwardedProto || "https"}://${forwardedHost}`;
  }
  return request.nextUrl.origin;
}

export function isSameOriginRequest(request: NextRequest) {
  if (!MUTATING_METHODS.has(request.method)) return true;
  const source =
    request.headers.get("origin") || request.headers.get("referer");
  if (!source) return false;
  try {
    return new URL(source).origin === publicOrigin(request);
  } catch {
    return false;
  }
}

export async function postUpstreamJson<T>(
  request: NextRequest,
  path: string,
  body: unknown
) {
  const response = await fetch(buildUpstreamUrl(path), {
    body: JSON.stringify(body),
    cache: "no-store",
    headers: upstreamRequestHeaders(
      request.headers,
      {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      trustProxyHeaders()
    ),
    method: "POST",
  });
  const payload = (await response.json()) as T;
  return { payload, response };
}

export async function proxyPublicRequest(
  request: NextRequest,
  pathSegments: readonly string[]
) {
  if (!isSameOriginRequest(request)) {
    return Response.json(
      { code: 403, message: "Invalid request origin" },
      { status: 403 }
    );
  }
  const isPublicApi = pathSegments[0] === "v1" && pathSegments[1] === "public";
  const isCommonConfig =
    request.method === "GET" &&
    pathSegments.join("/") === "v1/common/site/config";
  if (!(pathSegments.length && (isPublicApi || isCommonConfig))) {
    return Response.json(
      { code: 404, message: "Route not found" },
      { status: 404 }
    );
  }

  const cookie = request.cookies.get(authCookieName())?.value;
  const headers = upstreamRequestHeaders(
    request.headers,
    { Accept: "application/json" },
    trustProxyHeaders()
  );
  const contentType = request.headers.get("content-type");
  if (contentType) headers.set("Content-Type", contentType);
  if (cookie) headers.set("Authorization", cookie);

  const hasBody = !["GET", "HEAD"].includes(request.method);
  const declaredLength = Number(request.headers.get("content-length") || 0);
  if (
    Number.isFinite(declaredLength) &&
    declaredLength > MAX_PUBLIC_PROXY_BODY_BYTES
  ) {
    return Response.json(
      { code: 413, message: "Request body is too large" },
      { status: 413 }
    );
  }
  const requestBody = hasBody ? await request.arrayBuffer() : undefined;
  if (requestBody && requestBody.byteLength > MAX_PUBLIC_PROXY_BODY_BYTES) {
    return Response.json(
      { code: 413, message: "Request body is too large" },
      { status: 413 }
    );
  }
  const upstreamResponse = await fetch(
    buildUpstreamUrl(
      `${pathSegments.map(encodeURIComponent).join("/")}${
        request.nextUrl.pathname.endsWith("/") ? "/" : ""
      }`,
      request.nextUrl.search
    ),
    {
      body: requestBody,
      cache: "no-store",
      headers,
      method: request.method,
      redirect: "manual",
    }
  );

  const responseHeaders = new Headers({
    "Cache-Control": "no-store",
    "Content-Type":
      upstreamResponse.headers.get("content-type") ||
      "application/json; charset=utf-8",
    "X-Content-Type-Options": "nosniff",
  });

  const body = await upstreamResponse.arrayBuffer();
  let invalidSession = upstreamResponse.status === 401;
  if (
    cookie &&
    upstreamResponse.headers.get("content-type")?.includes("json")
  ) {
    try {
      const payload = JSON.parse(new TextDecoder().decode(body)) as {
        code?: number;
      };
      invalidSession ||= [40_002, 40_003, 40_004].includes(payload.code || 0);
    } catch {
      // Preserve non-standard upstream responses without treating them as auth failures.
    }
  }

  if (invalidSession && cookie) {
    responseHeaders.append(
      "Set-Cookie",
      `${authCookieName()}=; Path=/; HttpOnly; SameSite=Lax; Max-Age=0${
        authCookieSecure() ? "; Secure" : ""
      }`
    );
  }

  return new Response(body, {
    headers: responseHeaders,
    status: upstreamResponse.status,
  });
}
