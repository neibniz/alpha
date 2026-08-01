import type { Metadata } from "next";
import { OAuthBindCallback } from "@/components/auth/oauth-bind-callback";

export const metadata: Metadata = { title: "连接 GitHub" };

export default async function GitHubBindCallbackPage({
  searchParams,
}: Readonly<{ searchParams: Promise<{ code?: string; state?: string }> }>) {
  const params = await searchParams;
  return <OAuthBindCallback code={params.code} state={params.state} />;
}
