import { createLazyFileRoute } from "@tanstack/react-router";
import Payment from "@/sections/payment";

export const Route = createLazyFileRoute("/dashboard/payment/")({
  component: Payment,
});
