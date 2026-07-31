import { createLazyFileRoute } from "@tanstack/react-router";
import Servers from "@/sections/servers";

export const Route = createLazyFileRoute("/dashboard/servers")({
  component: Servers,
});
