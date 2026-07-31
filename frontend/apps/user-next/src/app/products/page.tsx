import type { Metadata } from "next";
import { cookies } from "next/headers";
import { AppShell } from "@/components/app-shell";
import { PageReveal } from "@/components/page-reveal";
import { ProductStorefront } from "@/components/products/product-storefront";
import { authCookieName, isServerDemoMode } from "@/lib/runtime";
import {
  getServerCurrentUser,
  getServerMembershipCard,
  getServerProducts,
  getServerSiteConfig,
  getServerSubscriptions,
} from "@/lib/server/api";
import {
  ServerHydration,
  type ServerPrefetchQuery,
} from "@/lib/server/hydration";

export const metadata: Metadata = { title: "购买套餐" };

export default async function ProductsPage() {
  const cookieStore = await cookies();
  const authenticated =
    isServerDemoMode() || Boolean(cookieStore.get(authCookieName())?.value);
  const queries: ServerPrefetchQuery[] = [
    {
      queryFn: () => getServerProducts(authenticated),
      queryKey: ["alpha-products", authenticated ? "user" : "guest"],
      staleTime: authenticated ? 15_000 : 60_000,
    },
    {
      queryFn: getServerSiteConfig,
      queryKey: ["alpha-site-config"],
      staleTime: 300_000,
    },
  ];
  if (authenticated) {
    queries.push(
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
      }
    );
  }
  return (
    <ServerHydration queries={queries}>
      <AppShell authenticated={authenticated}>
        <PageReveal className="min-w-0">
          <ProductStorefront authenticated={authenticated} />
        </PageReveal>
      </AppShell>
    </ServerHydration>
  );
}
