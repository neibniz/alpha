import type { NextRequest } from "next/server";
import { proxySubscriptionRequest } from "@/lib/server/upstream";

export async function GET(request: NextRequest) {
  try {
    return await proxySubscriptionRequest(request, "/v1/subscribe/config");
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
