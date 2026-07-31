import { createLazyFileRoute } from "@tanstack/react-router";
import SubscribeTrafficLogPage from "@/sections/log/subscribe-traffic";

export const Route = createLazyFileRoute("/dashboard/log/subscribe-traffic")({
  component: SubscribeTrafficLogPage,
});
