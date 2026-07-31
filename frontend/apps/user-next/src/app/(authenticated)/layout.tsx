import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";
import { AppShell } from "@/components/app-shell";
import { authCookieName, isServerDemoMode } from "@/lib/runtime";
import {
  getServerCurrentUser,
  getServerMembershipCard,
  getServerSubscriptions,
  isServerAuthError,
} from "@/lib/server/api";
import { ServerHydration } from "@/lib/server/hydration";

export default async function AuthenticatedLayout({
  children,
}: Readonly<{ children: ReactNode }>) {
  if (!isServerDemoMode()) {
    const cookieStore = await cookies();
    if (!cookieStore.get(authCookieName())?.value) redirect("/login");
    try {
      await getServerCurrentUser();
    } catch (error) {
      if (isServerAuthError(error)) redirect("/login?reason=session-expired");
    }
  }

  return (
    <ServerHydration
      queries={[
        {
          queryFn: getServerCurrentUser,
          queryKey: ["alpha-current-user"],
          staleTime: 30_000,
        },
        {
          queryFn: getServerSubscriptions,
          queryKey: ["alpha-subscriptions"],
          staleTime: 30_000,
        },
        {
          queryFn: getServerMembershipCard,
          queryKey: ["alpha-membership-card"],
          staleTime: 30_000,
        },
      ]}
    >
      <AppShell>{children}</AppShell>
    </ServerHydration>
  );
}
