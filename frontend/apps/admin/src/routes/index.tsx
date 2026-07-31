import { createFileRoute, redirect } from "@tanstack/react-router";
import { loadAdminSession } from "@/lib/admin-bootstrap";

export const Route = createFileRoute("/")({
  beforeLoad: async ({ context }) => {
    const user = await loadAdminSession(context.queryClient);
    if (user) {
      throw redirect({ to: "/dashboard" });
    }
  },
});
