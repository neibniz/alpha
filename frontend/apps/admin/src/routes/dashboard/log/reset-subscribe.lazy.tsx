import { createLazyFileRoute } from "@tanstack/react-router";
import ResetSubscribeLogPage from "@/sections/log/reset-subscribe";

export const Route = createLazyFileRoute("/dashboard/log/reset-subscribe")({
  component: ResetSubscribeLogPage,
});
