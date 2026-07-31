import "server-only";

import {
  dehydrate,
  HydrationBoundary,
  QueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import type { ReactNode } from "react";

export type ServerPrefetchQuery = {
  queryFn: () => Promise<unknown>;
  queryKey: QueryKey;
  staleTime?: number;
};

export async function ServerHydration({
  children,
  queries,
}: Readonly<{
  children: ReactNode;
  queries: ServerPrefetchQuery[];
}>) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: 15_000 },
    },
  });

  await Promise.all(queries.map((query) => queryClient.prefetchQuery(query)));

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      {children}
    </HydrationBoundary>
  );
}
