import { createLazyFileRoute } from "@tanstack/react-router";
import ServerTrafficLogPage from "@/sections/log/server-traffic";

export const Route = createLazyFileRoute("/dashboard/log/server-traffic")({
  component: ServerTrafficLogPage,
});
