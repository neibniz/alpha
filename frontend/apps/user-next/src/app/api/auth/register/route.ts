import { cookies } from "next/headers";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import {
  authCookieMaxAge,
  authCookieName,
  authCookieSecure,
} from "@/lib/runtime";
import { isSameOriginRequest, postUpstreamJson } from "@/lib/server/upstream";

type RegisterPayload = {
  code?: string;
  email?: string;
  invite?: string;
  password?: string;
};

type RegisterResponse = {
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
  const body = (await request.json()) as RegisterPayload;
  const email = body.email?.trim();
  if (
    !(email && body.password) ||
    body.password.length < 12 ||
    body.password.length > 72
  ) {
    return NextResponse.json(
      { message: "请输入有效邮箱和 12 至 72 位密码" },
      { status: 400 }
    );
  }

  try {
    const { payload } = await postUpstreamJson<RegisterResponse>(
      request,
      "v1/auth/register",
      {
        code: body.code,
        email,
        identifier: "",
        invite: body.invite?.trim() || undefined,
        password: body.password,
      }
    );
    const token = payload.data?.token;
    if (!token || (payload.code !== 0 && payload.code !== 200)) {
      return NextResponse.json(
        { message: payload.message || "注册失败" },
        { status: 400 }
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
      { message: "注册服务暂时不可用" },
      { status: 502 }
    );
  }
}
