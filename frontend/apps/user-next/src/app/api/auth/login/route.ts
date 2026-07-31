import { cookies } from "next/headers";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import {
  authCookieMaxAge,
  authCookieName,
  authCookieSecure,
} from "@/lib/runtime";
import { isSameOriginRequest, postUpstreamJson } from "@/lib/server/upstream";

type LoginPayload = {
  cf_token?: string;
  email?: string;
  password?: string;
};

type LoginResponse = {
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

  const body = (await request.json()) as LoginPayload;
  const email = body.email?.trim();
  const password = body.password;
  if (!(email && password) || password.length > 128) {
    return NextResponse.json(
      { message: "请输入有效的邮箱和密码" },
      { status: 400 }
    );
  }

  try {
    const { payload } = await postUpstreamJson<LoginResponse>(
      request,
      "v1/auth/login",
      {
        cf_token: body.cf_token,
        email,
        identifier: "",
        password,
      }
    );
    const token = payload.data?.token;
    if (!token || (payload.code !== 0 && payload.code !== 200)) {
      return NextResponse.json(
        { message: payload.message || "邮箱或密码错误" },
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
      { message: "登录服务暂时不可用" },
      { status: 502 }
    );
  }
}
