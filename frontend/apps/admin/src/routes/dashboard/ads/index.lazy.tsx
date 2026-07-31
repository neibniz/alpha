import { createLazyFileRoute } from "@tanstack/react-router";
import Ads from "@/sections/ads";

export const Route = createLazyFileRoute("/dashboard/ads/")({
  component: Ads,
});
