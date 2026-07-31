import type { NextRequest } from "next/server";
import { proxySubscriptionRequest } from "@/lib/server/upstream";
import { normalizeSubscriptionProxyPath } from "@/lib/subscription-proxy";

export async function GET(request: NextRequest) {
  try {
    return await proxySubscriptionRequest(
      request,
      normalizeSubscriptionProxyPath(process.env.ALPHA_SUBSCRIPTION_PROXY_PATH)
    );
  } catch {
    return new Response("Upstream subscription service unavailable", {
      headers: {
        "Cache-Control": "no-store",
        "Content-Type": "text/plain; charset=utf-8",
      },
      status: 502,
    });
  }
}
