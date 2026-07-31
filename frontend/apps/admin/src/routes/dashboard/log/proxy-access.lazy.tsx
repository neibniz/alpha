import { createLazyFileRoute } from "@tanstack/react-router";
import ProxyAccessLogPage from "@/sections/log/proxy-access";

export const Route = createLazyFileRoute("/dashboard/log/proxy-access")({
  component: ProxyAccessLogPage,
});
