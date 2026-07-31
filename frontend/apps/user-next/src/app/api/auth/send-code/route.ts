import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { isSameOriginRequest, postUpstreamJson } from "@/lib/server/upstream";

type CodeResponse = {
  code?: number;
  data?: { status?: boolean };
  message?: string;
};

export async function POST(request: NextRequest) {
  if (!isSameOriginRequest(request)) {
    return NextResponse.json(
      { message: "Invalid request origin" },
      { status: 403 }
    );
  }
  const body = (await request.json()) as { email?: string; type?: number };
  if (!(body.email?.trim() && [1, 2].includes(body.type || 0))) {
    return NextResponse.json(
      { message: "验证码请求参数无效" },
      { status: 400 }
    );
  }
  try {
    const { payload } = await postUpstreamJson<CodeResponse>(
      request,
      "v1/common/send_code",
      {
        email: body.email.trim(),
        type: body.type,
      }
    );
    if (
      (payload.code !== 0 && payload.code !== 200) ||
      payload.data?.status === false
    ) {
      return NextResponse.json(
        { message: payload.message || "验证码发送失败" },
        { status: 400 }
      );
    }
    return NextResponse.json({ sent: true });
  } catch {
    return NextResponse.json(
      { message: "验证码服务暂时不可用" },
      { status: 502 }
    );
  }
}
