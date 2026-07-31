import { createLazyFileRoute } from "@tanstack/react-router";
import SubscribeLogPage from "@/sections/log/subscribe";

export const Route = createLazyFileRoute("/dashboard/log/subscribe")({
  component: SubscribeLogPage,
});
