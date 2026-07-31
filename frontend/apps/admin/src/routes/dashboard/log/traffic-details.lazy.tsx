import { createLazyFileRoute } from "@tanstack/react-router";
import TrafficDetailsPage from "@/sections/log/traffic-details";

export const Route = createLazyFileRoute("/dashboard/log/traffic-details")({
  component: TrafficDetailsPage,
});
