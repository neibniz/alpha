import type { NextRequest } from "next/server";
import { proxyPublicRequest } from "@/lib/server/upstream";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

async function handler(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  try {
    return await proxyPublicRequest(request, path);
  } catch {
    return Response.json(
      { code: 502, message: "Upstream service unavailable" },
      { status: 502 }
    );
  }
}

export const GET = handler;
export const POST = handler;
export const PUT = handler;
export const PATCH = handler;
export const DELETE = handler;
