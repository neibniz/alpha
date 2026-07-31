import { createFileRoute, redirect } from "@tanstack/react-router";
import { loadAdminSession } from "@/lib/admin-bootstrap";
import { setRedirectUrl } from "@/utils/common";

export const Route = createFileRoute("/dashboard")({
  beforeLoad: async ({ context, location }) => {
    const user = await loadAdminSession(context.queryClient);
    if (user) return;

    setRedirectUrl(location.href);
    throw redirect({ to: "/" });
  },
});
