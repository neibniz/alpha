import { cookies } from "next/headers";
import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import {
  authCookieMaxAge,
  authCookieName,
  authCookieSecure,
} from "@/lib/runtime";
import { isSameOriginRequest, postUpstreamJson } from "@/lib/server/upstream";

type ResetPayload = {
  code?: string;
  email?: string;
  password?: string;
};

type ResetResponse = {
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
  const body = (await request.json()) as ResetPayload;
  if (
    !(body.email?.trim() && body.password) ||
    body.password.length < 12 ||
    body.password.length > 72 ||
    !body.code?.trim()
  ) {
    return NextResponse.json(
      { message: "请完整填写邮箱、验证码和新密码" },
      { status: 400 }
    );
  }
  try {
    const { payload } = await postUpstreamJson<ResetResponse>(
      request,
      "v1/auth/reset",
      {
        code: body.code.trim(),
        email: body.email.trim(),
        identifier: "",
        password: body.password,
      }
    );
    if (payload.code !== 0 && payload.code !== 200) {
      return NextResponse.json(
        { message: payload.message || "重置密码失败" },
        { status: 400 }
      );
    }
    if (payload.data?.token) {
      const cookieStore = await cookies();
      cookieStore.set(authCookieName(), payload.data.token, {
        httpOnly: true,
        maxAge: authCookieMaxAge(),
        path: "/",
        sameSite: "lax",
        secure: authCookieSecure(),
      });
    }
    return NextResponse.json({
      authenticated: Boolean(payload.data?.token),
      reset: true,
    });
  } catch {
    return NextResponse.json(
      { message: "密码重置服务暂时不可用" },
      { status: 502 }
    );
  }
}
