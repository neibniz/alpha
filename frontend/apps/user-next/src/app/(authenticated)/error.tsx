"use client";

import { RouteError } from "@/components/route-error";

export default function AuthenticatedError(
  props: Parameters<typeof RouteError>[0]
) {
  return <RouteError {...props} />;
}
