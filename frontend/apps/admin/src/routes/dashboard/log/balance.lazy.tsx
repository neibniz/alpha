import { createLazyFileRoute } from "@tanstack/react-router";
import BalanceLogPage from "@/sections/log/balance";

export const Route = createLazyFileRoute("/dashboard/log/balance")({
  component: BalanceLogPage,
});
