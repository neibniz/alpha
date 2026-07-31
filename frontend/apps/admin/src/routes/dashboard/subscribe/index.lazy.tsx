import { createLazyFileRoute } from "@tanstack/react-router";
import Subscribe from "@/sections/subscribe";

export const Route = createLazyFileRoute("/dashboard/subscribe/")({
  component: Subscribe,
});
