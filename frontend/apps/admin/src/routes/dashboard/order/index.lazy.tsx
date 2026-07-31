import { createLazyFileRoute } from "@tanstack/react-router";
import Order from "@/sections/order";

export const Route = createLazyFileRoute("/dashboard/order/")({
  component: Order,
});
