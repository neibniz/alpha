import { createLazyFileRoute } from "@tanstack/react-router";
import Dashboard from "@/sections/dashboard";

export const Route = createLazyFileRoute("/dashboard/")({
  component: Dashboard,
});
