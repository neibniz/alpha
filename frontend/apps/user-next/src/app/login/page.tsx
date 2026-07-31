import type { Metadata } from "next";
import { AuthScreen } from "@/components/auth/auth-screen";
import { getServerSiteConfig } from "@/lib/server/api";
import { ServerHydration } from "@/lib/server/hydration";

export const metadata: Metadata = { title: "登录" };
export const dynamic = "force-dynamic";

export default function LoginPage() {
  return (
    <ServerHydration
      queries={[
        {
          queryFn: getServerSiteConfig,
          queryKey: ["alpha-site-config"],
          staleTime: 300_000,
        },
      ]}
    >
      <AuthScreen />
    </ServerHydration>
  );
}
