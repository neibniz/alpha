import type { Metadata } from "next";
import { OAuthCallback } from "@/components/auth/oauth-callback";

export const metadata: Metadata = { title: "GitHub 登录" };

export default async function OAuthCallbackPage({
  searchParams,
}: Readonly<{ searchParams: Promise<{ code?: string; state?: string }> }>) {
  const params = await searchParams;
  return <OAuthCallback code={params.code} state={params.state} />;
}
