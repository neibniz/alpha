import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { githubOAuthCallbackUrl } from "@/lib/oauth-callback";
import {
  isSameOriginRequest,
  postUpstreamJson,
  publicOrigin,
} from "@/lib/server/upstream";

type OAuthStartResponse = {
  code?: number;
  data?: { redirect?: string };
  message?: string;
};

export async function POST(request: NextRequest) {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json(
      { message: "Invalid request origin" },
      { status: 403 }
    );
  }
  try {
    const { payload } = await postUpstreamJson<OAuthStartResponse>(
      request,
      "v1/auth/oauth/login",
      {
        method: "github",
        redirect: githubOAuthCallbackUrl(publicOrigin(request), "login"),
      }
    );
    if (
      !payload.data?.redirect ||
      (payload.code !== 0 && payload.code !== 200)
    ) {
      return NextResponse.json(
        { message: payload.message || "GitHub 登录暂时不可用" },
        { status: 400 }
      );
    }
    return NextResponse.json({ redirect: payload.data.redirect });
  } catch {
    return NextResponse.json(
      { message: "GitHub 登录服务暂时不可用" },
      { status: 502 }
    );
  }
}
