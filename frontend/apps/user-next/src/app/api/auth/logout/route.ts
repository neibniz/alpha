import { cookies } from "next/headers";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { authCookieName } from "@/lib/runtime";
import { isSameOriginRequest } from "@/lib/server/upstream";

export async function POST(request: NextRequest) {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json(
      { message: "Invalid request origin" },
      { status: 403 }
    );
  }
  const cookieStore = await cookies();
  cookieStore.delete(authCookieName());
  return NextResponse.json({ authenticated: false });
}
