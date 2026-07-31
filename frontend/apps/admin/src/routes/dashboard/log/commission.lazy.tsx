import { createLazyFileRoute } from "@tanstack/react-router";
import CommissionLogPage from "@/sections/log/commission";

export const Route = createLazyFileRoute("/dashboard/log/commission")({
  component: CommissionLogPage,
});
