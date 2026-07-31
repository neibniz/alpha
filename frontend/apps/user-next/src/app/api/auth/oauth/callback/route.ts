import { cookies } from "next/headers";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import {
  authCookieMaxAge,
  authCookieName,
  authCookieSecure,
} from "@/lib/runtime";
import { isSameOriginRequest, postUpstreamJson } from "@/lib/server/upstream";

type CallbackBody = {
  code?: string;
  state?: string;
};

type OAuthTokenResponse = {
  code?: number;
  data?: { token?: string };
  message?: string;
};

export async function POST(request: NextRequest) {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json(
      { message: "Invalid request origin" },
      { status: 403 }
    );
  }
  const body = (await request.json()) as CallbackBody;
  if (!(body.code && body.state)) {
    return NextResponse.json(
      { message: "GitHub 回调参数不完整" },
      { status: 400 }
    );
  }

  try {
    const { payload } = await postUpstreamJson<OAuthTokenResponse>(
      request,
      "v1/auth/oauth/login/token",
      {
        callback: { code: body.code, state: body.state },
        method: "github",
      }
    );
    const token = payload.data?.token;
    if (!token || (payload.code !== 0 && payload.code !== 200)) {
      return NextResponse.json(
        { message: payload.message || "GitHub 登录失败" },
        { status: 401 }
      );
    }
    const cookieStore = await cookies();
    cookieStore.set(authCookieName(), token, {
      httpOnly: true,
      maxAge: authCookieMaxAge(),
      path: "/",
      sameSite: "lax",
      secure: authCookieSecure(),
    });
    return NextResponse.json({ authenticated: true });
  } catch {
    return NextResponse.json(
      { message: "GitHub 登录服务暂时不可用" },
      { status: 502 }
    );
  }
}
