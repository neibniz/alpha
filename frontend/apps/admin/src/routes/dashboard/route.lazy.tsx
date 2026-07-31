import { createLazyFileRoute } from "@tanstack/react-router";
import DashboardLayout from "@/layout";

export const Route = createLazyFileRoute("/dashboard")({
  component: DashboardLayout,
});
